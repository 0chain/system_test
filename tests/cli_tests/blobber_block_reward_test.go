package cli_tests

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
)

func TestBlobberBlockRewards(testSetup *testing.T) {
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

	t.RunSequentiallyWithTimeout("Case : ", 500*time.Minute, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		blobber1AllocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 0,
			"expire": "20m",
		})
		fmt.Println("Allocation ID : ", blobber1AllocationID)

		blobber2AllocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 0,
			"expire": "20m",
		})
		fmt.Println("Allocation ID : ", blobber2AllocationID)

		allocationIDs := []string{blobber1AllocationID, blobber2AllocationID}

		// Uploading 10% of allocation
		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		for _, allocationId := range allocationIDs {
			err = createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = uploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
		}

		// download the file

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationIDs[0],
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		for i := 0; i < 3; i++ {
			err = os.Remove(filename)
			require.Nil(t, err)

			remoteFilepath := remotepath + filepath.Base(filename)

			output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationIDs[1],
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		}

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		fmt.Println("curBlock", curBlock)

		// 2. Get the block rewards for all the blobbers.

		blobberBlockRewards := getBlockRewards("", strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		// print all values
		fmt.Println("blobber1ProviderRewards", blobber1ProviderRewards)
		fmt.Println("blobber2ProviderRewards", blobber2ProviderRewards)
		fmt.Println("blobber1DelegateRewards", blobber1DelegateRewards)
		fmt.Println("blobber2DelegateRewards", blobber2DelegateRewards)
		fmt.Println("blobber1TotalRewards", blobber1TotalRewards)
		fmt.Println("blobber2TotalRewards", blobber2TotalRewards)

		// check if blobber1TotalRewards and blobber2TotalRewards are equal with error margin of 5%
		require.True(t, math.Abs(blobber1TotalRewards-blobber2TotalRewards) <= 0.05*blobber1TotalRewards, "blobber1TotalRewards and blobber2TotalRewards should be equal with error margin of 5%")
		// check if blobber1DelegateRewards and blobber2DelegateRewards are equal with error margin of 5%
		require.True(t, math.Abs(blobber1DelegateRewards-blobber2DelegateRewards) <= 0.05*blobber1DelegateRewards, "blobber1DelegateRewards and blobber2DelegateRewards should be equal with error margin of 5%")
		// check if blobber1ProviderRewards and blobber2ProviderRewards are equal with error margin of 5%
		require.True(t, math.Abs(blobber1ProviderRewards-blobber2ProviderRewards) <= 0.05*blobber1ProviderRewards, "blobber1ProviderRewards and blobber2ProviderRewards should be equal with error margin of 5%")

		prevBlock = getLatestFinalizedBlock(t)

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 1: Free Reads, one delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// download the file
		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		fmt.Println("curBlock", curBlock)

		// 2. Get the block rewards for all the blobbers.

		blobberBlockRewards := getBlockRewards("", strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1ProviderRewards := float64(blobberBlockRewards[0])
		blobber2ProviderRewards := float64(blobberBlockRewards[1])
		blobber1DelegateRewards := float64(blobberBlockRewards[2])
		blobber2DelegateRewards := float64(blobberBlockRewards[3])
		blobber1TotalRewards := float64(blobberBlockRewards[4])
		blobber2TotalRewards := float64(blobberBlockRewards[5])

		// print all values
		fmt.Println("blobber1ProviderRewards", blobber1ProviderRewards)
		fmt.Println("blobber2ProviderRewards", blobber2ProviderRewards)
		fmt.Println("blobber1DelegateRewards", blobber1DelegateRewards)
		fmt.Println("blobber2DelegateRewards", blobber2DelegateRewards)
		fmt.Println("blobber1TotalRewards", blobber1TotalRewards)
		fmt.Println("blobber2TotalRewards", blobber2TotalRewards)

		// check if blobber1TotalRewards and blobber2TotalRewards are equal with error margin of 5%
		require.True(t, math.Abs(blobber1TotalRewards-blobber2TotalRewards) <= 0.05*blobber1TotalRewards, "blobber1TotalRewards and blobber2TotalRewards should be equal with error margin of 5%")
		// check if blobber1DelegateRewards and blobber2DelegateRewards are equal with error margin of 5%
		require.True(t, math.Abs(blobber1DelegateRewards-blobber2DelegateRewards) <= 0.05*blobber1DelegateRewards, "blobber1DelegateRewards and blobber2DelegateRewards should be equal with error margin of 5%")
		// check if blobber1ProviderRewards and blobber2ProviderRewards are equal with error margin of 5%
		require.True(t, math.Abs(blobber1ProviderRewards-blobber2ProviderRewards) <= 0.05*blobber1ProviderRewards, "blobber1ProviderRewards and blobber2ProviderRewards should be equal with error margin of 5%")

		prevBlock = getLatestFinalizedBlock(t)

		//// get all challenges
		//challenges, _ := getAllChallenges(allocationId)
		//
		//for _, challenge := range challenges {
		//	require.True(t, challenge.Passed, "All Challenges should be passed")
		//}

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 2: Different Write Price, Equal Read Price, one delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		fmt.Println(output)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// download the file
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

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		// 2. Get the block rewards for all the blobbers.

		blobberBlockRewards := getBlockRewards("", strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1DelegateRewards := blobberBlockRewards[1]
		blobber1Rewards := blobberBlockRewards[0] + blobber1DelegateRewards
		blobber2DelegateRewards := blobberBlockRewards[3]
		blobber2Rewards := blobberBlockRewards[2] + blobber2DelegateRewards

		blobber1Percent := blobber1Rewards / (blobber1Rewards + blobber2Rewards) * 100
		blobber2Percent := blobber2Rewards / (blobber1Rewards + blobber2Rewards) * 100

		blobber1DelegatePercent := blobber1DelegateRewards / (blobber1Rewards + blobber2Rewards) * 100
		blobber2DelegatePercent := blobber2DelegateRewards / (blobber1Rewards + blobber2Rewards) * 100

		zetaBlobber1 := getZeta(0.1, 0.1)
		zetaBlobber2 := getZeta(0.2, 0.1)

		// print all values
		fmt.Println("Blobber 1 Rewards : ", blobber1Rewards)
		fmt.Println("Blobber 2 Rewards : ", blobber2Rewards)
		fmt.Println("Blobber 1 Delegate Rewards : ", blobber1DelegateRewards)
		fmt.Println("Blobber 2 Delegate Rewards : ", blobber2DelegateRewards)
		fmt.Println("Blobber 1 Percent : ", blobber1Percent)
		fmt.Println("Blobber 2 Percent : ", blobber2Percent)
		fmt.Println("Blobber 1 Delegate Percent : ", blobber1DelegatePercent)
		fmt.Println("Blobber 2 Delegate Percent : ", blobber2DelegatePercent)
		fmt.Println("Zeta Blobber 1 : ", zetaBlobber1)
		fmt.Println("Zeta Blobber 2 : ", zetaBlobber2)

		// match if difference between their percent is less than 1%
		require.InEpsilon(t, blobber1Rewards/blobber2Rewards, zetaBlobber1/zetaBlobber2, 0.1, "Difference between blobber1 and blobber2 rewards is more than 1%")

		prevBlock = getLatestFinalizedBlock(t)

		// get all challenges
		challenges, _ := getAllChallenges(allocationId)

		for _, challenge := range challenges {
			require.True(t, challenge.Passed, "All Challenges should be passed")
		}

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Case 3: Different Read Price, one delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		fmt.Println(output)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// download the file
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

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		// 2. Get the block rewards for all the blobbers.

		blobberBlockRewards := getBlockRewards("", strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1DelegateRewards := blobberBlockRewards[1]
		blobber1Rewards := blobberBlockRewards[0] + blobber1DelegateRewards
		blobber2DelegateRewards := blobberBlockRewards[3]
		blobber2Rewards := blobberBlockRewards[2] + blobber2DelegateRewards

		blobber1Percent := blobber1Rewards / (blobber1Rewards + blobber2Rewards) * 100
		blobber2Percent := blobber2Rewards / (blobber1Rewards + blobber2Rewards) * 100

		blobber1DelegatePercent := blobber1DelegateRewards / (blobber1Rewards + blobber2Rewards) * 100
		blobber2DelegatePercent := blobber2DelegateRewards / (blobber1Rewards + blobber2Rewards) * 100

		zetaBlobber1 := getZeta(0.1, 0)
		zetaBlobber2 := getZeta(0.1, 1)

		// print all values
		fmt.Println("Blobber 1 Rewards : ", blobber1Rewards)
		fmt.Println("Blobber 2 Rewards : ", blobber2Rewards)
		fmt.Println("Blobber 1 Delegate Rewards : ", blobber1DelegateRewards)
		fmt.Println("Blobber 2 Delegate Rewards : ", blobber2DelegateRewards)
		fmt.Println("Blobber 1 Percent : ", blobber1Percent)
		fmt.Println("Blobber 2 Percent : ", blobber2Percent)
		fmt.Println("Blobber 1 Delegate Percent : ", blobber1DelegatePercent)
		fmt.Println("Blobber 2 Delegate Percent : ", blobber2DelegatePercent)
		fmt.Println("Zeta Blobber 1 : ", zetaBlobber1)
		fmt.Println("Zeta Blobber 2 : ", zetaBlobber2)

		// match if difference between their percent is less than 1%
		require.InEpsilon(t, blobber1Rewards/blobber2Rewards, zetaBlobber1/zetaBlobber2, 0.1, "Difference between blobber1 and blobber2 rewards is more than 1%")

		prevBlock = getLatestFinalizedBlock(t)

		// get all challenges
		challenges, _ := getAllChallenges(allocationId)

		for _, challenge := range challenges {
			require.True(t, challenge.Passed, "All Challenges should be passed")
		}

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Case 4: Free Reads, one delegate each, unequal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 2, 1, 2,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		fmt.Println(output)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// download the file
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

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		// 2. Get the block rewards for all the blobbers.

		blobberBlockRewards := getBlockRewards("", strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

		blobber1DelegateRewards := blobberBlockRewards[1]
		blobber1Rewards := blobberBlockRewards[0]
		blobber2DelegateRewards := blobberBlockRewards[3]
		blobber2Rewards := blobberBlockRewards[2]

		blobber1Percent := blobber1Rewards / (blobber1Rewards + blobber2Rewards) * 100
		blobber2Percent := blobber2Rewards / (blobber1Rewards + blobber2Rewards) * 100

		blobber1DelegatePercent := blobber1DelegateRewards / (blobber1Rewards + blobber2Rewards) * 100
		blobber2DelegatePercent := blobber2DelegateRewards / (blobber1Rewards + blobber2Rewards) * 100

		// print all values
		fmt.Println("Blobber 1 Rewards : ", blobber1Rewards)
		fmt.Println("Blobber 2 Rewards : ", blobber2Rewards)
		fmt.Println("Blobber 1 Delegate Rewards : ", blobber1DelegateRewards)
		fmt.Println("Blobber 2 Delegate Rewards : ", blobber2DelegateRewards)
		fmt.Println("Blobber 1 Percent : ", blobber1Percent)
		fmt.Println("Blobber 2 Percent : ", blobber2Percent)
		fmt.Println("Blobber 1 Delegate Percent : ", blobber1DelegatePercent)
		fmt.Println("Blobber 2 Delegate Percent : ", blobber2DelegatePercent)

		require.Equal(t, blobber1Rewards, blobber2Rewards, "Blobber1 and Blobber2 rewards should be equal")
		require.InEpsilon(t, blobber1DelegateRewards/blobber2DelegateRewards, 1/2, 0.1, "Difference between blobber1 delegate and blobber2 delegate rewards is more than 1%")

		// match if difference between their percent is less than 1%
		require.True(t, blobber1Percent-blobber2Percent < 1, "Difference between blobber1 and blobber2 rewards is more than 1%")
		require.True(t, blobber1DelegatePercent-blobber2DelegatePercent < 1, "Difference between blobber1 delegate and blobber2 delegate rewards is more than 1%")

		prevBlock = getLatestFinalizedBlock(t)

		// get all challenges
		challenges, _ := getAllChallenges(allocationId)

		for _, challenge := range challenges {
			require.True(t, challenge.Passed, "All Challenges should be passed")
		}

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

}

func getAllBlockRewards(blobberList []climodel.BlobberInfo) map[string]int64 {

	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?reward_type=3"
	var response map[string]interface{}

	result := make(map[string]int64)

	for _, blobber := range blobberList {
		result[blobber.Id] = 0
	}

	res, _ := http.Get(url)

	// decode and save the res body to response
	json.NewDecoder(res.Body).Decode(&response)

	//fmt.Println(response)

	fmt.Println("Total Block Rewards : ", int64(response["sum"].(float64)))

	var blockData map[float64][]map[string]interface{}

	// initialize blockData
	blockData = make(map[float64][]map[string]interface{})

	var a int64
	var b int64
	a = 0
	b = 0

	// run loop to get all the amount in rps in the response
	for _, rps := range response["rps"].([]interface{}) {
		amount := int64(rps.(map[string]interface{})["amount"].(float64))
		blockNumber := rps.(map[string]interface{})["block_number"].(float64)
		providerId := rps.(map[string]interface{})["provider_id"].(string)

		if blockNumber < 5000 {
			a += amount
		} else {
			b += amount
		}

		// check if the block number is present in the map an if not then create a new entry
		if _, ok := blockData[blockNumber]; !ok {
			blockData[blockNumber] = make([]map[string]interface{}, 0)
		}

		// append the provider id and amount to the block number
		blockData[blockNumber] = append(blockData[blockNumber], map[string]interface{}{
			"provider_id": providerId,
			"amount":      amount,
		})

		//blockData[blockNumber] = append(blockData[blockNumber], map[string]interface{}{
		//	"provider_id": providerId,
		//	"amount":      amount,
		//})

		//fmt.Println("Block Number : ", blockNumber)
		//fmt.Println("Provider ID : ", providerId)
		//fmt.Println("Amount : ", amount)
		//
		//fmt.Println("\n\n----------------------------------------------------")

		result["provider_id"] += amount
	}

	fmt.Println("A : ", a)
	fmt.Println("B : ", b)

	for blockNumber, data := range blockData {
		fmt.Println("Block Number : ", blockNumber)

		for _, d := range data {
			fmt.Println("Provider ID : ", d["provider_id"])
			fmt.Println("Amount : ", d["amount"])
		}
		fmt.Println("\n\n----------------------------------------------------")
	}

	fmt.Println("Block Data : ", blockData)

	return result
}

func getTotalBlockRewardsByBlobberID(blobberID string) int64 {

	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?reward_type=3"
	var response map[string]interface{}

	var result int64

	res, _ := http.Get(url)

	// decode and save the res body to response
	json.NewDecoder(res.Body).Decode(&response)
	//
	//fmt.Println(response)

	//fmt.Println("Total Block Rewards : ", int64(response["sum"].(float64)))

	// run loop to get all the amount in rps in the response
	for _, rps := range response["rps"].([]interface{}) {
		amount := int64(rps.(map[string]interface{})["amount"].(float64))
		providerId := rps.(map[string]interface{})["provider_id"].(string)

		if providerId == blobberID {
			result += amount
		}
	}

	return result
}

func getBlockRewards(blockNumber, startBlockNumber, endBlockNumber, blobber1, blobber2 string) []int64 {
	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block-rewards?start_block_number=" + startBlockNumber + "&end_block_number=" + endBlockNumber
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

func getZeta(wp, rp float64) float64 {

	i := float64(1)
	k := float64(1)
	mu := float64(1)

	if wp == 0 {
		return 0
	}

	return i - (k * (rp / (rp + (mu * wp))))
}
