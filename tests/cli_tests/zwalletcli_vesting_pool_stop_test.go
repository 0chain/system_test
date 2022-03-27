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

func TestVestingPoolStop(t *testing.T) {
	t.Parallel()

	// get current valid vesting configs
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-STOP cases
	t.Run("Vesting pool stop for pool with one destination should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		// add a vesting pool for sending 0.1 to target wallet
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

		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
			"d":       targetWallet.ClientID,
		}), true)
		require.Nil(t, err, "error stopping vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Stop vesting for "+targetWallet.ClientID+".", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Destination should be removed from vp-info after stopping
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 11, "expected output of length 11 atleast")
		require.Equal(t, "destinations:", output[9])
		require.Equal(t, "client_id:    "+wallet.ClientID, output[10])
		canUnlockAmount, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[2]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		canUnlockUnit := regexp.MustCompile("[um]?ZCN").FindString(output[2])
		canUnlockAmount = unitToZCN(canUnlockAmount, canUnlockUnit)

		// token-accounting for this case: balance tokens should be unlockable
		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error unlocking tokens from vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN := unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		require.InEpsilonf(t, 0.9+canUnlockAmount, newBalanceInZCN, 0.00000000001, "expected balance to be [%v] but was [%v]", 0.9+canUnlockAmount, newBalanceInZCN)
	})

	// FIXME: this only stops last destination flag.
	t.Run("Vesting pool stop for multiple destinations should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWalletName3 := "targetWallet3" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName3)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching destination wallet")

		targetWallet3, err := getWalletForName(t, configPath, targetWalletName3)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2" + " --d " + targetWallet3.ClientID + ":0.3",
			"lock":     0.6,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// Stopping with multiple destinations
		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
			"d":       targetWallet.ClientID + " --d " + targetWallet2.ClientID,
		}), true)
		require.Nil(t, err, "error stopping vesting pool")
		// FIXME: output only shows stop vesting for last destination flag. Should show all stopped destinations
		require.Len(t, output, 2)
		require.Equal(t, "Stop vesting for "+targetWallet2.ClientID+".", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Destination should be removed from vp-info after stopping
		// FIXME: Multiple d flags don't work, only last flag passed is stopped.
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 23, "expected output of length 23")
		canUnlockAmount, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[2]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		canUnlockUnit := regexp.MustCompile("[um]?ZCN").FindString(output[2])
		canUnlockAmount = unitToZCN(canUnlockAmount, canUnlockUnit)

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error unlocking tokens from vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN := unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		require.InEpsilonf(t, 0.4+canUnlockAmount, newBalanceInZCN, 0.00000000001, "expected balance to be [%v] but was [%v]", 0.9+canUnlockAmount, newBalanceInZCN)
	})

	t.Run("Vesting pool stop for someone else's pool must fail", func(t *testing.T) {
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
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// Stopping with multiple destinations
		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
			"d":       targetWallet.ClientID,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "stop_vesting_failed: only owner can stop a vesting", output[0])
	})

	t.Run("Vesting pool stop without pool id must fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"d": targetWallet.ClientID,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})

	t.Run("Vesting pool stop without destination should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": "dummypoolid",
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'd' flag", output[0])
	})
}

func vestingPoolStop(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Stopping vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-stop "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-stop " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}
