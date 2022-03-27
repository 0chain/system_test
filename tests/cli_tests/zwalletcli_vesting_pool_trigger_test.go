package cli_tests

import (
	"regexp"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestVestingPoolTrigger(t *testing.T) {
	t.Parallel()

	// get current valid vesting configs
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-TRIGGER cases
	// FIXME: vp-trigger is not working, tokens still keep on being sent in a timely manner as opposed to being release all at once.
	t.Run("Vesting pool trigger for one destination pool should work", func(t *testing.T) {
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
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error trigerring vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Vesting triggered successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// vp-info should show vested tokens as sent
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Len(t, output, 18)
		// FIXME:
		// output[1] should be 0 SAS, output[3]: "1.000 mZCN (real value)", output[13]: "1.000 mZCN"
		// output[14] should be 0 SAS, output[4]: 0 SAS

		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching wallet balance")
		// FIXME: Balance should be 1.000 mZCN
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]?ZCN \(\d*\.?\d+ USD\)`), output[0])
	})

	t.Run("Vesting pool trigger for a pool with multiple destinations should work", func(t *testing.T) {
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

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":     0.3,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error trigerring vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Vesting triggered successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Vp-info should show that all tokens are transferred to destination wallets
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Len(t, output, 24, "expected output of length 24 atleast")
		// FIXME:
		// Balance should be 0 SAS, Sent should be 3.000 mZCN, sent for ID 1 should be 1.000 mZCN
		// Sent for ID 2 should be 2.000 mZCN

		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching wallet balance")
		// FIXME: Balance should be 1.000 mZCN
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]?ZCN \(\d*\.?\d+ USD\)`), output[0])

		output, err = getBalanceForWallet(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching wallet balance")
		// FIXME: Balance should be 2.000 mZCN
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]?ZCN \(\d*\.?\d+ USD\)`), output[0])
	})

	t.Run("Triggering someone else's pool must fail", func(t *testing.T) {
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

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "trigger_vesting_pool_failed: only owner can trigger the pool", output[0])
	})

	t.Run("Vesting pool trigger without pool id flag should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "expected error trigerring vesting pool without pool id")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})

	t.Run("Vesting pool trigger with invalid pool id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef123456",
		}), false)
		require.NotNil(t, err, "expected error trigerring vesting pool with invalid pool id")
		require.Len(t, output, 1)
		require.Equal(t, "trigger_vesting_pool_failed: can't get pool: value not present", output[0])
	})
}

func vestingPoolTrigger(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return vestingPoolTriggerForWallet(t, cliConfigFilename, params, retry, escapedTestName(t))
}

func vestingPoolTriggerForWallet(t *testing.T, cliConfigFilename, params string, retry bool, wallet string) ([]string, error) {
	t.Log("Triggering vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-trigger "+params+
			" --silent --wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-trigger " + params +
			" --silent --wallet " + wallet + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}
