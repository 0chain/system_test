package cli_tests

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutil "github.com/0chain/system_test/internal/cli/util"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

const (
	sSMSNumberMiners              = 4 // todo actually 11
	sSMSNumberSharders            = 2 // todo actually 3
	sSMSMinerSharderStakePoolSize = 1
	sSMSServiceCharge             = 0.2
)

// TestSpreadsheetMinerSharderCase1
// 11 miners, 3 sharders to represent a scaled version of 111 miners and 30 sharders for mainnet
// 1 delegate each, equal stake
// Loadtest is off
// Txn fee set to zero, Service charge set to 20%. Turn challenge off. No blobbers
// Miners and sharders should get equal rewards. May need to find the right share ratio
// Total rewards to all miners and sharders needs to be equal to the total minted tokens on the network.
// Each miner/sharder delegate income should be equal and is based on rewards minus the service charge.
// The delegate should also receive the service fee portion.
// Total Rewards = Rewards from all miners and sharders
func TestSpreadsheetMinerSharderCase1(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	//  t.RunWithTimeout("Spreadsheet miner sharder case 1", 500*time.Second, func(t *test.SystemTest) {
	_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

	if !confirmDebugBuild(t) {
		t.Skip("miner block rewards test skipped as it requires a debug event database")
	}
	sharderUrl := getSharderUrl(t)
	minerIds := getSortedMinerIds(t, sharderUrl)
	sharderIds := getSortedSharderIds(t, sharderUrl)

	SSMSCase1Setup(t, minerIds, sharderIds, sharderUrl)

	beforeMiners := getNodes(t, minerIds, sharderUrl)
	beforeSharders := getNodes(t, sharderIds, sharderUrl)
	// ----------------------------------- w
	time.Sleep(time.Second * 3)
	// ----------------------------------=
	afterMiners := getNodes(t, minerIds, sharderUrl)
	afterSharders := getNodes(t, sharderIds, sharderUrl)

	startRound, endRound := getStartAndEndRounds(
		t, beforeMiners.Nodes, afterMiners.Nodes, beforeSharders.Nodes, afterSharders.Nodes,
	)

	// Miners and sharders should get equal rewards
	minerScConfig := getMinerScMap(t)
	minerBlockReward, sharderBlockReward := blockRewards(startRound, minerScConfig)
	require.Equal(testSetup, minerBlockReward, sharderBlockReward,
		"miner and sharder should have equal block rewards")

	time.Sleep(time.Second) // give time for last round to be saved
	history := cliutil.NewHistory(startRound, endRound)
	history.Read(t, sharderUrl, true)

	balanceMintsAndRewards(
		t, startRound, endRound, beforeMiners.Nodes, afterMiners.Nodes, beforeSharders.Nodes, afterSharders.Nodes, history, sharderUrl,
	)

	balanceMinerRewards(t, startRound, endRound, minerIds, beforeMiners.Nodes, afterMiners.Nodes, history)
	balanceSharderRewards(t, startRound, endRound, minerIds, beforeSharders.Nodes, afterSharders.Nodes, history)
	balanceMinerIncome(t, startRound, endRound, minerIds, beforeMiners.Nodes, afterMiners.Nodes, history)
	balanceSharderIncome(t, startRound, endRound, minerIds, beforeSharders.Nodes, afterSharders.Nodes, history)

	//  })
}

func balanceMintsAndRewards(
	t *test.SystemTest,
	startRound, endRound int64,
	beforeMiners, afterMiners, beforeSharders, afterSharders []climodel.Node,
	history *cliutil.ChainHistory,
	sharderUrl string,
) {
	startSanapshot := getSnapshot(t, startRound, 1, sharderUrl)
	endSnapshot := getSnapshot(t, endRound, 1, sharderUrl)
	startSanapshot = startSanapshot
	endSnapshot = endSnapshot
}

func SSMSCase1Setup(t *test.SystemTest, minerIds, sharderIds []string, sharderUrl string) {
	// 11 miners, 3 sharders
	require.Len(t, minerIds, sSMSNumberMiners)
	require.Len(t, sharderIds, sSMSNumberSharders)

	// 1 delegate each, equal stake
	t.Log("locking tokens in new miner delegate pools")
	_ = createStakePools(t, minerIds, []float64{sSMSMinerSharderStakePoolSize}, climodel.ProviderMiner)
	t.Log("locking tokens in new sharder delegate pools")
	_ = createStakePools(t, sharderIds, []float64{sSMSMinerSharderStakePoolSize}, climodel.ProviderSharder)
	miners := getNodes(t, minerIds, sharderUrl)
	sharders := getNodes(t, sharderIds, sharderUrl)
	for i := range miners.Nodes {
		//require.Len(t, miners.Nodes[i].StakePool.Pools, 1,
		//	"there should be exactly one stake pool per miner")
		for _, pool := range miners.Nodes[i].StakePool.Pools {
			require.Equal(t, pool.Balance, int64(sSMSMinerSharderStakePoolSize)*1e10,
				"stake pools should be all have balance %v", sSMSMinerSharderStakePoolSize)
		}
	}
	for i := range sharders.Nodes {
		//require.Len(t, sharders.Nodes[i].StakePool.Pools, 1,
		//	"there should be exactly one stake pool per sharder")
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
	costCollectReward, ok := storageSettings.Numeric["cost.collect_reward"]
	costCollectReward = costCollectReward
	//require.Equal(t, 0, costCollectReward)

	// cost.challenge_request set to zero

	//  No blobbers
	blobbers := getBlobbers(t)
	require.Len(t, blobbers, 0)
}

func createStakePools(
	t *test.SystemTest, providerIds []string, tokens []float64, provider climodel.Provider,
) func() {
	require.True(t, len(tokens) > 0, "create greater than zero pools")
	for _, id := range providerIds {
		for delegate := 0; delegate < len(tokens); delegate++ {
			wallet := escapedTestName(t) + "_delegate_" + strconv.Itoa(delegate) + "_node_" + id
			registerWalletWithTokens(t, configPath, wallet, tokens[delegate])
			output, err := minerOrSharderLockForWallet(t, configPath, createParams(map[string]interface{}{
				provider.String() + "_id": id,
				"tokens":                  tokens[delegate],
			}), wallet, true)
			require.NoError(t, err, "lock tokens in %s's stake pool", id)
			require.Len(t, output, 1, "output, lock tokens in %s's stake pool", id)
		}
	}
	return func() {
		for _, id := range providerIds {
			for delegate := 0; delegate < len(tokens); delegate++ {
				wallet := escapedTestName(t) + "_delegate_" + strconv.Itoa(delegate) + "_node_" + id
				_, err := minerOrSharderUnlockForWallet(t, configPath, createParams(map[string]interface{}{
					"id": id,
				}), wallet, true)
				require.NoError(t, err, "unlock tokens in %s's stake pool", id)
			}
		}
	}
}

func getBlobbers(t *test.SystemTest) []climodel.BlobberDetails {
	output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1, strings.Join(output, "\n"))

	var blobberList []climodel.BlobberDetails
	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, strings.Join(output, "\n"))

	return blobberList
}

func getSnapshot(t *test.SystemTest, round int64, limit int, sharderBaseUrl string) []climodel.Snapshot {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + cliutils.StorageScAddress + "/replicate-snapshots")
	params := map[string]string{
		"round": strconv.FormatInt(round, 10),
		"limit": strconv.Itoa(limit),
	}
	return cliutils.ApiGetSlice[climodel.Snapshot](t, url, params)
}

//
