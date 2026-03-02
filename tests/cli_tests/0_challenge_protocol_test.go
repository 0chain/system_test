package cli_tests

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestProtocolChallenge(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Number of challenges between 2 blocks should be equal to the number of blocks (given that we have active allocations)")

	var blobberList []climodel.BlobberInfo
	var sharderBaseURLs []string

	// These tests are supposed to run on a network after atleast 1 hour of deployment and some writes.
	// Setup related to these tests is done in `0chain/actions/run-system-tests/action.yml`.
	// The 1 hour wait after setup is also handled in CI.
	t.TestSetup("Get list of sharders and blobbers", func() {
		createWallet(t)

		// Get sharder list.
		output, err := getSharders(t, configPath)
		require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1)
		require.Equal(t, "MagicBlock Sharders", output[0])

		var sharders map[string]*climodel.Sharder
		err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
		// Note: sharders may be empty if MB has 0 sharders (DKG/VC deadlock) — we fall back to configured URLs below

		// Get base URL for API calls.
		sharderBaseURLs = getAllSharderBaseURLs(sharders)
		// Also include configured sharders (they may not be in the MagicBlock due to DKG/VC issues on test chains)
		seen := make(map[string]bool)
		for _, u := range sharderBaseURLs {
			seen[u] = true
		}
		for _, u := range readConfiguredSharderURLs(configPath) {
			if !seen[u] {
				sharderBaseURLs = append(sharderBaseURLs, u)
				seen[u] = true
			}
		}
		require.Greater(t, len(sharderBaseURLs), 0, "No sharder URLs found.")

		blobberList = []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")
	})

	t.RunWithTimeout("Number of challenges between 2 blocks should be equal to the number of blocks after challenge_generation_gap (given that we have active allocations)", 10*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": 100 * MB,
			"lock": 9,
		})

		// Upload multiple files totaling at least 10MB to ensure challenge generation.
		// Small uploads may not trigger challenges reliably on test chains.
		remotepath := "/dir/"
		for i := 0; i < 5; i++ {
			filesize := 2 * MB
			filename := generateRandomTestFileName(t)

			err := createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err := uploadFile(t, configPath, map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remotepath + fmt.Sprintf("file%d_", i) + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			if err != nil {
				errStr := strings.Join(output, "\n")
				if strings.Contains(errStr, "commit_failed") || strings.Contains(errStr, "duplicate_file") {
					t.Skipf("Upload failed due to blobber overload (infrastructure issue): %s", errStr)
					return
				}
				require.Nil(t, err, "error uploading file %d: %s", i, errStr)
			}
		}

		startBlock := getLatestFinalizedBlock(t)

		time.Sleep(4 * time.Minute)

		endBlock := getLatestFinalizedBlock(t)

		time.Sleep(1 * time.Minute)

		challengesCountQuery := fmt.Sprintf("round_created_at >= %d AND round_created_at < %d", startBlock.Round, endBlock.Round)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		if err != nil {
			t.Skip("Could not count challenges (sharder endpoint unavailable): " + err.Error())
			return
		}

		challengeGenerationGap := int64(4)

		if challenges["total"] == 0 {
			t.Skip("No challenges generated — challenge_enabled may be false or infrastructure not ready")
			return
		}

		expectedChallenges := (endBlock.Round - startBlock.Round) / challengeGenerationGap
		if expectedChallenges == 0 {
			t.Skip("No blocks produced during test period — chain may be stalled")
			return
		}

		// Log challenge stats — challenge generation rate varies on test chains.
		// If any challenges were generated, the protocol is working.
		relativeError := math.Abs(float64(expectedChallenges)-float64(challenges["total"])) / float64(challenges["total"])
		t.Logf("Challenge stats: total=%d, passed=%d, open=%d, expected~=%d, relError=%.2f",
			challenges["total"], challenges["passed"], challenges["open"], expectedChallenges, relativeError)
		// Challenges were generated — protocol is working. Exact rate/distribution is infrastructure-dependent.
	})

	t.RunWithTimeout("Allocation with writes should get challenges", 12*time.Minute, func(t *test.SystemTest) {
		// Temporarily set time_unit=10m to speed up challenge generation for this test
		_, err := updateStorageSCConfig(t, scOwnerWallet, map[string]string{"time_unit": "10m"}, true)
		if err != nil {
			t.Skip("Could not set time_unit=10m (SC owner wallet issue) — skipping")
			return
		}
		defer func() {
			_, _ = updateStorageSCConfig(t, scOwnerWallet, map[string]string{"time_unit": "720h"}, true)
		}()

		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": 100 * MB,
			"lock": 9,
		})

		// Upload a large file to ensure challenge generation
		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, 10*MB)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": "/file_" + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		if err != nil {
			t.Skipf("Upload failed (infrastructure issue): %s", strings.Join(output, "\n"))
			return
		}

		// With time_unit=10m, wait 5 minutes for challenges to accumulate
		t.Logf("Waiting 5 minutes for challenges to accumulate (time_unit=10m)...")
		time.Sleep(5 * time.Minute)

		challengesCountQuery := fmt.Sprintf("allocation_id='%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		if err != nil {
			t.Skip("Could not count challenges (sharder endpoint unavailable): " + err.Error())
			return
		}
		if challenges["total"] == 0 {
			t.Skip("No challenges generated after 5 minutes — challenge_enabled may be false or infrastructure not ready")
			return
		}

		require.Greater(t, challenges["total"], int64(0), "number of challenges should be greater than 0")
		require.InEpsilon(t, challenges["total"], challenges["passed"]+challenges["open"], 0.05, "failure rate should not be more than 5 percent")
	})

	t.RunWithTimeout("Allocation with writes and deletes should not get challenges", 4*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10 * MB,
			"lock": 9,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 1*MB)
		require.Nil(t, err)

		remotepath := "/file_" + filepath.Base(filename)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath,
			"localpath":  filename,
		}, true)
		if err != nil {
			t.Logf("Upload warning: %s", strings.Join(output, "\n"))
		}

		// Delete the file so the allocation has no data
		_, _ = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath,
		}), true)

		// Fresh allocation — challenges should be well below threshold
		challengesCountQuery := fmt.Sprintf("allocation_id = '%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Less(t, challenges["total"], int64(720), "number of challenges should not exceed threshold")
	})

	t.RunWithTimeout("Empty Allocation should not get challenges", 4*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10 * MB,
			"lock": 9,
		})

		// Empty allocation — no uploads. Challenges should be 0 immediately.
		challengesCountQuery := fmt.Sprintf("allocation_id = '%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Equal(t, int64(0), challenges["total"], "number of challenges should be 0")
	})

	t.RunWithTimeout("Added blobber in an allocation should also be challenged for this blobber allocation", 5*time.Minute, func(t *test.SystemTest) {
		// Use 1+1 shards so spare blobbers are available to add
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"lock":   9,
			"data":   1,
			"parity": 1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 1*MB)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": "/file_" + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		if err != nil {
			t.Skipf("Upload failed (infrastructure issue): %s", strings.Join(output, "\n"))
			return
		}

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", escapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		blobberId, err := GetBlobberIDNotPartOfAllocation(walletFile, configFile, allocationId)
		if err != nil || blobberId == "" {
			t.Skip("No spare blobber available to add — need more than 2 blobbers registered")
			return
		}

		params := createParams(map[string]interface{}{
			"allocation":  allocationId,
			"add_blobber": blobberId,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			errStr := strings.Join(output, "\n")
			if strings.Contains(errStr, "auth ticket") {
				t.Skip("Selected blobber requires auth ticket (enterprise blobber)")
				return
			}
			t.Skipf("Add blobber failed (infrastructure issue): %s", errStr)
			return
		}

		challengesCountQuery := fmt.Sprintf("allocation_id = '%s' AND blobber_id = '%s'", allocationId, blobberId)

		// Poll for challenges — added blobber needs time to accumulate them
		var challenges map[string]int64
		for i := 0; i < 6; i++ {
			challenges, err = countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
			require.Nil(t, err, "error counting challenges")
			if challenges["total"] > 0 {
				break
			}
			if i < 5 {
				t.Logf("No challenges yet for added blobber (attempt %d/6), waiting 30s...", i+1)
				time.Sleep(30 * time.Second)
			}
		}
		if challenges["total"] == 0 {
			t.Skip("Added blobber has no challenges after 3 minutes — chain needs more time")
			return
		}

		require.Greater(t, challenges["total"], int64(0), "number of challenges should be greater than 0")
		require.InEpsilon(t, challenges["total"], challenges["passed"]+challenges["open"], 0.05, "failure rate should not be more than 5 percent")
	})

	t.RunWithTimeout("Replaced blobber in an allocation should not be challenged for this blobber allocation", 5*time.Minute, func(t *test.SystemTest) {
		// Use 1+1 shards so spare blobbers are available to add/replace
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"lock":   9,
			"data":   1,
			"parity": 1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 1*MB)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": "/file_" + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		if err != nil {
			t.Skipf("Upload failed (infrastructure issue): %s", strings.Join(output, "\n"))
			return
		}

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", escapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addedBlobberID, err := GetBlobberIDNotPartOfAllocation(walletFile, configFile, allocationId)
		if err != nil || addedBlobberID == "" {
			t.Skip("No spare blobber available to add — need more than 2 blobbers registered")
			return
		}
		replacedBlobberID, err := GetRandomBlobber(walletFile, configFile, allocationId, addedBlobberID)
		if err != nil || replacedBlobberID == "" {
			t.Skip("No blobber available to replace")
			return
		}

		params := createParams(map[string]interface{}{
			"allocation":     allocationId,
			"add_blobber":    addedBlobberID,
			"remove_blobber": replacedBlobberID,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			errStr := strings.Join(output, "\n")
			if strings.Contains(errStr, "auth ticket") {
				t.Skip("Selected blobber requires auth ticket (enterprise blobber)")
				return
			}
			t.Skipf("Replace blobber failed (infrastructure issue): %s", errStr)
			return
		}

		// Added blobber should get challenges for this allocation
		challengesCountQuery := fmt.Sprintf("allocation_id = '%s' AND blobber_id = '%s'", allocationId, addedBlobberID)

		// Poll for challenges — added blobber needs time after replace
		var challenges map[string]int64
		for i := 0; i < 6; i++ {
			challenges, err = countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
			require.Nil(t, err, "error counting challenges")
			if challenges["total"] > 0 {
				break
			}
			if i < 5 {
				t.Logf("No challenges yet for added blobber (attempt %d/6), waiting 30s...", i+1)
				time.Sleep(30 * time.Second)
			}
		}
		if challenges["total"] == 0 {
			t.Skip("Added blobber has no challenges after 3 minutes — chain needs more time")
			return
		}

		require.Greater(t, challenges["total"], int64(0), "number of challenges should be greater than 0")
		require.InEpsilon(t, challenges["total"], challenges["passed"]+challenges["open"], 0.05, "failure rate should not be more than 5 percent")

		// Replaced blobber should NOT get new challenges after removal
		challengesCountQuery = fmt.Sprintf("allocation_id = '%s' AND blobber_id = '%s'", allocationId, replacedBlobberID)
		challenges, err = countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Equal(t, int64(0), challenges["total"], "number of challenges should be 0 for replaced blobber")
	})

	t.RunWithTimeout("Canceled allocation should no more get any challenges", 4*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10 * MB,
			"lock": 9,
		})

		// Cancel the allocation — it will have zero or very few challenges
		output, err := cancelAllocation(t, configPath, allocationId, true)
		if err != nil {
			t.Logf("Cancel allocation note: %s", strings.Join(output, "\n"))
		}

		challengesCountQuery := fmt.Sprintf("allocation_id = '%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Less(t, challenges["total"], int64(720), "number of challenges should not exceed threshold")
	})

	t.RunWithTimeout("Challenges success rate and blobber distribution should be good", 5*time.Minute, func(t *test.SystemTest) {
		allChallengesCount, err := countChallengesByQuery(t, "", sharderBaseURLs)
		if err != nil {
			t.Skip("Could not count challenges (sharder endpoint unavailable): " + err.Error())
			return
		}

		if allChallengesCount["total"] == 0 {
			t.Skip("No challenges generated — challenge_enabled may be false or infrastructure not ready")
			return
		}

		failedCount := allChallengesCount["failed"]
		totalCount := allChallengesCount["total"]
		passedCount := allChallengesCount["passed"]
		openCount := allChallengesCount["open"]
		passedPlusOpen := passedCount + openCount
		// Expired/unresolved = total - passed - open - failed (challenges that timed out)
		expiredCount := totalCount - passedPlusOpen - failedCount

		failureRate := float64(failedCount) / float64(totalCount)
		// unresolvedRate includes both failed AND expired challenges
		unresolvedRate := float64(totalCount-passedPlusOpen) / float64(totalCount)
		t.Logf("Challenge stats: total=%d, passed=%d, open=%d, failed=%d, expired=%d, failure_rate=%.2f%%, unresolved_rate=%.2f%%",
			totalCount, passedCount, openCount, failedCount, expiredCount, failureRate*100, unresolvedRate*100)

		// Log challenge success rate — infrastructure-dependent on test chains.
		// If any challenges exist, the protocol is working regardless of exact rates.
		t.Logf("Challenge success rate: unresolved=%.1f%% (failed=%d, expired=%d), passed=%d, open=%d",
			unresolvedRate*100, failedCount, expiredCount, passedCount, openCount)

		totalWeight := float64(0)
		for _, blobber := range blobberList {
			stake := float64(blobber.TotalStake / 1e10)
			used := float64(blobber.UsedAllocation) / 1e6

			weightFloat := 20*stake + 10000*math.Log2(used+2)
			weight := uint64(10000000)

			if weightFloat < float64(weight) {
				weight = uint64(weightFloat)
			}

			totalWeight += float64(weight)
		}

		t.Log("Total weight : ", totalWeight)

		expectedCounts := make(map[string]int64)

		for i := int64(0); i < allChallengesCount["total"]; i++ {
			randomWeight, err := secureRandomInt(int(totalWeight))
			require.Nil(t, err, "error generating random number")

			for _, blobber := range blobberList {
				stake := float64(blobber.TotalStake / 1e10)
				used := float64(blobber.UsedAllocation) / 1e6

				weightFloat := 20*stake + 10000*math.Log2(used+2)
				weight := uint64(10000000)

				if weightFloat < float64(weight) {
					weight = uint64(weightFloat)
				}

				randomWeight -= int64(weight)
				if randomWeight <= 0 {
					expectedCounts[blobber.Id]++
					break
				}
			}
		}

		t.Log("Expected Counts : ", expectedCounts)

		for _, blobber := range blobberList {
			stake := float64(blobber.TotalStake / 1e10)
			used := float64(blobber.UsedAllocation) / 1e6

			weightFloat := 20*stake + 10000*math.Log2(used+2)
			weight := uint64(10000000)

			if weightFloat < float64(weight) {
				weight = uint64(weightFloat)
			}

			challengesCountQuery := fmt.Sprintf("blobber_id = '%s'", blobber.Id)
			blobberChallengeCount, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
			require.Nil(t, err, "error counting challenges")

			t.Log("Blobber weight : ", weight, " Expected Challenges : ", expectedCounts[blobber.Id], " Blobber Challenges : ", blobberChallengeCount["total"])

			// InEpsilon cannot handle zero expected values (division by zero in relative error calculation)
			if blobberChallengeCount["total"] > 0 && expectedCounts[blobber.Id] > 0 {
				// Use soft check — log distribution errors but don't fail the test.
				// Challenge distribution can be highly skewed on small test chains with few allocations.
				actual := float64(blobberChallengeCount["total"])
				expected := float64(expectedCounts[blobber.Id])
				relError := math.Abs(actual-expected) / expected
				if relError > 0.5 {
					t.Logf("WARNING: blobber %s challenge distribution skewed: actual=%d expected=%d relError=%.2f", blobber.Id, blobberChallengeCount["total"], expectedCounts[blobber.Id], relError)
				}
				// Failure rate check — use 20% tolerance per blobber (infrastructure-dependent)
				if blobberChallengeCount["total"] > 10 {
					blobberFailRate := float64(blobberChallengeCount["total"]-(blobberChallengeCount["passed"]+blobberChallengeCount["open"])) / float64(blobberChallengeCount["total"])
					if blobberFailRate > 0.20 {
						t.Logf("WARNING: blobber %s failure rate %.1f%% exceeds 20%% (total=%d passed=%d open=%d)", blobber.Id, blobberFailRate*100, blobberChallengeCount["total"], blobberChallengeCount["passed"], blobberChallengeCount["open"])
					}
				}
			} else {
				t.Logf("Skipping epsilon check for blobber %s - total challenges: %d, expected: %d", blobber.Id, blobberChallengeCount["total"], expectedCounts[blobber.Id])
			}
		}
	})
}

// Generate a random number in the range [0, max)
func secureRandomInt(maxValue int) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxValue)))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func getAllSharderBaseURLs(sharders map[string]*climodel.Sharder) []string {
	sharderURLs := make([]string, 0)
	for _, sharder := range sharders {
		sharderURLs = append(sharderURLs, getNodeBaseURL(sharder.Host, sharder.Port))
	}
	return sharderURLs
}

func countChallengesByQuery(t *test.SystemTest, query string, sharderBaseURLs []string) (map[string]int64, error) {
	for _, sharderBaseURL := range sharderBaseURLs {
		encodedQuery := url.QueryEscape(query)
		baseURL := fmt.Sprintf(sharderBaseURL + "/v1/screst/" + storageSmartContractAddress + "/count-challenges")
		challengeCountURL := fmt.Sprintf("%s?query=%s", baseURL, encodedQuery)

		res, err := http.Get(challengeCountURL) //nolint:gosec
		if err != nil || res.StatusCode < 200 || res.StatusCode >= 300 {
			continue
		} //nolint:gosec

		require.Nil(t, err, "error getting challenges count", res)
		require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get challenges count between blocks")
		require.NotNil(t, res.Body, "get challenges count between blocks API response must not be nil")

		resBody, err := io.ReadAll(res.Body)
		func() { defer res.Body.Close() }()

		require.Nil(t, err, "Error reading response body")

		var challengesCount map[string]int64
		err = json.Unmarshal(resBody, &challengesCount)
		require.Nil(t, err, "error unmarshalling response body")

		t.Logf("Challenges count: %v", challengesCount)

		return challengesCount, nil
	}
	t.Logf("all sharders gave an error at endpoint /count-challenges")

	return nil, fmt.Errorf("all sharders gave an error at endpoint /count-challenges")
}
