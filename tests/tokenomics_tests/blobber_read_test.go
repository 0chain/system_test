package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlobberReadReward(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var blobberListString []string
	for _, blobber := range blobberList {
		blobberListString = append(blobberListString, blobber.Id)
	}

	t.Log("blobberListString", blobberListString)

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

	blobber1 := blobberListString[0]
	blobber2 := blobberListString[1]

	t.RunSequentiallyWithTimeout("download one time, equal from both blobbers", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
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
			"expire": "10m",
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

		downloadCost := sizeInGB(int64(filesize)) * math.Pow10(8) * 4

		downloadRewards, err := getReadRewards(t, allocationId)
		require.Nil(t, err, "error getting read rewards")

		blobber1DownloadRewards := downloadRewards[blobber1].Amount
		blobber2DownloadRewards := downloadRewards[blobber2].Amount
		blobber1DelegatesDownloadRewards := downloadRewards[blobber1].Total - blobber1DownloadRewards
		blobber2DelegatesDownloadRewards := downloadRewards[blobber2].Total - blobber2DownloadRewards
		blobber1TotalDownloadRewards := downloadRewards[blobber1].Total
		blobber2TotalDownloadRewards := downloadRewards[blobber2].Total

		totalDownloadRewards := blobber1TotalDownloadRewards + blobber2TotalDownloadRewards

		// log all the values
		t.Log("downloadCost", downloadCost)
		t.Log("blobber1DownloadRewards", blobber1DownloadRewards)
		t.Log("blobber2DownloadRewards", blobber2DownloadRewards)
		t.Log("blobber1Delegate1DownloadRewards", blobber1DelegatesDownloadRewards)
		t.Log("blobber2Delegate1DownloadRewards", blobber2DelegatesDownloadRewards)
		t.Log("blobber1TotalDownloadRewards", blobber1TotalDownloadRewards)
		t.Log("blobber2TotalDownloadRewards", blobber2TotalDownloadRewards)
		t.Log("totalDownloadRewards", totalDownloadRewards)

		require.InEpsilon(t, downloadCost, totalDownloadRewards, 0.05, "Download cost and total download rewards are not equal")
		require.InEpsilon(t, blobber1DownloadRewards, blobber2DownloadRewards, 0.05, "Blobber 1 and Blobber 2 download rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesDownloadRewards, blobber2DelegatesDownloadRewards, 0.05, "Blobber 1 delegate 1 and Blobber 2 delegate 1 download rewards are not equal")
		require.InEpsilon(t, blobber1TotalDownloadRewards, blobber2TotalDownloadRewards, 0.05, "Blobber 1 total download rewards and Blobber 2 total download rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("download several times and checking if downloading fails after allocation expiry", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
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
			"expire": "10m",
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

			downloadCost := sizeInGB(int64(filesize)) * math.Pow10(8) * 4

			downloadRewards, err := getReadRewards(t, allocationId)
			require.Nil(t, err, "error getting read rewards")

			blobber1DownloadRewards := downloadRewards[blobber1].Amount
			blobber2DownloadRewards := downloadRewards[blobber2].Amount
			blobber1DelegatesDownloadRewards := downloadRewards[blobber1].Total - blobber1DownloadRewards
			blobber2DelegatesDownloadRewards := downloadRewards[blobber2].Total - blobber2DownloadRewards
			blobber1TotalDownloadRewards := downloadRewards[blobber1].Total
			blobber2TotalDownloadRewards := downloadRewards[blobber2].Total

			totalDownloadRewards := blobber1TotalDownloadRewards + blobber2TotalDownloadRewards

			// log all the values
			t.Log("downloadCost", downloadCost)
			t.Log("blobber1DownloadRewards", blobber1DownloadRewards)
			t.Log("blobber2DownloadRewards", blobber2DownloadRewards)
			t.Log("blobber1Delegate1DownloadRewards", blobber1DelegatesDownloadRewards)
			t.Log("blobber2Delegate1DownloadRewards", blobber2DelegatesDownloadRewards)
			t.Log("blobber1TotalDownloadRewards", blobber1TotalDownloadRewards)
			t.Log("blobber2TotalDownloadRewards", blobber2TotalDownloadRewards)
			t.Log("totalDownloadRewards", totalDownloadRewards)

			require.InEpsilon(t, downloadCost, totalDownloadRewards, 0.05, "Download cost and total download rewards are not equal")
			require.InEpsilon(t, blobber1DownloadRewards, blobber2DownloadRewards, 0.05, "Blobber 1 and Blobber 2 download rewards are not equal")
			require.InEpsilon(t, blobber1DelegatesDownloadRewards, blobber2DelegatesDownloadRewards, 0.05, "Blobber 1 delegate 1 and Blobber 2 delegate 1 download rewards are not equal")
			require.InEpsilon(t, blobber1TotalDownloadRewards, blobber2TotalDownloadRewards, 0.05, "Blobber 1 total download rewards and Blobber 2 total download rewards are not equal")

		}

		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)

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

func getReadRewards(t *test.SystemTest, allocationID string) (map[string]ProviderAllocationRewards, error) {
	var result map[string]ProviderAllocationRewards

	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/read-rewards?allocation_id=" + allocationID)

	t.Log("Allocation challenge rewards url: ", url)

	res, _ := http.Get(url)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(res.Body)

	body, _ := io.ReadAll(res.Body)

	err := json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func sizeInGB(size int64) float64 {
	return float64(size) / float64(GB)
}
