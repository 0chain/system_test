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
			"size": "1048576", // 1MB to ensure it's well above min_alloc_size (2KB) and works with 4 data + 2 parity shards
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
		// Use LessOrEqual to account for floating point rounding - balance might be exactly balanceAfterAlloc-1 due to fees
		require.LessOrEqual(t, balanceAfterLock, balanceAfterAlloc-1)
		// Also verify it's actually less (accounting for transaction fees)
		require.Less(t, balanceAfterLock, balanceAfterAlloc-0.99, "Balance should decrease by at least 1 ZCN plus fees")

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
		require.InEpsilon(t, balanceAfterCancel, balanceBeforeCancel+2.0-allocationCancellationChargeInZCN, 0.05)
	})

	t.Run("Should not be able to lock more write tokens than wallet balance", func(t *test.SystemTest) {
		createWallet(t)
		// Wallet is pre-funded with 1000 ZCN, no need for faucet

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1048576", // 1MB to ensure it's well above min_alloc_size (2KB) and works with 4 data + 2 parity shards
			"lock": "0.5",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance after allocation creation (0.5 lock + fees)
		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, balanceBefore-0.5-0.01, balanceAfter)
		balanceBefore = balanceAfter

		// Lock more tokens than available in wallet (wallet has ~999.5 ZCN, try to lock 2000)
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     2000.0, // Lock more than the remaining balance
		})
		output, err = writePoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked more tokens than in wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in write pool: write_pool_lock_failed: lock amount is greater than balance", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain same (- fee)
		balanceAfter, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, balanceBefore-0.01, balanceAfter)
	})

	t.Run("Should not be able to lock negative write tokens", func(t *test.SystemTest) {
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1048576", // 1MB to ensure it's well above min_alloc_size (2KB) and works with 4 data + 2 parity shards
			"lock": "0.5",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, balanceBefore-0.5-0.01, balanceAfter)
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

		// Wallet balance should remain same
		balanceAfter, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, balanceBefore, balanceAfter)
	})

	t.RunWithTimeout("Should not be able to lock zero write tokens", 60*time.Second, func(t *test.SystemTest) { //todo: slow
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1048576", // 1MB to ensure it's well above min_alloc_size (2KB) and works with 4 data + 2 parity shards
			"lock": "0.5",
		})
		output, err := createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		require.InEpsilon(t, balanceBefore-0.51, balanceAfter, 0.01)
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

		// Wallet balance should remain same (-fee)
		balanceAfter, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InEpsilon(t, balanceBefore-0.01, balanceAfter, 0.01)
	})

	t.Run("Missing tokens flag should result in error", func(t *test.SystemTest) {
		createWallet(t)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1048576", // 1MB to ensure it's well above min_alloc_size (2KB) and works with 4 data + 2 parity shards
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
