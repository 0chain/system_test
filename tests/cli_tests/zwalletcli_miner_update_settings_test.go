package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerUpdateSettings(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + minerNodeDelegateWallet + "_wallet.json"); err != nil {
		t.Skipf("blobber owner wallet located at %s is missing", "./config/"+minerNodeDelegateWallet+"_wallet.json")
	}

	mnConfig := getMinerSCConfiguration(t)

	t.Run("Miner update min_stake by delegate wallet should work", func(t *testing.T) {
		t.Parallel()

		output, err := listMiners(t, configPath, "--json")
		require.Nil(t, err, "error listing miners")
		require.Len(t, output, 1)

		var miners climodel.MinerSCNodes
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "error unmarshalling ls-miners json output")
		miner := miners.Nodes[2]

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}))
		require.Nil(t, err, "error fetching miner node info")
		require.Len(t, output, 1)

		var oldMinerInfo climodel.SimpleNode
		err = json.Unmarshal([]byte(output[0]), &oldMinerInfo)
		require.Nil(t, err, "error unmarshalling miner info")

		output, err = minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 1,
		}), true)
		require.Nil(t, err, "error reverting miner node settings after test")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		defer func() {
			output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
				"id":        miner.ID,
				"min_stake": oldMinerInfo.MinStake,
			}), true)
			require.Nil(t, err, "error reverting miner node settings after test")
			require.Len(t, output, 1)
			require.Equal(t, "settings updated", output[0])
		}()
		require.Nil(t, err, "error updating min stake in miner node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])
	})

	t.Run("Miner update num_delegates by delegate wallet should work", func(t *testing.T) {
		t.Parallel()

		output, err := listMiners(t, configPath, "--json")
		require.Nil(t, err, "error listing miners")
		require.Len(t, output, 1)

		var miners climodel.MinerSCNodes
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "error unmarshalling ls-miners json output")

		miner := miners.Nodes[2]
		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}))
		require.Nil(t, err, "error fetching miner node info")
		require.Len(t, output, 1)

		var oldMinerInfo climodel.SimpleNode
		err = json.Unmarshal([]byte(output[0]), &oldMinerInfo)
		require.Nil(t, err, "error unmarshalling miner info")

		output, err = minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
		}), true)
		defer func() {
			output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
				"id":            miner.ID,
				"num_delegates": oldMinerInfo.NumberOfDelegates,
			}), true)
			require.Nil(t, err, "error reverting miner node settings after test")
			require.Len(t, output, 1)
			require.Equal(t, "settings updated", output[0])
		}()
		require.Nil(t, err, "error updating num_delegates in miner node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])
	})

	t.Run("Miner update max_stake with delegate wallet should work", func(t *testing.T) {
		t.Parallel()

		output, err := listMiners(t, configPath, "--json")
		require.Nil(t, err, "error listing miners")
		require.Len(t, output, 1)

		var miners climodel.MinerSCNodes
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "error unmarshalling ls-miners json output")

		miner := miners.Nodes[2]
		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}))
		require.Nil(t, err, "error fetching miner node info")
		require.Len(t, output, 1)

		var oldMinerInfo climodel.SimpleNode
		err = json.Unmarshal([]byte(output[0]), &oldMinerInfo)
		require.Nil(t, err, "error unmarshalling miner info")

		output, err = minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": 99,
		}), true)
		defer func() {
			output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
				"id":        miner.ID,
				"max_stake": oldMinerInfo.MaxStake,
			}), true)
			require.Nil(t, err, "error reverting miner node settings after test")
			require.Len(t, output, 1)
			require.Equal(t, "settings updated", output[0])
		}()
		require.Nil(t, err, "error updating max_stake in miner node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])
	})

	t.Run("Miner update min_stake with less than global min stake should fail", func(t *testing.T) {
		t.Parallel()

		output, err := listMiners(t, configPath, "--json")
		require.Nil(t, err, "error listing miners")
		require.Len(t, output, 1)

		var miners climodel.MinerSCNodes
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "error unmarshalling ls-miners json output")

		miner := miners.Nodes[2]
		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}))
		require.Nil(t, err, "error fetching miner node info")
		require.Len(t, output, 1)

		var oldMinerInfo climodel.SimpleNode
		err = json.Unmarshal([]byte(output[0]), &oldMinerInfo)
		require.Nil(t, err, "error unmarshalling miner info")

		output, err = minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": mnConfig["min_stake"] - 1e-9,
		}), false)
		defer func() {
			output, err = minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
				"id":        miner.ID,
				"min_stake": oldMinerInfo.MinStake,
			}), true)
			require.Nil(t, err, "error reverting miner node settings after test")
			require.Len(t, output, 1)
			require.Equal(t, "settings updated", output[0])
		}()
		require.NotNil(t, err, "expected error when updating min_stake less than global min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})
}

func listMiners(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet ls-miners %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, minerNodeDelegateWallet, cliConfigFilename), 3, time.Second*2)
}

func minerUpdateSettings(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerUpdateSettingsForWallet(t, cliConfigFilename, params, minerNodeDelegateWallet, retry)
}

func minerUpdateSettingsForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("Updating miner settings...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-update-settings %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-update-settings %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}

func minerInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Fetching miner node info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, minerNodeDelegateWallet, cliConfigFilename), 3, time.Second*2)
}
