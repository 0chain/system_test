package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func confirmDebugBuild(t *test.SystemTest) {
	globalCfg := getGlobalConfiguration(t, true)
	value, found := globalCfg["server_chain.dbs.settings.debug"]
	require.True(t, found, "server_chain.dbs.settings.debug setting exists")
	debug, err := strconv.ParseBool(value.(string))
	require.NoError(t, err, "server_chain.dbs.settings.debug id bool setting")
	require.True(t, debug, "this test requires debug event database")
}

func TestBlockRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	confirmDebugBuild(t)

	t.Run("Miner share on block fees and rewards", func(t *test.SystemTest) {
		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

		sharderUrl := getSharderUrl(t)
		minersIds := getSortedMinerIds(t, sharderUrl)
		require.True(t, len(minersIds) > 0, "no miners found")
		minerScConfig := getMinerScMap(t)
		tokens := []float64{1, 0.5}
		cleanupFunc := createStakePools(t, minersIds, tokens)
		t.Cleanup(func() {
			cleanupFunc()
		})

		beforeMiners := getNodes(t, minersIds, sharderUrl)

		// -----------------------------------
		time.Sleep(time.Second * 2)
		// ----------------------------------=

		afterMiners := getNodes(t, minersIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round
		startRound := beforeMiners.Nodes[0].Round + 1
		endRound := afterMiners.Nodes[0].Round + 1
		for i := range beforeMiners.Nodes {
			if startRound < beforeMiners.Nodes[i].Round {
				startRound = beforeMiners.Nodes[i].Round
			}
			if endRound > afterMiners.Nodes[i].Round {
				endRound = afterMiners.Nodes[i].Round
			}
		}

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl)

		require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
			"epoch changed during test, start %v finish %v",
			startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

		minerBlockReward, _ := blockRewards(startRound, minerScConfig)

		for i, id := range minersIds {
			var rewards int64
			for round := beforeMiners.Nodes[i].Round + 1; round <= afterMiners.Nodes[i].Round; round++ {
				roundHistory := history.RoundHistory(t, round)
				for _, pReward := range roundHistory.ProviderRewards {
					if pReward.ProviderId != id {
						continue
					}
					switch pReward.RewardType {
					case climodel.BlockRewardMiner:
						require.Equal(t, pReward.ProviderId, roundHistory.Block.MinerID,
							"block reward only paid to round lottery winner")
						expectedServiceCharge := int64(float64(minerBlockReward) * beforeMiners.Nodes[i].Settings.ServiceCharge)
						require.InDeltaf(t, expectedServiceCharge, pReward.Amount, 1, "service charge round %d", round)
						rewards += pReward.Amount
					case climodel.FeeRewardMiner:
						rewards += pReward.Amount
					default:
						require.Failf(t, "check ,miner reward type %s", pReward.RewardType.String())
					}
				}
			}
			rewardDelta := afterMiners.Nodes[i].Reward - beforeMiners.Nodes[i].Reward
			require.InDeltaf(testSetup, rewardDelta, rewards, 1, "rewards, expected %v got %v", rewardDelta, rewards)
		}
	})
}

func initialiseTest(t *test.SystemTest, wallet string, funds bool) string {
	output, err := registerWallet(t, configPath)
	require.NoError(t, err, "registering wallet failed", strings.Join(output, "\n"))

	if funds {
		output, err = executeFaucetWithTokens(t, configPath, 10)
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

func getMinerScMap(t *test.SystemTest) map[string]float64 {
	output, err := getMinerSCConfig(t, configPath, true)
	require.NoError(t, err, "get miners sc config failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)
	_, floatMap := keyValuePairStringToMap(output)
	return floatMap
}

func blockRewards(round int64, minerScConfig map[string]float64) (minerReward, sharderReward int64) {
	epoch := round / int64(minerScConfig["epoch"])
	epochDecline := 1.0 - minerScConfig["reward_decline_rate"]
	declineRate := math.Pow(epochDecline, float64(epoch))
	blockReward := (minerScConfig["block_reward"] * float64(TOKEN_UNIT)) * declineRate
	minerReward = int64(blockReward * minerScConfig["share_ratio"])
	sharderReward = int64(blockReward) - minerReward
	return minerReward, sharderReward
}

func getSharderUrl(t *test.SystemTest) string {
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

func getNode(t *test.SystemTest, cliConfigFilename, nodeID string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet mn-info --silent --id "+nodeID+" --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getSortedMinerIds(t *test.SystemTest, sharderBaseURL string) []string {
	return getSortedNodeIds(t, "getMinerList", sharderBaseURL)
}

// todo need for sharder rewards test
//func getSortedSharderIds(t *test.SystemTest, sharderBaseURL string) []string { // nolint:
//	return getSortedNodeIds(t, "getSharderList", sharderBaseURL)
//}

func getSortedNodeIds(t *test.SystemTest, endpoint, sharderBaseURL string) []string {
	url := sharderBaseURL + "/v1/screst/" + minerSmartContractAddress + "/" + endpoint
	nodeList := cliutil.ApiGet[climodel.NodeList](t, url, nil)
	var nodeIds []string
	for i := range nodeList.Nodes {
		nodeIds = append(nodeIds, nodeList.Nodes[i].ID)
	}
	sort.Slice(nodeIds, func(i, j int) bool {
		return nodeIds[i] < nodeIds[j]
	})
	return nodeIds
}

func getNodes(t *test.SystemTest, ids []string, sharderBaseURL string) climodel.NodeList {
	url := sharderBaseURL + "/v1/screst/" + minerSmartContractAddress + "/nodeStat"
	params := map[string]string{
		"include_delegates": "true",
	}
	var nodes climodel.NodeList
	for _, id := range ids {
		params["id"] = id
		nodes.Nodes = append(nodes.Nodes, *cliutil.ApiGet[climodel.Node](t, url, params))
	}
	return nodes
}

func getSharders(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return getShardersForWallet(t, cliConfigFilename, escapedTestName(t))
}

func getShardersForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutil.RunCommandWithRawOutput("./zwallet ls-sharders --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func getNodeBaseURL(host string, port int) string {
	return fmt.Sprintf(`http://%s:%d`, host, port)
}

func getMinersForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutil.RunCommandWithRawOutput("./zwallet ls-miners --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func apiGetBalance(sharderBaseURL, clientID string) (*http.Response, error) {
	return http.Get(sharderBaseURL + "/v1/client/get/balance?client_id=" + clientID)
}

func apiGetBlock(sharderBaseURL string, round int64) (*http.Response, error) {
	return http.Get(fmt.Sprintf(sharderBaseURL+"/v1/block/get?content=full&round=%d", round))
}

func createStakePools(
	t *test.SystemTest, providerIds []string, tokens []float64,
) func() {
	require.True(t, len(tokens) > 0, "create greater than zero pools")
	for _, id := range providerIds {
		for delegate := 0; delegate < len(tokens); delegate++ {
			wallet := escapedTestName(t) + "_delegate_" + strconv.Itoa(delegate) + "_node_" + id
			registerWalletWithTokens(t, configPath, wallet, tokens[delegate])
			output, err := minerOrSharderLockForWallet(t, configPath, createParams(map[string]interface{}{
				"id":     id,
				"tokens": tokens[delegate],
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

func getMiners(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet ls-miners --json --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}
