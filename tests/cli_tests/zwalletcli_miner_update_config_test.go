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

	// Test Suite I - Testing min allowances 
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

	t.RunSequentially("successful update of config to maximum allowed value", func(t *test.SystemTest) {

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   []string{"max_n", "min_n", "max_s", "min_s","max_delegates", "reward_rate","block_reward","share_ratio","reward_decline_rate","t_percent","k_percent","x_percent","min_stake","max_stake","min_stake_per_delegate","start_rounds","contribute_rounds","share_rounds","publish_rounds","wait_rounds","max_charge","epoch","reward_decline_rate","reward_round_frequency","num_miner_delegates_rewarded","num_sharders_rewarded","num_sharder_delegates_rewarded"},
			"values": []string{"100", "3", "30", "1","200", "0", "0.9", "0", "0","0.66","0.75","0.70","0.0","20000.0","1","50","50","50","50","50","0.5","125000000","0","250","10","1","5"},
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to config parameters succeeded with min values")
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


	t.RunSequentially("successful update of add_miner cost", func(t *test.SystemTest) {
		configKey := "cost.add_miner"
		newValue := "361" // A valid cost value for adding a miner
	
		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to add_miner cost did not succeed with valid value")
	})
	
	t.RunSequentially("successful update of add_sharder cost", func(t *test.SystemTest) {
		configKey := "cost.add_sharder"
		newValue := "331" // A valid cost value for adding a sharder
	
		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to add_sharder cost did not succeed with valid value")
	})
			   
	t.RunSequentially("successful update of delete_miner cost", func(t *test.SystemTest) {
		configKey := "cost.delete_miner"
		newValue := "484" // A valid cost value for deleting a miner
	
		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to delete_miner cost did not succeed with valid value")
	})

	t.RunSequentially("successful update of delete_sharder cost", func(t *test.SystemTest) {
		configKey := "cost.delete_sharder"
		newValue := "335" // A valid cost value for deleting a sharder
	
		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to delete_sharder cost did not succeed with valid value")
	})

	t.RunSequentially("successful update of miner_health_check cost", func(t *test.SystemTest) {
		configKey := "cost.miner_health_check"
		newValue := "149" // A valid cost value for miner health check
	
		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, isUpdateSuccess(output), "Update to miner_health_check cost did not succeed with valid value")
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
