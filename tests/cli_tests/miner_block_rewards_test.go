package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const delta = 1.0

func TestMinerBlockRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	if !confirmDebugBuild(t) {
		t.Skip("miner block rewards test skipped as it requires a debug event database")
	}

	// Take a snapshot of the chains miners, then wait a few seconds, take another snapshot.
	// Examine the rewards paid between the two snapshot and confirm the self-consistency
	// of the block reward payments
	//
	// Each round a random miner is chosen to receive the block reward.
	// The miner's service charge is used to determine the fraction received by the miner's wallet.
	//
	// The remaining block reward is then distributed amongst the miner's delegates.
	//
	// A subset of the delegates chosen at random to receive a portion of the block reward.
	// The total received by each stake pool is proportional to the tokens they have locked
	// wither respect to the total locked by the chosen delegate pools.
	t.RunWithTimeout("Miner share of block fees and rewards", 240*time.Second, func(t *test.SystemTest) {
		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

		sharderUrl := getSharderUrl(t)
		minerIds := getSortedMinerIds(t, sharderUrl)
		require.True(t, len(minerIds) > 0, "no miners found")

		beforeMiners := getNodes(t, minerIds, sharderUrl)

		// ------------------------------------
		cliutils.Wait(t, 15*time.Second)
		// ------------------------------------

		afterMiners := getNodes(t, minerIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round

		startRound := beforeMiners.Nodes[0].RoundServiceChargeLastUpdated + 1
		endRound := afterMiners.Nodes[0].RoundServiceChargeLastUpdated + 1
		for i := range beforeMiners.Nodes {
			if startRound > beforeMiners.Nodes[i].RoundServiceChargeLastUpdated {
				startRound = beforeMiners.Nodes[i].RoundServiceChargeLastUpdated
			}
			if endRound < afterMiners.Nodes[i].RoundServiceChargeLastUpdated {
				endRound = afterMiners.Nodes[i].RoundServiceChargeLastUpdated
			}
		}

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl)

		minerScConfig := getMinerScMap(t)
		numMinerDelegatesRewarded := int(minerScConfig["num_miner_delegates_rewarded"])
		require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
			"epoch changed during test, start %v finish %v",
			startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

		minerBlockReward, _ := blockRewards(startRound, minerScConfig)

		// Each round one miner is chosen to receive a block reward.
		// The winning miner is stored in the block object.
		// The reward payments retrieved from the provider reward table.
		// The amount of the reward is a fraction of the block reward allocated to miners each
		// round. The fraction is the miner's service charge. If the miner has
		// no stake pools then the reward becomes the full block reward.
		//
		// Firstly we confirm the self-consistency of the block and reward tables.
		// We calculate the change in the miner rewards during and confirm that this
		// equals the total of the reward payments read from the provider rewards table.
		for i, id := range minerIds {
			var rewards int64
			for round := beforeMiners.Nodes[i].RoundServiceChargeLastUpdated + 1; round <= afterMiners.Nodes[i].RoundServiceChargeLastUpdated; round++ {
				roundHistory := history.RoundHistory(t, round)
				for _, pReward := range roundHistory.ProviderRewards {
					if pReward.ProviderId != id {
						continue
					}
					switch pReward.RewardType {
					case climodel.BlockRewardMiner:
						require.Equalf(t, pReward.ProviderId, roundHistory.Block.MinerID,
							"%s not round lottery winner %s but nevertheless paid with block reward."+
								"only the round lottery winner shold get a miner block reward",
							pReward.ProviderId, roundHistory.Block.MinerID)
						var expectedServiceCharge int64
						if len(beforeMiners.Nodes[i].StakePool.Pools) > 0 {
							expectedServiceCharge = int64(float64(minerBlockReward) * beforeMiners.Nodes[i].Settings.ServiceCharge)
						} else {
							expectedServiceCharge = minerBlockReward
						}
						require.InDeltaf(t, expectedServiceCharge, pReward.Amount, delta, "incorrect service charge %v for round %d"+
							" service charge should be block reward %v multiplied by service ratio %v."+
							" Start round %d end round %d",
							pReward.Amount, round, expectedServiceCharge, minerBlockReward, beforeMiners.Nodes[i].Settings.ServiceCharge,
							beforeMiners.Nodes[i].RoundServiceChargeLastUpdated+1, afterMiners.Nodes[i].RoundServiceChargeLastUpdated)
						rewards += pReward.Amount
					case climodel.FeeRewardMiner:
						rewards += pReward.Amount
					default:
						require.Failf(t, "reward type %s is not available for miners", pReward.RewardType.String())
					}
				}
			}
			actualReward := afterMiners.Nodes[i].Reward - beforeMiners.Nodes[i].Reward
			require.InDeltaf(t, actualReward, rewards, delta,
				"rewards expected %v, change in miners reward during the test is %v", actualReward, rewards)
		}

		// Each round there should be exactly one block reward payment
		// and this to the blocks' miner.
		for round := startRound + 1; round <= endRound-1; round++ {
			roundHistory := history.RoundHistory(t, round)
			foundBlockRewardPayment := false
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.RewardType == climodel.BlockRewardMiner {
					require.False(t, foundBlockRewardPayment, "blocker reward already paid, only pay miner block rewards once")
					foundBlockRewardPayment = true
					require.Equal(t, pReward.ProviderId, roundHistory.Block.MinerID,
						"round %d, start round %d, finish round %d, block reward paid to %s, should only be paid to round lottery winner %s",
						round, startRound, endRound, pReward.ProviderId, roundHistory.Block.MinerID)
				}
			}
			require.True(t, foundBlockRewardPayment,
				"miner block reward payment not recorded. block rewards should be paid every round.")
		}

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

		for i, id := range minerIds {
			delegateBlockReward := int64(float64(minerBlockReward) * (1 - beforeMiners.Nodes[i].Settings.ServiceCharge))
			numPools := len(afterMiners.Nodes[i].StakePool.Pools)
			rewards := make(map[string]int64, numPools)
			for poolId := range afterMiners.Nodes[i].StakePool.Pools {
				rewards[poolId] = 0
			}
			for round := beforeMiners.Nodes[i].RoundServiceChargeLastUpdated + 1; round <= afterMiners.Nodes[i].RoundServiceChargeLastUpdated; round++ {
				poolsBlockRewarded := make(map[string]int64)
				roundHistory := history.RoundHistory(t, round)
				for _, dReward := range roundHistory.DelegateRewards {
					if _, found := rewards[dReward.PoolID]; !found {
						continue
					}
					switch dReward.RewardType {
					case climodel.BlockRewardMiner:
						_, found := poolsBlockRewarded[dReward.PoolID]
						require.False(t, found, "delegate pool %s paid a block reward more than once on round %d",
							dReward.PoolID, round)
						poolsBlockRewarded[dReward.PoolID] = dReward.Amount
						rewards[dReward.PoolID] += dReward.Amount
					case climodel.FeeRewardMiner:
						rewards[dReward.PoolID] += dReward.Amount
					default:
						require.Failf(t, "reward type %s not paid to miner delegate pools", dReward.RewardType.String())
					}
				}
				if roundHistory.Block.MinerID != id {
					require.Len(t, poolsBlockRewarded, 0,
						"delegate pools should not get a block reward unless their parent miner won the round lottery")
				}
				confirmPoolPayments(
					t, delegateBlockReward, poolsBlockRewarded, afterMiners.Nodes[i].StakePool.Pools, numMinerDelegatesRewarded,
				)
			}
			for poolId := range afterMiners.Nodes[i].StakePool.Pools {
				actualReward := afterMiners.Nodes[i].StakePool.Pools[poolId].Reward - beforeMiners.Nodes[i].StakePool.Pools[poolId].Reward
				require.InDeltaf(t, actualReward, rewards[poolId], delta,
					"poolID %s, rewards expected %v change in pools reward during test %v", poolId, rewards[poolId], startRound,
				)
			}
		}
	})
}

func confirmPoolPayments(
	t *test.SystemTest,
	blockReward int64,
	poolsBlockRewarded map[string]int64,
	pools map[string]*climodel.DelegatePool,
	numRewards int,
) {
	if len(poolsBlockRewarded) == 0 {
		return
	}
	if numRewards > len(pools) {
		numRewards = len(pools)
	}
	require.Equal(t, len(poolsBlockRewarded), numRewards,
		"expected reward payments %d does not equal actual payment count %d", numRewards, len(poolsBlockRewarded))
	var total float64
	for id := range poolsBlockRewarded {
		total += float64(pools[id].Balance)
	}
	for id, reward := range poolsBlockRewarded {
		expectedReward := (float64(pools[id].Balance) / total) * float64(blockReward)
		require.InDeltaf(t, expectedReward, float64(reward), 1,
			"delegate rewards. delegates should be rewarded in proportion to their stake."+
				"total reward %d stake pools %v", blockReward, pools)
	}
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

func confirmDebugBuild(t *test.SystemTest) bool {
	globalCfg := getGlobalConfiguration(t, true)
	value, found := globalCfg["server_chain.dbs.settings.debug"]
	require.True(t, found, "server_chain.dbs.settings.debug setting does not exists")
	debug, err := strconv.ParseBool(value.(string))
	require.NoErrorf(t, err, "edb debug should be boolean, actual value %v", value)
	return debug
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
	t *test.SystemTest,
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
		t.Log("unexpect setting key", key, "value", value)
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
	t.Logf("getting sharder url...")
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
	t.Logf("getting a miner or sharder node...")
	return cliutil.RunCommand(t, "./zwallet mn-info --silent --id "+nodeID+" --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getSortedMinerIds(t *test.SystemTest, sharderBaseURL string) []string {
	return getSortedNodeIds(t, "getMinerList", sharderBaseURL)
}

func getSortedNodeIds(t *test.SystemTest, endpoint, sharderBaseURL string) []string {
	t.Logf("getting miner or sharder nodes...")
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
	t.Logf("getting miner or sharder nodes...")
	url := sharderBaseURL + "/test/screst/nodeStat"
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
	t.Logf("list sharder nodes...")
	return cliutil.RunCommandWithRawOutput("./zwallet ls-sharders --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func getNodeBaseURL(host string, port int) string {
	return fmt.Sprintf(`http://%s:%d`, host, port)
}

func getMinersForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	t.Log("list miner nodes...")
	return cliutil.RunCommandWithRawOutput("./zwallet ls-miners --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func apiGetBalance(t *test.SystemTest, sharderBaseURL, clientID string) (*http.Response, error) {
	t.Logf("Getting balance for %s...", clientID)
	return http.Get(sharderBaseURL + "/v1/client/get/balance?client_id=" + clientID)
}

func apiGetBlock(t *test.SystemTest, sharderBaseURL string, round int64) (*http.Response, error) {
	t.Logf("Gert block for round %d...", round)
	return http.Get(fmt.Sprintf(sharderBaseURL+"/v1/block/get?content=full&round=%d", round))
}
func getMiners(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	t.Log("Get miners...")
	return cliutil.RunCommand(t, "./zwallet ls-miners --json --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}
