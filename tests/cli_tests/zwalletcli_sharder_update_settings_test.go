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
	mnConfig := getMinerSCConfiguration(t)

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
	var sharder climodel.Sharder
	for _, sharder = range sharders {
		if sharder.ID == sharder01ID {
			found = true
			break
		}
	}

	if !found {
		t.Skip("Skipping update test settings as delegate wallet not found. Please the wallets on https://github.com/0chain/actions/blob/master/run-system-tests/action.yml match delegate wallets on rancher.")
	}

	cooldownPeriod := int64(mnConfig["cooldown_period"]) // Updating miner settings has a cooldown of this many rounds
	lastRoundOfSettingUpdate := int64(0)

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

		old_max_stake, err := oldSharderInfo.Settings.MaxStake.Int64()
		require.Nil(t, err)
		old_min_stake, err := oldSharderInfo.Settings.MinStake.Int64()
		require.Nil(t, err)
		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": oldSharderInfo.Settings.MaxNumDelegates,
			"max_stake":     intToZCN(old_max_stake),
			"min_stake":     intToZCN(old_min_stake),
		}), true)
		require.Nil(t, err, "error reverting sharder settings after test")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
	})

	t.RunSequentially("Sharder update min_stake by delegate wallet should work", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"min_stake": 1,
		}), true)
		require.Nil(t, err, "error reverting sharder node settings after test")
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
		require.Nil(t, err, "error unmarshalling sharder info")
		min_stake, err := sharderInfo.Settings.MinStake.Int64()
		require.Nil(t, err)
		require.Equal(t, 1, int(intToZCN(min_stake)))
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": 5,
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

	t.RunSequentially("Sharder update max_stake by delegate wallet should work", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"max_stake": 99,
		}), true)
		require.Nil(t, err, "error updating max_stake in sharder node")
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
		require.Nil(t, err, "error unmarshalling sharder info")
		max_stake, err := sharderInfo.Settings.MaxStake.Int64()
		require.Nil(t, err)
		require.Equal(t, 99, int(intToZCN(max_stake)))
	})

	t.RunSequentially("Sharder update multiple settings with delegate wallet should work", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": 8,
			"min_stake":     2,
			"max_stake":     98,
		}), true)
		require.Nil(t, err, "error updating multiple settings in sharder node")
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
		require.Nil(t, err, "error unmarshalling sharder info")
		require.Equal(t, 8, sharderInfo.Settings.MaxNumDelegates)
		min_stake, err := sharderInfo.Settings.MinStake.Int64()
		require.Nil(t, err)
		require.Equal(t, 2, int(intToZCN(min_stake)))
		max_stake, err := sharderInfo.Settings.MaxStake.Int64()
		require.Nil(t, err)
		require.Equal(t, 98, int(intToZCN(max_stake)))
	})

	t.RunSequentially("Sharder update with min_stake less than global min should fail", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"min_stake": mnConfig["min_stake"] - 1e-10,
		}), false)
		require.NotNil(t, err, "expected error when updating min_stake less than global min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		const expected = "update_sharder_settings: decoding request: json: cannot unmarshal number -1 into Go struct field Settings.stake_pool.settings.min_stake of type currency.Coin"
		require.Equal(t, expected, output[0], strings.Join(output, "\n"))
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": mnConfig["max_delegates"] + 1,
		}), false)
		require.NotNil(t, err, "expected error when updating num_delegates greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: number_of_delegates greater than max_delegates of SC: 201 \\u003e 200"
		require.Equal(t, expected, output[0])
	})

	t.RunSequentially("Sharder update max_stake more than global max_stake should fail", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"max_stake": mnConfig["max_stake"] + 1e-10,
		}), false)
		require.NotNil(t, err, "expected error when updating max_store greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: max_stake is greater than allowed by SC: 1000000000001 \\u003e 1000000000000"
		require.Equal(t, expected, output[0])
	})

	t.RunSequentially("Sharder update min_stake greater than max_stake should fail", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"max_stake": 48,
			"min_stake": 51,
		}), false)
		require.NotNil(t, err, "expected error when trying to update min_stake greater than max stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: invalid node request results in min_stake greater than max_stake: 510000000000 \\u003e 480000000000"
		require.Equal(t, expected, output[0])
	})

	t.RunSequentially("Sharder update min_stake negative value should fail", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"min_stake": -1,
		}), false)
		require.NotNil(t, err, "expected error when updating negative min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: decoding request: json: cannot unmarshal number -10000000000 into Go struct field Settings.stake_pool.settings.min_stake of type currency.Coin"
		require.Equal(t, expected, output[0])
	})

	t.RunSequentially("Sharder update max_stake negative value should fail", func(t *test.SystemTest) {
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"max_stake": -1,
		}), false)
		require.NotNil(t, err, "expected error when updating negative max_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		const expected = "update_sharder_settings: decoding request: json: cannot unmarshal number -10000000000 into Go struct field Settings.stake_pool.settings.max_stake of type currency.Coin"
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": -1,
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

		output, err := sharderUpdateSettings(t, configPath, "", false)
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

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id": sharder01ID,
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

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = sharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": 5,
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating sharder settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, sharderAccessDenied, output[0])

		output, err = sharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"max_stake": 99,
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating sharder settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, sharderAccessDenied, output[0])

		output, err = sharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        sharder01ID,
			"min_stake": 1,
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating sharder settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, sharderAccessDenied, output[0])
	})
}

func sharderUpdateSettings(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return sharderUpdateSettingsForWallet(t, cliConfigFilename, params, sharder01NodeDelegateWalletName, retry)
}

func sharderUpdateSettingsForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Logf("Updating Sharder node info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-update-node-settings --sharder %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-update-node-settings --sharder %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
