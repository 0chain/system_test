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

func TestSharderFeeRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	t.Skip("wait for duplicate transaction issue to be solved, https://github.com/0chain/0chain/issues/2348")

	// Take a snapshot of the chains sharders, then repeat a transaction with a fee a few times, take another snapshot.
	// Examine the rewards paid between the two snapshot and confirm the self-consistency
	// of the reward payments
	//
	// Each round a random sharder is chosen to receive the block reward.
	// The sharder's service charge is used to determine the fraction received by the sharder's wallet.
	// The remaining reward is then distributed amongst the sharder's delegates.
	// A subset of the delegates chosen at random to receive a portion of the block reward.
	// The total received by each stake pool is proportional to the tokens they have locked
	// wither respect to the total locked by the chosen delegate pools.
	t.RunSequentially("Sharder share of fee rewards for transactions", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		wallet, err := getWalletForName(t, configPath, escapedTestName(t)+"_TARGET")
		require.NoError(t, err, "error getting target wallet", strings.Join(output, "\n"))

		if !confirmDebugBuild(t) {
			t.Skip("sharder fee rewards test skipped as it requires a debug event database")
		}

		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		sharderUrl := getSharderUrl(t)
		var sharderIds []string
		var beforeSharders climodel.NodeList
		sharderIds, beforeSharders = waitForNSharder(t, sharderUrl, 1)

		// ------------------------------------
		const numPaidTransactions = 3
		const fee = 0.1
		for i := 0; i < numPaidTransactions; i++ {
			output, err := sendTokens(t, configPath, wallet.ClientID, 0.5, escapedTestName(t), fee)
			require.NoError(t, err, "error sending tokens", strings.Join(output, "\n"))
		}
		time.Sleep(time.Second) // give time for last round to be saved
		// ------------------------------------

		afterSharders := getNodes(t, sharderIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round
		startRound, endRound := getStartAndEndRounds(
			t, nil, nil, beforeSharders.Nodes, afterSharders.Nodes,
		)

		time.Sleep(time.Second) // give time for last round to be saved

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl, true)

		balanceSharderIncome(
			t, startRound, endRound, sharderIds, beforeSharders.Nodes, afterSharders.Nodes, history,
		)
	})
}

func balanceSharderIncome(
	t *test.SystemTest,
	startRound, endRound int64,
	sharderIds []string,
	beforeSharders, afterSharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
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
		beforeSharders, afterSharders,
		history,
	)
	checkSharderFeeRewardFrequency(
		t, startRound+1, endRound-1, numShardersRewarded, history,
	)
	checkSharderDelegatePoolFeeRewardFrequency(
		t,
		numSharderDelegatesRewarded,
		sharderIds,
		beforeSharders,
		history,
	)
	checkSharderDelegatePoolFeeAmounts(
		t,
		sharderIds,
		minerShare,
		numShardersRewarded, numSharderDelegatesRewarded,
		beforeSharders, afterSharders,
		history,
	)
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
		var startRound int64
		if beforeSharders[i].RoundServiceChargeLastUpdated+1 < history.From() {
			startRound = history.From()
		} else {
			startRound = beforeSharders[i].RoundServiceChargeLastUpdated + 1
		}
		for round := startRound; round <= afterSharders[i].RoundServiceChargeLastUpdated; round++ {
			var recordedRoundRewards int64
			fees := int64(float64(history.FeesForRound(t, round)) / float64(numShardersRewarded))
			roundHistory := history.RoundHistory(t, round)
			var feesForSharder int64
			if len(beforeSharders[i].StakePool.Pools) > 0 {
				feesForSharder = int64(float64(fees) * beforeSharders[i].Settings.ServiceCharge * (1 - minerShare))
			} else {
				feesForSharder = int64(float64(fees) * (1 - minerShare))
			}
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.ProviderId != id {
					continue
				}
				switch pReward.RewardType {
				case climodel.FeeRewardSharder:
					require.Falsef(t, beforeSharders[i].IsKilled,
						"killed sharders cannot receive fees, %s is killed", id)
					require.Greaterf(t, feesForSharder, int64(0), "fee reward with no fees, reward %v", pReward)
					feeRewards += pReward.Amount
					recordedRoundRewards += pReward.Amount
				case climodel.BlockRewardSharder:
					blockRewards += pReward.Amount
				default:
					require.Failf(t, "", "reward type %s is not available for sharders", pReward.RewardType.String())
				}
			}
			// If sharder is one of the chosen sharders, check fee payment is correct
			if recordedRoundRewards > 0 {
				require.InDeltaf(t, feesForSharder, recordedRoundRewards, delta,
					"incorrect service charge %v for round %d"+
						" service charge should be fees %d multiplied by service ratio %v."+
						"length stake pools %d",
					recordedRoundRewards, round, fees, beforeSharders[i].Settings.ServiceCharge,
					len(beforeSharders[i].StakePool.Pools))
			}
		}
		actualReward := afterSharders[i].Reward - beforeSharders[i].Reward
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
// There should be exactly `num_sharder_delegates_rewarded` delegates rewarded each round,
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
						"", "reward type %s not paid to sharder delegate pools", dReward.RewardType)
				}
			}
			if fees > 0 {
				confirmPoolPayments(
					t,
					delegateFeeRewards(
						fees,
						1-minerShare,
						beforeSharders[i].Settings.ServiceCharge,
						numShardersRewarded,
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
