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

func TestSharderUpdateSettings(t *testing.T) {
	t.Parallel()

	mnConfig := getMinerSCConfiguration(t)
	output, err := listMiners(t, configPath, "--json")
	require.Nil(t, err, "error listing miners")
	require.Len(t, output, 1)

	if _, err := os.Stat("./config/" + sharderNodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("Sharder node owner wallet located at %s is missing", "./config/"+sharderNodeDelegateWalletName+"_wallet.json")
	}

	sharderNodeDelegateWallet, err := getWalletForName(t, configPath, sharderNodeDelegateWalletName)
	require.Nil(t, err, "error fetching sharder wallet")

	output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
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

	t.Run("Sharder update num_delegates by delegate wallet should work", func(t *testing.T) {
		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder.ID,
			"num_delegates": 5,
		}), true)
		require.Nil(t, err, "error updating num_delegated in sharder node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true)
		require.Nil(t, err, "error fetching sharder info")
		require.Len(t, output, 1)

		var sharderInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nil(t, err, "error unmarhsalling sharder node info")
		require.Equal(t, 5, sharderInfo.NumberOfDelegates)
	})

	t.Run("Sharder update max_stake by delegate wallet should work", func(t *testing.T) {
		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder.ID,
			"max_stake": 101, // FIXME: should be 99
		}), true)
		require.Nil(t, err, "error updating max_stake in sharder node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true)
		require.Nil(t, err, "error fetching sharder info")
		require.Len(t, output, 1)

		var sharderInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nil(t, err, "error unmarshalling sharder info")
		require.Equal(t, 101, int(intToZCN(sharderInfo.MaxStake)))
	})

	t.Run("Sharder update multiple settings with delegate wallet should work", func(t *testing.T) {
		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder.ID,
			"num_delegates": 8,
			"min_stake":     2,
			"max_stake":     102, // should be 98
		}), true)
		require.Nil(t, err, "error updating multiple settings in sharder node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true)
		require.Nil(t, err, "error fetching sharder info")
		require.Len(t, output, 1)

		var sharderInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nil(t, err, "error unmarshalling sharder info")
		require.Equal(t, 8, sharderInfo.NumberOfDelegates)
		require.Equal(t, 2, int(intToZCN(sharderInfo.MinStake)))
		require.Equal(t, 102, int(intToZCN(sharderInfo.MaxStake)))
	})

	t.Run("Sharder update with min_stake less than global min should fail", func(t *testing.T) {
		t.Parallel()

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder.ID,
			"min_stake": mnConfig["min_stake"] - 1e-10,
		}), false)
		require.NotNil(t, err, "expected error when updating min_stake less than global min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Sharder update with num_delegates more than global max_delegates should fail", func(t *testing.T) {
		t.Parallel()

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            sharder.ID,
			"num_delegates": mnConfig["max_delegates"] + 1,
		}), false)
		require.NotNil(t, err, "expected error when updating num_delegates greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Sharder update max_stake more than global max_stake should fail", func(t *testing.T) {
		t.Parallel()

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder.ID,
			"max_stake": mnConfig["max_stake"] - 1e-10, // FIXME: should be +
		}), false)
		require.NotNil(t, err, "expected error when updating max_store greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Sharder update min_stake negative value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder.ID,
			"min_stake": -1,
		}), false)
		require.NotNil(t, err, "expected error when updating negative min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Sharder update max_stake value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := sharderUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        sharder.ID,
			"max_stake": -1,
		}), false)
		require.NotNil(t, err, "expected error when updating negative max_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
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
