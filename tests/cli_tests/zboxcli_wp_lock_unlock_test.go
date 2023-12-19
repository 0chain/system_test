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
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// get balance
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "2048",
			"lock": "1",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
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
		require.Less(t, balanceAfterLock, balanceAfterAlloc-1)

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
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1024",
			"lock": "0.5",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance before lock should be 4.5 ZCN
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.49, balance)

		// Lock 10 token in write pool should fail
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     10,
		})
		output, err = writePoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked more tokens than in wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in write pool: write_pool_lock_failed: lock amount is greater than balance", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain same (- fee)
		balance, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.48, balance)
	})

	t.Run("Should not be able to lock negative write tokens", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1024",
			"lock": "0.5",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance before lock should be 4.49 ZCN (0.01 fees)
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.49, balance)

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
		balance, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.49, balance)
	})

	t.RunWithTimeout("Should not be able to lock zero write tokens", 60*time.Second, func(t *test.SystemTest) { //todo: slow
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1024",
			"lock": "0.5",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance before lock should be 4.49 ZCN
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.49, balance)

		// Locking 0 token in write pool should not succeed
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0,
		})
		output, err = writePoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked 0 tokens", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in write pool: write_pool_lock_failed: insufficient amount to lock", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain same (- fee)
		balance, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.48, balance)
	})

	t.Run("Missing tokens flag should result in error", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"size": "1024",
			"lock": "0.5",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
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
