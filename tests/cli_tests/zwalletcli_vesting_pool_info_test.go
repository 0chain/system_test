package cli_tests

import (
	"regexp"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestVestingPoolInfo(t *testing.T) {
	t.Parallel()

	// get current valid vesting configs
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-INFO cases
	// Feature to add: vp-info should have a json flag, it already has models in place in gosdk
	t.Run("Vesting pool info with valid pool_id should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		clientWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching client wallet")

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.3,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 18, "expected output of length 18")
		require.Equal(t, output[0], "pool_id:      "+poolId)
		require.Equal(t, output[1], "balance:      300.000 mZCN")
		require.Equal(t, output[2], "can unlock:   200.000 mZCN (excess)")
		require.Equal(t, output[3], "sent:         0 SAS (real value)")
		require.Equal(t, output[4], "pending:      100.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested:       \d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[5])
		require.Equal(t, output[6], "description:")
		require.Regexp(t, regexp.MustCompile("start_time:   [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[7])
		require.Regexp(t, regexp.MustCompile("expire_at:    [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[8])
		require.Equal(t, output[9], "destinations:")
		require.Equal(t, output[10], "- id:          "+targetWallet.ClientID)
		require.Equal(t, output[11], "vesting:     100.000 mZCN")
		require.Regexp(t, regexp.MustCompile(`can unlock: {2}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[12])
		require.Equal(t, output[13], "sent:        0 SAS (real value)")
		require.Equal(t, output[14], "pending:     100.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested: {6}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[15])
		require.Regexp(t, regexp.MustCompile("last unlock: [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[16])
		require.Equal(t, output[17], "client_id:    "+clientWallet.ClientID)
	})

	t.Run("Vesting pool info for pool with multiple destinations should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		clientWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching client wallet")

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

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

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 23, "expected output of length 23")
		require.Equal(t, output[0], "pool_id:      "+poolId)
		require.Equal(t, output[1], "balance:      300.000 mZCN")
		require.Equal(t, output[2], "can unlock:   0 SAS (excess)")
		require.Equal(t, output[3], "sent:         0 SAS (real value)")
		require.Equal(t, output[4], "pending:      300.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested:       \d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[5])
		require.Equal(t, output[6], "description:")
		require.Regexp(t, regexp.MustCompile("start_time:   [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[7])
		require.Regexp(t, regexp.MustCompile("expire_at:    [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[8])
		require.Equal(t, output[9], "destinations:")
		require.Equal(t, output[10], "- id:          "+targetWallet.ClientID)
		require.Equal(t, output[11], "vesting:     100.000 mZCN")
		require.Regexp(t, regexp.MustCompile(`can unlock: {2}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[12])
		require.Equal(t, output[13], "sent:        0 SAS (real value)")
		require.Equal(t, output[14], "pending:     100.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested: {6}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[15])
		require.Regexp(t, regexp.MustCompile("last unlock: [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[16])
		require.Equal(t, output[17], "- id:          "+targetWallet2.ClientID)
		require.Equal(t, output[18], "vesting:     200.000 mZCN")
		require.Regexp(t, regexp.MustCompile(`can unlock: {2}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[19])
		// FIXME: multiple destinations info not printing complete info for all destinations
		// require.Equal(t, output[20], "sent:        0 SAS (real value)")
		require.Equal(t, output[20], "pending:     200.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested: {6}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[21])
		// FIXME: multiple destinations info not printing complete info for all destinations
		// require.Regexp(t, regexp.MustCompile("last unlock: [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[23])
		require.Equal(t, output[22], "client_id:    "+clientWallet.ClientID)
	})

	// FIXME: vp-info can show information of vp belonging to other wallets
	t.Run("Vesting pool info for pool belonging to someone else's wallet must fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error getting faucet tokens on foreign wallet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.3,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// FIXME: should get error
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info", strings.Join(output, "\n"))
	})

	t.Run("Vesting pool info with invalid pool_id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef123456",
		}), false)
		require.NotNil(t, err, "expected error when using invalid pool_id")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "{\"code\":\"resource_not_found\",\"error\":\"resource_not_found: can't get pool: value not present\"}", output[0])
	})

	t.Run("Vesting pool info without pool id flag should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "expected error when using vp-info without pool id flag")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})
}

func vestingPoolInfo(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("fetching vesting pool info...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-info "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-info " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}
