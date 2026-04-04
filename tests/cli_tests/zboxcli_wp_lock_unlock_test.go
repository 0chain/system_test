package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestWritePoolLock(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Creating allocation should move tokens from wallet to write pool, write lock and unlock should work")

	t.Parallel()

	t.Run("Creating allocation should move tokens from wallet to write pool, write lock should work", func(t *test.SystemTest) {
		createWallet(t)

		// get balance
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "2048",
			"lock": "1",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		t.Log("new allocation:", output)

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		balanceAfterAlloc, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Less(t, balanceAfterAlloc, balance-0.5)

		// Lock 1 token in Write pool amongst all blobbers
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     1,
		})
		output, err = writePoolLock(t, configPath, params, true)
		require.Nil(t, err, "Failed to lock write tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		balanceAfterLock, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		// Use InEpsilon instead of LessOrEqual to handle floating point rounding
		// (e.g., 15.99 vs 15.989999999999998)
		require.InEpsilon(t, balanceAfterAlloc-1, balanceAfterLock, 0.01,
			"balance after lock (%f) should be approximately %f", balanceAfterLock, balanceAfterAlloc-1)

		// Write pool balance should increment by 1
		allocation := getAllocation(t, allocationID)
		require.Equal(t, 2.0, intToZCN(allocation.WritePool))

		allocationCost := 0.0
		for _, blobber := range allocation.BlobberDetails {
			allocationCost += sizeInGB(1024) * float64(blobber.Terms.WritePrice)
		}
		allocationCancellationCharge := allocationCost * 0.2
		allocationCancellationChargeInZCN := allocationCancellationCharge / 1e10

		// get balance before cancel
		balanceBeforeCancel, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err)
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		balanceAfterCancel, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InEpsilon(t, balanceAfterCancel, balanceBeforeCancel+2.0-allocationCancellationChargeInZCN, 0.15)
	})

	t.RunWithTimeout("Should not be able to lock more write tokens than wallet balance", 15*time.Minute, func(t *test.SystemTest) {
		// Fund with 9 ZCN (max single-call amount before system caps at 1) to survive
		// multiple failed allocation retries during view changes (each retry burns ~1 ZCN fee).
		_, err := executeFaucetWithTokens(t, configPath, 9)
		require.NoError(t, err)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "2048", // Use 2048 to meet min_alloc_size requirement
			"lock": "0.5",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance should decrease by lock amount + fee
		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		balanceDiff := balanceBefore - balanceAfter
		require.GreaterOrEqual(t, balanceDiff, 0.5, "Balance should decrease by at least the locked amount")
		require.Less(t, balanceDiff, 3.0, "Balance decrease should not exceed lock + reasonable fee")
		balanceBefore = balanceAfter

		// Lock more tokens than wallet balance should fail (use balanceBefore+5 to exceed balance regardless of pre-funding)
		excessTokens := int(balanceBefore) + 5
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     excessTokens,
		})
		output, err = writePoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked more tokens than in wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.True(t, strings.Contains(output[0], "lock amount is greater than balance") ||
			strings.Contains(output[0], "write_pool_lock_failed") ||
			strings.Contains(output[0], "insufficient") ||
			strings.Contains(output[0], "not enough"),
			"expected lock failure error, got: %s", output[0])

		// Wallet balance should remain approximately the same (failed txn may still charge fee)
		balanceAfter, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InDelta(t, balanceBefore, balanceAfter, 2.0, "Failed lock should not significantly change balance")
	})

	t.Run("Should not be able to lock negative write tokens", func(t *test.SystemTest) {
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "2048", // Use 2048 to meet min_alloc_size requirement
			"lock": "0.5",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		balanceDiff := balanceBefore - balanceAfter
		require.GreaterOrEqual(t, balanceDiff, 0.5, "Balance should decrease by at least the locked amount")
		require.Less(t, balanceDiff, 3.0, "Balance decrease should not exceed lock + reasonable fee")
		balanceBefore = balanceAfter

		// Locking -1 token in write pool should not succeed
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     -1,
		})
		output, err = writePoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked negative tokens", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "invalid token amount: negative", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain approximately the same
		balanceAfter, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InDelta(t, balanceBefore, balanceAfter, 1.0, "Negative lock should not change balance")
	})

	t.RunWithTimeout("Should not be able to lock zero write tokens", 60*time.Second, func(t *test.SystemTest) { //todo: slow
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "2048", // Use 2048 to meet min_alloc_size requirement
			"lock": "0.5",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		balanceDiff := balanceBefore - balanceAfter
		require.GreaterOrEqual(t, balanceDiff, 0.5, "Balance should decrease by at least the locked amount")
		require.Less(t, balanceDiff, 3.0, "Balance decrease should not exceed lock + reasonable fee")
		balanceBefore = balanceAfter

		// Locking 0 token in write pool should not succeed
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0,
		})
		output, err = writePoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked 0 tokens", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in write pool: write_pool_lock_failed: insufficient amount to lock", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain approximately the same (failed txn may charge fee)
		balanceAfter, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InDelta(t, balanceBefore, balanceAfter, 2.0, "Zero lock should not significantly change balance")
	})

	t.Run("Missing tokens flag should result in error", func(t *test.SystemTest) {
		createWallet(t)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "2048", // Use 2048 to meet min_alloc_size requirement
			"lock": "0.5",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Not specifying amount to lock should not succeed
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
		})
		output, err = writePoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked tokens without providing amount to lock", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'tokens' flag", output[0])
	})
}

func writePoolLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return writePoolLockWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func writePoolLockWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Locking write tokens...")
	cmd := fmt.Sprintf("./zbox wp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
