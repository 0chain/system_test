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
	var sharder climodel.Sharder
	var mnConfig map[string]float64

	t.TestSetup("Get list of sharders", func() {
		mnConfig = getMinerSCConfiguration(t)

		if _, err := os.Stat("./config/" + sharder01NodeDelegateWalletName + "_wallet.json"); err != nil {
			t.Skipf("Sharder node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
		}

		output, err := minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder01ID,
		}), true)
		require.Nil(t, err, "error fetching sharder settings")
		require.Len(t, output, 1)

		var oldSharderInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &oldSharderInfo)
		require.Nil(t, err, "error unmarhsalling mn-info json output")
		require.NotEmpty(t, oldSharderInfo)

		sharders := getShardersListForWallet(t, sharder01NodeDelegateWalletName)

		found := false
		for _, sharder = range sharders {
			if sharder.ID == sharder01ID {
				found = true
				break
			}
		}

		if !found {
			t.Skip("Skipping update test settings as delegate wallet not found. Please the wallets on https://github.com/0chain/actions/blob/master/run-system-tests/action.yml match delegate wallets on rancher.")
		}

		cooldownPeriod = int64(mnConfig["cooldown_period"]) // Updating miner settings has a cooldown of this many rounds
		lastRoundOfSettingUpdate = int64(0)

		// revert sharder node settings after test
		t.Cleanup(func() {
			currRound := getCurrentRound(t)

			if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
					// dummy transactions to increase round
					for i := 0; i < 5; i++ {
						_, _ = executeFaucetWithTokens(t, configPath, 2.0)
					}
					currRound = getCurrentRound(t)
				}
			}

			output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
				"id":            sharder01ID,
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
		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				// dummy transactions to increase round
				for i := 0; i < 5; i++ {
					_, _ = executeFaucetWithTokens(t, configPath, 2.0)
				}
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": 5,
			"sharder":       "",
		}), true)
		require.Nil(t, err, "error updating num_delegated in sharder node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder01ID,
		}), true)
		require.Nil(t, err, "error fetching sharder info")
		require.Len(t, output, 1)

		var sharderInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nil(t, err, "error unmarhsalling sharder node info")
		require.Equal(t, 5, sharderInfo.Settings.MaxNumDelegates)
	})

	t.RunSequentially("Sharder update with num_delegates more than global max_delegates should fail", func(t *test.SystemTest) {
		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				// dummy transactions to increase round
				for i := 0; i < 5; i++ {
					_, _ = executeFaucetWithTokens(t, configPath, 2.0)
				}
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": mnConfig["max_delegates"] + 1,
			"sharder":       "",
		}), false)
		require.NotNil(t, err, "expected error when updating num_delegates greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: number_of_delegates greater than max_delegates of SC: 21 \\u003e 20"
		require.Equal(t, expected, output[0])
	})

	t.RunSequentially("Sharder update num_delegates negative value should fail", func(t *test.SystemTest) {
		currRound := getCurrentRound(t)

		if (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
			for (currRound - lastRoundOfSettingUpdate) < cooldownPeriod {
				// dummy transactions to increase round
				for i := 0; i < 5; i++ {
					_, _ = executeFaucetWithTokens(t, configPath, 2.0)
				}
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            sharder01ID,
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
				// dummy transactions to increase round
				for i := 0; i < 5; i++ {
					_, _ = executeFaucetWithTokens(t, configPath, 2.0)
				}
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
				// dummy transactions to increase round
				for i := 0; i < 5; i++ {
					_, _ = executeFaucetWithTokens(t, configPath, 2.0)
				}
				currRound = getCurrentRound(t)
			}
		}

		output, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":      sharder01ID,
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
				// dummy transactions to increase round
				for i := 0; i < 5; i++ {
					_, _ = executeFaucetWithTokens(t, configPath, 2.0)
				}
				currRound = getCurrentRound(t)
			}
		}

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = minerSharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": 5,
			"sharder":       "",
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating sharder settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, sharderAccessDenied, output[0])

		output, err = minerSharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"max_stake": 99,
			"sharder":   "",
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating sharder settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, sharderAccessDenied, output[0])

		output, err = minerSharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"min_stake": 1,
			"sharder":   "",
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
