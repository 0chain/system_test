package cli_tests

import (
	"fmt"
	"os"
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

	// Share Ratio - Test case for updating share_ratio to the minimum allowed value
	t.RunSequentially("successful update of share_ratio to minimum allowed value", func(t *test.SystemTest) {
		configKey := "share_ratio"
		minValue := "0" // The minimum allowed value for share_ratio

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": minValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to share_ratio did not succeed with minimum value")
	})

	// Share Ratio - Test case for updating share_ratio to a mid-interval value
	t.RunSequentially("successful update of share_ratio to mid-interval value", func(t *test.SystemTest) {
		configKey := "share_ratio"
		midValue := "0.5" // A valid mid-interval value for share_ratio

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": midValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to share_ratio did not succeed with mid-interval value")
	})

	//  Share Ratio - Test case for updating share_ratio to the maximum allowed value, exclusive of 1
	t.RunSequentially("successful update of share_ratio to maximum allowed value", func(t *test.SystemTest) {
		configKey := "share_ratio"
		maxValue := "0.999999" // A value just under 1 for the exclusive range

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": maxValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to share_ratio did not succeed with maximum allowed value")
	})

	// Reward Decline Rate - Test case for updating reward_decline_rate to the minimum allowed value
	t.RunSequentially("successful update of reward_decline_rate to minimum allowed value", func(t *test.SystemTest) {
		configKey := "reward_decline_rate"
		minValue := "0" // The minimum allowed value for reward_decline_rate

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": minValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to reward_decline_rate did not succeed with minimum value")
	})

	// Reward Decline Rate - Test case for updating reward_decline_rate to a mid-interval value
	t.RunSequentially("successful update of reward_decline_rate to mid-interval value", func(t *test.SystemTest) {
		configKey := "reward_decline_rate"
		midValue := "0.5" // A valid mid-interval value for reward_decline_rate

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": midValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to reward_decline_rate did not succeed with mid-interval value")
	})

	// Reward Decline Rate - Test case for updating reward_decline_rate to the maximum allowed value, exclusive of 1
	t.RunSequentially("successful update of reward_decline_rate to maximum allowed value", func(t *test.SystemTest) {
		configKey := "reward_decline_rate"
		maxValue := "0.999999" // A value just under 1 for the exclusive range

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": maxValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to reward_decline_rate did not succeed with maximum allowed value")
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
