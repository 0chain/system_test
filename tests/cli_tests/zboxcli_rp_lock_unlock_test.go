package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestReadPoolLockUnlock(t *testing.T) {
	t.Parallel()

	t.Run("Locking read pool tokens moves tokens from wallet to read pool", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.5)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Wallet balance before lock should be 1.5 ZCN
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.500 ZCN \(\d*\.?\d+ USD\)$`), output[0])

		// There should be no read pool before lock
		output, err = readPoolInfo(t, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "no tokens locked", output[0])

		// Lock 1 token in read pool distributed amongst all blobbers
		lockAmount := 1.0
		params := createParams(map[string]interface{}{
			"tokens": lockAmount,
		})
		output, err = readPoolLock(t, configPath, params, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Wallet balance should decrement from 1.5 to 0.5 ZCN
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// Read pool balance should increment to 1
		output, err = readPoolInfo(t, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		readPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &readPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Equal(t, lockAmount, readPool.OwnerBalance)

		output, err = readPoolUnlock(t, configPath, params, true)
		require.Nil(t, err, "Unable to unlock tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "unlocked", output[0])

		// Wallet balance should increment from 0.5 to 1.5 ZCN
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.500 ZCN \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("Should not be able to lock more read tokens than wallet balance", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Wallet balance before lock should be 0.5 ZCN
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.00\d ZCN \(\d*\.?\d+ USD\)$`), output[0])

		// Lock 1 token in read pool distributed amongst all blobbers
		params := createParams(map[string]interface{}{
			"tokens": 2,
		})
		output, err = readPoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked more tokens than in wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in read pool: read_pool_lock_failed: lock amount is greater than balance", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain same
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.00\d ZCN \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("Should not be able to lock negative read tokens", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Wallet balance before lock should be 0.5 ZCN
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// Locking -1 token in read pool should not succeed
		params := createParams(map[string]interface{}{
			"tokens": -1,
		})
		output, err = readPoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked negative tokens", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in read pool: [txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain same
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.00\d ZCN \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("Should not be able to lock zero read tokens", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Wallet balance before lock should be 0.5 ZCN
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.00\d ZCN \(\d*\.?\d+ USD\)$`), output[0])

		// Locking 0 token in read pool should not succeed
		params := createParams(map[string]interface{}{
			"tokens": 0,
		})
		output, err = readPoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked 0 tokens", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Failed to lock tokens in read pool: read_pool_lock_failed: insufficient amount to lock", output[0], strings.Join(output, "\n"))

		// Wallet balance should remain same
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.00\d ZCN \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("Missing tokens flag should result in error", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Wallet balance before lock should be 0.5 ZCN
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// Not specifying amount to lock should not succeed
		params := createParams(map[string]interface{}{
			"owner": "true",
		})
		output, err = readPoolLock(t, configPath, params, false)
		require.NotNil(t, err, "Locked tokens without providing amount to lock", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'tokens' flag", output[0])

		// Wallet balance should remain same
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])
	})
}

func readPoolUnlock(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Unlocking read tokens...")
	cmd := fmt.Sprintf("./zbox rp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
