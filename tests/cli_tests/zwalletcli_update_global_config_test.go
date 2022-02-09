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

	t.Run("Update Global Config - Update mutable config should work", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

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
		if oldValue == newValue {
			newValue = "201"
		}

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

	t.Run("Update Global Config - Update multiple mutable config should work", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey1 := "server_chain.block.proposal.max_wait_time"
		newValue1 := "190ms"

		configKey2 := "server_chain.smart_contract.setting_update_period"
		newValue2 := "210"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		cfgBefore := getGlobalConfiguration(t, true)
		oldValue1 := cfgBefore[configKey1].(string)
		if oldValue1 == newValue1 {
			newValue1 = "185ms"
		}

		oldValue2 := cfgBefore[configKey2].(string)
		if oldValue2 == newValue2 {
			newValue2 = "200"
		}

		// ensure revert in config is run regardless of test result
		defer func() {
			output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey1 + "," + configKey2,
				"values": oldValue1 + "," + oldValue2,
			}, true)
		}()

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey1 + "," + configKey2,
			"values": newValue1 + "," + newValue2,
		}, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "global settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		cliutils.Wait(t, 2*time.Second)

		cfgAfter := getGlobalConfiguration(t, true)

		require.Equal(t, newValue1, cfgAfter[configKey1], "new value %s for config %s was not set", newValue1, configKey1)
		require.Equal(t, newValue2, cfgAfter[configKey2], "new value %s for config %s was not set", newValue2, configKey2)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	t.Run("Update Global Config - Update immutable config must fail", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Failed to get wallet")

		configKey := "server_chain.owner"
		newValue := wallet.ClientID

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, "Setting immutable config must fail. but it didn't", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_globals: validation: server_chain.owner cannot be modified via a transaction", output[0], strings.Join(output, "\n"))
	})

	t.Run("Update Global Config - Update multiple config including 1 immutable config must fail", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Failed to get wallet")

		configKey1 := "server_chain.owner"
		newValue1 := wallet.ClientID

		configKey2 := "server_chain.smart_contract.setting_update_period"
		newValue2 := "210"

		cfgBefore := getGlobalConfiguration(t, true)

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey1 + "," + configKey2,
			"values": newValue1 + "," + newValue2,
		}, false)
		require.NotNil(t, err, "Setting immutable config must fail. but it didn't", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_globals: validation: server_chain.owner cannot be modified via a transaction", output[0], strings.Join(output, "\n"))

		cliutils.Wait(t, 2*time.Second)

		cfgAfter := getGlobalConfiguration(t, true)

		require.Equal(t, cfgBefore[configKey1], cfgAfter[configKey1], "new value %s for config %s must not be set", newValue1, configKey1)
		require.Equal(t, cfgBefore[configKey2], cfgAfter[configKey2], "new value %s for config %s must not be set", newValue2, configKey2)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	// FIXME! Maybe this is better to fail the command from zwallet in case of no parameters.
	// Currently in this case transaction is getting executed, but nothing is getting updated.
	t.Run("Update Global Config - update with suppliying no parameter must update nothing", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		cfgBefore := getGlobalConfiguration(t, true)

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{}, false)
		require.Nil(t, err, "Error in updating global config", strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "global settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		cliutils.Wait(t, 2*time.Second)

		cfgAfter := getGlobalConfiguration(t, true)

		for key, value := range cfgBefore {
			require.Equal(t, cfgBefore[key], cfgAfter[key], "The command should not update the values but it changes '%v' from %v to %v", key, value, cfgAfter[key])
		}
	})

	t.Run("Update Global Config - update with invalid key must fail", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		configKey := "invalid.key"
		newValue := "120ms"

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, "Setting config with invalid key must fail. but it didn't", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_globals: validation: 'invalid.key' is not a valid global setting", output[0], strings.Join(output, "\n"))
	})

	t.Run("Update Global Config - update with invalid value must fail", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		configKey := "server_chain.block.proposal.max_wait_time"
		newValue := "abc"

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateGlobalConfigWithWallet(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, "Setting config with invalid value must fail. but it didn't", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_globals: validation: server_chain.block.proposal.max_wait_time value abc cannot be parsed as a time.duration", output[0], strings.Join(output, "\n"))
	})

	t.Run("Update Global Config with a non-owner wallet Should Fail ", func(t *testing.T) {
		t.Parallel()

		configKey := "server_chain.smart_contract.setting_update_period"
		newValue := "215"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateGlobalConfigWithWallet(t, escapedTestName(t), map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_globals: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
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
