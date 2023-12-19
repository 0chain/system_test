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

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
)

const (
	delta          = 1.0
	restApiRetries = 3
)

func TestMinerBlockRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

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
	t.RunSequentially("Miner share of block rewards", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.NoError(t, err, "registering wallet failed", strings.Join(output, "\n"))

		if !confirmDebugBuild(t) {
			t.Skip("miner block rewards test skipped as it requires a debug event database")
		}

		sharderUrl := getSharderUrl(t)
		minerIds := getSortedMinerIds(t, sharderUrl)
		require.True(t, len(minerIds) > 0, "no miners found")

		beforeMiners := getNodes(t, minerIds, sharderUrl)

		// ------------------------------------
		cliutil.Wait(t, 3*time.Second)
		// ------------------------------------

		afterMiners := getNodes(t, minerIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round

		startRound, endRound := getStartAndEndRounds(
			t, beforeMiners.Nodes, afterMiners.Nodes, nil, nil,
		)

		time.Sleep(time.Second) // give time for last round to be saved
		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl, true)

		balanceMinerRewards(
			t, startRound, endRound, minerIds, beforeMiners.Nodes, afterMiners.Nodes, history,
		)
	})
}

func balanceMinerRewards(
	t *test.SystemTest,
	startRound, endRound int64,
	minerIds []string,
	beforeMiners, afterMiners []climodel.Node,
	history *cliutil.ChainHistory,
) {
	minerScConfig := getMinerScMap(t)
	require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
		"epoch changed during test, start %v finish %v",
		startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

	minerBlockReward, _ := blockRewards(startRound, minerScConfig)

	checkMinerBlockRewards(
		t,
		minerIds,
		minerBlockReward,
		beforeMiners, afterMiners,
		history,
	)

	countMinerBlockRewards(
		t, startRound+1, endRound-1, history,
	)

	checkMinerDelegatePoolBlockRewards(
		t,
		minerIds,
		int(minerScConfig["num_miner_delegates_rewarded"]),
		minerBlockReward,
		beforeMiners, afterMiners,
		history,
	)
}

// checkMinerBlockRewards
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
func checkMinerBlockRewards(
	t *test.SystemTest,
	minerIds []string,
	minerBlockReward int64,
	beforeMiners, afterMiners []climodel.Node,
	history *cliutil.ChainHistory,
) {
	for i, id := range minerIds {
		var feeRewards, blockRewards int64
		for round := beforeMiners[i].RoundServiceChargeLastUpdated + 1; round <= afterMiners[i].RoundServiceChargeLastUpdated; round++ {
			roundHistory := history.RoundHistory(t, round)
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.ProviderId != id {
					continue
				}
				switch pReward.RewardType {
				case climodel.BlockRewardMiner:
					require.Falsef(t, beforeMiners[i].IsKilled,
						"killed miners cannot receive rewards, %s is killed", id)
					var expectedServiceCharge int64
					if len(beforeMiners[i].StakePool.Pools) > 0 {
						expectedServiceCharge = int64(float64(minerBlockReward) * beforeMiners[i].Settings.ServiceCharge)
					} else {
						expectedServiceCharge = minerBlockReward
					}
					require.InDeltaf(t, expectedServiceCharge, pReward.Amount, delta,
						"incorrect service charge %v for round %d"+
							" service charge should be block reward %v multiplied by service ratio %v."+
							"length stake pools %d",
						pReward.Amount, round,
						minerBlockReward, beforeMiners[i].Settings.ServiceCharge,
						len(beforeMiners[i].StakePool.Pools))
					blockRewards += pReward.Amount
				case climodel.FeeRewardMiner:
					feeRewards += pReward.Amount
				default:
					require.Failf(t, "", "reward type %s is not available for miners", pReward.RewardType.String())
				}
			}
		}
		actualReward := afterMiners[i].Reward - beforeMiners[i].Reward
		require.InDeltaf(t, actualReward, feeRewards+blockRewards, delta,
			"rewards expected %v, change in miners reward during the test is %v", actualReward, feeRewards+blockRewards)
	}
}

// countMinerBlockRewards
// Each round there should be exactly one block reward payment
// and this to the blocks' miner.
func countMinerBlockRewards(
	t *test.SystemTest,
	start, end int64,
	history *cliutil.ChainHistory,
) {
	for round := start; round <= end; round++ {
		roundHistory := history.RoundHistory(t, round)
		foundBlockRewardPayment := false
		for _, pReward := range roundHistory.ProviderRewards {
			if pReward.RewardType == climodel.BlockRewardMiner {
				require.Falsef(t, foundBlockRewardPayment, "round %d, block reward already paid, only pay miner block rewards once", round)
				foundBlockRewardPayment = true
			}
		}
		if !foundBlockRewardPayment {
			t.Log(foundBlockRewardPayment)
		}
		require.Truef(t, foundBlockRewardPayment,
			"rond %d, miner block reward payment not recorded. block rewards should be paid every round.", round)
	}
	t.Log("about to test delegate pools")
}

// checkMinerDelegatePoolBlockRewards
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
func checkMinerDelegatePoolBlockRewards(
	t *test.SystemTest,
	minerIds []string,
	numMinerDelegatesRewarded int,
	minerBlockReward int64,
	beforeMiners, afterMiners []climodel.Node,
	history *cliutil.ChainHistory,
) {
	for i, id := range minerIds {
		delegateBlockReward := int64(float64(minerBlockReward) * (1 - beforeMiners[i].Settings.ServiceCharge))
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
				case climodel.BlockRewardMiner:
					_, found := poolsBlockRewarded[dReward.PoolID]
					require.False(t, found, "delegate pool %s paid a block reward more than once on round %d",
						dReward.PoolID, round)
					poolsBlockRewarded[dReward.PoolID] = dReward.Amount
					rewards[dReward.PoolID] += dReward.Amount
				case climodel.FeeRewardMiner:
					rewards[dReward.PoolID] += dReward.Amount
				default:
					require.Failf(t, "mismatched reward", "round %d, %s not available for miner", round, dReward.RewardType)
				}
			}
			if roundHistory.Block.MinerID != id {
				require.Len(t, poolsBlockRewarded, 0,
					"delegate pools should not get a block reward unless their parent miner won the round lottery")
			}
			confirmPoolPayments(
				t, delegateBlockReward, poolsBlockRewarded, afterMiners[i].StakePool.Pools, numMinerDelegatesRewarded,
			)
		}
		for poolId := range afterMiners[i].StakePool.Pools {
			actualReward := afterMiners[i].StakePool.Pools[poolId].Reward - beforeMiners[i].StakePool.Pools[poolId].Reward
			require.InDeltaf(t, actualReward, rewards[poolId], delta,
				"poolID %s, rewards expected %v change in pools reward during test", poolId, rewards[poolId],
			)
		}
	}
}

func confirmPoolPayments(
	t *test.SystemTest,
	totalReward int64,
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
		expectedReward := (float64(pools[id].Balance) / total) * float64(totalReward)
		require.InDeltaf(t, expectedReward, float64(reward), 1,
			"delegate rewards. delegates should be rewarded in proportion to their stake."+
				"total reward %d stake pools %v", totalReward, pools)
	}
}

func initialiseTest(t *test.SystemTest, wallet string, funds bool) string {
	output, err := createWalletForName(t, configPath, wallet)
	require.NoError(t, err, "error creating target wallet", strings.Join(output, "\n"))

	if funds {
		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))
	}

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

func getNodeSlice(t *test.SystemTest, endpoint, sharderBaseURL string) []climodel.Node {
	t.Logf("getting miner or sharder nodes...")
	url := sharderBaseURL + "/v1/screst/" + minerSmartContractAddress + "/" + endpoint
	nodeList := cliutil.ApiGetRetries[climodel.NodeList](t, url, nil, restApiRetries)
	return nodeList.Nodes
}

func getSortedNodeIds(t *test.SystemTest, endpoint, sharderBaseURL string) []string {
	nodeList := getNodeSlice(t, endpoint, sharderBaseURL)
	var nodeIds []string
	for i := range nodeList {
		nodeIds = append(nodeIds, nodeList[i].ID)
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
		nodes.Nodes = append(nodes.Nodes, *cliutil.ApiGetRetries[climodel.Node](t, url, params, restApiRetries))
	}
	return nodes
}

func getSharders(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return getShardersForWallet(t, cliConfigFilename, escapedTestName(t))
}

func getShardersForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	t.Logf("list sharder nodes...")
	return cliutil.RunCommandWithRawOutput("./zwallet ls-sharders --active --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func getNodeBaseURL(host string, port int) string {
	return fmt.Sprintf(`http://%s:%d`, host, port)
}

func getMinersForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	t.Log("list miner nodes...")
	return cliutil.RunCommandWithRawOutput("./zwallet ls-miners --active --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func apiGetBalance(t *test.SystemTest, sharderBaseURL, clientID string) (*http.Response, error) {
	t.Logf("Getting balance for %s...", clientID)
	return http.Get(sharderBaseURL + "/v1/client/get/balance?client_id=" + clientID)
}

func getMiners(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	t.Log("Get miners...")
	return cliutil.RunCommand(t, "./zwallet ls-miners --active --json --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getStartAndEndRounds(
	t *test.SystemTest,
	beforeMiners, afterMiners, beforeSharders, afterSharders []climodel.Node,
) (startRound, endRound int64) {
	require.Equal(t, len(beforeMiners), len(afterMiners), "miner count mismatch")
	require.Equal(t, len(beforeSharders), len(afterSharders), "sharder count mismatch")
	if len(beforeMiners) > 0 {
		startRound = beforeMiners[0].RoundServiceChargeLastUpdated - 1
		endRound = afterMiners[0].RoundServiceChargeLastUpdated + 1
	} else {
		require.True(t, len(beforeSharders) > 0, "no miners or sharders found")
		startRound = beforeSharders[0].RoundServiceChargeLastUpdated - 1
		endRound = afterSharders[0].RoundServiceChargeLastUpdated + 1
	}

	for i := range beforeMiners {
		if startRound > beforeMiners[i].RoundServiceChargeLastUpdated {
			startRound = beforeMiners[i].RoundServiceChargeLastUpdated
		}
		if endRound < afterMiners[i].RoundServiceChargeLastUpdated {
			endRound = afterMiners[i].RoundServiceChargeLastUpdated
		}
	}
	for i := range beforeSharders {
		if startRound > beforeSharders[i].RoundServiceChargeLastUpdated {
			startRound = beforeSharders[i].RoundServiceChargeLastUpdated
		}
		if endRound < afterSharders[i].RoundServiceChargeLastUpdated {
			endRound = afterSharders[i].RoundServiceChargeLastUpdated
		}
	}
	t.Logf("start round %d, end round %d", startRound, endRound)
	return startRound, endRound
}
