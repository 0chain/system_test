package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBlockRewards(t *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t.Run("Miner share on block fees and rewards", func(t *testing.T) {

		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

		sharderUrl := getSharderUrl(t)
		beforeMiners := getSortedMiners(t, sharderUrl)
		require.True(t, len(beforeMiners.Nodes) > 0, "no miners found")

		time.Sleep(time.Second * 10)

		afterMiners := getSortedMiners(t, sharderUrl)
		require.EqualValues(t, len(afterMiners.Nodes), len(beforeMiners.Nodes), "miner count changed during test")

		// we add rewards at the end of the round, and they don't appear until the next round
		startRound := beforeMiners.Nodes[0].Round + 1
		endRound := afterMiners.Nodes[0].Round + 1
		for i, m := range beforeMiners.Nodes {
			require.EqualValues(t, m.ID, afterMiners.Nodes[i].ID, "miners changed during test")
			require.EqualValues(t, startRound-1, m.Round)
			require.EqualValues(t, endRound-1, afterMiners.Nodes[i].Round)
		}

		minerScConfig := getMinerScMap(t)
		history := cliutil.NewHistory(startRound, endRound)
		history.ReadBlocks(t, sharderUrl)

		require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
			"epoch changed during test, start %v finish %v",
			startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

		minerBlockRewardPerRound, _ := blockRewards(t, startRound, minerScConfig)
		for i, beforeMiner := range beforeMiners.Nodes {
			id := beforeMiner.ID
			timesWon := history.TimesWonBestMiner(id)
			expectedBlockRewards := timesWon * minerBlockRewardPerRound
			recordedFees := history.TotalMinerFees(id)
			expectedRewards := expectedBlockRewards + recordedFees
			actualReward := afterMiners.Nodes[i].Reward - beforeMiner.Reward
			require.EqualValues(t, expectedRewards, actualReward, "actual rewards don't match expected rewards")
		}
	})

	t.Run("Sharder share on block fees and rewards", func(t *testing.T) {

		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

		sharderUrl := getSharderUrl(t)
		beforeSharders := getSortedSharders(t, sharderUrl)
		require.True(t, len(beforeSharders.Nodes) > 0, "no miners found")

		time.Sleep(time.Second * 10)

		afterSharders := getSortedSharders(t, sharderUrl)
		require.EqualValues(t, len(afterSharders.Nodes), len(beforeSharders.Nodes), "miner count changed during test")

		startRound := beforeSharders.Nodes[0].Round + 1
		endRound := afterSharders.Nodes[0].Round + 1
		for i, m := range beforeSharders.Nodes {
			require.EqualValues(t, m.ID, afterSharders.Nodes[i].ID, "miners changed during test")
			require.EqualValues(t, startRound-1, m.Round)
			require.EqualValues(t, endRound-1, afterSharders.Nodes[i].Round)
		}

		minerScConfig := getMinerScMap(t)
		require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
			"epoch changed during test, start %v finish %v",
			startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

		_, sharderBlockRewards := blockRewards(t, startRound, minerScConfig)
		numberOfRounds := endRound - startRound
		totalBlockRewardsPerSharder := numberOfRounds * sharderBlockRewards / int64(len(beforeSharders.Nodes))
		for i, beforeSharder := range beforeSharders.Nodes {
			expectedBlockRewards := totalBlockRewardsPerSharder
			actualReward := afterSharders.Nodes[i].Reward - beforeSharder.Reward
			require.EqualValues(t, expectedBlockRewards, actualReward)
		}
	})
}

func initialiseTest(t *testing.T, wallet string, funds bool) string {
	output, err := registerWallet(t, configPath)
	require.NoError(t, err, "registering wallet failed", strings.Join(output, "\n"))

	if funds {
		output, err = executeFaucetWithTokens(t, configPath, 3)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))
	}

	output, err = registerWalletForName(t, configPath, wallet)
	require.NoError(t, err, "error registering target wallet", strings.Join(output, "\n"))

	targetWallet, err := getWalletForName(t, configPath, wallet)
	require.NoError(t, err, "error getting target wallet", strings.Join(output, "\n"))
	return targetWallet.ClientID
}

func keyValuePairStringToMap(input []string) (stringMap map[string]string, floatMap map[string]float64) {
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

type settingMaps struct {
	Messages map[string]string
	Keys     map[string]string // keys are hexadecimal of length 64
	Numeric  map[string]float64
	Boolean  map[string]bool
	Duration map[string]int64
}

func newSettingMaps() *settingMaps {
	return &settingMaps{
		Messages: make(map[string]string),
		Keys:     make(map[string]string),
		Numeric:  make(map[string]float64),
		Boolean:  make(map[string]bool),
		Duration: make(map[string]int64),
	}
}

func keyValueSettingsToMap(
	input []string,
) settingMaps {
	const sdkPrefix = "0chain-core-sdk"
	const keyLength = 64
	var settings = newSettingMaps()
	for _, tapSeparatedKeyValuePair := range input {
		kvp := strings.Split(tapSeparatedKeyValuePair, "\t")
		var key, value string
		if len(kvp) == 2 {
			key = strings.TrimSpace(kvp[0])
			value = strings.TrimSpace(kvp[1])
		} else if len(kvp) == 1 {
			key = strings.TrimSpace(kvp[0])
			value = ""
		}
		float, err := strconv.ParseFloat(value, 64)
		if err == nil {
			settings.Numeric[key] = float
			continue
		}
		boolean, err := strconv.ParseBool(value)
		if err == nil {
			settings.Boolean[key] = boolean
			continue
		}
		duration, err := time.ParseDuration(value)
		if err == nil {
			settings.Duration[key] = int64(duration.Seconds())
			continue
		}
		if len(value) >= keyLength {
			if _, err := hex.DecodeString(value); err == nil {
				settings.Keys[key] = value
				continue
			}
		}
		if len(key) >= len(sdkPrefix) && key[:len(sdkPrefix)] == sdkPrefix {
			settings.Messages[key] = value
			continue
		}
		log.Println("unexpect setting key", key, "value", value)
	}
	return *settings
}

func getMinerScMap(t *testing.T) map[string]float64 {
	output, err := getMinerSCConfig(t, configPath, true)
	require.NoError(t, err, "get miners sc config failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)
	_, floatMap := keyValuePairStringToMap(output)
	return floatMap
}

func blockRewards(t *testing.T, round int64, minerScConfig map[string]float64) (int64, int64) {
	epoch := round / int64(minerScConfig["epoch"])
	epochDecline := 1.0 - minerScConfig["reward_decline_rate"]
	declineRate := math.Pow(epochDecline, float64(epoch))
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

	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

	return getNodeBaseURL(sharder.Host, sharder.Port)
}

func getNode(t *testing.T, cliConfigFilename, nodeID string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet mn-info --silent --id "+nodeID+" --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getMiners(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet ls-miners --json --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func apiGetMiners(sharderBaseURL string) (*http.Response, error) {
	return http.Get(sharderBaseURL + "/v1/screst/" + minerSmartContractAddress + "/getMinerList")
}

func apiGetSharders(sharderBaseURL string) (*http.Response, error) {
	return http.Get(sharderBaseURL + "/v1/screst/" + minerSmartContractAddress + "/getSharderList")
}

func getSortedMiners(t *testing.T, sharderBaseURL string) climodel.NodeList {
	res, err := apiGetMiners(sharderBaseURL)
	require.NoError(t, err, "retrieving miners")
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"gailed API request to get miners, status code: %d", res.StatusCode)
	require.NotNil(t, res.Body, "balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)

	var miners climodel.NodeList
	err = json.Unmarshal(resBody, &miners)
	require.NoError(t, err, "deserializing JSON string `%s`: %v", string(resBody), err)
	sort.Slice(miners.Nodes, func(i, j int) bool {
		return miners.Nodes[i].ID < miners.Nodes[j].ID
	})
	return miners
}

func getSortedSharders(t *testing.T, sharderBaseURL string) climodel.NodeList {
	res, err := apiGetSharders(sharderBaseURL)
	require.NoError(t, err, "retrieving miners")
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"gailed API request to get sharders, status code: %d", res.StatusCode)
	require.NotNil(t, res.Body, "balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)

	var sharders climodel.NodeList
	err = json.Unmarshal(resBody, &sharders)
	require.NoError(t, err, "deserializing JSON string `%s`: %v", string(resBody), err)
	sort.Slice(sharders.Nodes, func(i, j int) bool {
		return sharders.Nodes[i].ID < sharders.Nodes[j].ID
	})
	return sharders
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
