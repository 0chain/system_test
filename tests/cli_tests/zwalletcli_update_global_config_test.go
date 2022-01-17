package cli_tests

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestUpdateGlobalConfig(t *testing.T) {
	t.Parallel()

	t.Run("Update Global Config Should Work", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		// configKey := "server_chain.block.max_block_size"
		// newValue := "11"

		configKey := "server_chain.smart_contract.setting_update_period"
		newValue := "200"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		cfgBefore := getGlobalConfiguration(t, true)
		oldValue := cfgBefore[configKey]

		// ensure revert in config is run regardless of test result
		defer func() {
			output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": oldValue,
			}, true)
		}()

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "global settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		cliutils.Wait(t, 2*time.Second)

		cfgAfter := getGlobalConfiguration(t, true)

		require.Equal(t, newValue, cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	t.Run("Update Global Config with a non-owner wallet Should Fail ", func(t *testing.T) {
		t.Parallel()

		configKey := "server_chain.smart_contract.setting_update_period"
		newValue := "215"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// cfgBefore := getGlobalConfiguration(t, true)
		// oldValue := cfgBefore[configKey]

		output, err = updateGlobalConfigWithWallet(t, escapedTestName(t), map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "fatal:{\"error\": \"verify transaction failed\"}", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get Global Config Should Work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = getGlobalConfig(t, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfg := map[string]string{}
		for _, o := range output {
			configPair := strings.Split(o, "\t")
			cfg[strings.TrimSpace(configPair[0])] = strings.TrimSpace(configPair[1])
		}

		require.Greater(t, len(cfg), 0, "Configuration map must include some items")
	})
}

func getGlobalConfiguration(t *testing.T, retry bool) map[string]interface{} {
	res := map[string]interface{}{}
	output, err := getGlobalConfig(t, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	for _, tabSeparatedKeyValuePair := range output {
		kvp := strings.Split(tabSeparatedKeyValuePair, "\t")
		key := strings.TrimSpace(kvp[0])
		var val string
		if len(kvp) > 1 {
			val = strings.TrimSpace(kvp[1])
		}
		res[key] = val
	}
	return res
}

func getGlobalConfig(t *testing.T, retry bool) ([]string, error) {
	return getGlobalConfigWithWallet(t, escapedTestName(t), true)
}

func getGlobalConfigWithWallet(t *testing.T, walletName string, retry bool) ([]string, error) {
	t.Logf("Retrieving global config...")

	cmd := "./zwallet global-config --silent --wallet " + walletName + "_wallet.json --configDir ./config --config " + configPath

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateGlobalConfigWithWallet(t *testing.T, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Updating global config...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zwallet global-update-config %s --silent --wallet %s --configDir ./config --config %s",
		p,
		walletName+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
