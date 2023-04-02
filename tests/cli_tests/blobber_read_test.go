package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBlobberReadReward(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	prevBlock := getLatestFinalizedBlock(t)

	fmt.Println("prevBlock", prevBlock)

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var validatorList []climodel.Validator
	output, err = listValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	t.RunSequentiallyWithTimeout("Case 1 : 1 delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		remotepath := "/dir/"
		filesize := 50 * MB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		fmt.Println(output)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		fmt.Println("Sleeping for 30 seconds")
		time.Sleep(30 * time.Second)

		downloadCost, _ := getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
		}), true)

		fmt.Println(downloadCost)

		curBlock := getLatestFinalizedBlock(t)

		totalDownloadRewards := getReadRewards("", strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		fmt.Println(totalDownloadRewards)

		require.Equal(t, downloadCost, totalDownloadRewards, "Download cost and total download rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	//t.Skip()

	//t.RunSequentiallyWithTimeout("Case 2 : 1 delegate each, equal stake, upload multiple times", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
	//
	//	stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)
	//
	//	output, err := registerWallet(t, configPath)
	//	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))
	//
	//	// 1. Create an allocation with 1 data shard and 1 parity shard.
	//	allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	//		"size":   500 * MB,
	//		"tokens": 1,
	//		"data":   1,
	//		"parity": 1,
	//		"expire": "20m",
	//	})
	//
	//	remotepath := "/dir/"
	//	filesize := 50 * MB
	//	filename := generateRandomTestFileName(t)
	//
	//	err = createFileWithSize(filename, int64(filesize))
	//	require.Nil(t, err)
	//
	//	output, err = uploadFile(t, configPath, map[string]interface{}{
	//		"allocation": allocationId,
	//		"remotepath": remotepath + filepath.Base(filename),
	//		"localpath":  filename,
	//	}, true)
	//	require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
	//
	//	err = os.Remove(filename)
	//	require.Nil(t, err)
	//
	//	remoteFilepath := remotepath + filepath.Base(filename)
	//
	//	output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
	//		"allocation": allocationId,
	//		"remotepath": remoteFilepath,
	//		"localpath":  os.TempDir() + string(os.PathSeparator),
	//	}), true)
	//	fmt.Println(output)
	//	require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
	//
	//	fmt.Println("Sleeping for 30 seconds")
	//	time.Sleep(30 * time.Second)
	//
	//	downloadCost, _ := getDownloadCost(t, configPath, createParams(map[string]interface{}{
	//		"allocation": allocationId,
	//		"remotepath": remoteFilepath,
	//	}), true)
	//
	//	fmt.Println(downloadCost)
	//
	//	curBlock := getLatestFinalizedBlock(t)
	//
	//	totalDownloadRewards := getReadRewards("", strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)
	//
	//	fmt.Println(totalDownloadRewards)
	//
	//	require.Equal(t, downloadCost, totalDownloadRewards, "Download cost and total download rewards are not equal")
	//})
}

func getTotalRewardsByRewardType(rewardType int) int {
	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward_type=" + strconv.Itoa(rewardType)

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println(err)
		return 0
	}

	fmt.Println(resp.Body)

	var response map[string]interface{}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	err = json.Unmarshal(body, &response)

	fmt.Println(response)

	sum := response["sum"].(float64)

	return int(sum)

}

func getReadRewards(blockNumber, startBlockNumber, endBlockNumber, blobber1, blobber2 string) []int64 {
	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/read-rewards?start_block_number=" + startBlockNumber + "&end_block_number=" + endBlockNumber
	var response map[string]interface{}

	res, _ := http.Get(url)

	// decode and save the res body to response
	json.NewDecoder(res.Body).Decode(&response)

	var result []int64

	var blobber1TotalReward int64
	blobber1TotalReward = 0
	var blobber2TotalReward int64
	blobber2TotalReward = 0

	var blobber1ProviderReward int64
	blobber1ProviderReward = 0
	var blobber2ProviderReward int64
	blobber2ProviderReward = 0

	for _, providerReward := range response["provider_rewards"].([]interface{}) {
		providerId := providerReward.(map[string]interface{})["provider_id"].(string)
		amount := int64(providerReward.(map[string]interface{})["amount"].(float64))

		if providerId == blobber1 {
			blobber1TotalReward += amount
			blobber1ProviderReward += amount
		} else if providerId == blobber2 {
			blobber2TotalReward += amount
			blobber2ProviderReward += amount
		}
	}

	var blobber1DelegateReward int64
	blobber1DelegateReward = 0
	var blobber2DelegateReward int64
	blobber2DelegateReward = 0

	for _, delegateRewards := range response["delegate_rewards"].([]interface{}) {
		providerId := delegateRewards.(map[string]interface{})["provider_id"].(string)
		amount := int64(delegateRewards.(map[string]interface{})["amount"].(float64))

		if providerId == blobber1 {
			blobber1TotalReward += amount
			blobber1DelegateReward += amount
		} else if providerId == blobber2 {
			blobber2TotalReward += amount
			blobber2DelegateReward += amount
		}
	}

	result = append(result, blobber1ProviderReward)
	result = append(result, blobber2ProviderReward)
	result = append(result, blobber1DelegateReward)
	result = append(result, blobber2DelegateReward)

	result = append(result, blobber1TotalReward)
	result = append(result, blobber2TotalReward)

	return result
}
