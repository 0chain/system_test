package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	blockRewardConfigKey       = "block_reward"
	epochConfigKey             = "epoch"
	rewardDeclineRateConfigKey = "reward_decline_rate"
	rewardRateConfigKey        = "reward_rate"
	shareRatioConfigKey        = "share_ratio"
)

func TestBlockRewards(t *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t.Run("Miner share on block fees and rewards", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 3)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		// Get miner list.
		output, err = getMiners(t, configPath)
		require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var miners climodel.NodeList
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
		require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))

		//var miner climodel.Node
		//err = json.Unmarshal([]byte(strings.Join(output, "")), &miner)
		//require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		//require.NotEmpty(t, miner, "No node found: %v", strings.Join(output, "\n"))

		type minerChange struct {
			id            string
			before, after *climodel.Node
		}

		var changes []*minerChange
		for _, m := range miners.Nodes {
			before := getMinersDetail(t, m.ID)
			require.NotNil(t, before, "can't get information for miner ", m.ID)
			changes = append(changes, &minerChange{
				id:     m.ID,
				before: before,
			})
		}
		fee := 0.1
		for i := 0; i < 5; i++ {
			output, err = sendTokens(t, configPath, targetWallet.ClientID, 0.5, escapedTestName(t), fee)
			require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))
		}
		for _, c := range changes {
			c.after = getMinersDetail(t, c.id)
			require.NotNil(t, c.after, "can't get information for miner ", c.id)
		}

		minerScConfig := getMinerScMap(t)
		history := cliutil.NewHistory(changes[0].before.Round, changes[len(changes)-1].after.Round)
		history.ReadBlocks(t, getSharderUrl(t)) //  getNodeBaseURL(sharder.Host, sharder.Port))
		history.DumpTransactions()
		fmt.Println("-----------------------------------")
		//sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)
		//for round := changes[0].before.Round; round < changes[0].after.Round; round++ {
		//	block := getBlock(t, sharderBaseUrl, round)
		//	for _, tx := range block.Block.Transactions {
		//		fmt.Println("tx round", round, "fee", tx.TransactionFee, "data", tx.TransactionData)
		//	}
		//}

		for _, change := range changes {
			minerBlockRewardPerRound, _ := blockRewards(t, change.after.Round, minerScConfig)
			timesWon := history.TimesWonBestMiner(change.id, change.before.Round, change.after.Round)
			expectedBlockRewards := timesWon * minerBlockRewardPerRound
			recordedFees := history.TotalMinerFees(change.id, change.before.Round, change.after.Round)
			expectedRewards := expectedBlockRewards + recordedFees
			actualReward := change.after.Reward - change.before.Reward
			difference := expectedRewards - actualReward

			totalFees := history.TotalFees(change.after.Reward, change.before.Reward)
			fmt.Println("expected Rewards", expectedRewards, "expected block rewards", expectedBlockRewards,
				"recorded fees", recordedFees, "actual rewards", actualReward, "difference expected - actual", difference, "miner")
			fmt.Println("total fees", totalFees)
			require.EqualValues(t, expectedRewards, actualReward)
		}

	})

	t.Run("Sharder share on block fees and rewards", func(t *testing.T) {
		t.Skip("Need investigation")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 3)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		// Get MinerSC Global Config
		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, "get miners sc config failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)

		// Get sharder list.
		output, err = getSharders(t, configPath)
		require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1)
		require.Equal(t, "MagicBlock Sharders", output[0])

		var sharders map[string]climodel.Sharder
		err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
		require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

		// Use first sharder from map.
		require.Greater(t, len(reflect.ValueOf(sharders).MapKeys()), 0)
		selectedSharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

		// Get sharder's node details (this has the total_stake and pools populated).
		output, err = getNode(t, configPath, selectedSharder.ID)
		require.Nil(t, err, "get node %s failed", selectedSharder.ID, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var sharder climodel.Node
		err = json.Unmarshal([]byte(strings.Join(output, "")), &sharder)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.NotEmpty(t, sharder, "No node found: %v", strings.Join(output, "\n"))

		fromMiner := getMinersDetail(t, sharder.ID)
		// Do 5 send transactions with fees
		fee := 0.1
		for i := 0; i < 5; i++ {
			output, err = sendTokens(t, configPath, targetWallet.ClientID, 0.5, escapedTestName(t), fee)
			require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))
		}
		toMiner := getMinersDetail(t, sharder.ID)

		sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)
		history := cliutil.NewHistory(fromMiner.Round, toMiner.Round)
		history.ReadBlocks(t, sharderBaseUrl)
		//	history.TotalFees()

	})
}

func keyValuePairStringToMap(t *testing.T, input []string) (stringMap map[string]string, floatMap map[string]float64) {
	stringMap = map[string]string{}
	floatMap = map[string]float64{}
	for _, tapSeparatedKeyValuePair := range input {
		kvp := strings.Split(tapSeparatedKeyValuePair, "\t")
		var key, val string
		if len(kvp) == 2 {
			key = strings.TrimSpace(kvp[0])
			val = strings.TrimSpace(kvp[1])
		} else if len(kvp) == 1 {
			key = strings.TrimSpace(kvp[0])
			val = ""
		}

		float, err := strconv.ParseFloat(val, 64)
		if err == nil {
			floatMap[key] = float
		}
		stringMap[key] = val
	}
	return
}

func getMinerScMap(t *testing.T) map[string]float64 {
	output, err := getMinerSCConfig(t, configPath, true)
	require.NoError(t, err, "get miners sc config failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)
	_, configAsFloat := keyValuePairStringToMap(t, output)
	return configAsFloat
}

func blockRewards(t *testing.T, round int64, minerScConfig map[string]float64) (int64, int64) {
	epoch := round / int64(minerScConfig["epoch"])
	declineRate := math.Pow(minerScConfig["reward_decline_rate"], float64(epoch))
	blockReward := (minerScConfig["block_reward"] * float64(TOKEN_UNIT)) * declineRate
	minerReward := int64(blockReward * minerScConfig["share_ratio"])
	sharderReward := int64(blockReward) - minerReward
	return minerReward, sharderReward
}

func getSharderUrl(t *testing.T) string {
	// Get sharder list.
	output, err := getSharders(t, configPath)
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 1)
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	//require.Greater(t, len(reflect.ValueOf(sharders).MapKeys()), 0)
	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

	return getNodeBaseURL(sharder.Host, sharder.Port)
}

func getNode(t *testing.T, cliConfigFilename, nodeID string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet mn-info --silent --id "+nodeID+" --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getMiners(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet ls-miners --json --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getSharders(t *testing.T, cliConfigFilename string) ([]string, error) {
	return getShardersForWallet(t, cliConfigFilename, escapedTestName(t))
}

func getShardersForWallet(t *testing.T, cliConfigFilename, wallet string) ([]string, error) {
	return cliutil.RunCommandWithRawOutput("./zwallet ls-sharders --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func getNodeBaseURL(host string, port int) string {
	return fmt.Sprintf(`http://%s:%d`, host, port)
}

func apiGetBalance(sharderBaseURL, clientID string) (*http.Response, error) {
	return http.Get(sharderBaseURL + "/v1/client/get/balance?client_id=" + clientID)
}

func apiGetBlock(sharderBaseURL string, round int64) (*http.Response, error) {
	return http.Get(fmt.Sprintf(sharderBaseURL+"/v1/block/get?content=full&round=%d", round))
}

func getBlock(t *testing.T, sharderBaseUrl string, round int64) apimodel.Block {
	res, err := apiGetBlock(sharderBaseUrl, round)
	require.Nil(t, err, "Error retrieving block %d", round)
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get block %d details: %d", round, res.StatusCode)
	require.NotNil(t, res.Body, "Balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body: %v", err)

	var block apimodel.Block
	err = json.Unmarshal(resBody, &block)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

	return block
}
