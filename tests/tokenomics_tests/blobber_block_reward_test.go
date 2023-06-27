package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
)

func TestBlockRewardsForBlobbers(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	prevBlock := utils.GetLatestFinalizedBlock(t)

	t.Log("prevBlock", prevBlock)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
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

	totalData := 0.1 * GB

	var descriptions []string
	descriptions = append(descriptions, "Blobber Block Reward Test - 1")

	t.RunSequentiallyWithTimeout("All conditions same", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stake := []float64{1.0, 1.0, 1.0, 1.0}
		readData := []int{1, 1}

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, stake, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   1,
			"parity": 1,
			"tokens": 99,
			"expire": "10m",
		})

		t.Log("Allocation ID : ", allocationId)

		remotepath := "/dir/"
		filesize := totalData
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		for i := 0; i < readData[0]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[0].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		for i := 0; i < readData[1]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[1].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
		curBlock := utils.GetLatestFinalizedBlock(t)

		blobber1PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[0].Id)
		blobber2PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[1].Id)

		blobberBlockRewards := getBlockRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		blobber1Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[0])*totalData, stake[0], blobber1PassedChallenges)
		blobber2Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[1])*totalData, stake[1], blobber2PassedChallenges)

		// print all values
		t.Log("blobber1ProviderRewards", blobber1ProviderRewards)
		t.Log("blobber2ProviderRewards", blobber2ProviderRewards)
		t.Log("blobber1DelegateRewards", blobber1DelegateRewards)
		t.Log("blobber2DelegateRewards", blobber2DelegateRewards)
		t.Log("blobber1TotalRewards", blobber1TotalRewards)
		t.Log("blobber2TotalRewards", blobber2TotalRewards)

		require.InEpsilon(t, blobber1TotalRewards/blobber2TotalRewards, blobber1Weight/blobber2Weight, 0.05, "Total rewards not distributed correctly")

		prevBlock = utils.GetLatestFinalizedBlock(t)

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

	t.RunSequentiallyWithTimeout("Verify free reads", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Updating blobber 2 read price
		blobber2 := blobberList[1]
		utils.ExecuteFaucetWithTokensForWallet(t, blobber2Wallet, configPath, 9)
		output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(2), utils.CreateParams(map[string]interface{}{"blobber_id": blobber2.Id, "read_price": utils.IntToZCN(0)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		blobber1 := blobberList[0]
		utils.ExecuteFaucetWithTokensForWallet(t, blobber1Wallet, configPath, 9)
		output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber1.Id, "read_price": utils.IntToZCN(0)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		stake := []float64{1.0, 1.0, 1.0, 1.0}
		readData := []int{9, 9}

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, stake, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   1,
			"parity": 1,
			"tokens": 99,
			"expire": "10m",
		})

		t.Log("Allocation ID : ", allocationId)

		remotepath := "/dir/"
		filesize := totalData
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		for i := 0; i < readData[0]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[0].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		for i := 0; i < readData[1]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[1].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
		curBlock := utils.GetLatestFinalizedBlock(t)

		blobber1PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[0].Id)
		blobber2PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[1].Id)

		blobberBlockRewards := getBlockRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		blobber1Weight := calculateWeight(1000000000, 10000000000, totalData, float64(readData[0])*totalData, stake[0], blobber1PassedChallenges)
		blobber2Weight := calculateWeight(1000000000, 0, totalData, float64(readData[1])*totalData, stake[1], blobber2PassedChallenges)

		// print all values
		t.Log("blobber1ProviderRewards", blobber1ProviderRewards)
		t.Log("blobber2ProviderRewards", blobber2ProviderRewards)
		t.Log("blobber1DelegateRewards", blobber1DelegateRewards)
		t.Log("blobber2DelegateRewards", blobber2DelegateRewards)
		t.Log("blobber1TotalRewards", blobber1TotalRewards)
		t.Log("blobber2TotalRewards", blobber2TotalRewards)

		require.InEpsilon(t, blobber1TotalRewards/blobber2TotalRewards, blobber1Weight/blobber2Weight, 0.05, "Total rewards not distributed correctly")

		prevBlock = utils.GetLatestFinalizedBlock(t)

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

	t.RunSequentiallyWithTimeout("Verify write price diff changes", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 9)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "read_price": utils.IntToZCN(1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 9)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "write_price": utils.IntToZCN(int64(math.Pow10(count)) * 1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		stake := []float64{1.0, 1.0, 1.0, 1.0}
		readData := []int{9, 9}

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, stake, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   1,
			"parity": 1,
			"tokens": 99,
			"expire": "10m",
		})

		t.Log("Allocation ID : ", allocationId)

		remotepath := "/dir/"
		filesize := totalData
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		for i := 0; i < readData[0]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[0].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		for i := 0; i < readData[1]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[1].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
		curBlock := utils.GetLatestFinalizedBlock(t)

		blobber1PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[0].Id)
		blobber2PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[1].Id)

		blobberBlockRewards := getBlockRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		blobber1Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[0])*totalData, stake[0], blobber1PassedChallenges)
		blobber2Weight := calculateWeight(10000000000, 1000000000, totalData, float64(readData[1])*totalData, stake[1], blobber2PassedChallenges)

		// print all values
		t.Log("blobber1ProviderRewards", blobber1ProviderRewards)
		t.Log("blobber2ProviderRewards", blobber2ProviderRewards)
		t.Log("blobber1DelegateRewards", blobber1DelegateRewards)
		t.Log("blobber2DelegateRewards", blobber2DelegateRewards)
		t.Log("blobber1TotalRewards", blobber1TotalRewards)
		t.Log("blobber2TotalRewards", blobber2TotalRewards)

		require.InEpsilon(t, blobber1TotalRewards/blobber2TotalRewards, blobber1Weight/blobber2Weight, 0.05, "Total rewards not distributed correctly")

		prevBlock = utils.GetLatestFinalizedBlock(t)

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

	t.RunSequentiallyWithTimeout("Check read price ratio", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 99)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "read_price": utils.IntToZCN(int64(math.Pow10(count)) * 1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 99)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "write_price": utils.IntToZCN(1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		stake := []float64{1.0, 1.0, 1.0, 1.0}
		readData := []int{1, 1}

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, stake, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   1,
			"parity": 1,
			"tokens": 99,
			"expire": "10m",
		})

		t.Log("Allocation ID : ", allocationId)

		remotepath := "/dir/"
		filesize := totalData
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		for i := 0; i < readData[0]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[0].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		for i := 0; i < readData[1]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[1].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
		curBlock := utils.GetLatestFinalizedBlock(t)

		blobber1PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[0].Id)
		blobber2PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[1].Id)

		blobberBlockRewards := getBlockRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		blobber1Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[0])*totalData, stake[0], blobber1PassedChallenges)
		blobber2Weight := calculateWeight(1000000000, 10000000000, totalData, float64(readData[1])*totalData, stake[1], blobber2PassedChallenges)

		// print all values
		t.Log("blobber1ProviderRewards", blobber1ProviderRewards)
		t.Log("blobber2ProviderRewards", blobber2ProviderRewards)
		t.Log("blobber1DelegateRewards", blobber1DelegateRewards)
		t.Log("blobber2DelegateRewards", blobber2DelegateRewards)
		t.Log("blobber1TotalRewards", blobber1TotalRewards)
		t.Log("blobber2TotalRewards", blobber2TotalRewards)

		require.InEpsilon(t, blobber1TotalRewards/blobber2TotalRewards, blobber1Weight/blobber2Weight, 0.05, "Total rewards not distributed correctly")

		prevBlock = utils.GetLatestFinalizedBlock(t)

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

	t.RunSequentiallyWithTimeout("Verify stake ratio", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 99)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "read_price": utils.IntToZCN(1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 99)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "write_price": utils.IntToZCN(1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		stake := []float64{1.0, 3.0, 1.0, 3.0}
		readData := []int{1, 1}

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, stake, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   1,
			"parity": 1,
			"tokens": 99,
			"expire": "10m",
		})

		t.Log("allocationId", allocationId)

		remotepath := "/dir/"
		filesize := totalData
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		for i := 0; i < readData[0]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[0].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		for i := 0; i < readData[1]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[1].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
		curBlock := utils.GetLatestFinalizedBlock(t)

		blobber1PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[0].Id)
		blobber2PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[1].Id)

		blobberBlockRewards := getBlockRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		blobber1Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[0])*totalData, stake[0], blobber1PassedChallenges)
		blobber2Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[1])*totalData, stake[1], blobber2PassedChallenges)

		// print all values
		t.Log("blobber1ProviderRewards", blobber1ProviderRewards)
		t.Log("blobber2ProviderRewards", blobber2ProviderRewards)
		t.Log("blobber1DelegateRewards", blobber1DelegateRewards)
		t.Log("blobber2DelegateRewards", blobber2DelegateRewards)
		t.Log("blobber1TotalRewards", blobber1TotalRewards)
		t.Log("blobber2TotalRewards", blobber2TotalRewards)

		require.InEpsilon(t, blobber1TotalRewards/blobber2TotalRewards, blobber1Weight/blobber2Weight, 0.05, "Total rewards not distributed correctly")

		prevBlock = utils.GetLatestFinalizedBlock(t)

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

	t.RunSequentiallyWithTimeout("Check ratio with respect to total read data", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 9)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "read_price": utils.IntToZCN(1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		for count, blobber := range blobberList {
			utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 9)
			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": blobber.Id, "write_price": utils.IntToZCN(1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		stake := []float64{1.0, 1.0, 1.0, 1.0}
		readData := []int{1, 9}

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, stake, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   1,
			"parity": 1,
			"tokens": 99,
			"expire": "10m",
		})

		t.Log("Allocation ID : ", allocationId)

		remotepath := "/dir/"
		filesize := totalData
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		for i := 0; i < readData[0]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[0].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		for i := 0; i < readData[1]; i++ {
			err = os.Remove(filename)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
				"blobber_id": blobberList[1].Id,
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
		curBlock := utils.GetLatestFinalizedBlock(t)

		blobber1PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[0].Id)
		blobber2PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[1].Id)

		blobberBlockRewards := getBlockRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		blobber1Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[0])*totalData, stake[0], blobber1PassedChallenges)
		blobber2Weight := calculateWeight(1000000000, 1000000000, totalData, float64(readData[1])*totalData, stake[1], blobber2PassedChallenges)

		// print all values
		t.Log("blobber1ProviderRewards", blobber1ProviderRewards)
		t.Log("blobber2ProviderRewards", blobber2ProviderRewards)
		t.Log("blobber1DelegateRewards", blobber1DelegateRewards)
		t.Log("blobber2DelegateRewards", blobber2DelegateRewards)
		t.Log("blobber1TotalRewards", blobber1TotalRewards)
		t.Log("blobber2TotalRewards", blobber2TotalRewards)

		require.InEpsilon(t, blobber1TotalRewards/blobber2TotalRewards, blobber1Weight/blobber2Weight, 0.05, "Total rewards not distributed correctly")

		prevBlock = utils.GetLatestFinalizedBlock(t)

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

}

func getBlockRewards(t *test.SystemTest, startBlockNumber, endBlockNumber, blobber1, blobber2 string) []int64 {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/block-rewards?start_block_number=" + startBlockNumber + "&end_block_number=" + endBlockNumber)
	var response []int64

	res, _ := http.Get(url)

	// decode and save the res body to response
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil
	}

	return response
}

func getZeta(wp, rp float64) float64 {

	i := float64(1)
	k := float64(0.9)
	mu := float64(0.2)

	if wp == 0 {
		return 0
	}

	return i - (k * (rp / (rp + (mu * wp))))
}

func getGamma(X, R float64) float64 {

	A := float64(10)
	B := float64(1)
	alpha := float64(0.2)

	if X == 0 {
		return 0
	}

	factor := math.Abs((alpha*X - R) / (alpha*X + R))
	return A - B*factor
}

func calculateWeight(wp, rp, X, R, stakes, challenges float64) float64 {

	zeta := getZeta(wp, rp)
	gamma := getGamma(X, R)

	return (zeta*gamma + 1) * stakes * challenges
}

func countPassedChallengesForBlobberAndAllocation(t *test.SystemTest, allocationID, blobberID string) float64 {

	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/passed-challenges?allocation_id=" + allocationID)
	var response map[string]int

	res, _ := http.Get(url)

	// decode and save the res body to response
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return 0
	}

	return float64(response[blobberID])
}
