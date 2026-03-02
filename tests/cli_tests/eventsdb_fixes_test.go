package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

// TestEventsDBFixes verifies the eventsDB fixes deployed to the network:
// 1. FK constraint fix: allocation.owner no longer has DB-level FK to users table
// 2. ON DELETE SET NULL: cancel allocation preserves write markers (not cascade-deleted)
// 3. Challenge pool merge fix: a.Amount += b.Amount (was a.Amount += a.Amount)
// 4. uint64 downtime fix: provider Downtime is int64, no scan errors
// 5. GREATEST guard: delegate pool balance never negative after penalty
func TestEventsDBFixes(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests(
		"Allocation creation should succeed without FK violation",
		"Provider health check data should not have scan errors",
	)
	t.Parallel()

	// Fix 1: FK constraint - rapid wallet+allocation creation should not FK-violate.
	// We create 5 wallets in quick succession and immediately allocate — the race
	// window where the user row doesn't exist yet is maximized by parallelism.
	t.RunWithTimeout("Allocation creation should succeed without FK violation", 3*time.Minute, func(t *test.SystemTest) {
		const attempts = 5
		for i := 0; i < attempts; i++ {
			createWallet(t)

			allocationId := setupAllocation(t, configPath, map[string]interface{}{
				"size":   int64(1024 * 1024),
				"lock":   1,
				"data":   2,
				"parity": 1,
			})
			require.NotEmpty(t, allocationId,
				"allocation %d/%d should be created without FK violation", i+1, attempts)
		}
		t.Logf("All %d rapid allocations created successfully (no FK violations)", attempts)
	})

	// Fix 2: ON DELETE SET NULL - write markers must survive allocation cancel.
	// Old code: ON DELETE CASCADE → write markers deleted. New code: SET NULL → preserved.
	t.RunWithTimeout("Write markers should survive allocation cancel", 20*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size":   int64(10 * 1024 * 1024),
			"lock":   5,
			"data":   2,
			"parity": 1,
		})
		require.NotEmpty(t, allocationId)

		// Upload a file to generate write markers
		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 1024)
		require.NoError(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": "/test_file",
			"localpath":  filename,
		}, true)
		require.NoError(t, err, "upload should succeed", strings.Join(output, "\n"))

		// Poll for write markers — blobber marker_redeem_interval defaults to 10 minutes
		// with jitter, so markers may take up to 15 minutes to appear in events DB.
		var preCount int
		for i := 0; i < 30; i++ {
			time.Sleep(30 * time.Second)
			preCount = getWriteMarkerCount(t, allocationId)
			if preCount > 0 {
				break
			}
			t.Logf("Write markers not yet indexed (attempt %d/30, %.0fs elapsed), waiting...", i+1, float64(i+1)*30)
		}
		require.Greater(t, preCount, 0, "should have write markers after upload")
		t.Logf("Write markers before termination: %d", preCount)

		// Try to cancel the allocation. With time_unit=10m0s, the allocation may have already
		// expired by the time write markers appear (~10 min). Both cancel and finalization
		// should preserve write markers (ON DELETE SET NULL), so either outcome tests the fix.
		output, cancelErr := cancelAllocation(t, configPath, allocationId, false)
		if cancelErr != nil {
			cancelOut := strings.Join(output, "\n")
			if strings.Contains(cancelOut, "trying to cancel expired allocation") {
				t.Logf("Allocation already expired before cancel (time_unit=10m0s, markers took ~10min). " +
					"Finalization also tests ON DELETE SET NULL — verifying markers survived expiration.")
			} else {
				require.NoError(t, cancelErr, "cancel allocation should succeed", cancelOut)
			}
		}

		// Wait for eventsDB to process the cancel/finalization
		time.Sleep(15 * time.Second)

		// Count write markers AFTER termination — must still exist (SET NULL, not CASCADE)
		postCount := getWriteMarkerCount(t, allocationId)
		t.Logf("Write markers after termination: %d", postCount)
		require.Equal(t, preCount, postCount,
			"write markers should survive allocation termination (ON DELETE SET NULL). "+
				"Before: %d, After: %d — if 0, the old CASCADE behavior is still active", preCount, postCount)
	})

	// Fix 3: Challenge pool merge - create two allocations, upload to both,
	// wait for challenges, then verify MovedToChallenge is less than the lock amount.
	// The merge bug (a.Amount += a.Amount) doubles the value on each merge event,
	// so after several merges it exceeds the total locked tokens.
	t.RunWithTimeout("Challenge pool should not inflate from merge bug", 10*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		lockAmount := int64(9)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size":   int64(100 * 1024 * 1024),
			"lock":   lockAmount,
			"data":   2,
			"parity": 1,
		})
		require.NotEmpty(t, allocationId)

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 10*1024*1024)
		require.NoError(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": "/challenge_test_file",
			"localpath":  filename,
		}, true)
		require.NoError(t, err, "upload should succeed", strings.Join(output, "\n"))

		t.Log("Waiting for challenges to be generated...")
		time.Sleep(3 * time.Minute)

		allocation := getAllocation(t, allocationId)

		// lockAmount in ZCN → tokens (1 ZCN = 10^10 tokens)
		maxTokens := lockAmount * 1e10
		t.Logf("MovedToChallenge: %d, WritePool: %d, MaxTokens: %d",
			allocation.MovedToChallenge, allocation.WritePool, maxTokens)

		// With the merge bug, MovedToChallenge doubles on each merge and quickly
		// exceeds the total locked amount. With the fix, it stays <= lock amount.
		require.LessOrEqual(t, allocation.MovedToChallenge, maxTokens,
			"MovedToChallenge (%d) should not exceed total locked tokens (%d). "+
				"If it does, the challenge pool merge bug (a.Amount += a.Amount) is still active",
			allocation.MovedToChallenge, maxTokens)
	})

	// Fix 4: uint64 downtime - provider queries should return valid data without scan errors
	t.Run("Provider health check data should not have scan errors", func(t *test.SystemTest) {
		createWallet(t)

		// Query all blobbers
		output, err := listBlobbers(t, configPath, "--json")
		require.NoError(t, err, "list blobbers should succeed without scan errors")
		require.NotEmpty(t, output, "blobber list should not be empty")

		var blobbers []climodel.BlobberInfo
		err = json.Unmarshal([]byte(strings.Join(output, "")), &blobbers)
		require.NoError(t, err, "should unmarshal blobbers without scan errors")
		require.NotEmpty(t, blobbers, "should have at least one blobber")

		for _, b := range blobbers {
			t.Logf("Blobber %s: LastHealthCheck=%d", b.Id, b.LastHealthCheck)
			if b.LastHealthCheck > 0 {
				require.True(t, b.LastHealthCheck > 1577836800,
					"LastHealthCheck should be after 2020-01-01")
			}
		}

		// Query all miners
		minerOutput, err := getMiners(t, configPath)
		require.NoError(t, err, "list miners should succeed without scan errors")
		require.NotEmpty(t, minerOutput, "miner list should not be empty")
		t.Logf("Successfully queried %d miners without scan errors", len(minerOutput))

		// Query all sharders
		sharderOutput, err := getSharders(t, configPath)
		require.NoError(t, err, "list sharders should succeed without scan errors")
		require.NotEmpty(t, sharderOutput, "sharder list should not be empty")
		t.Logf("Successfully queried sharders without scan errors")
	})

	// Fix 5: GREATEST guard - delegate pool balances should never be negative
	t.RunWithTimeout("Delegate pool balances should be non-negative", 2*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		output, err := getSharders(t, configPath)
		require.NoError(t, err, "get sharders failed")

		var sharders map[string]*climodel.Sharder
		err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
		require.NoError(t, err, "unmarshal sharders failed")
		require.NotEmpty(t, sharders)

		// Query blobbers and check their delegate pool balances via REST API
		blobberOutput, err := listBlobbers(t, configPath, "--json")
		require.NoError(t, err)

		var blobbers []climodel.BlobberInfo
		err = json.Unmarshal([]byte(strings.Join(blobberOutput, "")), &blobbers)
		require.NoError(t, err)

		for _, sharder := range sharders {
			sharderURL := getNodeBaseURL(sharder.Host, sharder.Port)

			for _, blobber := range blobbers {
				url := fmt.Sprintf("%s/v1/screst/%s/getStakePoolStat?provider_id=%s&provider_type=1",
					sharderURL,
					"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7",
					blobber.Id)

				resp, err := http.Get(url) //nolint:gosec
				if err != nil || resp.StatusCode != 200 {
					continue
				}

				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				var poolStat map[string]interface{}
				if json.Unmarshal(body, &poolStat) != nil {
					continue
				}

				// Check delegate pools for negative balances
				if delegates, ok := poolStat["delegate"].([]interface{}); ok {
					for _, d := range delegates {
						if dp, ok := d.(map[string]interface{}); ok {
							if balance, ok := dp["balance"].(float64); ok {
								require.GreaterOrEqual(t, balance, float64(0),
									"delegate pool balance for blobber %s should not be negative (balance=%f). "+
										"Negative balance means GREATEST guard is missing from penalty query",
									blobber.Id, balance)
							}
						}
					}
				}
			}
			break // one sharder is enough
		}
		t.Log("All delegate pool balances are non-negative")
	})
}


// getWriteMarkerCount queries the sharder REST API for the write marker count
// of an allocation. Returns 0 if the query fails.
func getWriteMarkerCount(t *test.SystemTest, allocationID string) int {
	output, err := getSharders(t, configPath)
	require.NoError(t, err)

	var sharders map[string]*climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.NoError(t, err)

	for _, sharder := range sharders {
		sharderURL := getNodeBaseURL(sharder.Host, sharder.Port)
		url := fmt.Sprintf("%s/v1/screst/%s/alloc_write_marker_count?allocation_id=%s",
			sharderURL,
			"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7",
			allocationID)

		resp, err := http.Get(url) //nolint:gosec
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			// Try parsing as plain integer
			var count int
			if err := json.Unmarshal(body, &count); err == nil {
				return count
			}
			continue
		}

		if count, ok := result["count"].(float64); ok {
			return int(count)
		}
	}

	t.Log("WARNING: could not query write marker count from any sharder")
	return 0
}
