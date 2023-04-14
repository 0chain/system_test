package cli_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// TestSpreadsheetMinerSharderCase3
// 11 miners, 3 sharders to represent a scaled version of 111 miners and 30 sharders for mainnet
// 2 delegate each, one twice the other's stake, one delegate collects rewards
// Loadtest is off
// Txn fee set to zero, Service charge set to 20%. Turn challenge off. No blobbers
// Miners and sharders should get equal rewards. May need to find the right share ratio
// Total rewards to all miners and sharders needs to be equal to the total minted tokens on the network.
// Each miner/sharder delegate income should be equal and is based on rewards minus the service charge.
// The delegate should also receive the service fee portion.
// Total Rewards = Rewards from all miners and sharders
func TestSpreadsheetMinerSharderCase3(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
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

	walletsMiners, walletsSharders := SSMSCase3Setup(t, minerIds, sharderIds, sharderUrl)

	// ----------------------------------- w
	time.Sleep(time.Second * 1)
	for i := range walletsMiners {
		require.Len(t, walletsMiners[i], 2)
		output, err := collectRewardsMinerSharder(t, configPath, createParams(map[string]interface{}{
			"provider_type": climodel.ProviderMiner.String(),
			"provider_id":   minerIds[i],
		}), false, walletsMiners[i][0])
		require.NoError(t, err)
		require.Len(t, output, 1)
		require.True(t, strings.HasPrefix(output[0], "locked with"))
	}
	for i := range walletsSharders {
		require.Len(t, walletsMiners[i], 2)
		output, err := collectRewardsMinerSharder(t, configPath, createParams(map[string]interface{}{
			"provider_type": climodel.ProviderSharder.String(),
			"provider_id":   sharderIds[i],
		}), false, walletsSharders[i][0])
		require.NoError(t, err)
		require.Len(t, output, 1)
		require.True(t, strings.HasPrefix(output[0], "locked with"))
	}
	// ----------------------------------=

	afterMiners := waitForRound(t, sSMNumberOfRounds, sharderUrl, minerIds)
	afterSharders := getNodes(t, sharderIds, sharderUrl)

	time.Sleep(time.Second) // give time for last round to be saved

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

func SSMSCase3Setup(t *test.SystemTest, minerIds, sharderIds []string, sharderUrl string) (minerWallets, sharderWallets [][]string) {
	// 11 miners, 3 sharders
	require.Len(t, minerIds, sSMSNumberMiners)
	require.Len(t, sharderIds, sSMSNumberSharders)

	// 2 delegate each, equal stake
	t.Log("locking tokens in new miner delegate pools")
	minerWallets = createStakePools(t, minerIds, []float64{sSMSMinerSharderStakePoolSize, 2 * sSMSMinerSharderStakePoolSize}, climodel.ProviderMiner)
	t.Log("locking tokens in new sharder delegate pools")
	sharderWallets = createStakePools(t, sharderIds, []float64{sSMSMinerSharderStakePoolSize, 2 * sSMSMinerSharderStakePoolSize}, climodel.ProviderSharder)

	time.Sleep(time.Second)

	miners := getNodes(t, minerIds, sharderUrl)
	sharders := getNodes(t, sharderIds, sharderUrl)
	for i := range miners.Nodes {
		require.Len(t, miners.Nodes[i].StakePool.Pools, 2,
			"there should be exactly one stake pool per miner")
	}
	for i := range sharders.Nodes {
		require.Len(t, sharders.Nodes[i].StakePool.Pools, 2,
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
	return minerWallets, sharderWallets
}
