package cli_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// TestSpreadsheetMinerSharderCase0
// 11 miners, 3 sharders to represent a scaled version of 111 miners and 30 sharders for mainnet
// no delegate each,
// Loadtest is off
// Txn fee set to zero, Service charge set to 20%. Turn challenge off. No blobbers
// Miners and sharders should get equal rewards. May need to find the right share ratio
// Total rewards to all miners and sharders needs to be equal to the total minted tokens on the network.
// Each miner/sharder delegate income should be equal and is based on rewards minus the service charge.
// The delegate should also receive the service fee portion.
// Total Rewards = Rewards from all miners and sharders
func TestSpreadsheetMinerSharderCase0(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	//  t.RunWithTimeout("Spreadsheet miner sharder case 1", 500*time.Second, func(t *test.SystemTest) {
	_ = initialiseTest(t, escapedTestName(t)+"_TARGET", false)
	time.Sleep(time.Second * 3)

	if !confirmDebugBuild(t) {
		t.Skip("miner block rewards test skipped as it requires a debug event database")
	}
	sharderUrl := getSharderUrl(t)
	minerIds := getSortedMinerIds(t, sharderUrl)
	sharderIds := getSortedSharderIds(t, sharderUrl)

	SSMSCase0Setup(t, minerIds, sharderIds, sharderUrl)

	// ----------------------------------- w
	time.Sleep(time.Second * 3)
	// ----------------------------------=

	time.Sleep(time.Second) // give time for last round to be saved

	afterMiners := getNodes(t, minerIds, sharderUrl)
	afterSharders := getNodes(t, sharderIds, sharderUrl)

	time.Sleep(time.Second) // give time for last round to be saved

	// Miners and sharders should get equal rewards
	minerScConfig := getMinerScMap(t)
	minerBlockReward, sharderBlockReward := blockRewards(1, minerScConfig)
	require.Equal(testSetup, minerBlockReward, sharderBlockReward,
		"miner and sharder should have equal block rewards")

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

	//  })
}

func SSMSCase0Setup(t *test.SystemTest, minerIds, sharderIds []string, sharderUrl string) {
	// 11 miners, 3 sharders
	require.Len(t, minerIds, sSMSNumberMiners)
	require.Len(t, sharderIds, sSMSNumberSharders)

	// 0 delegate each
	miners := getNodes(t, minerIds, sharderUrl)
	sharders := getNodes(t, sharderIds, sharderUrl)
	for i := range miners.Nodes {
		require.Len(t, miners.Nodes[i].StakePool.Pools, 0,
			"there should be exactly one stake pool per miner")
	}
	for i := range sharders.Nodes {
		require.Len(t, sharders.Nodes[i].StakePool.Pools, 0,
			"there should be exactly one stake pool per sharder")
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
	costCollectReward, ok := storageSettings.Numeric["cost.collect_reward"]
	costCollectReward = costCollectReward
	//require.Equal(t, 0, costCollectReward)

	// cost.challenge_request set to zero

	//  No blobbers
	blobbers := getBlobbers(t)
	require.Len(t, blobbers, 0)
}

//
