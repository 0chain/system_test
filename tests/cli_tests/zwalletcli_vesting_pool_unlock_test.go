package cli_tests

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestVestingPoolUnlock(t *testing.T) {
	t.Parallel()

	// get current valid vesting configs
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-UNLOCK cases
	t.Run("Vesting pool unlock with excess tokens in pool should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.2,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		cliutils.Wait(t, time.Second)

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 2, "expected output of length 1")
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Vp-info should show (can unlock) as 0, wallet should have increased by 0.1
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Equal(t, "can unlock:   0 SAS (excess)", output[2])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching wallet balance")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 900.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("Vesting pool unlock by destination wallet should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		cliutils.Wait(t, time.Second*5)

		output, err = vestingPoolUnlockForWallet(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true, targetWalletName)
		require.Nil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 2, "expected output of length 1")
		require.Equal(t, "Tokens unlocked successfully.", output[0])

		// Vp-info should show (can unlock) as 0, wallet should have increased by 0.1
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Equal(t, "can unlock:   0 SAS (excess)", output[2])

		// Target wallet balance should get unlocked tokens
		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching wallet balance")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]ZCN \(\d*\.?\d+ USD\)$`), output[0])
		balance, err := strconv.ParseFloat(regexp.MustCompile(`\d*\.?\d+`).FindString(output[0]), 64)
		require.Nil(t, err, "error parsing float from balance")
		require.Greater(t, balance, 0.0)
	})

	t.Run("Unlocking someone else's vesting pool should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error registering new wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		reg := regexp.MustCompile("unlock_vesting_pool_failed: vesting pool: destination [a-z0-9]{64} not found in the pool")
		require.Regexp(t, reg, output[0])
	})

	t.Run("Vesting pool unlock for one destination and no excess tokens in pool should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		cliutils.Wait(t, time.Second)

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "unlock_vesting_pool_failed: draining pool: no excess tokens to unlock", output[0])
	})

	t.Run("Vesting unlock without pool id must fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})

	t.Run("Vesting unlock with invalid pool id must fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef123456",
		}), false)
		require.NotNil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "unlock_vesting_pool_failed: can't get pool: value not present", output[0])
	})
}

func vestingPoolUnlock(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return vestingPoolUnlockForWallet(t, cliConfigFilename, params, retry, escapedTestName(t))
}

func vestingPoolUnlockForWallet(t *testing.T, cliConfigFilename, params string, retry bool, wallet string) ([]string, error) {
	t.Log("Unlocking a vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-unlock "+params+
			" --silent --wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-unlock " + params +
			" --silent --wallet " + wallet + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}
