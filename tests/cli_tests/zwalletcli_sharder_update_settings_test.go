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

func TestSharderUpdateSettings(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + sharderNodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("Sharder node owner wallet located at %s is missing", "./config/"+sharderNodeDelegateWalletName+"_wallet.json")
	}

	sharderNodeDelegateWallet, err := getWalletForName(t, configPath, sharderNodeDelegateWalletName)
	require.Nil(t, err, "error fetching sharder wallet")

	output, err := minerInfo(t, configPath, createParams(map[string]interface{}{
		"id": sharderNodeDelegateWallet.ClientID,
	}), true)
	require.Nil(t, err, "error fetching sharder settings")
	require.Len(t, output, 1)

	var oldSharderInfo climodel.Node
	err = json.Unmarshal([]byte(output[0]), &oldSharderInfo)
	require.Nil(t, err, "error unmarhsalling mn-info json output")

	sharders := getShardersListForWallet(t, sharderNodeDelegateWalletName)
	var sharder climodel.Sharder
	for _, sharder = range sharders {
		if sharder.ID == sharderNodeDelegateWallet.ClientID {
			break
		}
	}

	defer func() {
		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder.ID,
			"num_delegates": oldSharderInfo.NumberOfDelegates,
			"max_stake":     intToZCN(oldSharderInfo.MaxStake),
			"min_stake":     intToZCN(oldSharderInfo.MinStake),
		}), true)
		require.Nil(t, err, "error reverting sharder settings after test")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])
	}()

	t.Run("Sharder update min_stake by delegate wallet should work", func(t *testing.T) {
		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder.ID,
			"min_stake": 1,
		}), true)
		require.Nil(t, err, "error reverting sharder node settings after test")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true)
		require.Nil(t, err, "error fwtching sharder info")
		require.Len(t, output, 1)

		var sharderInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nil(t, err, "error unmarshalling sharder info")
		require.Equal(t, 1, int(intToZCN(sharderInfo.MinStake)))
	})
}

func sharderUpdateSettings(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return sharderUpdateSettingsForWallet(t, cliConfigFilename, params, sharderNodeDelegateWalletName, retry)
}

func sharderUpdateSettingsForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Logf("Updating Sharder node info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-update-node-settings %s --sharder --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-update-node-settings %s --sharder --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
