package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
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
	t.TestSetupWithTimeout("Get list of sharders and blobbers", 2*time.Hour, func() {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		// Get sharder list.
		output, err = getSharders(t, configPath)
		require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1)
		require.Equal(t, "MagicBlock Sharders", output[0])

		var sharders map[string]*climodel.Sharder
		err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
		require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

		// Get base URL for API calls.
		sharderBaseURLs = getAllSharderBaseURLs(sharders)
		require.Greater(t, len(sharderBaseURLs), 0, "No sharder URLs found.")

		blobberList = []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

		for i := 0; i < 40; i++ {
			allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"data":   1,
				"parity": 1,
				"size":   10 * MB,
				"tokens": 9,
			})

			remotepath := "/dir/"
			filesize := 1 * MB
			filename := generateRandomTestFileName(t)

			err := createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err := uploadFile(t, configPath, map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
		}

		t.Log("Waiting for 1 hour to let the challenge protocol run")
		time.Sleep(30 * time.Minute)
	})

	t.RunWithTimeout("Number of challenges between 2 blocks should be equal to the number of blocks (given that we have active allocations)", 5*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		startBlock := getLatestFinalizedBlock(t)

		time.Sleep(2 * time.Minute)

		endBlock := getLatestFinalizedBlock(t)

		time.Sleep(1 * time.Minute)

		challengesCountQuery := fmt.Sprintf("round_created_at >= %d AND round_created_at < %d", startBlock.Round, endBlock.Round)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Equal(t, endBlock.Round-startBlock.Round, challenges["total"], "number of challenges should be equal to the number of blocks")
		require.InEpsilon(t, challenges["total"], challenges["passed"]+challenges["open"], 0.05, "failure rate should not be more than 5 percent")
		require.Less(t, challenges["open"], int64(720), "number of open challenges should be lesser than 720")
	})

	t.RunWithTimeout("Allocation with writes should get challenges", 4*time.Minute, func(t *test.SystemTest) {
		// read allocation id in first line of challenge_allocations.txt

		file := "challenge_allocations.txt"
		allocationId := readLineFromFile(t, file, 0)

		challengesCountQuery := fmt.Sprintf("allocation_id='%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Greater(t, challenges["total"], int64(0), "number of challenges should be greater than 0")
		require.InEpsilon(t, challenges["total"], challenges["passed"]+challenges["open"], 0.05, "failure rate should not be more than 5 percent")
	})

	t.RunWithTimeout("Allocation with writes and deletes should not get challenges", 4*time.Minute, func(t *test.SystemTest) {
		// read allocation id in second line of challenge_allocations.txt

		file := "challenge_allocations.txt"
		allocationId := readLineFromFile(t, file, 1)

		challengesCountQuery := fmt.Sprintf("allocation_id = '%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.NotEqual(t, int64(0), challenges["total"], "number of challenges should not be complete 0")
		require.Less(t, challenges["total"], int64(720), "number of challenges should not more increase after a threshold")
	})

	t.RunWithTimeout("Empty Allocation should not get challenges", 4*time.Minute, func(t *test.SystemTest) {
		// read allocation id in third line of challenge_allocations.txt

		file := "challenge_allocations.txt"
		allocationId := readLineFromFile(t, file, 2)

		challengesCountQuery := fmt.Sprintf("allocation_id = '%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Equal(t, int64(0), challenges["total"], "number of challenges should be 0")
	})

	t.RunWithTimeout("Added blobber in an allocation should also be challenged for this blobber allocation", 4*time.Minute, func(t *test.SystemTest) {
		challengeAllocationFile := "challenge_allocations.txt"
		challengeBlobberFile := "challenge_blobbers.txt"

		// read allocation id in fourth line of challenge_allocations.txt
		allocationId := readLineFromFile(t, challengeAllocationFile, 3)

		// read blobber id in first line of challenge_blobbers.txt
		blobberId := readLineFromFile(t, challengeBlobberFile, 0)

		challengesCountQuery := fmt.Sprintf("allocation_id = '%s' AND blobber_id = '%s'", allocationId, blobberId)

		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Greater(t, challenges["total"], int64(0), "number of challenges should be greater than 0")
		require.InEpsilon(t, challenges["total"], challenges["passed"]+challenges["open"], 0.05, "failure rate should not be more than 5 percent")
	})

	t.RunWithTimeout("Replaced blobber in an allocation should not be challenged for this blobber allocation", 4*time.Minute, func(t *test.SystemTest) {
		challengeAllocationFile := "challenge_allocations.txt"
		challengeBlobberFile := "challenge_blobbers.txt"

		// read allocation id in fifth line of challenge_allocations.txt
		allocationId := readLineFromFile(t, challengeAllocationFile, 4)

		// read blobber id in second line of challenge_blobbers.txt
		addedBlobberID := readLineFromFile(t, challengeBlobberFile, 1)
		replacedBlobberID := readLineFromFile(t, challengeBlobberFile, 2)

		// Added Blobber should get challenges for this allocation

		challengesCountQuery := fmt.Sprintf("allocation_id = '%s' AND blobber_id = '%s'", allocationId, addedBlobberID)

		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Greater(t, challenges["total"], int64(0), "number of challenges should be greater than 0")
		require.InEpsilon(t, challenges["total"], challenges["passed"]+challenges["open"], 0.05, "failure rate should not be more than 5 percent")

		// Replaced Blobber should not get challenges for this allocation

		challengesCountQuery = fmt.Sprintf("allocation_id = '%s' AND blobber_id = '%s'", allocationId, replacedBlobberID)

		challenges, err = countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.Equal(t, int64(0), challenges["total"], "number of challenges should be 0")
	})

	t.RunWithTimeout("Canceled allocation should no more get any challenges", 4*time.Minute, func(t *test.SystemTest) {
		// read allocation id in sixth line of challenge_allocations.txt

		file := "challenge_allocations.txt"
		allocationId := readLineFromFile(t, file, 5)

		challengesCountQuery := fmt.Sprintf("allocation_id = '%s'", allocationId)
		challenges, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.NotEqual(t, int64(0), challenges["total"], "number of challenges should not be complete 0")
		require.Less(t, challenges["total"], int64(720), "number of challenges should not more increase after a threshold")
	})

	t.RunWithTimeout("Challenges success rate and blobber distribution should be good", 5*time.Minute, func(t *test.SystemTest) {
		allChallengesCount, err := countChallengesByQuery(t, "", sharderBaseURLs)
		require.Nil(t, err, "error counting challenges")

		require.InEpsilonf(t, allChallengesCount["total"], allChallengesCount["passed"]+allChallengesCount["open"], 0.05, "Challenge Failure rate should not be more than 5%")

		totalUsed := int64(0)
		for _, blobber := range blobberList {
			totalUsed += blobber.UsedAllocation
		}

		for _, blobber := range blobberList {
			challengesCountQuery := fmt.Sprintf("blobber_id = '%s'", blobber.Id)
			blobberChallengeCount, err := countChallengesByQuery(t, challengesCountQuery, sharderBaseURLs)
			require.Nil(t, err, "error counting challenges")

			require.InEpsilon(t, (allChallengesCount["total"]*blobber.UsedAllocation)/totalUsed, blobberChallengeCount["total"], 0.15, "blobber distribution should within tolerance")
			require.InEpsilon(t, blobberChallengeCount["total"], blobberChallengeCount["passed"]+blobberChallengeCount["open"], 0.05, "failure rate should not be more than 5 percent")
		}
	})
}

func getAllSharderBaseURLs(sharders map[string]*climodel.Sharder) []string {
	sharderURLs := make([]string, 0)
	for _, sharder := range sharders {
		sharderURLs = append(sharderURLs, getNodeBaseURL(sharder.Host, sharder.Port))
	}
	return sharderURLs
}

func countChallengesByQuery(t *test.SystemTest, query string, sharderBaseURLs []string) (map[string]int64, error) {
	t.Logf("Counting challenges with query: %s", query)

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
	t.Errorf("all sharders gave an error at endpoint /count-challenges")

	return nil, nil
}

func readLineFromFile(t *test.SystemTest, file string, line int) string {
	output, err := os.ReadFile(file)
	require.Nil(t, err, "error reading file", file)

	lines := strings.Split(string(output), "\n")
	require.Greater(t, len(lines), line, "file should have at least %d lines", line)

	return lines[line]
}
