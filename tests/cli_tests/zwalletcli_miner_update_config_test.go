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

func TestMinerUpdateConfig(t *testing.T) {
	t.Parallel()

	t.Run("should allow update of interest_rate", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "interest_rate"
		newValue := "0.1"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgBefore, _ := keyValuePairStringToMap(t, output)

		// ensure revert in config is run regardless of test result
		defer func() {
			oldValue := cfgBefore[configKey]
			output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": oldValue,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			// FIXME should say miner instead of storagesc
			require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
			require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
		}()

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		// FIXME should say miner instead of storagesc
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newValue, cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	t.Run("update interest_rate to invalid value should fail", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "interest_rate"
		newValue := "x"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "fatal:{\"error\": \"verify transaction failed\"}", output[0], strings.Join(output, "\n"))
	})

	t.Run("update by non-smartcontract owner should fail", func(t *testing.T) {
		t.Parallel()

		configKey := "reward_rate"
		newValue := "0.1"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateMinerSCConfig(t, escapedTestName(t), map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with bad config key should fail", func(t *testing.T) {
		t.Parallel()

		t.Skip("Skipping test for now as it causes miners to restart and cause test failures to others")

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "unknown_key"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "fatal:{\"error\": \"verify transaction failed\"}", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with missing keys param should fail", func(t *testing.T) {
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

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with missing values param should fail", func(t *testing.T) {
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

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys": "reward_rate",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})
}

func getMinerSCConfig(t *testing.T, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Retrieving miner config...")

	cmd := "./zwallet mn-config --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateMinerSCConfig(t *testing.T, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
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
