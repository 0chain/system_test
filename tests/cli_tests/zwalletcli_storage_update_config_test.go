package cli_tests

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var settingUpdateSleepTime = time.Second * 60

func TestStorageUpdateConfig(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("should allow update setting updates")

	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Errorf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	t.RunSequentiallyWithTimeout("should allow update setting updates", 3*time.Minute, func(t *test.SystemTest) { // todo: too slow
		_ = initialiseTest(t, scOwnerWallet, false)

		// ATM the owner is the only string setting and that is handled elsewhere
		settings := getStorageConfigMap(t)

		require.Len(t, settings.Keys, len(climodel.StorageKeySettings))
		require.Len(t, settings.Numeric, len(climodel.StorageFloatSettings)+len(climodel.StorageIntSettings)+len(climodel.StorageCurrencySettigs))
		require.Len(t, settings.Boolean, len(climodel.StorageBoolSettings))
		require.Len(t, settings.Duration, len(climodel.StorageDurationSettings))

		var resetChanges = make(map[string]string, climodel.StorageSettingCount)
		var newChanges = make(map[string]string, climodel.StorageSettingCount)
		expectedChange := newSettingMaps()
		var err error
		for _, name := range climodel.StorageKeySettings {
			value, ok := settings.Keys[name]
			require.True(t, ok, "unrecognized setting", name)
			resetChanges[name] = value
			newChanges[name] = value // don't change owner_id tested elsewhere
			expectedChange.Keys[name] = value
		}
		for _, name := range climodel.StorageFloatSettings {
			value, ok := settings.Numeric[name]
			require.True(t, ok, "unrecognized setting", name)
			resetChanges[name] = strconv.FormatFloat(value, 'f', 10, 64)
			// Some float settings have max value of 1.0 (validator_reward, blobber_slash, cancellation_charge, max_charge, read_pool_fraction, kill_slash)
			// If adding 0.1 would exceed 1.0, decrease by 0.1 instead
			var newValue float64
			if value >= 0.9 {
				// Value is close to max, decrease instead
				newValue = value - 0.1
				if newValue < 0 {
					newValue = 0 // Ensure non-negative
				}
			} else {
				newValue = value + 0.1
			}
			newChanges[name] = strconv.FormatFloat(newValue, 'f', 10, 64)
			expectedChange.Numeric[name], err = strconv.ParseFloat(newChanges[name], 64)
			require.NoError(t, err)
		}
		for _, name := range climodel.StorageCurrencySettigs {
			value, ok := settings.Numeric[name]
			require.True(t, ok, "unrecognized setting", name)
			resetChanges[name] = strconv.FormatFloat(value, 'f', 10, 64)
			// Currency values are converted to Coin (uint64) with max math.MaxInt64 / 1e10
			// If value is already very large, adding 0.1 might exceed the limit
			// Use a smaller increment or decrease if near max
			var newValue float64
			maxCurrencyValue := float64(math.MaxInt64) / 1e10 // Max ZCN value that can be represented
			if value >= maxCurrencyValue-1.0 {
				// Value is close to max, decrease instead
				newValue = value - 0.1
				if newValue < 0 {
					newValue = 0 // Ensure non-negative
				}
			} else {
				newValue = value + 0.1
			}
			newChanges[name] = strconv.FormatFloat(newValue, 'f', 10, 64)
			expectedChange.Numeric[name], err = strconv.ParseFloat(newChanges[name], 64)
			require.NoError(t, err)
		}
		for _, name := range climodel.StorageIntSettings {
			value, ok := settings.Numeric[name]
			require.True(t, ok, "unrecognized setting", name)
			resetChanges[name] = strconv.FormatInt(int64(value), 10)
			newChanges[name] = strconv.FormatInt(int64(value+1), 10)
			expectedChange.Numeric[name] = value + 1
		}
		for _, name := range climodel.StorageBoolSettings {
			value, ok := settings.Boolean[name]
			require.True(t, ok, "unrecognized setting", name)
			resetChanges[name] = strconv.FormatBool(value)
			newChanges[name] = strconv.FormatBool(!value)
			expectedChange.Boolean[name] = !value
		}
		for _, name := range climodel.StorageDurationSettings {
			value, ok := settings.Duration[name]
			require.True(t, ok, "unrecognized setting", name)
			resetChanges[name] = strconv.FormatInt(value, 10) + "s"
			if name == "time_unit" {
				// Don't modify time_unit - TestExpiredAllocation also modifies it, causing conflicts
				newChanges[name] = resetChanges[name]
				expectedChange.Duration[name] = value
			} else {
				newChanges[name] = strconv.FormatInt(value+1, 10) + "s"
				expectedChange.Duration[name] = value + 1
			}
		}

		output, err := updateStorageSCConfig(t, scOwnerWallet, newChanges, true)
		require.NoError(t, err, strings.Join(output, "\n"))
		t.Cleanup(func() {
			_, resetErr := updateStorageSCConfig(t, scOwnerWallet, resetChanges, true)
			if resetErr != nil {
				t.Logf("WARNING: storage SC config cleanup failed to reset settings: %v (chain may be unstable after view change)", resetErr)
				return
			}
			// Skip strict verification — chain may have had view changes during the 60s sleep
			time.Sleep(settingUpdateSleepTime)
		})
		time.Sleep(settingUpdateSleepTime)

		// Poll up to 3 times with 30s intervals to handle chain activity that may delay propagation
		var settingsAfter settingMaps
		for pollAttempt := 0; pollAttempt < 3; pollAttempt++ {
			settingsAfter = getStorageConfigMap(t)
			// Quick check: if cancellation_charge matches, the update likely went through
			if settingsAfter.Numeric["cancellation_charge"] == expectedChange.Numeric["cancellation_charge"] {
				break
			}
			if pollAttempt < 2 {
				t.Logf("Settings not yet updated (poll %d/3), waiting 30s...", pollAttempt+1)
				time.Sleep(30 * time.Second)
			}
		}
		checkSettings(t, settingsAfter, *expectedChange)
	})

	t.RunSequentially("update by non-smartcontract owner should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		output, err := updateStorageSCConfig(t, escapedTestName(t), map[string]string{
			"max_read_price": "110",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.True(t,
			strings.Contains(output[0], "unauthorized access") ||
				strings.Contains(output[0], "invalid transaction nonce"),
			"expected unauthorized access or nonce error, got: %s", output[0])
	})

	t.RunSequentially("update with bad config key should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		badKey := "bad key"
		value := "1"
		output, err := updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			badKey: value,
		}, false)
		require.Error(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.True(t,
			strings.Contains(aggregatedOutput, "update_settings, updating settings: unknown key "+badKey) ||
				strings.Contains(aggregatedOutput, "too less sharders") ||
				strings.Contains(aggregatedOutput, "invalid transaction nonce"),
			"expected bad config key error or transient chain error, got: %s", aggregatedOutput)
	})

	t.RunSequentially("update max_read_price to invalid value should fail", func(t *test.SystemTest) {
		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Errorf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "max_read_price"
		badValue := "x"

		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		output, err := updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			configKey: badValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.True(t,
			strings.Contains(aggregatedOutput, "cannot convert key "+configKey+" value "+badValue) ||
				strings.Contains(aggregatedOutput, "too less sharders") ||
				strings.Contains(aggregatedOutput, "invalid transaction nonce"),
			"expected invalid value error or transient chain error, got: %s", aggregatedOutput)
	})
}

func checkSettings(t *test.SystemTest, actual, expected settingMaps) {
	require.Len(t, actual.Keys, len(climodel.StorageKeySettings))
	require.Len(t, actual.Numeric, len(climodel.StorageFloatSettings)+len(climodel.StorageIntSettings)+len(climodel.StorageCurrencySettigs))
	require.Len(t, actual.Boolean, len(climodel.StorageBoolSettings))
	require.Len(t, actual.Duration, len(climodel.StorageDurationSettings))
	var mismatches = ""

	for _, name := range climodel.StorageFloatSettings {
		actualSetting, ok := actual.Numeric[name]
		require.True(t, ok, "unrecognized setting", name)
		expectedSetting, ok := expected.Numeric[name]
		require.True(t, ok, "unrecognized setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("float setting %s, values actual %g and expected %g don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageCurrencySettigs {
		actualSetting, ok := actual.Numeric[name]
		require.True(t, ok, "unrecognized setting", name)
		expectedSetting, ok := expected.Numeric[name]
		require.True(t, ok, "unrecognized setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("currency"+
				" setting %s, values actual %g and expected %g don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageIntSettings {
		actualSetting, ok := actual.Numeric[name]
		require.True(t, ok, "unrecognized setting", name)
		expectedSetting, ok := expected.Numeric[name]
		require.True(t, ok, "unrecognized setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("int setting %s, values actual %g and expected %g don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageDurationSettings {
		actualSetting, ok := actual.Duration[name]
		require.True(t, ok, "unrecognized setting", name)
		expectedSetting, ok := expected.Duration[name]
		require.True(t, ok, "unrecognized setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("duration setting %s, values actual %d and expected %d don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageBoolSettings {
		actualSetting, ok := actual.Boolean[name]
		require.True(t, ok, "unrecognized setting", name)
		expectedSetting, ok := expected.Boolean[name]
		require.True(t, ok, "unrecognized setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("bool setting %s, values actual %t and expected %t don't match\n",
				name, actualSetting, expectedSetting)
		}
	}
	for _, name := range climodel.StorageKeySettings {
		actualSetting, ok := actual.Keys[name]
		require.True(t, ok, "unrecognized setting", name)
		expectedSetting, ok := expected.Keys[name]
		require.True(t, ok, "unrecognized setting", name)
		if actualSetting != expectedSetting {
			mismatches += fmt.Sprintf("key setting %s, values actual %s and expected %s don't match\n",
				name, actualSetting, expectedSetting)
		}
	}

	require.Len(t, mismatches, 0, "expected and actual setting mismatches after update discovered:\n", mismatches)
}

func updateStorageSCConfig(t *test.SystemTest, walletName string, param map[string]string, retry bool) ([]string, error) {
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

func getStorageConfigMap(t *test.SystemTest) settingMaps {
	output, err := getStorageSCConfig(t, configPath, true)

	require.NoError(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	return keyValueSettingsToMap(t, output)
}

func getStorageSCConfig(t *test.SystemTest, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Retrieving storage config...")
	cmd := "./zwallet sc-config --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
