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

func TestMinerUpdateSettings(testSetup *testing.T) { // nolint cyclomatic complexity 44
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Miner update num_delegates by delegate wallet should work")

	var cooldownPeriod int64
	var lastRoundOfSettingUpdate int64
	var miners climodel.MinerSCNodes
	var miner climodel.Node
	var mnConfig map[string]float64
	t.TestSetup("Register wallet, get miner info", func() {
		if _, err := os.Stat("./config/" + miner01NodeDelegateWalletName + "_wallet.json"); err != nil {
			t.Skipf("miner node owner wallet located at %s is missing", "./config/"+miner01NodeDelegateWalletName+"_wallet.json")
		}

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = createWalletForName(t, configPath, miner01NodeDelegateWalletName)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		mnConfig = getMinerSCConfiguration(t)
		output, err = listMiners(t, configPath, "--json")
		require.Nil(t, err, "error listing miners")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "error unmarshalling ls-miners json output")

		found := false
		for _, miner = range miners.Nodes {
			if miner.ID == miner01ID {
				found = true
				break
			}
		}

		if !found {
			t.Skip("Skipping update test settings as delegate wallet not found. Please the wallets on https://github.com/0chain/actions/blob/master/run-system-tests/action.yml match delegate wallets on rancher.")
		}

		// Revert miner settings after test is complete
		t.Cleanup(func() {
			t.Log("start revert")
			output, err := minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
				"id":            miner.ID,
				"num_delegates": miner.Settings.MaxNumDelegates,
			}), true)
			require.Nil(t, err, "error reverting miner settings after test")
			require.Len(t, output, 2)
			require.Equal(t, "settings updated", output[0])
			require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
			t.Log("end revert")
		})

		cooldownPeriod = int64(mnConfig["cooldown_period"]) // Updating miner settings has a cooldown of this many rounds
		lastRoundOfSettingUpdate = int64(0)
	})

	t.RunSequentiallyWithTimeout("Miner update num_delegates by delegate wallet should work", 60*time.Second, func(t *test.SystemTest) {
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
		output, err := minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
		}), true)

		lastRoundOfSettingUpdate = getCurrentRound(t)

		require.Nil(t, err, "error updating num_delegates in miner node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 5, minerInfo.Settings.MaxNumDelegates)
	})

	t.RunSequentiallyWithTimeout("Miner update num_delegates greater than global max_delegates should fail", 60*time.Second, func(t *test.SystemTest) {
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

		output, err := minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": mnConfig["max_delegates"] + 1,
		}), false)

		lastRoundOfSettingUpdate = getCurrentRound(t)

		require.NotNil(t, err, "expected error when updating num_delegates greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: number_of_delegates greater than max_delegates of SC: 21 \\u003e 20", output[0])
	})

	t.RunSequentially("Miner update num_delegate negative value should fail", func(t *test.SystemTest) {
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

		output, err := minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": -1,
		}), false)

		lastRoundOfSettingUpdate = getCurrentRound(t)

		require.NotNil(t, err, "expected error on negative num_delegates but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: invalid non-positive number_of_delegates: -1", output[0])
	})

	t.RunSequentially("Miner update without miner id flag should fail", func(t *test.SystemTest) {
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

		output, err := minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, "", false)
		require.NotNil(t, err, "expected error trying to update miner node settings without id, but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing id flag", output[0])

		lastRoundOfSettingUpdate = getCurrentRound(t)
	})

	t.RunSequentially("Miner update with nothing to update should fail", func(t *test.SystemTest) {
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

		output, err := minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id": miner.ID,
		}), false)
		lastRoundOfSettingUpdate = getCurrentRound(t)

		// FIXME: some indication that no param has been selected to update should be given
		require.Nil(t, err)
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		t.Log("end test")
	})

	t.RunSequentially("Miner update settings from non-delegate wallet should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = minerSharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
		}), escapedTestName(t), false)

		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: access denied", output[0])

		output, err = minerSharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 1,
		}), escapedTestName(t), false)

		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: access denied", output[0])

		output, err = minerSharderUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": 99,
		}), escapedTestName(t), false)

		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: access denied", output[0])
	})
}

func listMiners(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet ls-miners %s --active --silent --wallet %s_wallet.json --configDir ./config --config %s", params, miner01NodeDelegateWalletName, cliConfigFilename), 3, time.Second*2)
}

func minerInfo(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Fetching miner node info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, miner01NodeDelegateWalletName, cliConfigFilename), 3, time.Second*2)
}

func getCurrentRound(t *test.SystemTest) int64 {
	return getLatestFinalizedBlock(t).Round
}
