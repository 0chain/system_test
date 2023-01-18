package cli_tests

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBlobberBlockRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	// if !confirmDebugBuild(t) {
	// 	t.Skip("blobber block rewards test skipped as it requires a debug event database")
	// }

	// Take a snapshot of the chains blobbers, then wait a few seconds, take another snapshot.
	// Examine the rewards paid between the two snapshot and confirm the self-consistency
	// of the block reward payments.
	//
	// Each round, we get all blobbers passed challenge from last challenge period.
	//
	// Using the transaction hash+prev block hash, we create a seed. This seed is used to get
	// random partitions from the above.
	//
	// Then Stake pool are iterated through, and weights are saved. Rewards are distributed according to
	// these weights.
	//
	// The remaining block reward (if any) is then distributed amongst all stakepools.

	t.Run("Blobber share of block rewards should be distribute correctly", func(t *test.SystemTest) {
		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

		sharderUrl := getSharderUrl(t)
		blobberIds := getSortedBlobberIds(t, sharderUrl)
		require.True(t, len(blobberIds) > 0, "no blobbers found")

		beforeBlobberStakePools := getBlobberStakepools(t, sharderUrl, blobberIds)

		// ------------------------------------
		cliutils.Wait(t, 3*time.Minute)
		// ------------------------------------

		afterBlobberStakePools := getBlobberStakepools(t, sharderUrl, blobberIds)

		fmt.Print(beforeBlobberStakePools, afterBlobberStakePools)
		// we add rewards at the end of the round, and they don't appear until the next round
		// 	startRound := beforeBlobbers.Nodes[0].Round + 1
		// 	endRound := afterMiners.Nodes[0].Round + 1
		// 	for i := range beforeBlobbers.Nodes {
		// 		if startRound < beforeBlobbers.Nodes[i].Round {
		// 			startRound = beforeBlobbers.Nodes[i].Round
		// 		}
		// 		if endRound > afterMiners.Nodes[i].Round {
		// 			endRound = afterMiners.Nodes[i].Round
		// 		}
		// 	}

		// 	history := cliutil.NewHistory(startRound, endRound)
		// 	history.Read(t, sharderUrl)

		// 	minerScConfig := getMinerScMap(t)
		// 	numMinerDelegatesRewarded := int(minerScConfig["num_miner_delegates_rewarded"])
		// 	require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
		// 		"epoch changed during test, start %v finish %v",
		// 		startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

		// 	minerBlockReward, _ := blockRewards(startRound, minerScConfig)

		// 	// Each round one miner is chosen to receive a block reward.
		// 	// The winning miner is stored in the block object.
		// 	// The reward payments retrieved from the provider reward table.
		// 	// The amount of the reward is a fraction of the block reward allocated to miners each
		// 	// round. The fraction is the miner's service charge. If the miner has
		// 	// no stake pools then the reward becomes the full block reward.
		// 	//
		// 	// Firstly we confirm the self-consistency of the block and reward tables.
		// 	// We calculate the change in the miner rewards during and
		// 	// confirm that this equals the total of the reward payments
		// 	// read from the provider rewards table.
		// 	for i, id := range blobberIds {
		// 		var rewards int64
		// 		for round := beforeBlobbers.Nodes[i].Round + 1; round <= afterMiners.Nodes[i].Round; round++ {
		// 			roundHistory := history.RoundHistory(t, round)
		// 			for _, pReward := range roundHistory.ProviderRewards {
		// 				if pReward.ProviderId != id {
		// 					continue
		// 				}
		// 				switch pReward.RewardType {
		// 				case climodel.BlockRewardMiner:
		// 					require.Equalf(t, pReward.ProviderId, roundHistory.Block.MinerID,
		// 						"%s not round lottery winner %s but nevertheless paid with block reward."+
		// 							"only the round lottery winner shold get a miner block reward",
		// 						pReward.ProviderId, roundHistory.Block.MinerID)
		// 					var expectedServiceCharge int64
		// 					if len(beforeBlobbers.Nodes[i].StakePool.Pools) > 0 {
		// 						expectedServiceCharge = int64(float64(minerBlockReward) * beforeBlobbers.Nodes[i].Settings.ServiceCharge)
		// 					} else {
		// 						expectedServiceCharge = minerBlockReward
		// 					}
		// 					require.InDeltaf(t, expectedServiceCharge, pReward.Amount, delta, "incorrect service charge %v for round %d"+
		// 						" service charge should be block reward %v multiplied by service ratio %v",
		// 						pReward.Amount, round, expectedServiceCharge, minerBlockReward, beforeBlobbers.Nodes[i].Settings.ServiceCharge)
		// 					rewards += pReward.Amount
		// 				case climodel.FeeRewardMiner:
		// 					rewards += pReward.Amount
		// 				default:
		// 					require.Failf(t, "reward type %s is not available for miners", pReward.RewardType.String())
		// 				}
		// 			}
		// 		}
		// 		actualReward := afterMiners.Nodes[i].Reward - beforeBlobbers.Nodes[i].Reward
		// 		require.InDeltaf(t, actualReward, rewards, delta,
		// 			"rewards expected %v, change in miners reward during the test is %v", actualReward, rewards)
		// 	}

		// 	// Each round there should be exactly one block reward payment
		// 	// and this to the blocks' miner.
		// 	for round := history.From(); round <= history.To(); round++ {
		// 		roundHistory := history.RoundHistory(t, round)
		// 		foundBlockRewardPayment := false
		// 		for _, pReward := range roundHistory.ProviderRewards {
		// 			if pReward.RewardType == climodel.BlockRewardMiner {
		// 				require.False(t, foundBlockRewardPayment, "blocker reward already paid, only pay miner block rewards once")
		// 				foundBlockRewardPayment = true
		// 				require.Equal(t, pReward.ProviderId, roundHistory.Block.MinerID,
		// 					"block reward paid to %s, should only be paid to round lottery winner %s",
		// 					pReward.ProviderId, roundHistory.Block.MinerID)
		// 			}
		// 		}
		// 		require.True(t, foundBlockRewardPayment,
		// 			"miner block reward payment not recorded. block rewards should be paid every round.")
		// 	}

		// 	// Each round confirm payments to delegates or the blocks winning miner.
		// 	// There should be exactly `num_miner_delegates_rewarded` delegates rewarded each round,
		// 	// or all delegates if less.
		// 	//
		// 	// Delegates should be rewarded in proportional to their locked tokens.
		// 	// We check the self-consistency of the reward payments each round using
		// 	// the delegate reward table.
		// 	//
		// 	// Next we compare the actual change in rewards to each miner delegate, with the
		// 	// change expected from the delegate reward table.

		// 	for i, id := range blobberIds {
		// 		delegateBlockReward := int64(float64(minerBlockReward) * (1 - beforeBlobbers.Nodes[i].Settings.ServiceCharge))
		// 		numPools := len(afterMiners.Nodes[i].StakePool.Pools)
		// 		rewards := make(map[string]int64, numPools)
		// 		for poolId := range afterMiners.Nodes[i].StakePool.Pools {
		// 			rewards[poolId] = 0
		// 		}
		// 		for round := beforeBlobbers.Nodes[i].Round + 1; round <= afterMiners.Nodes[i].Round; round++ {
		// 			poolsBlockRewarded := make(map[string]int64)
		// 			roundHistory := history.RoundHistory(t, round)
		// 			for _, dReward := range roundHistory.DelegateRewards {
		// 				if _, found := rewards[dReward.PoolID]; !found {
		// 					continue
		// 				}
		// 				switch dReward.RewardType {
		// 				case climodel.BlockRewardMiner:
		// 					_, found := poolsBlockRewarded[dReward.PoolID]
		// 					require.False(t, found, "delegate pool %s paid a block reward more than once on round %d",
		// 						dReward.PoolID, round)
		// 					poolsBlockRewarded[dReward.PoolID] = dReward.Amount
		// 					rewards[dReward.PoolID] += dReward.Amount
		// 				case climodel.FeeRewardMiner:
		// 					rewards[dReward.PoolID] += dReward.Amount
		// 				default:
		// 					require.Failf(t, "reward type %s not paid to miner delegate pools", dReward.RewardType.String())
		// 				}
		// 			}
		// 			if roundHistory.Block.MinerID != id {
		// 				require.Len(t, poolsBlockRewarded, 0,
		// 					"delegate pools should not get a block reward unless their parent miner won the round lottery")
		// 			}
		// 			confirmPoolPayments(
		// 				t, delegateBlockReward, poolsBlockRewarded, afterMiners.Nodes[i].StakePool.Pools, numMinerDelegatesRewarded,
		// 			)
		// 		}
		// 		for poolId := range afterMiners.Nodes[i].StakePool.Pools {
		// 			actualReward := afterMiners.Nodes[i].StakePool.Pools[poolId].Reward - beforeBlobbers.Nodes[i].StakePool.Pools[poolId].Reward
		// 			require.InDeltaf(t, actualReward, rewards[poolId], delta,
		// 				"rewards, expected %v change in pools reward during test %v", actualReward, rewards[poolId])
		// 		}
		// 	}
	})
}

func getSortedBlobberIds(t *test.SystemTest, sharderBaseURL string) []string {
	t.Logf("getting sorted blobber nodes...")
	blobbers := getBlobbersList(t)
	var nodeIds []string
	for _, blobber := range blobbers {
		nodeIds = append(nodeIds, blobber.Id)
	}
	sort.Slice(nodeIds, func(i, j int) bool {
		return nodeIds[i] < nodeIds[j]
	})
	return nodeIds
}

func getBlobberStakepools(t *test.SystemTest, sharderBaseURL string, blobbers []string) map[string]climodel.StakePoolStat {
	t.Logf("getting blobber stake pools...")

	nodePoolStats := map[string]climodel.StakePoolStat{}
	url := fmt.Sprintf(sharderBaseURL + "/v1/screst/" + cliutils.StorageScAddress + "/getStakePoolStat")

	for _, blobberId := range blobbers {
		params := map[string]string{
			"provider_id":   blobberId,
			"provider_type": "3",
		}

		nodePoolStat := cliutils.ApiGet[climodel.StakePoolStat](t, url, params)

		nodePoolStats[blobberId] = *nodePoolStat
	}
	return nodePoolStats
}
