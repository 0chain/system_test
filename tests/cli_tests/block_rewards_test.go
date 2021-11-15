package cli_tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	t.Parallel()

	t.Run("Miner share on block fees and rewards", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Get MinerSC Global Config
		output, err = getMinerSCConfig(t, configPath)
		require.Nil(t, err, "get miners sc config failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)

		mconfig := map[string]float64{}
		for _, o := range output {
			configPair := strings.Split(o, "\t")
			val, err := strconv.ParseFloat(strings.TrimSpace(configPair[1]), 64)
			require.Nil(t, err, "config val %s for %s is unexpected not float64: %s", configPair[1], configPair[0], strings.Join(output, "\n"))
			mconfig[strings.TrimSpace(configPair[0])] = val
		}

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

		var nodeRes climodel.Node
		err = json.Unmarshal([]byte(strings.Join(output, "")), &nodeRes)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.NotEmpty(t, nodeRes, "No node found: %v", strings.Join(output, "\n"))

		miner := nodeRes.SimpleNode

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
		sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

		// Get base URL for API calls.
		sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)

		// Get the starting balance for miner's delegate wallet.
		res, err := apiGetBalance(sharderBaseUrl, miner.ID)
		require.Nil(t, err, "Error retrieving miner %s balance", miner.ID)
		require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to check miner %s balance: %d", miner.ID, res.StatusCode)
		require.NotNil(t, res.Body, "Balance API response must not be nil")

		resBody, err := ioutil.ReadAll(res.Body)
		require.Nil(t, err, "Error reading response body")

		var startBalance apimodel.Balance
		err = json.Unmarshal(resBody, &startBalance)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)
		require.NotEmpty(t, startBalance.Txn, "Balance txn is unexpectedly empty: %s", string(resBody))
		require.Positive(t, startBalance.Balance, "Balance is unexpectedly zero or negative: %d", startBalance.Balance)
		require.Positive(t, startBalance.Round, "Round of balance is unexpectedly zero or negative: %d", startBalance.Round)

		// Do 5 lock transactions with fees
		for i := 0; i < 5; i++ {
			output, err = lockInterest(t, configPath, true, 1, false, 0, true, 0.1, 0.1, false)
			require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.100000) locked successfully", output[0])
		}

		// Get the ending balance for miner's delegate wallet.
		res, err = apiGetBalance(sharderBaseUrl, miner.ID)
		require.Nil(t, err, "Error retrieving miner %s balance", miner.ID)
		require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to check miner %s balance: %d", miner.ID, res.StatusCode)
		require.NotNil(t, res.Body, "Balance API response must not be nil")

		resBody, err = ioutil.ReadAll(res.Body)
		require.Nil(t, err, "Error reading response body")

		var endBalance apimodel.Balance
		err = json.Unmarshal(resBody, &endBalance)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)
		require.NotEmpty(t, endBalance.Txn, "Balance txn is unexpectedly empty: %s", string(resBody))
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)
		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)

		totalRewardsAndFees := int64(0)
		// Calculate the total rewards and fees for this miner.
		for round := startBalance.Round + 1; round <= endBalance.Round; round++ {
			// Get round details
			res, err := apiGetBlock(sharderBaseUrl, round)
			require.Nil(t, err, "Error retrieving block %d", round)
			require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get block %d details: %d", round, res.StatusCode)
			require.NotNil(t, res.Body, "Balance API response must not be nil")

			resBody, err = ioutil.ReadAll(res.Body)
			require.Nil(t, err, "Error reading response body: %v", err)

			var block apimodel.Block
			err = json.Unmarshal(resBody, &block)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

			// No expected rewards for this miner if not the generator of block.
			if block.Block.MinerId != miner.ID {
				continue
			}

			// Get total block fees
			blockFees := int64(0)
			for _, txn := range block.Block.Transactions {
				blockFees += txn.TransactionFee
			}

			// reward rate declines per epoch
			// new reward ratio = current reward rate * (1.0 - reward decline rate)
			epochs := round / int64(mconfig[epochConfigKey])
			rewardRate := mconfig[rewardRateConfigKey] * math.Pow(1.0-mconfig[rewardDeclineRateConfigKey], float64(epochs))

			// block reward (mint) = block reward (configured) * reward rate
			blockRewardMint := mconfig[blockRewardConfigKey] * 1e10 * rewardRate

			// generator rewards = block reward * share ratio
			generatorRewards := blockRewardMint * mconfig[shareRatioConfigKey]

			// generator reward service charge = generator rewards * service charge
			generatorRewardServiceCharge := generatorRewards * miner.ServiceCharge
			generatorRewardsRemaining := generatorRewards - generatorRewardServiceCharge

			// generator fees = block fees * share ratio
			generatorFees := float64(blockFees) * mconfig[shareRatioConfigKey]

			// generator fee service charge = generator fees * service charge
			generatorFeeServiceCharge := generatorFees * miner.ServiceCharge
			generatorFeeRemaining := generatorFees - generatorFeeServiceCharge

			totalRewardsAndFees += int64(generatorRewardServiceCharge)
			totalRewardsAndFees += int64(generatorFeeServiceCharge)

			// if none staked at node, node gets all rewards.
			// otherwise, then remaining are distributed to stake holders.
			if miner.TotalStake == 0 {
				totalRewardsAndFees += int64(generatorRewardsRemaining)
				totalRewardsAndFees += int64(generatorFeeRemaining)
			}
		}

		wantBalanceDiff := totalRewardsAndFees
		gotBalanceDiff := endBalance.Balance - startBalance.Balance
		assert.InEpsilonf(t, wantBalanceDiff, gotBalanceDiff, 0.0000001, "expected total share is not close to actual share: want %d, got %d", wantBalanceDiff, gotBalanceDiff)
	})

	t.Run("Sharder share on block fees and rewards", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Get MinerSC Global Config
		output, err = getMinerSCConfig(t, configPath)
		require.Nil(t, err, "get miners sc config failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)

		mconfig := map[string]float64{}
		for _, o := range output {
			configPair := strings.Split(o, "\t")
			val, err := strconv.ParseFloat(strings.TrimSpace(configPair[1]), 64)
			require.Nil(t, err, "config val %s for %s is unexpected not float64: %s", configPair[1], configPair[0], strings.Join(output, "\n"))
			mconfig[strings.TrimSpace(configPair[0])] = val
		}

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
		selectedSharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

		// Get sharder's node details (this has the total_stake and pools populated).
		output, err = getNode(t, configPath, selectedSharder.ID)
		require.Nil(t, err, "get node %s failed", selectedSharder.ID, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var nodeRes climodel.Node
		err = json.Unmarshal([]byte(strings.Join(output, "")), &nodeRes)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.NotEmpty(t, nodeRes, "No node found: %v", strings.Join(output, "\n"))

		sharder := nodeRes.SimpleNode

		// Get base URL for API calls.
		sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)

		// Get the starting balance for sharder's delegate wallet.
		res, err := apiGetBalance(sharderBaseUrl, sharder.ID)
		require.Nil(t, err, "Error retrieving sharder %s balance", sharder.ID)
		require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to check sharder %s balance: %d", sharder.ID, res.StatusCode)
		require.NotNil(t, res.Body, "Balance API response must not be nil")

		resBody, err := ioutil.ReadAll(res.Body)
		require.Nil(t, err, "Error reading response body")

		var startBalance apimodel.Balance
		err = json.Unmarshal(resBody, &startBalance)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)
		require.NotEmpty(t, startBalance.Txn, "Balance txn is unexpectedly empty: %s", string(resBody))
		require.Positive(t, startBalance.Balance, "Balance is unexpectedly zero or negative: %d", startBalance.Balance)
		require.Positive(t, startBalance.Round, "Round of balance is unexpectedly zero or negative: %d", startBalance.Round)

		// Do 5 lock transactions with fees
		for i := 0; i < 5; i++ {
			output, err = lockInterest(t, configPath, true, 1, false, 0, true, 0.1, 0.1, false)
			require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.100000) locked successfully", output[0])
		}

		// Get the ending balance for sharder's delegate wallet.
		res, err = apiGetBalance(sharderBaseUrl, sharder.ID)
		require.Nil(t, err, "Error retrieving sharder %s balance", sharder.ID)
		require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to check sharder %s balance: %d", sharder.ID, res.StatusCode)
		require.NotNil(t, res.Body, "Balance API response must not be nil")

		resBody, err = ioutil.ReadAll(res.Body)
		require.Nil(t, err, "Error reading response body")

		var endBalance apimodel.Balance
		err = json.Unmarshal(resBody, &endBalance)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)
		require.NotEmpty(t, endBalance.Txn, "Balance txn is unexpectedly empty: %s", string(resBody))
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)
		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)

		totalRewardsAndFees := int64(0)
		// Calculate the total rewards and fees for this sharder.
		for round := startBalance.Round + 1; round <= endBalance.Round; round++ {
			// Get round details
			res, err := apiGetBlock(sharderBaseUrl, round)
			require.Nil(t, err, "Error retrieving block %d", round)
			require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get block %d details: %d", round, res.StatusCode)
			require.NotNil(t, res.Body, "Balance API response must not be nil")

			resBody, err = ioutil.ReadAll(res.Body)
			require.Nil(t, err, "Error reading response body: %v", err)

			var block apimodel.Block
			err = json.Unmarshal(resBody, &block)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

			// Get total block fees
			blockFees := int64(0)
			for _, txn := range block.Block.Transactions {
				blockFees += txn.TransactionFee
			}

			// reward rate declines per epoch
			// new reward ratio = current reward rate * (1.0 - reward decline rate)
			epochs := round / int64(mconfig[epochConfigKey])
			rewardRate := mconfig[rewardRateConfigKey] * math.Pow(1.0-mconfig[rewardDeclineRateConfigKey], float64(epochs))

			// block reward (mint) = block reward (configured) * reward rate
			blockRewardMint := mconfig[blockRewardConfigKey] * 1e10 * rewardRate

			// generator rewards = block reward * share ratio
			// sharders rewards  = block reward - generator rewards
			sharderRewards := blockRewardMint * (1 - mconfig[shareRatioConfigKey])
			sharderRewardsShare := sharderRewards / float64(len(sharders))

			// sharder reward service charge = sharders rewards * service charge
			sharderRewardServiceCharge := sharderRewardsShare * sharder.ServiceCharge
			sharderRewardsRemaining := sharderRewardsShare - sharderRewardServiceCharge

			// generator fees = block fees * share ratio
			// sharders fees  = block fees - generator fees
			sharderFees := float64(blockFees) * (1 - mconfig[shareRatioConfigKey])
			sharderFeesShare := sharderFees / float64(len(sharders))

			// sharder fee service charge = sharders fees * service charge
			sharderFeeServiceCharge := sharderFeesShare * sharder.ServiceCharge
			sharderFeeRemaining := sharderFeesShare - sharderFeeServiceCharge

			totalRewardsAndFees += int64(sharderRewardServiceCharge)
			totalRewardsAndFees += int64(sharderFeeServiceCharge)

			// if none staked at node, node gets all rewards
			// otherwise, then remaining are distributed to stake holders.
			if sharder.TotalStake == 0 {
				totalRewardsAndFees += int64(sharderRewardsRemaining)
				totalRewardsAndFees += int64(sharderFeeRemaining)
			}
		}

		wantBalanceDiff := totalRewardsAndFees
		gotBalanceDiff := endBalance.Balance - startBalance.Balance
		assert.InEpsilonf(t, wantBalanceDiff, gotBalanceDiff, 0.0000001, "expected total share is not close to actual share: want %d, got %d", wantBalanceDiff, gotBalanceDiff)
	})
}

func getNode(t *testing.T, cliConfigFilename, nodeID string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet mn-info --silent --id "+nodeID+" --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getMiners(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet ls-miners --json --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getMinerSCConfig(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cliutil.RunCommand(t, "./zwallet mn-config --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getSharders(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cliutil.RunCommandWithRawOutput("./zwallet ls-sharders --json --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
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
