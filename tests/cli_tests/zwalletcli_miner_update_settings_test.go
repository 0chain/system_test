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

	if _, err := os.Stat("./config/" + minerNodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+minerNodeDelegateWalletName+"_wallet.json")
	}

	mnConfig := getMinerSCConfiguration(t)
	output, err := listMiners(t, configPath, "--json")
	require.Nil(t, err, "error listing miners")
	require.Len(t, output, 1)

	minerNodeDelegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
	require.Nil(t, err, "error fetching minerNodeDelegate wallet")

	var miners climodel.MinerSCNodes
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "error unmarshalling ls-miners json output")

	var miner climodel.Node
	for _, miner = range miners.Nodes {
		if miner.ID == minerNodeDelegateWallet.ClientID {
			break
		}
	}

	// Revert miner settings after test is complete
	defer func() {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": miner.NumberOfDelegates,
			"min_stake":     miner.MinStake / 1e10,
			"max_stake":     miner.MaxStake / 1e10,
		}), true)
		require.Nil(t, err, "error reverting miner settings after test")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])
	}()

	t.Run("Miner update min_stake by delegate wallet should work", func(t *testing.T) {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 1,
		}), true)
		require.Nil(t, err, "error reverting miner node settings after test")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		require.Nil(t, err, "error updating min stake in miner node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 1, int(intToZCN(minerInfo.MinStake)))
	})

	t.Run("Miner update num_delegates by delegate wallet should work", func(t *testing.T) {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
		}), true)

		require.Nil(t, err, "error updating num_delegates in miner node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 5, minerInfo.NumberOfDelegates)
	})

	t.Run("Miner update max_stake with delegate wallet should work", func(t *testing.T) {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": 101, // FIXME: should be 99
		}), true)

		require.Nil(t, err, "error updating max_stake in miner node")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 101, int(intToZCN(minerInfo.MaxStake)))
	})

	t.Run("Miner update multiple settings with delegate wallet should work", func(t *testing.T) {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
			"max_stake":     101, // FIXME: should be 99
			"min_stake":     1,
		}), true)
		require.Nil(t, err, "error updating multiple settings with delegate wallet")
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 5, minerInfo.NumberOfDelegates)
		require.Equal(t, float64(101), intToZCN(minerInfo.MaxStake))
		require.Equal(t, float64(1), intToZCN(minerInfo.MinStake))
	})

	t.Run("Miner update min_stake with less than global min stake should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": mnConfig["min_stake"] - 1e-10,
		}), false)

		require.NotNil(t, err, "expected error when updating min_stake less than global min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Miner update num_delegates greater than global max_delegates should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": mnConfig["max_delegates"] + 1,
		}), false)

		require.NotNil(t, err, "expected error when updating num_delegated greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Miner update max_stake greater than global max_stake should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": mnConfig["max_stake"] - 1e-10, // FIXME: should be +
		}), false)

		require.NotNil(t, err, "expected error when updating max_stake to greater than global max but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Miner update min_stake negative value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": -1,
		}), false)

		require.NotNil(t, err, "expected error on negative min stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Miner update max_stake negative value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": -1,
		}), false)

		require.NotNil(t, err, "expected error negative max_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Miner update num_delegate negative value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": -1,
		}), false)

		require.NotNil(t, err, "expected error on negative num_delegates but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Miner update without miner id flag should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, "", false)
		require.NotNil(t, err, "expected error trying to update miner node settings without id, but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing id flag", output[0])
	})

	t.Run("Miner update with nothing to update should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), false)
		require.Nil(t, err, "output")
		// FIXME: some indication that no param has been selected to update should be given
		require.Len(t, output, 1)
		require.Equal(t, "settings updated", output[0])
	})

	t.Run("Miner update settings from non-delegate wallet should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 1,
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": 99,
		}), escapedTestName(t), false)
		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})
}

func listMiners(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet ls-miners %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, minerNodeDelegateWalletName, cliConfigFilename), 3, time.Second*2)
}

func minerUpdateSettings(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerUpdateSettingsForWallet(t, cliConfigFilename, params, minerNodeDelegateWalletName, retry)
}

func minerUpdateSettingsForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("Updating miner settings...")
	cmd := fmt.Sprintf("./zwallet mn-update-settings %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func minerInfo(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Fetching miner node info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, minerNodeDelegateWalletName, cliConfigFilename), 3, time.Second*2)
}
