package cli_tests

import (
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSharderBlockRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	if !confirmDebugBuild(t) {
		t.Skip("sharder block rewards test skipped as it requires a debug event database")
	}

	t.Run("Sharder share of block fees and rewards", func(t *test.SystemTest) {
		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

		sharderUrl := getSharderUrl(t)
		sharderIds := getSortedSharderIds(t, sharderUrl)
		require.True(t, len(sharderIds) > 1, "this test needs at least two sharders")

		// todo piers remove
		//tokens := []float64{1, 0.5}
		//_ = createStakePools(t, sharderIds, tokens)
		//t.Cleanup(func() {
		//	cleanupFunc()
		//})

		beforeSharders := getNodes(t, sharderIds, sharderUrl)

		// ----------------------------------- w
		time.Sleep(time.Second * 2)
		// ----------------------------------=

		afterShardedrs := getNodes(t, sharderIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round
		startRound := beforeSharders.Nodes[0].Round + 1
		endRound := afterShardedrs.Nodes[0].Round + 1
		for i := range beforeSharders.Nodes {
			if startRound < beforeSharders.Nodes[i].Round {
				startRound = beforeSharders.Nodes[i].Round
			}
			if endRound > afterShardedrs.Nodes[i].Round {
				endRound = afterShardedrs.Nodes[i].Round
			}
		}

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl)

		minerScConfig := getMinerScMap(t)
		numSharderDelegatesRewarded := int(minerScConfig["num_sharder_delegates_rewarded"])
		numShardersRewarded := int(minerScConfig["num_sharders_rewarded"])
		if numShardersRewarded > len(sharderIds) {
			numShardersRewarded = len(sharderIds)
		}
		if numShardersRewarded == 0 {
			return
		}
		require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
			"epoch changed during test, start %v finish %v",
			startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

		// The num_sharders_rewarded smart contract setting determines how many sharder
		// we rewarded each round, or all sharders if less.
		//
		// The amount of the reward is a fraction of the block reward allocated to sharders each
		// round. The fraction is the sharder's service charge. If the sharder has
		// no stake pools then the reward becomes the full block reward.
		_, sharderBlockReward := blockRewards(startRound, minerScConfig)

		bwPerSharder := int64(float64(sharderBlockReward) / float64(numShardersRewarded))
		for i, id := range sharderIds {
			var rewards int64
			for round := beforeSharders.Nodes[i].Round + 1; round <= afterShardedrs.Nodes[i].Round; round++ {
				roundHistory := history.RoundHistory(t, round)
				for _, pReward := range roundHistory.ProviderRewards {
					if pReward.ProviderId != id {
						continue
					}
					switch pReward.RewardType {
					case climodel.BlockRewardSharder:
						var expectedServiceCharge int64
						payAllToSharder := len(beforeSharders.Nodes[i].StakePool.Pools) == 0 || numSharderDelegatesRewarded == 0
						if payAllToSharder {
							expectedServiceCharge = bwPerSharder
						} else {
							expectedServiceCharge = int64(float64(bwPerSharder) * beforeSharders.Nodes[i].Settings.ServiceCharge)
						}
						require.InDeltaf(t, expectedServiceCharge, pReward.Amount, delta, "sharder service charge incorrect value on round %d", round)
						rewards += pReward.Amount
					case climodel.FeeRewardSharder:
						rewards += pReward.Amount
					default:
						require.Failf(t, "reward type %s not available to sharders", pReward.RewardType.String())
					}
				}
			}
			actualReward := afterShardedrs.Nodes[i].Reward - beforeSharders.Nodes[i].Reward
			if actualReward != rewards {
				require.InDeltaf(t, actualReward, rewards, delta,
					"rewards expected %v change in sharders reward during test %v", actualReward, rewards)
			}
		}

		//sharderMap := make(map[string]climodel.Node, len(sharderIds))
		//for i := 0; i < len(beforeSharders.Nodes); i++ {
		//	sharderMap[beforeSharders.Nodes[i].ID] = beforeSharders.Nodes[i]
		//}

		// Each round there should be exactly num_sharders_rewarded sharder block reward payment
		// and this to the blocks' miner.
		// We confirm that the count of rewarded sharders is correct.
		for round := history.From(); round <= history.To(); round++ {
			roundHistory := history.RoundHistory(t, round)
			shardersPaid := make(map[string]bool)
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.RewardType == climodel.BlockRewardSharder {
					_, found := shardersPaid[pReward.ProviderId]
					require.Falsef(t, found, "sharder %s receives more than one block reward on round %d", pReward.ProviderId, round)
					shardersPaid[pReward.ProviderId] = true
				}
			}
			require.Equal(t, numShardersRewarded, len(shardersPaid),
				"mismatch between expected count of sharders rewarded and actual number on round %d", round)
		}

		// Each round each sharder rewarded should have num_sharder_delegates_rewarded of
		// their delegates rewarded, or all delegates if less.
		for round := history.From(); round <= history.To(); round++ {
			roundHistory := history.RoundHistory(t, round)
			for i, id := range sharderIds {
				poolsPaid := make(map[string]bool)
				for poolId := range beforeSharders.Nodes[i].Pools {
					for _, dReward := range roundHistory.DelegateRewards {
						if dReward.RewardType != climodel.BlockRewardSharder || dReward.PoolID != poolId {
							continue
						}
						_, found := poolsPaid[poolId]
						if found {
							require.Falsef(t, found, "pool %s should have only received block reward once, round %d", poolId, round)
						}
						poolsPaid[poolId] = true
					}
				}
				numShouldPay := numSharderDelegatesRewarded
				if numShouldPay > len(beforeSharders.Nodes[i].Pools) {
					numShouldPay = len(beforeSharders.Nodes[i].Pools)
				}
				require.Len(t, poolsPaid, numShouldPay,
					"should pay %d pools for shader %s on round %d; %d pools actually paid",
					numShouldPay, id, round, len(poolsPaid))
			}
		}

		// Compare the actual change in rewards to each miner delegate, with the
		// change expected from the delegate reward table.
		for i, _ := range sharderIds {
			delegateBlockReward := int64(float64(bwPerSharder) * (1 - beforeSharders.Nodes[i].Settings.ServiceCharge))
			numPools := len(afterShardedrs.Nodes[i].StakePool.Pools)
			rewards := make(map[string]int64, numPools)
			for poolId := range afterShardedrs.Nodes[i].StakePool.Pools {
				rewards[poolId] = 0
			}
			for round := beforeSharders.Nodes[i].Round + 1; round <= afterShardedrs.Nodes[i].Round; round++ {
				poolsBlockRewards := make(map[string]int64)
				roundHistory := history.RoundHistory(t, round)
				for _, dReward := range roundHistory.DelegateRewards {
					if _, found := rewards[dReward.PoolID]; !found {
						continue
					}
					switch dReward.RewardType {
					case climodel.BlockRewardSharder:
						_, found := poolsBlockRewards[dReward.PoolID]
						require.False(t, found, "pool %s gets more than one block reward on round %d",
							dReward.PoolID, round)
						poolsBlockRewards[dReward.PoolID] = dReward.Amount
						rewards[dReward.PoolID] += dReward.Amount
					case climodel.FeeRewardSharder:
						rewards[dReward.PoolID] += dReward.Amount
					default:
						require.Failf(t, "reward type %s not available to sharders stake pools;"+
							" received by sharder %s on round %d", dReward.RewardType.String(), &dReward.PoolID, round)
					}
				}
				confirmPoolPayments(
					t, delegateBlockReward, poolsBlockRewards, afterShardedrs.Nodes[i].StakePool.Pools, numSharderDelegatesRewarded,
				)
			}
			for poolId := range afterShardedrs.Nodes[i].StakePool.Pools {
				actualReward := afterShardedrs.Nodes[i].StakePool.Pools[poolId].Reward - beforeSharders.Nodes[i].StakePool.Pools[poolId].Reward
				require.InDeltaf(t, actualReward, rewards[poolId], delta,
					"rewards expected %v, change in rewards during test %v", actualReward, rewards[poolId])
			}
		}
	})
}

func getSortedSharderIds(t *test.SystemTest, sharderBaseURL string) []string {
	return getSortedNodeIds(t, "getSharderList", sharderBaseURL)
}
