package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
	}()

	t.Run("Miner update min_stake by delegate wallet should work", func(t *testing.T) {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 1,
		}), true)

		require.Nil(t, err, "error updating min stake in miner node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

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
		require.Equal(t, 5, minerInfo.NumberOfDelegates)
	})

	t.Run("Miner update max_stake with delegate wallet should work", func(t *testing.T) {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": 99,
		}), true)

		require.Nil(t, err, "error updating max_stake in miner node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 99, int(intToZCN(minerInfo.MaxStake)))
	})

	t.Run("Miner update multiple settings with delegate wallet should work", func(t *testing.T) {
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
			"max_stake":     99,
			"min_stake":     1,
		}), true)
		require.Nil(t, err, "error updating multiple settings with delegate wallet")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 5, minerInfo.NumberOfDelegates)
		require.Equal(t, float64(99), intToZCN(minerInfo.MaxStake))
		require.Equal(t, float64(1), intToZCN(minerInfo.MinStake))
	})

	t.Run("Miner update min_stake with less than global min stake should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": mnConfig["min_stake"] - 1e-10,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableErrorDelegateMiner(t, output, "update_miner_settings:min_stake is less than allowed by SC: -1 \\u003e 0")
	})

	t.Run("Miner update num_delegates greater than global max_delegates should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": mnConfig["max_delegates"] + 1,
		}), false)

		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableErrorDelegateMiner(t, output, "update_miner_settings:number_of_delegates greater than max_delegates of SC: 201 \\u003e 200")
	})

	t.Run("Miner update max_stake greater than global max_stake should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": mnConfig["max_stake"] + 1e-10,
		}), false)

		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableErrorDelegateMiner(t, output, "update_miner_settings:max_stake is greater than allowed by SC: 1000000000001 \\u003e 1000000000000")
	})

	t.Run("Miner update max_stake less than min_stake should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 51,
			"max_stake": 48,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableErrorDelegateMiner(t, output, "update_miner_settings:invalid node request results in min_stake greater than max_stake: 510000000000 \\u003e 480000000000")
	})

	t.Run("Miner update min_stake negative value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": -1,
		}), false)

		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableErrorDelegateMiner(t, output, "update_miner_settings:min_stake is less than allowed by SC: -10000000000 \\u003e 0")
	})

	t.Run("Miner update max_stake negative value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": -1,
		}), false)

		require.Nil(t, err, "expected error negative max_stake but got output:", strings.Join(output, "\n"))
		assertChargeableErrorDelegateMiner(t, output, "update_miner_settings:invalid negative min_stake: 0 or max_stake: -10000000000")
	})

	t.Run("Miner update num_delegate negative value should fail", func(t *testing.T) {
		t.Parallel()

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": -1,
		}), false)

		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableErrorDelegateMiner(t, output, "update_miner_settings:invalid non-positive number_of_delegates: -1")
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
		// FIXME: some indication that no param has been selected to update should be given
		require.Nil(t, err)
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
	})

	t.Run("Miner update settings from non-delegate wallet should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": 5,
		}), escapedTestName(t), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableError(t, output, "update_miner_settings:access denied")

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"min_stake": 1,
		}), escapedTestName(t), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		assertChargeableError(t, output, "update_miner_settings:access denied")

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        miner.ID,
			"max_stake": 99,
		}), escapedTestName(t), false)
		require.Nil(t, err, "Unexpected error", strings.Join(output, "\n"))
		assertChargeableError(t, output, "update_miner_settings:access denied")
	})
}

func assertChargeableErrorDelegateMiner(t *testing.T, output []string, msg string) {
	require.Len(t, output, 2, strings.Join(output, "\n"))

	split := strings.Split(output[1], ":")
	require.Len(t, split, 2, strings.Join(split, "\n"))
	output, _ = verifyTransaction(t, configPath, split[1])
	require.Len(t, output, 7, strings.Join(output, "\n"))
	confStatus := strings.Trim(strings.Split(output[5], ":")[1], " ") //TransactionStatus: 2
	require.Equal(t, "2", confStatus, strings.Join(output, "\n"))
	errString := strings.Trim(strings.Trim(strings.SplitN(output[6], ":", 2)[1], " "), "\"") //TransactionOutput: "update_settings:key max_pour_amount, unable to convert x to state.balance
	require.Equal(t, msg, errString, strings.Join(output, "\n"))
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
