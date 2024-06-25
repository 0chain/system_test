package cli_tests

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestMinerUpdateConfig(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("update by non-smartcontract owner should fail")

	// Test Suite I - Testing min allowances   [ Positive test cases ]
	// Max Miner Count - Test case for updating max_n to the maximum allowed value
	// 					"100" The maximum allowed value for max_n
	// Min Miner Count - Test case for updating min_n to the minimum allowed value
	// 					"3" The minimum allowed value for min_n
	// Max Sharder Count - Test case for updating max_s to the maximum allowed value
	// 					"30" The maximum allowed value for max_s
	// Min Sharder Count - Test case for updating min_s to the minimum allowed value
	//					"1" The minimum allowed value for min_s
	// Max Delegates -  Test case for updating max_delegates
	//					"200"
	// Reward Rate - Test cases for updating reward_rate
	//					"0"  Setting reward rate to zero to test close interval of range [0; 1)
	// Block Reward - Test case for updating block_reward to any flointing point value
	// 				"0.1" min value block reward
	// Share Ratio - Test case for updating share_ratio
	//				"0" Setting reward rate to 0  to test close interval of range [0; 1)
	// Reward Decline Rate - Test case for updating reward_decline_rate to the minimum allowed value
	//				"0" The minimum allowed value for reward_decline_rate
	// DKG - Test case for updating t_percent to a valid value
	// 				"0.66" A min value for t_percent
	// DKG - Test case for updating k_percent to a valid value
	//				"0.75" A min value for k_percent
	// DKG - Test case for updating x_percent to a valid value
	//				"0.70" A min value for x_percent
	// ETC - Test case for updating min_stake to min value
	//				"0.0" min value for min_stake
	// ETC - Test case for updating max_stake to max value
	//				"20000.0" max value for max_stake
	// ETC - Test case for updating min_stake_per_delegate
	//				"1"
	// ETC - Test case for updating start_rounds
	//				"50"
	// ETC - Test case for updating contribute_rounds
	//				"50"
	// ETC - Test case for updating share_rounds
	//				"50"
	// ETC - Test case for updating publish_rounds
	//				"50"
	// ETC - Test case for updating wait_rounds
	//				"50"
	// Max Charge by Generator - Test case for updating max_charge
	//				"0.5" max charge
	// Epoch - Test case for epoch
	//  			"125000000" # rounds
	// Reward Decline Rate - Test case for updating reward_decline_rate
	//				"0" Setting reward decline rate to 0  to test close interval of range [0; 1)
	// Reward Round Frequency - Test case for updating reward_round_frequency
	//				"250"
	// Num miner delegates rewarded - Test case for updating num_miner_delegates_rewarded
	// 				"10"
	// Num sharders rewarded each round - Test case for updating num_sharders_rewarded
	//				"1"
	// Num sharder delegates to get paid each round when paying fees and rewards - Test case for updating num_sharder_delegates_rewarded
	//				"5"
	// Test case for cost of adding miner
	//	"361" // A valid cost value for adding a miner
	// Test case for cost of adding sharder
	//	"331" // A valid cost value for adding a sharder
	// Test case for cost of deleting miner
	//	newValue := "484" // A valid cost value for deleting a miner
	// Test case for cost of deleting sharder
	// newValue := "335" // A valid cost value for deleting a sharder

	t.RunSequentiallyWithTimeout("successful update of config to minimum allowed value", 200*time.Minute, func(t *test.SystemTest) {
		// Get storage config and store them for later comparison

		keys := []string{
			"max_n",
			"min_n",
			"max_s",
			"min_s",
			"max_delegates",
			"reward_rate",
			"block_reward",
			"share_ratio",
			"reward_decline_rate",
			"min_stake",
			"max_stake",
			"min_stake_per_delegate",
			"max_charge",
			"epoch",
			"reward_decline_rate",
			"reward_round_frequency",
			"num_miner_delegates_rewarded",
			"num_sharders_rewarded",
			"num_sharder_delegates_rewarded",
			"cost.add_miner",
			"cost.add_sharder",
			"cost.miner_health_check",
			"cost.sharder_health_check",
			"cost.contributempk",
			"cost.sharesignsorshares",
			"cost.wait",
			"cost.update_globals",
			"cost.update_settings",
			"cost.update_miner_settings",
			"cost.update_sharder_settings",
			"cost.payfees",
			"cost.feespaid",
			"cost.mintedtokens",
			"cost.addtodelegatepool",
			"cost.deletefromdelegatepool",
			"cost.sharder_keep",
			"cost.kill_miner",
			"cost.kill_sharder",
		}
		values := []string{
			"100",
			"3",
			"30",
			"1",
			"200",
			"0",
			"0.1",
			"0",
			"0",
			"0",
			"20000",
			"1",
			"0.5",
			"125000000",
			"0",
			"250",
			"10",
			"1",
			"5",
			"361",
			"331",
			"149",
			"400",
			"200",
			"509",
			"100",
			"269",
			"136",
			"137",
			"134",
			"1356",
			"100",
			"100",
			"186",
			"150",
			"211",
			"146",
			"140",
		}

		// Save original ( prev ) config values in order to restore system to its previous state
		configMapBefore := getMinerScConfigsForKeys(t, configPath, keys)
		var beforeKeys, beforeValues []string
		for k, v := range configMapBefore {
			beforeKeys = append(beforeKeys, k)
			beforeValues = append(beforeValues, v)
		}

		t.Logf("original (prior to update ) values")
		t.Logf("***start***")
		t.Logf("Keys : %s", beforeKeys)
		t.Logf("Values : %s", beforeValues)
		t.Logf("***end***")

		t.Logf("update and verify updated values")

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   strings.Join(keys, ","),
			"values": strings.Join(values, ","),
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to config parameters succeeded with min values")

		updatedConfigMap := getMinerScConfigsForKeys(t, configPath, keys)

		// Assert that each updated value matches the expected value
		for i, key := range keys {
			expectedValue := values[i]
			actualValue, exists := updatedConfigMap[key]
			t.Logf("Config parameter to be compared: %s", key)
			t.Logf("Expected config: %s", expectedValue)
			t.Logf("Actual config: %s", actualValue)
			require.True(t, exists, fmt.Sprintf("Config key %s does not exist", key))
			require.Equal(t, expectedValue, actualValue, fmt.Sprintf("Config key %s does not match expected value. Expected: %s, Got: %s", key, expectedValue, actualValue))
		}

		t.Logf("update and verify original values")

		// Update config to previous values and then compare them
		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   strings.Join(beforeKeys, ","),
			"values": strings.Join(beforeValues, ","),
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to config parameters succeeded with min values")

		prevConfigMap := getMinerScConfigsForKeys(t, configPath, keys)

		// Assert that each updated value matches the original ( prev ) config values
		for i, key := range beforeKeys {
			expectedValue := beforeValues[i]
			actualValue, exists := prevConfigMap[key]
			t.Logf("Config parameter to be compared: %s", key)
			t.Logf("Expected config: %s", expectedValue)
			t.Logf("Actual config: %s", actualValue)
			require.True(t, exists, fmt.Sprintf("Config key %s does not exist", key))
			require.Equal(t, expectedValue, actualValue, fmt.Sprintf("Config key %s does not match expected value. Expected: %s, Got: %s", key, expectedValue, actualValue))
		}
	})

	// Test Suite II - Testing mid-value allowances  [ Positive test cases ]
	// Reward Rate - Test cases for updating reward_rate
	//					"0.5"  Setting reward rate to mid-interval value to test range [0; 1)
	// Block Reward - Test case for updating block_reward to any flointing point value
	// 				"0.5" mid value block reward
	// Share Ratio - Test case for updating share_ratio
	//				"0.5" Setting reward rate to mid-interval value to test rangerange [0; 1)
	// Reward Decline Rate - Test case for updating reward_decline_rate to the mid value
	//				"0.5" The mid value for reward_decline_rate to test range [0; 1)

	t.RunSequentially("successful update of config to mid-level allowed value", func(t *test.SystemTest) {
		keys := []string{"reward_rate", "block_reward", "share_ratio", "reward_decline_rate"}
		values := []string{"0.5", "0.5", "0.5", "0.5"}

		// Save original ( prev ) config values in order to restore system to its previous state
		configMapBefore := getMinerScConfigsForKeys(t, configPath, keys)
		var beforeKeys, beforeValues []string
		for k, v := range configMapBefore {
			beforeKeys = append(beforeKeys, k)
			beforeValues = append(beforeValues, v)
		}

		t.Logf("update and verify updated values")

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   strings.Join(keys, ","),
			"values": strings.Join(values, ","),
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to config parameters succeeded with min values")

		updatedConfigMap := getMinerScConfigsForKeys(t, configPath, keys)

		// Assert that each updated value matches the expected value
		for i, key := range keys {
			expectedValue := values[i]
			actualValue, exists := updatedConfigMap[key]
			t.Logf("Config parameter to be compared: %s", key)
			t.Logf("Expected config: %s", expectedValue)
			t.Logf("Actual config: %s", actualValue)
			require.True(t, exists, fmt.Sprintf("Config key %s does not exist", key))
			require.Equal(t, expectedValue, actualValue, fmt.Sprintf("Config key %s does not match expected value. Expected: %s, Got: %s", key, expectedValue, actualValue))
		}

		t.Logf("update and verify original values")

		// Update config to previous values and then compare them
		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   strings.Join(beforeKeys, ","),
			"values": strings.Join(beforeValues, ","),
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to config parameters succeeded with min values")

		prevConfigMap := getMinerScConfigsForKeys(t, configPath, keys)

		// Assert that each updated value matches the original ( prev ) config values
		for i, key := range beforeKeys {
			expectedValue := beforeValues[i]
			actualValue, exists := prevConfigMap[key]
			t.Logf("Config parameter to be compared: %s", key)
			t.Logf("Expected config: %s", expectedValue)
			t.Logf("Actual config: %s", actualValue)
			require.True(t, exists, fmt.Sprintf("Config key %s does not exist", key))
			require.Equal(t, expectedValue, actualValue, fmt.Sprintf("Config key %s does not match expected value. Expected: %s, Got: %s", key, expectedValue, actualValue))
		}
	})

	// Test Suite III - Testing max allowances  [ Positive test cases ]
	// Reward Rate - Test cases for updating reward_rate
	//					"0.999999" // A value just under 1 for the exclusive range
	// Block Reward - Test case for updating block_reward to any flointing point value
	// 				"0.9" max value block reward
	// Share Ratio - Test case for updating share_ratio
	//				"0.999999" // A value just under 1 for the exclusive range
	// Reward Decline Rate - Test case for updating reward_decline_rate to the minimum allowed value
	//				"0.999999" // A value just under 1 for the exclusive range

	t.RunSequentially("successful update of config to maximum allowed value", func(t *test.SystemTest) {
		keys := []string{"reward_rate", "block_reward", "share_ratio", "reward_decline_rate"}
		values := []string{"0.999999", "0.9", "0.999999", "0.999999"}

		// Save original ( prev ) config values in order to restore system to its previous state
		configMapBefore := getMinerScConfigsForKeys(t, configPath, keys)
		var beforeKeys, beforeValues []string
		for k, v := range configMapBefore {
			beforeKeys = append(beforeKeys, k)
			beforeValues = append(beforeValues, v)
		}

		t.Logf("update and verify updated values")

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   strings.Join(keys, ","),
			"values": strings.Join(values, ","),
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to config parameters succeeded with min values")

		updatedConfigMap := getMinerScConfigsForKeys(t, configPath, keys)

		// Assert that each updated value matches the expected value
		for i, key := range keys {
			expectedValue := values[i]
			actualValue, exists := updatedConfigMap[key]
			t.Logf("Config parameter to be compared: %s", key)
			t.Logf("Expected config: %s", expectedValue)
			t.Logf("Actual config: %s", actualValue)
			require.True(t, exists, fmt.Sprintf("Config key %s does not exist", key))
			require.Equal(t, expectedValue, actualValue, fmt.Sprintf("Config key %s does not match expected value. Expected: %s, Got: %s", key, expectedValue, actualValue))
		}

		t.Logf("update and verify original values")

		// Update config to previous values and then compare them
		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   strings.Join(beforeKeys, ","),
			"values": strings.Join(beforeValues, ","),
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to config parameters succeeded with min values")

		prevConfigMap := getMinerScConfigsForKeys(t, configPath, keys)

		// Assert that each updated value matches the original ( prev ) config values
		for i, key := range beforeKeys {
			expectedValue := beforeValues[i]
			actualValue, exists := prevConfigMap[key]
			t.Logf("Config parameter to be compared: %s", key)
			t.Logf("Expected config: %s", expectedValue)
			t.Logf("Actual config: %s", actualValue)
			require.True(t, exists, fmt.Sprintf("Config key %s does not exist", key))
			require.Equal(t, expectedValue, actualValue, fmt.Sprintf("Config key %s does not match expected value. Expected: %s, Got: %s", key, expectedValue, actualValue))
		}
	})

	// Test Suite IV - Testing invalid values [ Negative test cases ]
	// Reward Rate - Test cases for updating reward_rate
	//					"-1" // An invalid value of 1 for the exclusive range
	// Block Reward - Test case for updating block_reward to any flointing point value
	// 				"-1"
	// Share Ratio - Test case for updating share_ratio
	//				"-1" // An invalid value 1 for the exclusive range
	// Reward Decline Rate - Test case for updating reward_decline_rate to the minimum allowed value
	//				"-1" // A value of 1 for the exclusive range

	t.RunSequentially("unsuccessful update of config to invalid values", func(t *test.SystemTest) {
		keys := []string{"reward_rate", "block_reward", "share_ratio", "reward_decline_rate"}
		values := []string{"-1", "-1", "-1", "-1"}

		// Convert slices to comma-separated strings
		keysStr := strings.Join(keys, ",")
		valuesStr := strings.Join(values, ",")

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   keysStr,
			"values": valuesStr,
		}, false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, !isUpdateSuccess(output), "Update to config parameters failed with invalid values")
	})

	/* Exclusive values ( upper bound ) tests failing for below parameters - https://github.com/0chain/0chain/issues/3168 */
	// Test Suite V - Testing out of bounds [ Negative test cases ]
	// Reward Rate - Test cases for updating reward_rate
	//					"1" // A value of 1 for the exclusive range
	// Share Ratio - Test case for updating share_ratio
	//				"1" // A value of 1 for the exclusive range
	// Reward Decline Rate - Test case for updating reward_decline_rate to the minimum allowed value
	//				"1" // A value of 1 for the exclusive range

	t.RunSequentially("unsuccessful update of config to out of bounds value", func(t *test.SystemTest) {
		t.Skip("skip till the issue is fixed")

		keys := []string{"reward_rate", "share_ratio", "reward_decline_rate"}
		values := []string{"1", "1", "1"}

		// Convert slices to comma-separated strings
		keysStr := strings.Join(keys, ",")
		valuesStr := strings.Join(values, ",")

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   keysStr,
			"values": valuesStr,
		}, false)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, !isUpdateSuccess(output), "Update to config parameters failed with out of bounds values")
	})

	t.RunSequentially("update by non-smartcontract owner should fail", func(t *test.SystemTest) {
		configKey := "reward_rate"
		newValue := "0.1"

		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		output, err := updateMinerSCConfig(t, escapedTestName(t), map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update with bad config key should fail", func(t *test.SystemTest) {
		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "unknown_key"

		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unsupported key unknown_key", output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update with missing keys param should fail", func(t *test.SystemTest) {
		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update with missing values param should fail", func(t *test.SystemTest) {
		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys": "reward_rate",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})

	// Max Miner Count - Test case for updating max_n to the maximum allowed value
	t.RunSequentially("successful update of max_n to maximum allowed value", func(t *test.SystemTest) {
		configKey := "max_n"
		maxValue := "100" // The maximum allowed value for max_n

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": maxValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to max_n did not succeed with maximum value")
	})

	// Min Miner Count - Test case for updating min_n to the minimum allowed value
	t.RunSequentially("successful update of min_n to minimum allowed value", func(t *test.SystemTest) {
		configKey := "min_n"
		minValue := "3" // The minimum allowed value for min_n

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": minValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to min_n did not succeed with minimum value")
	})

	// Max Sharder Count - Test case for updating max_s to the maximum allowed value
	t.RunSequentially("successful update of max_s to maximum allowed value", func(t *test.SystemTest) {
		configKey := "max_s"
		maxValue := "30" // The maximum allowed value for max_s

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": maxValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to max_s did not succeed with maximum value")
	})

	// Min Sharder Count - Test case for updating min_s to the minimum allowed value
	t.RunSequentially("successful update of min_s to minimum allowed value", func(t *test.SystemTest) {
		configKey := "min_s"
		minValue := "1" // The minimum allowed value for min_s

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": minValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to min_s did not succeed with minimum value")
	})

	// Reward Rate - Test cases for updating reward_rate
	t.RunSequentially("successful update of reward_rate to zero", func(t *test.SystemTest) {
		configKey := "reward_rate"
		newValue := "0" // Setting reward rate to zero to test open interval of range [0; 1)

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to reward_rate did not succeed with value zero")
	})

	// Reward Rate - Test cases for updating reward_rate
	t.RunSequentially("successful update of reward_rate to 0.5", func(t *test.SystemTest) {
		configKey := "reward_rate"
		newValue := "0.5" // Setting reward rate to 0.5  to test mid range [0; 1)

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to reward_rate did not succeed with value 0.5")
	})

	// Block Reward - Test case for updating block_reward
	t.RunSequentially("successful update of block_reward", func(t *test.SystemTest) {
		configKey := "block_reward"
		newValue := "0.9" // Flointing point value block reward

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to block_reward did not succeed with specified value")
	})
       

}

func getMinerSCConfig(t *test.SystemTest, cliConfigFilename string, retry bool) ([]string, error) {
	t.Logf("Retrieving miner config...")

	cmd := "./zwallet mn-config --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateMinerSCConfig(t *test.SystemTest, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
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

// isUpdateSuccess checks if the output contains a "updated" message indicating that the update was successful.
func isUpdateSuccess(output []string) bool {
	successMsg := "minersc smart contract settings updated"
	for _, line := range output {
		if strings.Contains(line, successMsg) {
			return true
		}
	}
	return false
}

// func updateMinerSCConfig(t *test.SystemTest, walletName string, param map[string]interface{}, nonce int64, retry bool) ([]string, error) {
// 	t.Logf("Updating miner config...")
// 	p := createParams(param)
// 	cmd := fmt.Sprintf(
// 		"./zwallet mn-update-config %s --silent --wallet %s --configDir ./config --config %s",
// 		p,
// 		walletName+"_wallet.json",
// 		configPath,
// 	)

// 	if retry {
// 		return cliutils.RunCommand(t, cmd, 3, time.Second*5)
// 	} else {
// 		return cliutils.RunCommandWithoutRetry(cmd)
// 	}
// }

func getMinerScConfigsForKeys(t *test.SystemTest, configPath string, keys []string) map[string]string {
	updatedConfig, err := getMinerSCConfig(t, configPath, true)
	require.Nil(t, err)
	t.Logf("!!!!! error msg : %s", err)
	t.Logf("Updated config string : %s", updatedConfig)

	// Convert updatedConfig to a map for easier comparison
	updatedConfigMap := make(map[string]string)
	for _, key := range keys {
		// Find the key in the output and extract the value that follows it
		r := regexp.MustCompile(fmt.Sprintf(`\b%s\s+(\S+)`, key))
		matches := r.FindStringSubmatch(strings.Join(updatedConfig, " "))
		if len(matches) >= 2 {
			updatedConfigMap[key] = matches[1]
		} else {
			log.Println("No match found for key: ", key)
		}
	}
	return updatedConfigMap
}
