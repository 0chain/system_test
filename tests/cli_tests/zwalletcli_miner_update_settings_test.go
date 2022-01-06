package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
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

	t.Run("miner update min_stake by delegate wallet should work", func(t *testing.T) {
		t.Parallel()

		output, err := listMiners(t, configPath, "--json")
		require.Nil(t, err, "error listing miners")
		require.Len(t, output, 1)

		var miners climodel.MinerSCNodes
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "error unmarshalling ls-miners json output")
		miner := miners.Nodes[2]

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
		}))
		require.Nil(t, err, "error fetching miner node info")
		require.Len(t, output, 1)

		var oldMinerInfo climodel.SimpleNode
		err = json.Unmarshal([]byte(output[0]), &oldMinerInfo)
		require.Nil(t, err, "error unmarshalling miner info")

		output, err = minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 1,
		}))
		require.Nil(t, err, "error reverting miner node settings after test")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		defer func() {
			output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
				"id":        miner.ID,
				"min_stake": oldMinerInfo.MinStake,
			}))
			require.Nil(t, err, "error reverting miner node settings after test")
			require.Len(t, output, 1)
			require.Equal(t, "settings updated", output[0])
		}()
		require.Nil(t, err, "error updating min stake in miner node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])
	})
}

func listMiners(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet ls-miners %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, minerNodeDelegateWallet, cliConfigFilename), 3, time.Second*2)
}

func minerUpdateSettings(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return minerUpdateSettingsForWallet(t, cliConfigFilename, params, minerNodeDelegateWallet)
}

func minerUpdateSettingsForWallet(t *testing.T, cliConfigFilename, params, wallet string) ([]string, error) {
	t.Log("Updating miner settings...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-update-settings %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}

func minerInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Fetching miner node info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, minerNodeDelegateWallet, cliConfigFilename), 3, time.Second*2)
}
