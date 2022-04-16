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

func TestVestingPoolDelete(t *testing.T) {
	t.Parallel()

	// get current valid vesting configs
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-DELETE cases
	t.Run("Vesting pool delete should work", func(t *testing.T) {
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

		cliutils.Wait(t, time.Second)

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error deleting vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Vesting pool deleted successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Wallet balance should be greater than 0.9 since non-vested tokens should return
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error getting wallet balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		balance, err := strconv.ParseFloat(regexp.MustCompile(`\d*\.?\d+`).FindString(output[0]), 64)
		require.Nil(t, err, "error parsing float from balance")
		require.Greater(t, balance, 900.000)
	})

	t.Run("Vesting pool delete with invalid pool_id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{
			"pool_id": "invalidPoolId",
		}), false)
		require.NotNil(t, err, "expected error when deleting invalid vesting pool id")
		require.Len(t, output, 1)
		require.Equal(t, "delete_vesting_pool_failed: can't get pool: value not present", output[0])
	})

	t.Run("Deleting someone else's vesting pool should fail", func(t *testing.T) {
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

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "delete_vesting_pool_failed: only pool owner can delete the pool", output[0])
	})

	t.Run("Vesting pool delete without pool id flag should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "expected error using vp-delete without pool id")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})
}

func vestingPoolDelete(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Deleting vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-delete "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-delete " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}
