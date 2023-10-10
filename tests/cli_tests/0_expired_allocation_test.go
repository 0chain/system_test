package cli_tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

func TestExpiredAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Finalize Expired Allocation Should Work after challenge completion time + expiry")

	t.TestSetup("register wallet and get blobbers", func() {
		output, err := updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "1m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.Cleanup(func() {
		output, err := updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "1h",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Finalize Expired Allocation Should Work after challenge completion time + expiry", 5*time.Minute, func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		output, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		allocationID, _ := setupAndParseAllocation(t, configPath, map[string]interface{}{})

		time.Sleep(90 * time.Second)

		allocations := parseListAllocations(t, configPath)
		_, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)

		cliutils.Wait(t, 2*time.Minute)

		output, err = finalizeAllocation(t, configPath, allocationID, true)

		require.Nil(t, err, "unexpected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		matcher := regexp.MustCompile("Allocation finalized with txId .*$")
		require.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")
	})

	t.RunWithTimeout("Cancel Expired Allocation Should Fail", 4*time.Minute, func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		allocationID, _ := setupAndParseAllocation(t, configPath, map[string]interface{}{})

		time.Sleep(2 * time.Minute)
		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.LessOrEqual(t, ac.ExpirationDate, time.Now().Unix())

		// Cancel the expired allocation
		output, err := cancelAllocation(t, configPath, allocationID, false)
		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))

		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: trying to cancel expired allocation", output[0])
	})

	t.Run("Download File using Expired Allocation Should Fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		time.Sleep(2 * time.Minute)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "consensus_not_met")
	})

	t.RunWithTimeout("Update Expired Allocation Should Fail", 7*time.Minute, func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath, map[string]interface{}{})

		time.Sleep(2 * time.Minute)

		// Update expired alloc's duration
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: can't update expired allocation", output[len(output)-1])

		// Update the expired allocation's size
		size := int64(2048)
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: can't update expired allocation", output[0])
	})

	t.RunWithTimeout("Unlocking tokens from finalized allocation should work", 11*time.Minute, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "2048",
			"lock": "1",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance before lock should be 4.5 ZCN
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 1 token in Write pool amongst all blobbers
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     1,
		})
		output, err = writePoolLock(t, configPath, params, true)
		require.Nil(t, err, "Failed to lock write tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// get balance after lock
		balanceAfterLock, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// assert balance reduced by 1 ZCN and txn fee
		require.Less(t, balanceAfterLock, balance-1)

		// Write pool balance should increment by 1
		allocation := getAllocation(t, allocationID)
		require.Equal(t, 2.0, intToZCN(allocation.WritePool))

		allocationCost := 0.0
		for _, blobber := range allocation.BlobberDetails {
			allocationCost += sizeInGB(1024) * float64(blobber.Terms.Write_price)
		}
		allocationCancellationCharge := allocationCost * 0.2 // 20% of total allocation cost

		// get balance before finalize
		balanceBeforeFinalize, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Wait for allocation to expire
		cliutils.Wait(t, time.Minute*2)

		output, err = finalizeAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "unexpected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile("Allocation finalized with txId .*$"), output[0])

		// get balance after unlock
		balanceAfterFinalize, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// assert after unlock, balance is greater than before finalize, but need to pay fee
		require.InEpsilon(t, balanceAfterFinalize, balanceBeforeFinalize+2.0-allocationCancellationCharge, 0.05)
	})
}

func sizeInGB(size int64) float64 {
	return float64(size) / (1024 * 1024 * 1024)
}
