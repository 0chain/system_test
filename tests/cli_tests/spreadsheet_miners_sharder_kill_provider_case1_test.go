package cli_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	killRound = int64(1000)
)

// TestSpreadsheetMinerSharderKillProviderCase1
// 11 miners, 3 sharders to represent a scaled version of 111 miners and 30 sharders for mainnet
// 1 delegate each, equal stake
// Loadtest is off. Kill one miner, one sharder after 1000 rounds.
// Txn fee set to zero, Service charge set to 20%. Turn challenge off. No blobbers
// Miners and sharders should get equal rewards except one miner and sharder who will not receive after 1000 rounds.
// You can have the test done for 2000 rounds and compare, in which case, it will be half as much
// Total rewards to all miners and sharders needs to be equal to the total minted tokens on the network.
// Each miner/sharder delegate income should be equal and is based on rewards minus the service charge.
// The delegate should also receive the service fee portion.
// Total Rewards = Rewards from all miners and sharders
func TestSpreadsheetMinerSharderKillProviderCase1(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	t.Skip("waiting for kill miner and kill sharder functionality")

	t.RunWithTimeout("Spreadsheet miner sharder case 1", 500*time.Second, func(t *test.SystemTest) {
		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", false)
		time.Sleep(time.Second * 3)

		if !confirmDebugBuild(t) {
			t.Skip("miner block rewards test skipped as it requires a debug event database")
		}
		sharderUrl := getSharderUrl(t)
		minerIds := getSortedMinerIds(t, sharderUrl)
		sharderIds := getSortedSharderIds(t, sharderUrl)

		SSMKPSCase1Setup(t, minerIds, sharderIds, sharderUrl)

		_ = waitForRound(t, killRound, sharderUrl, minerIds)

		// todo put code im here to kill a miner and a sharder.

		afterMiners := waitForRound(t, sSMNumberOfRounds, sharderUrl, minerIds)
		afterSharders := getNodes(t, sharderIds, sharderUrl)

		var endRound int64
		for i := range afterMiners.Nodes {
			if endRound < afterMiners.Nodes[i].RoundServiceChargeLastUpdated {
				endRound = afterMiners.Nodes[i].RoundServiceChargeLastUpdated
			}
		}
		for i := range afterSharders.Nodes {
			if endRound < afterSharders.Nodes[i].RoundServiceChargeLastUpdated {
				endRound = afterSharders.Nodes[i].RoundServiceChargeLastUpdated
			}
		}
		history := cliutil.NewHistory(1, endRound)
		history.Read(t, sharderUrl, true)

		displayMetricsMinerSharders(
			t, endRound, afterMiners.Nodes, afterSharders.Nodes, history, sharderUrl,
		)

	})
}

func SSMKPSCase1Setup(t *test.SystemTest, minerIds, sharderIds []string, sharderUrl string) {
	// 11 miners, 3 sharders
	require.Len(t, minerIds, sSMSNumberMiners)
	require.Len(t, sharderIds, sSMSNumberSharders)
	require.True(t, killRound < sSMNumberOfRounds)

	// 1 delegate each, equal stake
	t.Log("locking tokens in new miner delegate pools")
	_ = createStakePools(t, minerIds, []float64{sSMSMinerSharderStakePoolSize}, climodel.ProviderMiner)
	t.Log("locking tokens in new sharder delegate pools")
	_ = createStakePools(t, sharderIds, []float64{sSMSMinerSharderStakePoolSize}, climodel.ProviderSharder)
	miners := getNodes(t, minerIds, sharderUrl)
	sharders := getNodes(t, sharderIds, sharderUrl)
	for i := range miners.Nodes {
		require.Len(t, miners.Nodes[i].StakePool.Pools, 1,
			"there should be exactly one stake pool per miner")
		for _, pool := range miners.Nodes[i].StakePool.Pools {
			require.Equal(t, pool.Balance, int64(sSMSMinerSharderStakePoolSize)*1e10,
				"stake pools should be all have balance %v", sSMSMinerSharderStakePoolSize)
		}
	}
	for i := range sharders.Nodes {
		require.Len(t, sharders.Nodes[i].StakePool.Pools, 1,
			"there should be exactly one stake pool per sharder")
		for _, pool := range sharders.Nodes[i].StakePool.Pools {
			require.Equal(t, pool.Balance, int64(sSMSMinerSharderStakePoolSize)*1e10,
				"stake pools should be all have balance %v", sSMSMinerSharderStakePoolSize)
		}
	}

	// Service charge set to 20%
	for i := range miners.Nodes {
		require.Equalf(t, miners.Nodes[i].Settings.ServiceCharge, sSMSServiceCharge, "")
	}
	for i := range sharders.Nodes {
		require.Equalf(t, sharders.Nodes[i].Settings.ServiceCharge, sSMSServiceCharge, "")
	}

	// Turn challenge off
	storageSettings := getStorageConfigMap(t)
	challengesEnabled, ok := storageSettings.Boolean["challenge_enabled"]
	require.True(t, ok, "missing challenge enabled setting")
	require.Falsef(t, challengesEnabled, "challenge enabled setting should be false")

	// costs all set to zero
	checkCostValues(t, storageSettings.Numeric, 0)
	checkCostValues(t, getMinerScMap(t), 0)

	//  No blobbers
	blobbers := getBlobbers(t)
	require.Len(t, blobbers, 0, " should be no blobbers")

	//No validators
	validators := getValidators(t)
	require.Len(t, validators, 0, "should be no validators")
}
