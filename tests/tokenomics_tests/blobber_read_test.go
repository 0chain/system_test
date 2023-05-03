package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"math"
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

	prevBlock := utils.GetLatestFinalizedBlock(t)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	t.RunSequentiallyWithTimeout("Case 1 : 1 delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		remotepath := "/dir/"
		filesize := 50 * MB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		time.Sleep(30 * time.Second)

		downloadCost := sizeInGB(int64(filesize)) * math.Pow10(9) * 4

		curBlock := utils.GetLatestFinalizedBlock(t)

		downloadRewards := getReadRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1DownloadRewards := float64(downloadRewards[0])
		blobber2DownloadRewards := float64(downloadRewards[1])
		blobber1Delegate1DownloadRewards := float64(downloadRewards[2])
		blobber2Delegate1DownloadRewards := float64(downloadRewards[3])
		blobber1TotalDownloadRewards := float64(downloadRewards[4])
		blobber2TotalDownloadRewards := float64(downloadRewards[5])

		totalDownloadRewards := blobber1TotalDownloadRewards + blobber2TotalDownloadRewards

		// log all the values
		t.Log("downloadCost", downloadCost)
		t.Log("blobber1DownloadRewards", blobber1DownloadRewards)
		t.Log("blobber2DownloadRewards", blobber2DownloadRewards)
		t.Log("blobber1Delegate1DownloadRewards", blobber1Delegate1DownloadRewards)
		t.Log("blobber2Delegate1DownloadRewards", blobber2Delegate1DownloadRewards)
		t.Log("blobber1TotalDownloadRewards", blobber1TotalDownloadRewards)
		t.Log("blobber2TotalDownloadRewards", blobber2TotalDownloadRewards)
		t.Log("totalDownloadRewards", totalDownloadRewards)

		require.InEpsilon(t, downloadCost/totalDownloadRewards, 1, 0.05, "Download cost and total download rewards are not equal")
		require.InEpsilon(t, blobber1DownloadRewards/blobber2DownloadRewards, 1, 0.05, "Blobber 1 and Blobber 2 download rewards are not equal")
		require.InEpsilon(t, blobber1Delegate1DownloadRewards/blobber2Delegate1DownloadRewards, 1, 0.05, "Blobber 1 delegate 1 and Blobber 2 delegate 1 download rewards are not equal")
		require.InEpsilon(t, blobber1TotalDownloadRewards/blobber2TotalDownloadRewards, 1, 0.05, "Blobber 1 total download rewards and Blobber 2 total download rewards are not equal")

		prevBlock = curBlock

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Case 2 : 1 delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		remotepath := "/dir/"
		filesize := 50 * MB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		for i := 0; i < 3; i++ {

			err = os.Remove(filename)
			require.Nil(t, err)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

			time.Sleep(30 * time.Second)

			downloadCost := sizeInGB(int64(filesize)) * math.Pow10(9) * 4

			curBlock := utils.GetLatestFinalizedBlock(t)

			downloadRewards := getReadRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

			blobber1DownloadRewards := float64(downloadRewards[0])
			blobber2DownloadRewards := float64(downloadRewards[1])
			blobber1Delegate1DownloadRewards := float64(downloadRewards[2])
			blobber2Delegate1DownloadRewards := float64(downloadRewards[3])
			blobber1TotalDownloadRewards := float64(downloadRewards[4])
			blobber2TotalDownloadRewards := float64(downloadRewards[5])

			totalDownloadRewards := blobber1TotalDownloadRewards + blobber2TotalDownloadRewards

			// log all the values
			t.Log("downloadCost", downloadCost)
			t.Log("blobber1DownloadRewards", blobber1DownloadRewards)
			t.Log("blobber2DownloadRewards", blobber2DownloadRewards)
			t.Log("blobber1Delegate1DownloadRewards", blobber1Delegate1DownloadRewards)
			t.Log("blobber2Delegate1DownloadRewards", blobber2Delegate1DownloadRewards)
			t.Log("blobber1TotalDownloadRewards", blobber1TotalDownloadRewards)
			t.Log("blobber2TotalDownloadRewards", blobber2TotalDownloadRewards)
			t.Log("totalDownloadRewards", totalDownloadRewards)

			require.InEpsilon(t, downloadCost/totalDownloadRewards, 1, 0.05, "Download cost and total download rewards are not equal")
			require.InEpsilon(t, blobber1DownloadRewards/blobber2DownloadRewards, 1, 0.05, "Blobber 1 and Blobber 2 download rewards are not equal")
			require.InEpsilon(t, blobber1Delegate1DownloadRewards/blobber2Delegate1DownloadRewards, 1, 0.05, "Blobber 1 delegate 1 and Blobber 2 delegate 1 download rewards are not equal")
			require.InEpsilon(t, blobber1TotalDownloadRewards/blobber2TotalDownloadRewards, 1, 0.05, "Blobber 1 total download rewards and Blobber 2 total download rewards are not equal")

			prevBlock = curBlock
		}

		// Sleep for 20 minutes
		time.Sleep(20 * time.Minute)

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.NotNil(t, err, "File should not be downloaded from expired allocation", strings.Join(output, "\n"))

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

}

func getReadRewards(t *test.SystemTest, startBlockNumber, endBlockNumber, blobber1, blobber2 string) []int64 {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/read-rewards?start_block_number=" + startBlockNumber + "&end_block_number=" + endBlockNumber)

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

func sizeInGB(size int64) float64 {
	return float64(size) / float64(GB)
}
