package cli_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
)

func TestMinerFeeRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	// Take a snapshot of the chains miners, repeat a transaction with a fee a few times,
	// take another snapshot.
	// Examine the rewards paid between the two snapshot and confirm the self-consistency
	// of the block reward payments
	//
	// Each round a random miner is chosen to receive the rewards for transactions.
	// The miner's service charge is used to determine the fraction received by the miner's wallet.
	// The remaining block reward is then distributed amongst the miner's delegates as follows.
	// A subset of the delegates chosen at random to receive a portion of the block reward.
	// The total received by each stake pool is proportional to the tokens they have locked
	// wither respect to the total locked by the chosen delegate pools.
	t.RunSequentially("Miner share of fee rewards for transactions", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		wallet, err := getWalletForName(t, configPath, escapedTestName(t)+"_TARGET")
		require.NoError(t, err, "error getting target wallet", strings.Join(output, "\n"))

		if !confirmDebugBuild(t) {
			t.Skip("miner fee rewards test skipped as it requires a debug event database")
		}
		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		sharderUrl := getSharderUrl(t)
		minerIds := getSortedMinerIds(t, sharderUrl)
		require.True(t, len(minerIds) > 0, "no miners found")

		beforeMiners := getNodes(t, minerIds, sharderUrl)

		// ------------------------------------
		const numPaidTransactions = 3
		const fee = 0.1
		for i := 0; i < numPaidTransactions; i++ {
			output, err := sendTokens(t, configPath, wallet.ClientID, 0.5, escapedTestName(t), fee)
			require.NoError(t, err, "error sending tokens", strings.Join(output, "\n"))
		}
		time.Sleep(time.Second) // give time for last round to be saved
		// ------------------------------------

		afterMiners := getNodes(t, minerIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round

		startRound, endRound := getStartAndEndRounds(
			t, beforeMiners.Nodes, afterMiners.Nodes, nil, nil,
		)

		time.Sleep(time.Second) // give time for last round to be saved

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl, true)

		balanceMinerIncome(
			t, startRound, endRound, minerIds, beforeMiners.Nodes, afterMiners.Nodes, history,
		)
	})
}

func balanceMinerIncome(
	t *test.SystemTest,
	startRound, endRound int64,
	minerIds []string,
	beforeMiners, afterMiners []climodel.Node,
	history *cliutil.ChainHistory,
) {
	minerScConfig := getMinerScMap(t)
	checkMinerFeeAmounts(
		t,
		minerIds,
		minerScConfig["share_ratio"],
		beforeMiners, afterMiners,
		history,
	)
	checkMinerFeeRewardFrequency(
		t, startRound+1, endRound-1, history,
	)
	checkMinerDelegatePoolFeeAmounts(
		t,
		minerIds,
		int(minerScConfig["num_miner_delegates_rewarded"]),
		minerScConfig["share_ratio"],
		beforeMiners, afterMiners,
		history,
	)
}

// checkMinerRewards
// Each round one miner is chosen to receive a that rounds rewards.
// The winning miner is stored in the block object.
// The reward payments retrieved from the provider reward table.
// The amount of the reward is a fraction of the rewards allocated to miners each
// round. The fraction is the miner's service charge. If the miner has
// no stake pools then the reward becomes the full block reward.
//
// Firstly we confirm the self-consistency of the block and reward payments.
// We calculate the change in each miner's rewards during and confirm that this
// equals the total of the reward payments read from the rewards table.
func checkMinerFeeAmounts(
	t *test.SystemTest,
	minerIds []string,
	minerShare float64,
	beforeMiners, afterMiners []climodel.Node,
	history *cliutil.ChainHistory,
) {
	t.Log("checking miner fee payment amounts...")
	for i, id := range minerIds {
		var blockRewards, feeRewards int64
		for round := beforeMiners[i].RoundServiceChargeLastUpdated + 1; round <= afterMiners[i].RoundServiceChargeLastUpdated; round++ {
			var recordedRoundRewards int64
			var roundFees = history.FeesForRound(t, round)
			roundHistory := history.RoundHistory(t, round)
			var fees int64
			if len(beforeMiners[i].StakePool.Pools) > 0 {
				fees = int64(float64(roundFees) * beforeMiners[i].Settings.ServiceCharge * minerShare)
			} else {
				fees = int64(float64(roundFees) * minerShare)
			}
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.ProviderId != id {
					continue
				}
				switch pReward.RewardType {
				case climodel.FeeRewardMiner:
					require.Falsef(t, beforeMiners[i].IsKilled,
						"killed miners cannot receive fees, %s is killed", id)
					feeRewards += pReward.Amount
					recordedRoundRewards += pReward.Amount
				case climodel.BlockRewardMiner:
					blockRewards += pReward.Amount
				default:
					require.Failf(t, "", "reward type %s is not available for miners", pReward.RewardType.String())
				}
			}
			// if this miner is the round miner check fees add up
			if id == roundHistory.Block.MinerID {
				require.InDeltaf(t, fees, recordedRoundRewards, delta,
					"incorrect service charge %v for round %d"+
						" service charge should be fees %d multiplied by service ratio %v."+
						"length stake pools %d",
					"round history %v",
					recordedRoundRewards, round, fees, beforeMiners[i].Settings.ServiceCharge,
					len(beforeMiners[i].StakePool.Pools), roundHistory)
			}
		}
		actualReward := afterMiners[i].Reward - beforeMiners[i].Reward
		require.InDeltaf(t, actualReward, blockRewards+feeRewards, delta,
			"rewards expected %v, change in miners reward during the test is %v", actualReward, blockRewards+feeRewards)
	}
}

// checkCountOfFeePayments
// Each round there should be zero or one fee reward payment depending on the count of
// transaction with a fee. This should be paid to the blocks' miner and its delegates.
func checkMinerFeeRewardFrequency(
	t *test.SystemTest,
	start, end int64,
	history *cliutil.ChainHistory,
) {
	t.Log("checking number of fee payments...")
	for round := start; round <= end; round++ {
		roundHistory := history.RoundHistory(t, round)
		isAFeePayment := history.FeesForRound(t, round) > 0
		foundFeeRewardPayment := false
		for _, pReward := range roundHistory.ProviderRewards {
			if pReward.RewardType == climodel.FeeRewardMiner {
				require.Falsef(t, foundFeeRewardPayment, "round %d, block reward already paid, only pay miner block rewards once", round)
				foundFeeRewardPayment = true
			}
		}
		require.EqualValues(t, foundFeeRewardPayment, isAFeePayment,
			"rond %d, incorrect number of miner fee reward payments", round)
	}
}

// checkMinerDelegatePoolRewards
// Each round confirm payments to delegates or the blocks winning miner.
// There should be exactly `num_miner_delegates_rewarded` delegates rewarded each round,
// or all delegates if less.
//
// Delegates should be rewarded in proportional to their locked tokens.
// We check the self-consistency of the reward payments each round using
// the delegate reward table.
//
// Next we compare the actual change in rewards to each miner delegate, with the
// change expected from the delegate reward table.
func checkMinerDelegatePoolFeeAmounts(
	t *test.SystemTest,
	minerIds []string,
	numMinerDelegatesRewarded int,
	minerShare float64,
	beforeMiners, afterMiners []climodel.Node,
	history *cliutil.ChainHistory,
) {
	t.Log("checking delegate pool fee payment amounts...")
	for i, id := range minerIds {
		numPools := len(afterMiners[i].StakePool.Pools)
		rewards := make(map[string]int64, numPools)
		for poolId := range afterMiners[i].StakePool.Pools {
			rewards[poolId] = 0
		}
		for round := beforeMiners[i].RoundServiceChargeLastUpdated + 1; round <= afterMiners[i].RoundServiceChargeLastUpdated; round++ {
			poolsBlockRewarded := make(map[string]int64)
			roundHistory := history.RoundHistory(t, round)
			for _, dReward := range roundHistory.DelegateRewards {
				if dReward.ProviderID != id {
					continue
				}
				_, isMinerPool := rewards[dReward.PoolID]
				require.Truef(t, isMinerPool, "round %d, invalid pool id, reward %v", round, dReward)
				switch dReward.RewardType {
				case climodel.FeeRewardMiner:
					_, found := poolsBlockRewarded[dReward.PoolID]
					require.False(t, found, "delegate pool %s paid a fee reward more than once on round %d",
						dReward.PoolID, round)
					poolsBlockRewarded[dReward.PoolID] = dReward.Amount
					rewards[dReward.PoolID] += dReward.Amount
				case climodel.BlockRewardMiner:
					rewards[dReward.PoolID] += dReward.Amount
				default:
					require.Failf(t, "", "reward type %s not paid to miner delegate pools", dReward.RewardType.String())
				}
			}
			if roundHistory.Block.MinerID != id {
				require.Len(t, poolsBlockRewarded, 0,
					"delegate pools should not get a block reward unless their parent miner won the round lottery")
			}
			fees := history.FeesForRound(t, round)
			if fees == 0 {
				require.Equal(t, len(poolsBlockRewarded), 0)
			} else {
				confirmPoolPayments(
					t,
					delegateFeeRewards(
						fees,
						minerShare,
						beforeMiners[i].Settings.ServiceCharge,
						1,
					),
					poolsBlockRewarded,
					afterMiners[i].StakePool.Pools,
					numMinerDelegatesRewarded,
				)
			}
		}
		for poolId := range afterMiners[i].StakePool.Pools {
			actualReward := afterMiners[i].StakePool.Pools[poolId].Reward - beforeMiners[i].StakePool.Pools[poolId].Reward
			require.InDeltaf(t, actualReward, rewards[poolId], delta,
				"poolID %s, rewards expected %v change in pools reward during test", poolId, rewards[poolId],
			)
		}
	}
}

func delegateFeeRewards(total int64, share, serviceCharge float64, numProvidersPaid int) int64 {
	return int64(float64(total) * share * (1 - serviceCharge) / float64(numProvidersPaid))
}
