package cli_tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestReadPoolLockUnlock(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Locking read pool tokens moves tokens from wallet to read pool")

	t.Skip()

	t.Parallel()

	t.RunWithTimeout("Locking read pool tokens moves tokens from wallet to read pool", 90*time.Second, func(t *test.SystemTest) { //TOOD: slow
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		lockAmount := 1.0
		readPoolParams := createParams(map[string]interface{}{
			"tokens": lockAmount,
		})
		output, err := readPoolLock(t, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Wallet balance should decrement from 5 to 3.9 (0.01 is fees) ZCN
		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InEpsilon(t, balanceBefore-1.01, balanceAfter, 0.01)
		balanceBefore = balanceAfter

		// Read pool balance should increment to 1
		readPool := getReadPoolInfo(t)
		require.Equal(t, ConvertToValue(lockAmount), readPool.Balance, "Read Pool balance must be equal to locked amount")

		output, err = readPoolUnlock(t, configPath, "", true)
		require.Nil(t, err, "Unable to unlock tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "unlocked", output[0])

		// Wallet balance should increment from 4 to 4.98 (0.01 fees for unlocking) ZCN
		balanceAfter, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)

		t.Log("balanceBefore : ", balanceBefore, " balanceAfter : ", balanceAfter)
		require.InEpsilon(t, balanceBefore+0.99, balanceAfter, 0.01)
	})

	t.Run("Should not be able to lock more read tokens than wallet balance", func(t *test.SystemTest) {
		_, err := executeFaucetWithTokens(t, configPath, 1)
		require.NoError(t, err)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		readPoolParams := createParams(map[string]interface{}{
			"tokens": 10,
		})
		output, err := readPoolLock(t, configPath, readPoolParams, false)
		require.NotNil(t, err, "Locked more tokens than in wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in read pool: read_pool_lock_failed: lock amount is greater than balance", output[0], strings.Join(output, "\n"))

		// Wallet balance reduced due to chargeable error (0.1 fees)
		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InEpsilon(t, balanceBefore-0.01, balanceAfter, 0.01)
	})

	t.Run("Should not be able to lock negative read tokens", func(t *test.SystemTest) {
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Locking -1 token in read pool should not succeed
		readPoolParams := createParams(map[string]interface{}{
			"tokens": -1,
		})
		output, err := readPoolLock(t, configPath, readPoolParams, false)
		require.NotNil(t, err, "Locked negative tokens", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "invalid token amount: negative", output[0], strings.Join(output, "\n"))

		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, balanceBefore, balanceAfter)
	})

	t.Run("Should not be able to lock zero read tokens", func(t *test.SystemTest) {
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Locking 0 token in read pool should not succeed
		readPoolParams := createParams(map[string]interface{}{
			"tokens": 0,
		})

		output, err := readPoolLock(t, configPath, readPoolParams, false)
		require.NotNil(t, err, "Locked 0 tokens", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in read pool: read_pool_lock_failed: invalid amount to lock [ensure token > 0].", output[0], strings.Join(output, "\n"))

		// Wallet balance gets reduced due to chargeable error (0.1 fees)
		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.InEpsilon(t, balanceBefore-0.01, balanceAfter, 0.01)
	})

	t.Run("Missing tokens flag in rp-lock should result in error", func(t *test.SystemTest) {
		createWallet(t)

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Not specifying amount to lock should not succeed
		readPoolParams := createParams(map[string]interface{}{})
		output, err := readPoolLock(t, configPath, readPoolParams, false)
		require.NotNil(t, err, "Locked tokens without providing amount to lock", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'tokens' flag", output[0])

		// Wallet balance should remain same
		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, balanceBefore, balanceAfter)
	})
}

func readPoolUnlock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Unlocking read tokens...")
	cmd := fmt.Sprintf("./zbox rp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getReadPoolInfo(t *test.SystemTest) climodel.ReadPoolInfo {
	output, err := readPoolInfo(t, configPath)
	require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
	require.Lenf(t, output, 1, "ouptut: %v", output)

	var readPool climodel.ReadPoolInfo
	err = json.Unmarshal([]byte(output[0]), &readPool)
	require.Nil(t, err, "Error unmarshalling read pool %s", strings.Join(output, "\n"))
	return readPool
}
