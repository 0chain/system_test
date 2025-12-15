package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerSCUserPoolInfo(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Getting MinerSC Stake pools of a wallet before and after locking against a miner should work")

	t.RunSequentially("Getting MinerSC Stake pools of a wallet before and after locking against a miner should work", func(t *test.SystemTest) {
		createWallet(t)

		w, err := getWallet(t, configPath)
		require.NoError(t, err)

		// before locking tokens against a miner
		output, err := stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Empty(t, poolsInfo.Pools)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner01ID,
			"tokens":   5,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])

		// after locking tokens against a miner
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools[miner01ID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[miner01ID][0].ID)
		require.Equal(t, float64(5), intToZCN(poolsInfo.Pools[miner01ID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner01ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.RunSequentially("Getting MinerSC Stake pools of a wallet before and after locking against a sharder should work", func(t *test.SystemTest) {
		createWallet(t)

		// before locking tokens against a sharder
		output, err := stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Empty(t, poolsInfo.Pools)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder01ID,
			"tokens":     5,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		w, err := getWallet(t, configPath)
		require.NoError(t, err)

		// after locking tokens against sharder
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools[sharder01ID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[sharder01ID][0].ID)
		require.Equal(t, float64(5), intToZCN(poolsInfo.Pools[sharder01ID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder01ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.RunSequentially("Getting MinerSC pools info for a different client id than wallet owner should work", func(t *test.SystemTest) { //TODO: slow
		createWallet(t)

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet")

		targetWalletName := escapedTestName(t) + "_target"
		createWalletForName(targetWalletName)
		require.Nil(t, err, "error creating wallet")

		// Get stakable miners list to find a valid miner (not genesis)
		stakableMiners := getStakableMinersList(t)
		require.NotEmpty(t, stakableMiners.Nodes, "No stakable miners found")
		selectedMinerID := stakableMiners.Nodes[0].ID
		// Try to use miner01ID if it exists in the list, otherwise use the first available miner
		for _, minerNode := range stakableMiners.Nodes {
			if minerNode.ID == miner01ID {
				selectedMinerID = miner01ID
				break
			}
		}

		// Get stakable sharders list to find a valid sharder
		stakableSharders := getStakableSharderList(t)
		require.NotEmpty(t, stakableSharders, "No stakable sharders found")
		selectedSharderID := stakableSharders[0].ID
		// Try to use sharder01ID if it exists in the list, otherwise use the first available sharder
		for _, sharderNode := range stakableSharders {
			if sharderNode.ID == sharder01ID {
				selectedSharderID = sharder01ID
				break
			}
		}

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
			"tokens":   4,
		}), true)
		require.Nil(t, err, "error locking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		waitForStakePoolActive(t)
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": selectedSharderID,
			"tokens":     4,
		}), true)
		require.Nil(t, err, "error locking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		output, err = stakePoolsInMinerSCInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"client_id": wallet.ClientID,
		}), targetWalletName, true)
		require.Nil(t, err, "error fetching Miner SC User Pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pools")

		w, err := getWallet(t, configPath)
		require.NoError(t, err)

		require.Len(t, poolsInfo.Pools[selectedMinerID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[selectedMinerID][0].ID)
		require.Equal(t, float64(4), intToZCN(poolsInfo.Pools[selectedMinerID][0].Balance))

		require.Len(t, poolsInfo.Pools[selectedSharderID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[selectedSharderID][0].ID)
		require.Equal(t, float64(4), intToZCN(poolsInfo.Pools[selectedSharderID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}

		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": selectedSharderID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})
}

func stakePoolsInMinerSCInfo(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return stakePoolsInMinerSCInfoForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func stakePoolsInMinerSCInfoForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("fetching mn-user-info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-user-info %s --json --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-user-info %s --json --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
