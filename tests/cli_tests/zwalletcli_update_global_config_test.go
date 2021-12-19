package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestUpdateGlobalConfig(t *testing.T) {
	t.Parallel()

	t.Run("Get Global Config Should Work", func(t *testing.T) {
		t.Parallel()

		// if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		// 	t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		// }

		configKey := "max_pour_amount"
		newValue := "15"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// // register SC owner wallet
		// output, err = registerWalletForName(t, configPath, scOwnerWallet)
		// require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = getGlobalConfig(t, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgBefore := map[string]string{}
		for _, o := range output {
			configPair := strings.Split(o, "\t")
			cfgBefore[strings.TrimSpace(configPair[0])] = strings.TrimSpace(configPair[1])
		}

		// ensure revert in config is run regardless of test result
		defer func() {
			oldValue := cfgBefore[configKey]
			output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": oldValue,
			}, true)
		}()

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "faucet smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		output, err = getGlobalConfig(t, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter := map[string]string{}
		for _, o := range output {
			configPair := strings.Split(o, "\t")
			cfgAfter[strings.TrimSpace(configPair[0])] = strings.TrimSpace(configPair[1])
		}

		require.Equal(t, newValue, cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})
}

func getGlobalConfig(t *testing.T, retry bool) ([]string, error) {
	return getGlobalConfigWithWallet(t, escapedTestName(t), true)
}

func getGlobalConfigWithWallet(t *testing.T, walletName string, retry bool) ([]string, error) {
	t.Logf("Retrieving miner config...")

	cmd := "./zwallet global-config --silent --wallet " + walletName + "_wallet.json --configDir ./config --config " + configPath

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateGlobalConfigWithWallet(t *testing.T, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Updating miner config...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zwallet mn-update-config %s --silent --wallet %s --configDir ./config --config %s",
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
