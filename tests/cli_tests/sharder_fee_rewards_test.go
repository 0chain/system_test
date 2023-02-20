package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSharderFeeRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	if !confirmDebugBuild(t) {
		t.Skip("miner block rewards test skipped as it requires a debug event database")
	}

	// Take a snapshot of the chains sharders, then repeat a transaction with a fee a few times, take another snapshot.
	// Examine the rewards paid between the two snapshot and confirm the self-consistency
	// of the reward payments
	//
	// Each round a random sharder is chosen to receive the block reward.
	// The sharder's service charge is used to determine the fraction received by the miner's wallet.
	// The remaining reward is then distributed amongst the sharder's delegates.
	// A subset of the delegates chosen at random to receive a portion of the block reward.
	// The total received by each stake pool is proportional to the tokens they have locked
	// wither respect to the total locked by the chosen delegate pools.
	t.RunWithTimeout("Sharder share of fee rewards for transactions", 500*time.Second, func(t *test.SystemTest) {
		walletId := initialiseTest(t, escapedTestName(t)+"_TARGET", true)
		output, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))
		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		sharderUrl := getSharderUrl(t)
		sharderIds := getSortedSharderIds(t, sharderUrl)
		require.True(t, len(sharderIds) > 1, "this test needs at least two sharders")

		beforeSharders := getNodes(t, sharderIds, sharderUrl)

		// ------------------------------------
		const numPaidTransactions = 3
		const fee = 0.1
		for i := 0; i < numPaidTransactions; i++ {
			output, err := sendTokens(t, configPath, walletId, 0.5, escapedTestName(t), fee)
			require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))
		}
		// ------------------------------------

		afterSharders := getNodes(t, sharderIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round
		startRound := beforeSharders.Nodes[0].RoundServiceChargeLastUpdated + 1
		endRound := afterSharders.Nodes[0].RoundServiceChargeLastUpdated + 1
		for i := range beforeSharders.Nodes {
			if startRound < beforeSharders.Nodes[i].RoundServiceChargeLastUpdated {
				startRound = beforeSharders.Nodes[i].RoundServiceChargeLastUpdated
			}
			if endRound > afterSharders.Nodes[i].RoundServiceChargeLastUpdated {
				endRound = afterSharders.Nodes[i].RoundServiceChargeLastUpdated
			}
			t.Logf("miner %s delegates pools %d", beforeSharders.Nodes[i].ID, len(beforeSharders.Nodes[i].Pools))
		}
		t.Logf("start round %d, end round %d", startRound, endRound)

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl, true)

		minerScConfig := getMinerScMap(t)
		numSharderDelegatesRewarded := int(minerScConfig["num_sharder_delegates_rewarded"])
		var numShardersRewarded int
		if len(sharderIds) > int(minerScConfig["num_sharders_rewarded"]) {
			numShardersRewarded = int(minerScConfig["num_sharders_rewarded"])
		} else {
			numShardersRewarded = len(sharderIds)
		}

		minerShare := minerScConfig["share_ratio"]

		checkSharderFeeAmounts(
			t,
			sharderIds,
			minerShare,
			numShardersRewarded,
			beforeSharders.Nodes, afterSharders.Nodes,
			history,
		)
		checkSharderFeeRewardFrequency(
			t, startRound+1, endRound-1, numShardersRewarded, history,
		)
		checkSharderDelegatePoolFeeRewardFrequency(
			t,
			numSharderDelegatesRewarded,
			sharderIds,
			beforeSharders.Nodes,
			history,
		)
		checkSharderDelegatePoolFeeAmounts(
			t,
			sharderIds,
			minerShare,
			numShardersRewarded, numSharderDelegatesRewarded,
			beforeSharders.Nodes, afterSharders.Nodes,
			history,
		)
	})
}

// checkSharderFeeAmounts
// Each round we select a subset of sharders to receive that round's rewards.
//
// The reward payments retrieved from the provider reward table.
// The reward is evenly spread between the sharders receiving rewards each
// round. Each round each sharder receiving a reward gets a fraction
// determined by that sharder's service charge. If the sharder has
// no stake pools then the reward becomes the full block reward.
//
// Firstly we confirm the self-consistency of the reward tables.
// We calculate the change in each sharder's rewards during and confirm that this
// equals the total of the reward payments as read from the provider rewards table.
func checkSharderFeeAmounts(
	t *test.SystemTest,
	sharderIds []string,
	minerShare float64,
	numShardersRewarded int,
	beforeSharders, afterSharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
	t.Log("checking sharder fee payment amounts...")
	for i, id := range sharderIds {
		var blockRewards, feeRewards int64
		for round := beforeSharders[i].RoundServiceChargeLastUpdated + 1; round <= afterSharders[i].RoundServiceChargeLastUpdated; round++ {
			feesPerSharder := int64(float64(history.FeesForRound(t, round)) / float64(numShardersRewarded))
			roundHistory := history.RoundHistory(t, round)
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.ProviderId != id {
					continue
				}
				switch pReward.RewardType {
				case climodel.FeeRewardSharder:
					require.Greater(t, feesPerSharder, int64(0), "fee reward with no fees")
					var fees int64
					if len(beforeSharders[i].StakePool.Pools) > 0 {
						fees = int64(float64(feesPerSharder) * beforeSharders[i].Settings.ServiceCharge * (1 - minerShare))
					} else {
						fees = int64(float64(feesPerSharder) * (1 - minerShare))
					}
					if fees != pReward.Amount {
						fmt.Println("fees", fees, "reward", pReward.Amount)
					}
					require.InDeltaf(t, fees, pReward.Amount, delta,
						"incorrect service charge %v for round %d"+
							" service charge should be fees %d multiplied by service ratio %v."+
							"length stake pools %d",
						pReward.Amount, round, fees, beforeSharders[i].Settings.ServiceCharge,
						len(beforeSharders[i].StakePool.Pools))
					feeRewards += pReward.Amount
				case climodel.BlockRewardSharder:
					blockRewards += pReward.Amount
				default:
					require.Failf(t, "reward type %s is not available for miners", pReward.RewardType.String())
				}
			}
		}
		actualReward := afterSharders[i].Reward - beforeSharders[i].Reward
		if actualReward != blockRewards+feeRewards {
			fmt.Println("piers actual rewards", actualReward, "block rewards", blockRewards, "fee rewards", feeRewards)
		}

		require.InDeltaf(t, actualReward, blockRewards+feeRewards, delta,
			"rewards expected %v, change in sharder reward during the test is %v", actualReward, blockRewards+feeRewards)
	}
}

// Each round there is a fee, there should be exactly num_sharders_rewarded sharder fee reward payment.
func checkSharderFeeRewardFrequency(
	t *test.SystemTest,
	start, end int64,
	numShardersRewarded int,
	history *cliutil.ChainHistory,
) {
	t.Log("checking number of fee payments...")
	for round := start; round <= end; round++ {
		if history.FeesForRound(t, round) == 0 {
			continue
		}
		roundHistory := history.RoundHistory(t, round)
		shardersPaid := make(map[string]bool)
		for _, pReward := range roundHistory.ProviderRewards {
			if pReward.RewardType == climodel.FeeRewardSharder {
				_, found := shardersPaid[pReward.ProviderId]
				require.Falsef(t, found, "sharder %s receives more than one block reward on round %d", pReward.ProviderId, round)
				shardersPaid[pReward.ProviderId] = true
			}
		}
		require.Equal(t, numShardersRewarded, len(shardersPaid),
			"mismatch between expected count of sharders rewarded and actual number on round %d", round)

	}
}

// checkSharderDelegatePoolFeeRewardFrequency
// Each round there is a fee each sharder rewarded should have num_sharder_delegates_rewarded of
// their delegates rewarded, or all delegates if less.
func checkSharderDelegatePoolFeeRewardFrequency(
	t *test.SystemTest,
	numSharderDelegatesRewarded int,
	sharderIds []string,
	sharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
	t.Log("checking delegate pool reward frequencies...")
	for round := history.From(); round <= history.To(); round++ {
		if history.FeesForRound(t, round) == 0 {
			continue
		}
		roundHistory := history.RoundHistory(t, round)
		for i, id := range sharderIds {
			poolsPaid := make(map[string]bool)
			for poolId := range sharders[i].Pools {
				for _, dReward := range roundHistory.DelegateRewards {
					if dReward.RewardType != climodel.FeeRewardSharder || dReward.PoolID != poolId {
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
			if numShouldPay > len(sharders[i].Pools) {
				numShouldPay = len(sharders[i].Pools)
			}
			require.Len(t, poolsPaid, numShouldPay,
				"should pay %d pools for shader %s on round %d; %d pools actually paid",
				numShouldPay, id, round, len(poolsPaid))
		}
	}
}

// checkSharderDelegatePoolFeeAmounts
// Each round confirm payments to delegates of the selected sharders.
// There should be exactly `num_miner_delegates_rewarded` delegates rewarded each round,
// or all delegates if less.
//
// Delegates should be rewarded in proportional to their locked tokens.
// We check the self-consistency of the reward payments each round using
// the delegate reward table.
//
// Next we compare the actual change in rewards to each sharder's delegates, with the
// change as read from the delegate reward table.
func checkSharderDelegatePoolFeeAmounts(
	t *test.SystemTest,
	sharderIds []string,
	minerShare float64,
	numShardersRewarded, numSharderDelegatesRewarded int,
	beforeSharders, afterSharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
	t.Log("checking sharder delegate pools fee rewards")
	for i, id := range sharderIds {
		numPools := len(afterSharders[i].StakePool.Pools)
		rewards := make(map[string]int64, numPools)
		for poolId := range afterSharders[i].StakePool.Pools {
			rewards[poolId] = 0
		}
		for round := beforeSharders[i].RoundServiceChargeLastUpdated + 1; round <= afterSharders[i].RoundServiceChargeLastUpdated; round++ {
			fees := history.FeesForRound(t, round)
			poolsBlockRewarded := make(map[string]int64)
			roundHistory := history.RoundHistory(t, round)
			for _, dReward := range roundHistory.DelegateRewards {
				if dReward.ProviderID != id {
					continue
				}
				_, isSharderPool := rewards[dReward.PoolID]
				require.Truef(t, isSharderPool, "round %d, invalid pool id, reward %v", round, dReward)
				switch dReward.RewardType {
				case climodel.FeeRewardSharder:
					require.Greater(t, fees, int64(0), "fee reward with no fees")
					_, found := poolsBlockRewarded[dReward.PoolID]
					require.False(t, found, "delegate pool %s paid a fee reward more than once on round %d",
						dReward.PoolID, round)
					poolsBlockRewarded[dReward.PoolID] = dReward.Amount
					rewards[dReward.PoolID] += dReward.Amount
				case climodel.BlockRewardSharder:
					rewards[dReward.PoolID] += dReward.Amount
				default:
					require.Failf(t, "mismatched reward type",
						"reward type %s not paid to miner delegate pools", dReward.RewardType)
				}
			}
			if fees > 0 {
				confirmPoolPayments(
					t,
					delegateSharderFeesRewards(
						numShardersRewarded,
						fees,
						beforeSharders[i].Settings.ServiceCharge,
						1-minerShare,
					),
					poolsBlockRewarded,
					afterSharders[i].StakePool.Pools,
					numSharderDelegatesRewarded,
				)
			}
		}
		for poolId := range afterSharders[i].StakePool.Pools {
			actualReward := afterSharders[i].StakePool.Pools[poolId].Reward - beforeSharders[i].StakePool.Pools[poolId].Reward
			require.InDeltaf(t, actualReward, rewards[poolId], delta,
				"poolID %s, rewards expected %v change in pools reward during test", poolId, rewards[poolId],
			)
		}
	}
}

func delegateSharderFeesRewards(numberSharders int, fee int64, serviceCharge, sharderShare float64) int64 {
	fmt.Println("num sharders", numberSharders, "fee", fee,
		"service charge", serviceCharge, "share", sharderShare, "result",
		int64(float64(fee)*(1-serviceCharge)*sharderShare/float64(numberSharders)))
	return int64(float64(fee) * (1 - serviceCharge) * sharderShare / float64(numberSharders))
}
