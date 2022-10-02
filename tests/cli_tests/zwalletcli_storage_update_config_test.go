package cli_tests

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var settings = []string{}

func TestStorageUpdateConfig(t *testing.T) {
	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	t.Run("should allow update setting updates", func(t *testing.T) {
		_ = initialiseTest(t, scOwnerWallet, false)

		// ATM the owner is the only string setting and that is handled elsewhere
		settings := getStorageConfigMap(t)

		require.Len(t, settings.Keys, len(climodel.StorageKeySettings))
		require.Len(t, settings.Numeric, len(climodel.StorageFloatSettings)+len(climodel.StorageIntSettings))
		require.Len(t, settings.Boolean, len(climodel.StorageBoolSettings))
		require.Len(t, settings.Duration, len(climodel.StorageDurationSettings))

		var resetChanges = make(map[string]string, climodel.StorageSettingCount)
		var newChanges = make(map[string]string, climodel.StorageSettingCount)
		expectedChange := newSettingMaps()
		var err error
		for _, name := range climodel.StorageKeySettings {
			value, ok := settings.Keys[name]
			require.True(t, ok, "unrecognised setting", name)
			resetChanges[name] = value
			newChanges[name] = value // don't change owner_id tested elsewhere
			expectedChange.Keys[name] = value
		}
		for _, name := range climodel.StorageFloatSettings {
			value, ok := settings.Numeric[name]
			require.True(t, ok, "unrecognised setting", name)
			resetChanges[name] = strconv.FormatFloat(value, 'f', 10, 64)
			newChanges[name] = strconv.FormatFloat(value+0.1, 'f', 10, 64)
			expectedChange.Numeric[name] = value + 0.1
		}
		for _, name := range climodel.StorageIntSettings {
			value, ok := settings.Numeric[name]
			require.True(t, ok, "unrecognised setting", name)
			resetChanges[name] = strconv.FormatInt(int64(value), 10)
			newChanges[name] = strconv.FormatInt(int64(value+1), 10)
			expectedChange.Numeric[name] = value + 1
			fmt.Println("setting name", name,
				"reset", strconv.FormatInt(int64(value), 10),
				"new", strconv.FormatInt(int64(value+1), 10),
				"expected", value+1,
			)
		}
		for _, name := range climodel.StorageBoolSettings {
			value, ok := settings.Boolean[name]
			require.True(t, ok, "unrecognised setting", name)
			resetChanges[name] = strconv.FormatBool(value)
			newChanges[name] = strconv.FormatBool(!value)
			expectedChange.Boolean[name] = !value
		}
		for _, name := range climodel.StorageDurationSettings {
			value, ok := settings.Duration[name]
			require.True(t, ok, "unrecognised setting", name)
			resetChanges[name] = strconv.FormatInt(value, 10) + "s"
			newChanges[name] = strconv.FormatInt(value+1, 10) + "s"
			expectedChange.Duration[name] = value + 1
		}

		fmt.Println("change storage sc settings")
		output, err := updateStorageSCConfig(t, scOwnerWallet, newChanges, true)
		require.NoError(t, err, strings.Join(output, "\n"))

		time.Sleep(time.Second * 10)

		settingsAfter := getStorageConfigMap(t)
		checkSettings(t, settingsAfter, *expectedChange)

		fmt.Println("reset storage sc settings to original values")
		_, err = updateStorageSCConfig(t, scOwnerWallet, resetChanges, true)
		require.NoError(t, err, strings.Join(output, "\n"))

		time.Sleep(time.Second * 10)

		settingsReset := getStorageConfigMap(t)
		checkSettings(t, settingsReset, settings)
		fmt.Println("piers done")

	})
	/*
		t.Run("should allow update of max_read_price", func(t *testing.T) {
			t.Skip("Skip till fixed...")
			configKey := "max_read_price"
			newValue := 99

			// unused wallet, just added to avoid having the creating new wallet outputs
			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			// register SC owner wallet
			output, err = registerWalletForName(t, configPath, scOwnerWallet)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = getStorageSCConfig(t, configPath, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Greater(t, len(output), 0, strings.Join(output, "\n"))

			cfgBefore, _, _, _ := keyValuePairStringToMap(output)

			t.Cleanup(func() {
				oldValue := cfgBefore[configKey]
				output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
					"keys":   configKey,
					"values": oldValue,
				}, true)
				require.Nil(t, err, strings.Join(output, "\n"))
				require.Len(t, output, 2, strings.Join(output, "\n"))
				require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
				require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
			})

			output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": newValue,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
			require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

			cliutils.Wait(t, 5*time.Second)

			output, err = getStorageSCConfig(t, configPath, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Greater(t, len(output), 0, strings.Join(output, "\n"))

			cfgAfter, _, _, _ := keyValuePairStringToMap(output)

			require.Equal(t, fmt.Sprint(newValue), cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

			// test transaction to verify chain is still working
			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
		})

		t.Run("update by non-smartcontract owner should fail", func(t *testing.T) {
			t.Skip("piers")
			configKey := "max_read_price"
			newValue := "110"

			// unused wallet, just added to avoid having the creating new wallet outputs
			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = updateStorageSCConfig(t, escapedTestName(t), map[string]interface{}{
				"keys":   configKey,
				"values": newValue,
			}, false)
			require.NotNil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1, strings.Join(output, "\n"))
			require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
		})

		t.Run("update with bad config key should fail", func(t *testing.T) {
			t.Skip("piers")
			configKey := "unknown_key"

			// unused wallet, just added to avoid having the creating new wallet outputs
			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": 1,
			}, false)
			require.NotNil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1, strings.Join(output, "\n"))
			require.Equal(t, "update_settings, updating settings: unknown key unknown_key, can't set value 1", output[0], strings.Join(output, "\n"))
		})

		t.Run("update with missing keys param should fail", func(t *testing.T) {
			t.Skip("piers")
			// unused wallet, just added to avoid having the creating new wallet outputs
			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
				"values": 1,
			}, false)
			require.NotNil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1, strings.Join(output, "\n"))
			require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
		})

		t.Run("update with missing values param should fail", func(t *testing.T) {
			t.Skip("piers")
			// unused wallet, just added to avoid having the creating new wallet outputs
			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys": "max_read_price",
			}, false)
			require.NotNil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1, strings.Join(output, "\n"))
			require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
		})

		t.Run("update max_read_price to invalid value should fail", func(t *testing.T) {
			t.Skip("piers")
			t.Parallel()

			if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
				t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
			}

			configKey := "max_read_price"
			newValue := "x"

			// unused wallet, just added to avoid having the creating new wallet outputs
			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			// register SC owner wallet
			output, err = registerWalletForName(t, configPath, scOwnerWallet)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": newValue,
			}, false)
			require.NotNil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1, strings.Join(output, "\n"))
			require.Equal(t, "update_settings, updating settings: cannot convert key max_read_price value x to state.balance: strconv.ParseFloat: parsing \\\"x\\\": invalid syntax", output[0], strings.Join(output, "\n"))
		})
	*/
}

func checkSettings(t *testing.T, actual, expected settingMaps) {
	require.Len(t, actual.Keys, len(climodel.StorageKeySettings))
	require.Len(t, actual.Numeric, len(climodel.StorageFloatSettings)+len(climodel.StorageIntSettings))
	require.Len(t, actual.Boolean, len(climodel.StorageBoolSettings))
	require.Len(t, actual.Duration, len(climodel.StorageDurationSettings))
	var mismatches string

	for _, name := range climodel.StorageFloatSettings {
		actualSetting, ok := actual.Numeric[name]
		require.True(t, ok, "unrecognised setting", name)
		expectedSetting, ok := expected.Numeric[name]
		require.True(t, ok, "unrecognised setting", name)
		// add in the check for floats after 0chain fix
		if actualSetting != expectedSetting {
			fmt.Println(fmt.Sprintf("float setting %s, values actual %g and expected %g don't match",
				name, actualSetting, expectedSetting))
		}
	}
	for _, name := range climodel.StorageIntSettings {
		actualSetting, ok := actual.Numeric[name]
		require.True(t, ok, "unrecognised setting", name)
		expectedSetting, ok := expected.Numeric[name]
		require.True(t, ok, "unrecognised setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("int setting %s, values actual %g and expected %g don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageDurationSettings {
		actualSetting, ok := actual.Duration[name]
		require.True(t, ok, "unrecognised setting", name)
		expectedSetting, ok := expected.Duration[name]
		require.True(t, ok, "unrecognised setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("duration setting %s, values actual %d and expected %d don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageBoolSettings {
		actualSetting, ok := actual.Boolean[name]
		require.True(t, ok, "unrecognised setting", name)
		expectedSetting, ok := expected.Boolean[name]
		require.True(t, ok, "unrecognised setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("bool setting %s, values actual %t and expected %t don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageKeySettings {
		actualSetting, ok := actual.Keys[name]
		require.True(t, ok, "unrecognised setting", name)
		expectedSetting, ok := expected.Keys[name]
		require.True(t, ok, "unrecognised setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("key setting %s, values actual %s and expected %s don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	require.Len(t, mismatches, 0, mismatches)
}

func updateStorageSCConfig(t *testing.T, walletName string, param map[string]string, retry bool) ([]string, error) {
	t.Logf("Updating storage config...")
	p := createKeyValueParams(param)
	cmd := fmt.Sprintf(
		"./zwallet sc-update-config %s --silent --wallet %s --configDir ./config --config %s",
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

func getStorageConfigMap(t *testing.T) settingMaps {
	output, err := getStorageSCConfig(t, configPath, true)

	require.NoError(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	return keyValueSettingsToMap(output)
}

func getStorageSCConfig(t *testing.T, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Retrieving storage config...")
	cmd := "./zwallet sc-config --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
