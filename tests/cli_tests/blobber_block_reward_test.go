package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlobberBlockRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

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

	//t.RunSequentially("Just read all the data", func(t *test.SystemTest) {
	//
	//	//for i, blobber := range blobberList {
	//	//	fmt.Println("Blobber ", i, " : ", blobber.Id)
	//	//	fmt.Println(getTotalBlockRewardsByBlobberID(blobber.Id))
	//	//}
	//
	//	fmt.Println(getAllBlockRewards(blobberList))
	//})
	//
	//t.Skip()

	t.RunSequentiallyWithTimeout("Case 1: Free Reads, one delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		//stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "15m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 50 * MB
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

		//// download the file
		//err = os.Remove(filename)
		//require.Nil(t, err)
		//
		//remoteFilepath := remotepath + filepath.Base(filename)
		//
		//output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
		//	"allocation": allocationId,
		//	"remotepath": remoteFilepath,
		//	"localpath":  os.TempDir() + string(os.PathSeparator),
		//}), true)
		//fmt.Println(output)
		//require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		// sleep for 2 minutes
		time.Sleep(6 * time.Minute)

		for i, blobber := range blobberList {
			fmt.Println("Blobber ", i, " : ", blobber.Id)
			fmt.Println(getTotalBlockRewardsByBlobberID(blobber.Id))
		}

		//2. Get the block rewards for all the blobbers.
		blockRewards := getAllBlockRewards(blobberList)

		for blobberId, amount := range blockRewards {
			fmt.Println("Blobber ID : ", blobberId)
			fmt.Println("Block Reward : ", amount)
		}

	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 4: Free Reads, One delegate each, unequal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, false)

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"tokens": 9.0,
		}), false)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "10m",
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 50 * MB
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

		// sleep for 2 minutes
		time.Sleep(6 * time.Minute)

		// 2. Get the block rewards for all the blobbers.
		blockRewards := getAllBlockRewards(blobberList)

		for blobberId, amount := range blockRewards {
			fmt.Println("Blobber ID : ", blobberId)
			fmt.Println("Block Reward : ", amount)
		}
	})

	t.RunSequentiallyWithTimeout("Case 6: Free Reads, One delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		//for _, blobber := range blobberList {
		//	getTotalBlockRewardsByBlobberID(t, blobber.ID, configPath, true)
		//}

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"tokens": 9.0,
		}), false)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "10m",
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 50 * MB
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

		// sleep for 2 minutes
		time.Sleep(3 * time.Minute)

		// 2. Get all the block rewards for blobber1.
		totalBlockRewardForBlobber1 := getTotalBlockRewardsByBlobberID(blobberList[0].Id)
		totalBlockRewardForBlobber2 := getTotalBlockRewardsByBlobberID(blobberList[1].Id)

		// 3. Stop the blobber1.
		killProvider(blobberList[0].Id)

		// 4. Sleep for 3 minutes.
		time.Sleep(3 * time.Minute)

		// 5. Get all the block rewards for blobber1.
		totalBlockRewardForBlobber1AfterStop := getTotalBlockRewardsByBlobberID(blobberList[0].Id)
		totalBlockRewardForBlobber2AfterStop := getTotalBlockRewardsByBlobberID(blobberList[1].Id)

		fmt.Println("Total Block Reward for Blobber 1 : ", totalBlockRewardForBlobber1)
		fmt.Println("Total Block Reward for Blobber 2 : ", totalBlockRewardForBlobber2)
		fmt.Println("Total Block Reward for Blobber 1 After Stop : ", totalBlockRewardForBlobber1AfterStop)
		fmt.Println("Total Block Reward for Blobber 2 After Stop : ", totalBlockRewardForBlobber2AfterStop)
	})

	t.RunSequentiallyWithTimeout("Case 7: Free Reads, One delegate each, equal stake, client uploads nothing", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"tokens": 9.0,
		}), false)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "10m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// sleep for 2 minutes
		time.Sleep(3 * time.Minute)

		// 2. Get all the block rewards for blobber1.
		totalBlockRewardForBlobber1 := getTotalBlockRewardsByBlobberID(blobberList[0].Id)
		totalBlockRewardForBlobber2 := getTotalBlockRewardsByBlobberID(blobberList[1].Id)

		// 3. Stop the blobber1.
		killProvider(blobberList[0].Id)

		// 4. Sleep for 3 minutes.
		time.Sleep(3 * time.Minute)

		// 5. Get all the block rewards for blobber1.
		totalBlockRewardForBlobber1AfterStop := getTotalBlockRewardsByBlobberID(blobberList[0].Id)
		totalBlockRewardForBlobber2AfterStop := getTotalBlockRewardsByBlobberID(blobberList[1].Id)

		fmt.Println("Total Block Reward for Blobber 1 : ", totalBlockRewardForBlobber1)
		fmt.Println("Total Block Reward for Blobber 2 : ", totalBlockRewardForBlobber2)

		fmt.Println("Total Block Reward for Blobber 1 After Stop : ", totalBlockRewardForBlobber1AfterStop)
		fmt.Println("Total Block Reward for Blobber 2 After Stop : ", totalBlockRewardForBlobber2AfterStop)

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
