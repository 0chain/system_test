package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const sharderAccessDenied = "update_sharder_settings: access denied"

func TestSharderUpdateSettings(testSetup *testing.T) { //nolint cyclomatic complexity 50 of func `
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Sharder update num_delegates by delegate wallet should work")

	var cooldownPeriod int64
	var lastRoundOfSettingUpdate int64
	var selectedSharderID string
	var oldSharderInfo climodel.Node
	var mnConfig map[string]float64

	t.TestSetup("Get list of sharders", func() {
		mnConfig = getMinerSCConfiguration(t)

		if _, err := os.Stat("./config/" + sharder01NodeDelegateWalletName + "_wallet.json"); err != nil {
			t.Skipf("Sharder node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
		}

		// First get the list of sharders for the wallet
		sharders := getShardersListForWallet(t, sharder01NodeDelegateWalletName)
		require.NotEmpty(t, sharders, "No sharders found for wallet")

		// Try to find sharder01ID in the list, otherwise use the first available sharder
		selectedSharderID = sharder01ID
		found := false
		for _, s := range sharders {
			if s.ID == sharder01ID {
				found = true
				break
			}
		}

		if !found {
			// Use the first available sharder if sharder01ID is not found
			for _, s := range sharders {
				selectedSharderID = s.ID
				break
			}
			t.Logf("sharder01ID not found, using first available sharder: %s", selectedSharderID)
		}

		// Fetch sharder info using the selected sharder ID
		output, err := minerInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"id": selectedSharderID,
		}), sharder01NodeDelegateWalletName, true)
		require.Nil(t, err, "error fetching sharder settings")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &oldSharderInfo)
		require.Nil(t, err, "error unmarhsalling mn-info json output")
		require.NotEmpty(t, oldSharderInfo)

		cooldownPeriod = int64(mnConfig["cooldown_period"]) // Updating miner settings has a cooldown of this many rounds
		lastRoundOfSettingUpdate = int64(0)

		// revert sharder node settings after test
		t.Cleanup(func() {
			currRound := getCurrentRound(t)

			if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
					time.Sleep(10 * time.Second)
					currRound = getCurrentRound(t)
				}
			}

			output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
				"id":            selectedSharderID,
				"num_delegates": oldSharderInfo.Settings.MaxNumDelegates,
				"sharder":       "",
			}), true)
			require.Nil(t, err, "error reverting sharder settings after test")
			require.Len(t, output, 2)
			require.Equal(t, "settings updated", output[0])
			require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		})
	})

	t.RunSequentially("Sharder update num_delegates by delegate wallet should work", func(t *test.SystemTest) {
		// Check balance before attempting update
		balance, err := getBalanceZCN(t, configPath, sharder01NodeDelegateWalletName)
		require.NoError(t, err, "Error fetching balance for sharder delegate wallet")
		require.GreaterOrEqual(t, balance, 0.2, "Sharder delegate wallet must have at least 0.2 ZCN to pay for transaction fees")

		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				time.Sleep(10 * time.Second)
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            selectedSharderID,
			"num_delegates": 5,
			"sharder":       "",
		}), true)
		require.Nil(t, err, "error updating num_delegated in sharder node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = minerInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"id": selectedSharderID,
		}), sharder01NodeDelegateWalletName, true)
		require.Nil(t, err, "error fetching sharder info")
		require.Len(t, output, 1)

		var sharderInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nil(t, err, "error unmarhsalling sharder node info")
		require.Equal(t, 5, sharderInfo.Settings.MaxNumDelegates)
	})

	t.RunSequentially("Sharder update with num_delegates more than global max_delegates should fail", func(t *test.SystemTest) {
		// Check balance before attempting update
		balance, err := getBalanceZCN(t, configPath, sharder01NodeDelegateWalletName)
		require.NoError(t, err, "Error fetching balance for sharder delegate wallet")
		require.GreaterOrEqual(t, balance, 0.2, "Sharder delegate wallet must have at least 0.2 ZCN to pay for transaction fees")

		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				time.Sleep(10 * time.Second)
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            selectedSharderID,
			"num_delegates": mnConfig["max_delegates"] + 1,
			"sharder":       "",
		}), false)
		require.NotNil(t, err, "expected error when updating num_delegates greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: number_of_delegates greater than max_delegates of SC: 21 > 20"
		require.Equal(t, expected, output[0])
	})

	t.RunSequentially("Sharder update num_delegates negative value should fail", func(t *test.SystemTest) {
		// Check balance before attempting update
		balance, err := getBalanceZCN(t, configPath, sharder01NodeDelegateWalletName)
		require.NoError(t, err, "Error fetching balance for sharder delegate wallet")
		require.GreaterOrEqual(t, balance, 0.2, "Sharder delegate wallet must have at least 0.2 ZCN to pay for transaction fees")

		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				time.Sleep(10 * time.Second)
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            selectedSharderID,
			"num_delegates": -1,
			"sharder":       "",
		}), false)
		require.NotNil(t, err, "expected error when updating negative num_delegates but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: invalid non-positive number_of_delegates: -1"
		require.Equal(t, expected, output[0])
	})

	t.RunSequentially("Sharder update without sharder id flag should fail", func(t *test.SystemTest) {
		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				time.Sleep(10 * time.Second)
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, "--sharder", false)
		require.NotNil(t, err, "expected error trying to update sharder node without id, but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing id flag", output[0])
	})

	t.RunSequentially("Sharder update with nothing to update should fail", func(t *test.SystemTest) {
		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				time.Sleep(10 * time.Second)
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":      selectedSharderID,
			"sharder": "",
		}), false)
		// FIXME: some indication that no param has been selected to update should be given
		require.Nil(t, err)
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
	})

	t.RunSequentially("Sharder update settings from non-delegate wallet should fail", func(t *test.SystemTest) {
		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				time.Sleep(10 * time.Second)
				currRound = getCurrentRound(t)
			}
		}

		createWallet(t)

		// Verify sharder exists before attempting update
		output, err := minerInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"id": selectedSharderID,
		}), escapedTestName(t), true)
		if err != nil {
			t.Skipf("Sharder %s not found, skipping test", selectedSharderID)
			return
		}

		output, err = minerSharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":            selectedSharderID,
			"num_delegates": 5,
			"sharder":       "",
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating sharder settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, sharderAccessDenied, output[0])
	})
}

func minerSharderUpdateSettings(t *test.SystemTest, cliConfigFilename, wallet, params string, retry bool) ([]string, error) {
	return minerSharderUpdateSettingsForWallet(t, cliConfigFilename, params, wallet, retry)
}

func minerSharderUpdateSettingsForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Logf("Updating Miner/Sharder node info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-update-settings %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-update-settings %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
