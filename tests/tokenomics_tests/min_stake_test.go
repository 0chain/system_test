package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type Reward int

const (
	MinLockDemandReward Reward = iota
	BlockRewardMiner
	BlockRewardSharder
	BlockRewardBlobber
	FeeRewardMiner
	FeeRewardAuthorizer
	FeeRewardSharder
	ValidationReward
	FileDownloadReward
	ChallengePassReward
	ChallengeSlashPenalty
	CancellationChargeReward
	NumOfRewards
)

func TestMinStakeForProviders(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.TestSetup("set storage config to use time_unit as 10 minutes", func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.Cleanup(func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "1h",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	var blobberList []climodel.BlobberInfo
	output, err := utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 2)

	err = json.Unmarshal([]byte(output[1]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var blobberListString []string
	for _, blobber := range blobberList {
		blobberListString = append(blobberListString, blobber.Id)
	}

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	var validatorListString []string
	for _, validator := range validatorList {
		validatorListString = append(validatorListString, validator.ID)
	}

	t.Log("Blobber List: ", blobberListString)
	t.Log("Validator List: ", validatorListString)

	t.RunWithTimeout("miner rewards", 10*time.Minute, func(t *test.SystemTest) {
		sharderUrl := utils.GetSharderUrl(t)
		minerIds := utils.GetSortedMinerIds(t, sharderUrl)

		// When there are no stakes
		for _, minerId := range minerIds {
			blockRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d", minerId, BlockRewardMiner)
			blockReward, err := getQueryRewards(t, blockRewardQuery)
			require.Nil(t, err, "Error getting block reward", blockRewardQuery)
			require.Equal(t, 0.0, blockReward.TotalReward, "Block reward should be 0 for miner %s", minerId)

			feeRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d", minerId, FeeRewardMiner)
			feeReward, err := getQueryRewards(t, feeRewardQuery)
			require.Nil(t, err, "Error getting fee reward", feeRewardQuery)
			require.Equal(t, 0.0, feeReward.TotalReward, "Fee reward should be 0 for miner %s", minerId)
		}

		// When there are stakes less than min stakes per delegate pool
		for _, minerId := range minerIds {
			_, err := utils.ExecuteFaucetWithTokens(t, configPath, 2.0)
			require.Nil(t, err, "error executing faucet")

			output, err = utils.MinerOrSharderLock(t, configPath, utils.CreateParams(map[string]interface{}{
				"miner_id": minerId,
				"tokens":   1.0,
			}), true)
			require.Nil(t, err, "error staking tokens against a node")
			require.Len(t, output, 1)
		}

		time.Sleep(30 * time.Second)

		for _, minerId := range minerIds {
			blockRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d", minerId, BlockRewardMiner)
			blockReward, err := getQueryRewards(t, blockRewardQuery)
			require.Nil(t, err, "Error getting block reward", blockRewardQuery)
			require.Equal(t, 0.0, blockReward.TotalReward, "Block reward should be 0 for miner %s", minerId)

			feeRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d", minerId, FeeRewardMiner)
			feeReward, err := getQueryRewards(t, feeRewardQuery)
			require.Nil(t, err, "Error getting fee reward", feeRewardQuery)
			require.Equal(t, 0.0, feeReward.TotalReward, "Fee reward should be 0 for miner %s", minerId)
		}

		// When there are stakes more than min stakes per delegate pool
		for _, minerId := range minerIds {
			_, err := utils.ExecuteFaucetWithTokens(t, configPath, 15.0)
			require.Nil(t, err, "error executing faucet")

			output, err = utils.MinerOrSharderLock(t, configPath, utils.CreateParams(map[string]interface{}{
				"miner_id": minerId,
				"tokens":   10.0,
			}), true)
			require.Nil(t, err, "error staking tokens against a node")
			require.Len(t, output, 1)
		}

		time.Sleep(30 * time.Second)

		for _, minerId := range minerIds {
			blockRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d", minerId, BlockRewardMiner)
			blockReward, err := getQueryRewards(t, blockRewardQuery)
			require.Nil(t, err, "Error getting block reward", blockRewardQuery)
			require.Greater(t, blockReward.TotalReward, 0.0, "Block reward should be greater than 0 for miner %s", minerId)
			require.Greater(t, blockReward.TotalProviderReward, 0.0, "Block reward should be greater than 0 for miner %s", minerId)
			require.Greater(t, blockReward.TotalDelegateReward, 0.0, "Block reward should be greater than 0 for miner %s", minerId)

			feeRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d", minerId, FeeRewardMiner)
			feeReward, err := getQueryRewards(t, feeRewardQuery)
			require.Nil(t, err, "Error getting fee reward", feeRewardQuery)
			require.Greater(t, feeReward.TotalReward, 0.0, "Fee reward should be greater than 0 for miner %s", minerId)
			require.Greater(t, feeReward.TotalProviderReward, 0.0, "Fee reward should be greater than 0 for miner %s", minerId)
			require.Greater(t, feeReward.TotalDelegateReward, 0.0, "Fee reward should be greater than 0 for miner %s", minerId)
		}
	})

	t.RunWithTimeout("sharder rewards", 10*time.Minute, func(t *test.SystemTest) {

	})

	t.RunWithTimeout("blobber and validator rewards", 10*time.Minute, func(t *test.SystemTest) {

	})
}

type QueryRewardsResponse struct {
	TotalProviderReward float64 `json:"total_provider_reward"`
	TotalDelegateReward float64 `json:"total_delegate_reward"`
	TotalReward         float64 `json:"total_reward"`
}

func getQueryRewards(t *test.SystemTest, query string) (QueryRewardsResponse, error) {
	var result QueryRewardsResponse

	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/query-rewards?query=" + query)

	res, _ := http.Get(url) //nolint:gosec

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)

	body, _ := io.ReadAll(res.Body)

	err := json.Unmarshal(body, &result)
	if err != nil {
		return QueryRewardsResponse{}, err
	}

	return result, nil
}
