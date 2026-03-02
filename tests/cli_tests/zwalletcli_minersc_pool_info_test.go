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
	t.Parallel()
	t.SetSmokeTests("Getting MinerSC Stake pools of a wallet before and after locking against a miner should work")

	t.RunSequentially("Getting MinerSC Stake pools of a wallet before and after locking against a miner should work", func(t *test.SystemTest) {
		createWallet(t)

		_, err := getWallet(t, configPath)
		require.NoError(t, err)

		// Find an active miner to stake against
		stakableMiners := getStakableMinersList(t)
		require.NotEmpty(t, stakableMiners.Nodes, "No stakable miners found")
		selectedMinerID := stakableMiners.Nodes[0].ID
		for _, m := range stakableMiners.Nodes {
			if m.ID == miner01ID {
				selectedMinerID = miner01ID
				break
			}
		}

		// Pre-unlock any stale pool from previous runs so pool count starts at 0
		_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
		}), true)
		cliutils.Wait(t, 5*time.Second)

		// before locking tokens against a miner
		output, err := stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		initialMinerPoolCount := len(poolsInfo.Pools[selectedMinerID])

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
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
		require.Greater(t, len(poolsInfo.Pools[selectedMinerID]), initialMinerPoolCount, "expected more pools after locking")

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.RunSequentially("Getting MinerSC Stake pools of a wallet before and after locking against a sharder should work", func(t *test.SystemTest) {
		createWallet(t)

		// Find an active sharder to stake against
		stakableSharders := getStakableSharderList(t)
		if len(stakableSharders) == 0 {
			t.Errorf("No stakable sharders found - test infrastructure should have sharders with available delegate slots")
		}
		selectedSharderID := stakableSharders[0].ID
		for _, s := range stakableSharders {
			if s.ID == sharder01ID {
				selectedSharderID = sharder01ID
				break
			}
		}

		// Pre-unlock any stale pool from previous runs so pool count starts at 0
		_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": selectedSharderID,
		}), true)
		cliutils.Wait(t, 5*time.Second)

		// before locking tokens against a sharder
		output, err := stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		initialSharderPoolCount := len(poolsInfo.Pools[selectedSharderID])

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": selectedSharderID,
			"tokens":     5,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		_, err = getWallet(t, configPath)
		require.NoError(t, err)

		// after locking tokens against sharder
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Greater(t, len(poolsInfo.Pools[selectedSharderID]), initialSharderPoolCount, "expected more pools after locking")

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": selectedSharderID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.RunSequentiallyWithTimeout("Getting MinerSC pools info for a different client id than wallet owner should work", 5*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		// Fund main wallet to cover two lock operations (4+4 ZCN) plus transaction fees
		_, err := executeFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "error funding wallet for pool info test")

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet")

		targetWalletName := escapedTestName(t) + "_target"
		createWalletForName(targetWalletName)
		_, err = executeFaucetWithTokensForWallet(t, targetWalletName, configPath, 9)
		require.Nil(t, err, "error funding target wallet")

		// Get stakable miners list to find a valid miner (not genesis)
		stakableMiners := getStakableMinersList(t)
		require.NotEmpty(t, stakableMiners.Nodes, "No stakable miners found")
		selectedMinerID := stakableMiners.Nodes[0].ID
		for _, minerNode := range stakableMiners.Nodes {
			if minerNode.ID == miner01ID {
				selectedMinerID = miner01ID
				break
			}
		}

		// Get stakable sharders list to find a valid sharder
		stakableSharders := getStakableSharderList(t)
		if len(stakableSharders) == 0 {
			t.Errorf("No stakable sharders found - test infrastructure should have sharders with available delegate slots")
		}
		selectedSharderID := stakableSharders[0].ID
		for _, sharderNode := range stakableSharders {
			if sharderNode.ID == sharder01ID {
				selectedSharderID = sharder01ID
				break
			}
		}

		// Ensure cleanup: unstake after test regardless of outcome
		t.Cleanup(func() {
			_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
				"miner_id": selectedMinerID,
			}), true)
			_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
				"sharder_id": selectedSharderID,
			}), true)
		})

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
			"tokens":   4,
		}), true)
		require.Nil(t, err, "error locking tokens against miner")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		// Wait for miner lock to confirm before sharder lock
		cliutils.Wait(t, 15*time.Second)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": selectedSharderID,
			"tokens":     4,
		}), true)
		require.Nil(t, err, "error locking tokens against sharder")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		// Wait for sharder lock to confirm
		cliutils.Wait(t, 10*time.Second)

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

		// Find our pool by client_id (other tests may have staked against the same miner)
		require.NotEmpty(t, poolsInfo.Pools[selectedMinerID], "expected at least one pool for miner %s", selectedMinerID)
		var minerPool *climodel.MinerSCDelegatePoolInfo
		for _, p := range poolsInfo.Pools[selectedMinerID] {
			if p.ID == w.ClientID {
				minerPool = p
				break
			}
		}
		require.NotNil(t, minerPool, "expected to find pool for client_id %s in miner %s pools", w.ClientID, selectedMinerID)
		require.GreaterOrEqual(t, intToZCN(minerPool.Balance), float64(4), "miner pool balance should be at least 4 ZCN")

		require.NotEmpty(t, poolsInfo.Pools[selectedSharderID], "expected at least one pool for sharder %s", selectedSharderID)
		var sharderPool *climodel.MinerSCDelegatePoolInfo
		for _, p := range poolsInfo.Pools[selectedSharderID] {
			if p.ID == w.ClientID {
				sharderPool = p
				break
			}
		}
		require.NotNil(t, sharderPool, "expected to find pool for client_id %s in sharder %s pools", w.ClientID, selectedSharderID)
		require.GreaterOrEqual(t, intToZCN(sharderPool.Balance), float64(4), "sharder pool balance should be at least 4 ZCN")
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
