package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
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

		delegateWallets := loadDelegateWallets(t)

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Get MinerSC Global Config
		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, "get miners sc config failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)

		_, configAsFloat := keyValuePairStringToMap(t, output)

		// Get miner list.
		output, err = getMiners(t, configPath)
		require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var miners climodel.NodeList
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
		require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))

		// Use first miner
		selectedMiner := miners.Nodes[0].SimpleNode

		// Get miner's node details (this has the total_stake and pools populated).
		output, err = getNode(t, configPath, selectedMiner.ID)
		require.Nil(t, err, "get node %s failed", selectedMiner.ID, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var miner climodel.Node
		err = json.Unmarshal([]byte(strings.Join(output, "")), &miner)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.NotEmpty(t, miner, "No node found: %v", strings.Join(output, "\n"))

		// Get sharder list.
		//TODO use sharder delegate wallets instead
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
		sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

		// Get base URL for API calls.
		sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)

		startBeforeRound := getCurrentRound(t)
		//TODO use delegate wallets instead
		require.NotNil(t, delegateWallets[miner.Settings.DelegateWallet], "Miner id is not from delegate wallet list")
		delegateId := delegateWallets[miner.Settings.DelegateWallet].ClientID
		startReward := getBalanceFromSharders(t, delegateId)
		startAfterRound := getCurrentRound(t)
		// Do 5 lock transactions with fees
		params := createParams(map[string]interface{}{
			"durationMin": 1,
			"tokens":      0.1,
			"fee":         0.1,
		})
		for i := 0; i < 5; i++ {
			output, err = lockInterest(t, configPath, params, true)
			require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
			require.Len(t, output, 2)
			require.Equal(t, "Tokens (0.100000) locked successfully", output[0])
			require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		}

		beforeAfterRound := getCurrentRound(t)
		endRewardStr, err := getBalanceForWallet(t, configPath, delegateId)
		require.NoError(t, err)
		endReward, err := strconv.ParseInt(endRewardStr[0], 10, 64)
		require.NoError(t, err)
		endAfterRound := getCurrentRound(t)

		maxTotalRewardsAndFees := int64(0)
		minTotalRewardsAndFees := int64(0)
		// Calculate the total rewards and fees for this miner.
		for round := startBeforeRound + 1; round <= endAfterRound; round++ {
			block := getBlock(t, sharderBaseUrl, round)
			// No expected rewards for this miner if not the generator of block.
			if block.Block.MinerId != miner.ID && block.Block.MinerId != delegateId {
				continue
			}

			// Get total block fees
			blockFees := int64(0)
			for _, txn := range block.Block.Transactions {
				blockFees += txn.TransactionFee
			}

			// reward rate declines per epoch
			// new reward ratio = current reward rate * (1.0 - reward decline rate)
			epochs := round / int64(configAsFloat[epochConfigKey])
			rewardRate := configAsFloat[rewardRateConfigKey] * math.Pow(1.0-configAsFloat[rewardDeclineRateConfigKey], float64(epochs))

			// block reward (mint) = block reward (configured) * reward rate
			blockRewardMint := configAsFloat[blockRewardConfigKey] * 1e10 * rewardRate

			// generator rewards = block reward * share ratio
			generatorRewards := blockRewardMint * configAsFloat[shareRatioConfigKey]

			// generator reward service charge = generator rewards * service charge
			generatorRewardServiceCharge := generatorRewards * miner.Settings.ServiceCharge
			generatorRewardsRemaining := generatorRewards - generatorRewardServiceCharge

			// generator fees = block fees * share ratio
			generatorFees := float64(blockFees) * configAsFloat[shareRatioConfigKey]

			// generator fee service charge = generator fees * service charge
			generatorFeeServiceCharge := generatorFees * miner.Settings.ServiceCharge
			generatorFeeRemaining := generatorFees - generatorFeeServiceCharge

			maxTotalRewardsAndFees += int64(generatorRewardServiceCharge)
			maxTotalRewardsAndFees += int64(generatorFeeServiceCharge)
			minTotalRewardsAndFees += int64(generatorRewardServiceCharge)
			minTotalRewardsAndFees += int64(generatorFeeServiceCharge)
			// if none staked at node, node gets all rewards.
			// otherwise, then remaining are distributed to stake holders.
			if miner.TotalStake == 0 {
				maxTotalRewardsAndFees += int64(generatorRewardsRemaining)
				maxTotalRewardsAndFees += int64(generatorFeeRemaining)
				minTotalRewardsAndFees += int64(generatorRewardsRemaining)
				minTotalRewardsAndFees += int64(generatorFeeRemaining)
			}
			if round < startAfterRound || beforeAfterRound < round {
				maxTotalRewardsAndFees += int64(generatorRewardServiceCharge)
				maxTotalRewardsAndFees += int64(generatorFeeServiceCharge)
				// if none staked at node, node gets all rewards.
				// otherwise, then remaining are distributed to stake holders.
				if miner.TotalStake == 0 {
					maxTotalRewardsAndFees += int64(generatorRewardsRemaining)
					maxTotalRewardsAndFees += int64(generatorFeeRemaining)
				}
			}

		}
		delta := float64(maxTotalRewardsAndFees - minTotalRewardsAndFees)
		rewardEarned := endReward - startReward
		assert.InDeltaf(t, minTotalRewardsAndFees, rewardEarned, delta, "total share difference %d is not within range %d", rewardEarned, delta)
	})

	t.Run("Sharder share on block fees and rewards", func(t *testing.T) {
		t.Skip("fails too often needs investigation")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		delegateWallets := loadDelegateWallets(t)

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Get MinerSC Global Config
		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, "get miners sc config failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)

		_, configAsFloat := keyValuePairStringToMap(t, output)

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

		// Get base URL for API calls.
		sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)

		startBeforeRound := getCurrentRound(t)
		require.NotNil(t, delegateWallets[sharder.Settings.DelegateWallet], "Sharder id is not from delegate wallet list")
		sharderDelegate := delegateWallets[sharder.Settings.DelegateWallet].ClientID
		startReward := getBalanceFromSharders(t, sharderDelegate)
		startAfterRound := getCurrentRound(t)

		// Do 5 lock transactions with fees
		params := createParams(map[string]interface{}{
			"durationMin": 1,
			"tokens":      0.1,
			"fee":         0.1,
		})
		for i := 0; i < 5; i++ {
			output, err = lockInterest(t, configPath, params, true)
			require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
			require.Len(t, output, 2)
			require.Equal(t, "Tokens (0.100000) locked successfully", output[0])
			require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		}

		beforeAfterRound := getCurrentRound(t)
		endReward := getBalanceFromSharders(t, sharderDelegate)
		endAfterRound := getCurrentRound(t)

		maxTotalRewardsAndFees := int64(0)
		minTotalRewardsAndFees := int64(0)
		// Calculate the total rewards and fees for this sharder.
		for round := startBeforeRound + 1; round <= endAfterRound; round++ {
			block := getBlock(t, sharderBaseUrl, round)

			// Get total block fees
			blockFees := int64(0)
			for _, txn := range block.Block.Transactions {
				blockFees += txn.TransactionFee
			}

			// reward rate declines per epoch
			// new reward ratio = current reward rate * (1.0 - reward decline rate)
			epochs := round / int64(configAsFloat[epochConfigKey])
			rewardRate := configAsFloat[rewardRateConfigKey] * math.Pow(1.0-configAsFloat[rewardDeclineRateConfigKey], float64(epochs))

			// block reward (mint) = block reward (configured) * reward rate
			blockRewardMint := configAsFloat[blockRewardConfigKey] * 1e10 * rewardRate

			// generator rewards = block reward * share ratio
			// sharders rewards  = block reward - generator rewards
			sharderRewards := blockRewardMint * (1 - configAsFloat[shareRatioConfigKey])
			sharderRewardsShare := sharderRewards / float64(len(sharders))

			// sharder reward service charge = sharders rewards * service charge
			sharderRewardServiceCharge := sharderRewardsShare * sharder.Settings.ServiceCharge
			sharderRewardsRemaining := sharderRewardsShare - sharderRewardServiceCharge

			// generator fees = block fees * share ratio
			// sharders fees  = block fees - generator fees
			sharderFees := float64(blockFees) * (1 - configAsFloat[shareRatioConfigKey])
			sharderFeesShare := sharderFees / float64(len(sharders))

			// sharder fee service charge = sharders fees * service charge
			sharderFeeServiceCharge := sharderFeesShare * sharder.Settings.ServiceCharge
			sharderFeeRemaining := sharderFeesShare - sharderFeeServiceCharge

			maxTotalRewardsAndFees += int64(sharderRewardServiceCharge)
			maxTotalRewardsAndFees += int64(sharderFeeServiceCharge)
			minTotalRewardsAndFees += int64(sharderRewardServiceCharge)
			minTotalRewardsAndFees += int64(sharderFeeServiceCharge)
			// if none staked at node, node gets all rewards
			// otherwise, then remaining are distributed to stake holders.
			if sharder.TotalStake == 0 {
				maxTotalRewardsAndFees += int64(sharderRewardsRemaining)
				maxTotalRewardsAndFees += int64(sharderFeeRemaining)
				minTotalRewardsAndFees += int64(sharderRewardsRemaining)
				minTotalRewardsAndFees += int64(sharderFeeRemaining)
			}

			if round < startAfterRound || beforeAfterRound < round {
				maxTotalRewardsAndFees += int64(sharderRewardServiceCharge)
				maxTotalRewardsAndFees += int64(sharderFeeServiceCharge)
				// if none staked at node, node gets all rewards
				// otherwise, then remaining are distributed to stake holders.
				if sharder.TotalStake == 0 {
					maxTotalRewardsAndFees += int64(sharderRewardsRemaining)
					maxTotalRewardsAndFees += int64(sharderFeeRemaining)
				}
			}
		}
		delta := float64(maxTotalRewardsAndFees - minTotalRewardsAndFees)
		rewardEarned := endReward - startReward
		assert.InDelta(t, minTotalRewardsAndFees, rewardEarned, delta, "total share difference %d is not within range %d", rewardEarned, delta)
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
