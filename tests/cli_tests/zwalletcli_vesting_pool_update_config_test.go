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

func TestVestingPoolUpdateConfig(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	t.Run("should allow update of max_destinations", func(t *testing.T) {
		configKey := "max_destinations"
		newValue := "4"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgBefore, _ := keyValuePairStringToMap(t, output)

		// ensure revert in config is run regardless of test result
		defer func() {
			oldValue := cfgBefore[configKey]
			output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": oldValue,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
			require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
		}()

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)
		require.Equal(t, newValue, cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	t.Run("should allow update of owner", func(t *testing.T) {
		ownerKey := "owner_id"
		newOwner := "22e412a350036944f9762a3d6b5687ee4f64d20d2cf6faf2571a490defd10f17"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwner,
		}, true)
		defer func() {
			output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		}()
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newOwner, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwner)

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": oldOwner,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
	})

	t.Run("update max_destinations to invalid value should fail", func(t *testing.T) {
		configKey := "max_destinations"
		newValue := "x"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: value x cannot be converted to time.Duration, failing to set config key max_destinations",
			output[0], strings.Join(output, "\n"))
	})

	t.Run("update owner and update max_destinations after with old owner should fail", func(t *testing.T) {
		configKey := "max_destinations"
		newValue := "x"

		ownerKey := "owner_id"
		newOwner := "22e412a350036944f9762a3d6b5687ee4f64d20d2cf6faf2571a490defd10f17"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwner,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": oldOwner,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
	})

	t.Run("update by non-smartcontract owner should fail", func(t *testing.T) {
		configKey := "max_destinations"
		newValue := "4"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, escapedTestName(t), map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with bad config key should fail", func(t *testing.T) {
		configKey := "unknown_key"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: config setting unknown_key not found", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with missing keys param should fail", func(t *testing.T) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with missing values param should fail", func(t *testing.T) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys": "max_destinations",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})
}

func getVestingPoolSCConfig(t *testing.T, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Retrieving vesting config...")

	cmd := "./zwallet vp-config --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateVestingPoolSCConfig(t *testing.T, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Updating vesting config...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zwallet vp-update-config %s --silent --wallet %s --configDir ./config --config %s",
		p,
		walletName+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
