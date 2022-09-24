package cli_tests

import (
	"math"
	"strings"
	"testing"
	"time"

	util "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerAndSharderFeesPayment(t *testing.T) {
	t.Run("Miner and sharder share of fee payments and rewards", func(t *testing.T) {
		clientId := initialiseTest(t)
		sharderUrl := getSharderUrl(t)

		beforeMiners := getSortedMiners(t, sharderUrl)
		require.True(t, len(beforeMiners.Nodes) > 0, "no miners found")
		beforeSharders := getSortedSharders(t, sharderUrl)
		require.True(t, len(beforeSharders.Nodes) > 0, "no sharders found")

		fee := 0.1
		for i := 0; i < 5; i++ {
			output, err := sendTokens(t, configPath, clientId, 0.5, escapedTestName(t), fee)
			require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))
		}

		time.Sleep(time.Second * 5)

		afterMiners := getSortedMiners(t, sharderUrl)
		require.EqualValues(t, len(afterMiners.Nodes), len(beforeMiners.Nodes), "miner count changed during test")
		afterSharders := getSortedSharders(t, sharderUrl)
		require.EqualValues(t, len(afterSharders.Nodes), len(beforeSharders.Nodes), "sharder count changed during test")

		startRound := beforeSharders.Nodes[0].Round + 1
		endRound := afterMiners.Nodes[0].Round + 1

		for i, m := range beforeMiners.Nodes {
			require.EqualValues(t, m.ID, afterMiners.Nodes[i].ID, "miners changed during test")
		}
		for i, s := range beforeMiners.Nodes {
			require.EqualValues(t, s.ID, afterMiners.Nodes[i].ID, "sharders changed during test")
		}

		minerScConfig := getMinerScMap(t)
		history := util.NewHistory(startRound, endRound)
		history.ReadBlocks(t, sharderUrl)

		require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
			"epoch changed during test, start %v finish %v",
			startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

		minerBlockRewardPerRound, sharderBlockRewards := blockRewards(t, startRound, minerScConfig)

		for i, beforeMiner := range beforeMiners.Nodes {
			id := beforeMiner.ID
			timesWon := history.TimesWonBestMiner(id)
			expectedBlockRewards := timesWon * minerBlockRewardPerRound
			recordedFees := history.TotalMinerFees(id)
			expectedFees := int64(float64(recordedFees) * minerScConfig["share_ratio"])
			expectedRewards := expectedBlockRewards + expectedFees
			actualReward := afterMiners.Nodes[i].Reward - beforeMiner.Reward
			require.EqualValues(t, expectedRewards, actualReward, "actual rewards don't match expected rewards. before",
				beforeMiner.Reward, "after", afterMiners.Nodes[i].Reward, "difference", expectedRewards-actualReward, "miner id", id)
		}

		numberOfRounds := endRound - startRound
		expectedSharderBlockRewards := numberOfRounds * sharderBlockRewards / int64(len(beforeSharders.Nodes))
		totalFees := float64(history.TotalFees() / int64(len(beforeSharders.Nodes)))
		expectedSharderFees := totalFees * (1 - minerScConfig["share_ratio"])
		expectRewards := int64(math.Round(float64(expectedSharderBlockRewards) + expectedSharderFees))
		for i, beforeSharder := range beforeSharders.Nodes {
			actualReward := afterSharders.Nodes[i].Reward - beforeSharder.Reward
			require.EqualValues(t, expectRewards, actualReward)
		}
	})
}
