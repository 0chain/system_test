package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerUpdateSettings(t *testing.T) {
	if _, err := os.Stat("./config/" + miner01NodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+miner01NodeDelegateWalletName+"_wallet.json")
	}
	if _, err := os.Stat("./config/" + minerNodeWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+minerNodeWalletName+"_wallet.json")
	}

	mnConfig := getMinerSCConfiguration(t)
	output, err := listMiners(t, configPath, "--json")
	require.Nil(t, err, "error listing miners")
	require.Len(t, output, 1)

	require.Nil(t, err, "error fetching minerNodeDelegate nonce")
	ret, err := getNonceForWallet(t, configPath, miner01NodeDelegateWalletName, true)
	require.Nil(t, err, "error fetching minerNodeDelegate nonce")
	nonceStr := strings.Split(ret[0], ":")[1]
	nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
	require.Nil(t, err, "error converting nonce to in")

	require.Nil(t, err, "error fetching minerNode wallet")
	minerNodeWallet, err := getWalletForName(t, configPath, minerNodeWalletName)
	require.Nil(t, err, "error fetching minerNode wallet")

	var miners climodel.MinerSCNodes
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "error unmarshalling ls-miners json output")

	found := false
	var miner climodel.Node
	for _, miner = range miners.Nodes {
		if miner.ID == minerNodeWallet.ClientID {
			found = true
			break
		}
	}

	if !found {
		t.Skip("Skipping update test settings as delegate wallet not found. Please the wallets on https://github.com/0chain/actions/blob/master/run-system-tests/action.yml match delegate wallets on rancher.")
	}

	// Revert miner settings after test is complete
	revertChanges := func(nonce int64) {
		t.Log("start revert")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            miner.ID,
			"num_delegates": miner.Settings.MaxNumDelegates,
			"min_stake":     miner.Settings.MinStake / 1e10,
			"max_stake":     miner.Settings.MaxStake / 1e10,
		}), nonce, true)
		require.Nil(t, err, "error reverting miner settings after test")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		t.Log("end revert")
	}

	cooldownPeriod := int64(mnConfig["cooldown_period"]) // Updating miner settings has a cooldown of this many rounds
	lastRoundOfSettingUpdate := int64(0)

	// Get base URL for API calls.
	t.Run("Miner update min_stake by delegate wallet should work", func(t *testing.T) {
		// Get the starting balance for miner's delegate wallet.
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

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)

		t.Log("start test")

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"min_stake": 1,
		}), n, true)

		lastRoundOfSettingUpdate = getCurrentRound(t)

		require.Nil(t, err, "error updating min stake in miner node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": minerNodeWallet.ClientID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 1, int(intToZCN(minerInfo.Settings.MinStake)))
	})

	t.Run("Miner update num_delegates by delegate wallet should work", func(t *testing.T) {
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

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            minerNodeWallet.ClientID,
			"num_delegates": 5,
		}), n, true)

		lastRoundOfSettingUpdate = getCurrentRound(t)

		require.Nil(t, err, "error updating num_delegates in miner node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": minerNodeWallet.ClientID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 5, minerInfo.Settings.MaxNumDelegates)
	})

	t.Run("Miner update max_stake with delegate wallet should work", func(t *testing.T) {
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

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"max_stake": 99,
		}), n, true)

		lastRoundOfSettingUpdate = getCurrentRound(t)

		require.Nil(t, err, "error updating max_stake in miner node")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": minerNodeWallet.ClientID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 99, int(intToZCN(minerInfo.Settings.MaxStake)))
	})

	t.Run("Miner update multiple settings with delegate wallet should work", func(t *testing.T) {
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

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)

		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            minerNodeWallet.ClientID,
			"num_delegates": 5,
			"max_stake":     99,
			"min_stake":     1,
		}), n, true)

		lastRoundOfSettingUpdate = getCurrentRound(t)

		require.Nil(t, err, "error updating multiple settings with delegate wallet")
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": minerNodeWallet.ClientID,
		}), true)
		require.Nil(t, err, "error fetching miner info")
		require.Len(t, output, 1)

		var minerInfo climodel.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo)
		require.Nil(t, err, "error unmarshalling miner info")
		require.Equal(t, 5, minerInfo.Settings.MaxNumDelegates)
		require.Equal(t, float64(99), intToZCN(minerInfo.Settings.MaxStake))
		require.Equal(t, float64(1), intToZCN(minerInfo.Settings.MinStake))
	})

	t.Run("Miner update min_stake with less than global min stake should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"min_stake": mnConfig["min_stake"] - 1e-10,
		}), n, false)

		require.NotNil(t, err, "expected error when updating min_stake less than global min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: min_stake is less than allowed by SC: -1 \\u003e 0", output[0])
		t.Log("end test")
	})

	t.Run("Miner update num_delegates greater than global max_delegates should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            minerNodeWallet.ClientID,
			"num_delegates": mnConfig["max_delegates"] + 1,
		}), n, false)

		require.NotNil(t, err, "expected error when updating num_delegates greater than max allowed but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: number_of_delegates greater than max_delegates of SC: 201 \\u003e 200", output[0])
		t.Log("end test")
	})

	t.Run("Miner update max_stake greater than global max_stake should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"max_stake": mnConfig["max_stake"] + 1e-10,
		}), n, false)

		require.NotNil(t, err, "expected error when updating max_stake to greater than global max but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: max_stake is greater than allowed by SC: 1000000000001 \\u003e 1000000000000", output[0])
		t.Log("end test")
	})

	t.Run("Miner update max_stake less than min_stake should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"min_stake": 51,
			"max_stake": 48,
		}), n, false)
		require.NotNil(t, err, "Expected error when trying to update max_stake to less than min_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: invalid node request results in min_stake greater than max_stake: 510000000000 \\u003e 480000000000", output[0])
		t.Log("end test")
	})

	t.Run("Miner update min_stake negative value should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"min_stake": -1,
		}), n, false)

		require.NotNil(t, err, "expected error on negative min stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: min_stake is less than allowed by SC: -10000000000 \\u003e 0", output[0])
		t.Log("end test")
	})

	t.Run("Miner update max_stake negative value should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"max_stake": -1,
		}), n, false)

		require.NotNil(t, err, "expected error negative max_stake but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.True(t, strings.HasPrefix(output[0], "update_miner_settings: invalid negative min_stake:"), "Expected ["+output[0]+"] to start with [update_miner_settings: invalid negative min_stake:]")
		t.Log("end test")
	})

	t.Run("Miner update num_delegate negative value should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":            minerNodeWallet.ClientID,
			"num_delegates": -1,
		}), n, false)

		require.NotNil(t, err, "expected error on negative num_delegates but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: invalid non-positive number_of_delegates: -1", output[0])
		t.Log("end test")
	})

	t.Run("Miner update without miner id flag should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 1)
		revertChanges(n)
		t.Log("start test")
		//doesn't matter what nonce is used
		output, err := minerUpdateSettings(t, configPath, "", 0, false)
		require.NotNil(t, err, "expected error trying to update miner node settings without id, but got output:", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing id flag", output[0])
		t.Log("end test")
	})

	t.Run("Miner update with nothing to update should fail", func(t *testing.T) {
		//t.Parallel()

		n := atomic.AddInt64(&nonce, 2)
		revertChanges(n - 1)
		t.Log("start test")
		output, err := minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id": minerNodeWallet.ClientID,
		}), n, false)
		// FIXME: some indication that no param has been selected to update should be given
		require.Nil(t, err)
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		t.Log("end test")
	})

	t.Run("Miner update settings from non-delegate wallet should fail", func(t *testing.T) {
		//t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		n := int64(2)

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":            minerNodeWallet.ClientID,
			"num_delegates": 5,
		}), escapedTestName(t), n, false)
		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: access denied", output[0])

		n++

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"min_stake": 1,
		}), escapedTestName(t), n, false)
		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: access denied", output[0])

		n++

		output, err = minerUpdateSettingsForWallet(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeWallet.ClientID,
			"max_stake": 99,
		}), escapedTestName(t), n, false)
		require.NotNil(t, err, "expected error when updating miner settings from non delegate wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_miner_settings: access denied", output[0])
	})
}

func listMiners(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet ls-miners %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, miner01NodeDelegateWalletName, cliConfigFilename), 3, time.Second*2)
}

func minerUpdateSettings(t *testing.T, cliConfigFilename, params string, nonce int64, retry bool) ([]string, error) {
	return minerUpdateSettingsForWallet(t, cliConfigFilename, params, miner01NodeDelegateWalletName, nonce, retry)
}

func minerUpdateSettingsForWallet(t *testing.T, cliConfigFilename, params, wallet string, nonce int64, retry bool) ([]string, error) {
	t.Log("Updating miner settings...")
	t.Log(t.Name(), nonce)
	cmd := fmt.Sprintf("./zwallet mn-update-settings %s --silent --withNonce %v --wallet %s_wallet.json --configDir ./config --config %s", params, nonce, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*10)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
func getNonceForWallet(t *testing.T, cliConfigFilename, wallet string, retry bool) ([]string, error) {
	t.Log("Updating miner settings...")
	cmd := fmt.Sprintf("./zwallet getnonce --silent --wallet %s_wallet.json --configDir ./config --config %s", wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func minerInfo(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Fetching miner node info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, miner01NodeDelegateWalletName, cliConfigFilename), 3, time.Second*2)
}

func getCurrentRound(t *testing.T) int64 {
	return getLatestFinalizedBlock(t).Round
}
