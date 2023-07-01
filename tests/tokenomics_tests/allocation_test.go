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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const tokenUnit float64 = 1e+10

func TestAllocationRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	var blobberDetailList []climodel.BlobberDetails
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	err = json.Unmarshal([]byte(output[0]), &blobberDetailList)
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

	stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
		1, 1, 1, 1,
	}, 1)

	t.RunWithTimeout("Create + Upload + Cancel equal read price 0.1", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)

		allocSize := 10 * MB

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   allocSize,
			"data":   1,
			"lock":   2,
			"parity": 1,
			"expire": "5m",
		}))
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocationId, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "Error getting allocation ID", strings.Join(output, "\n"))

		t.Log("allocationId", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		t.Log("Uploading file ", filename)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		time.Sleep(1 * time.Minute)

		alloc := utils.GetAllocation(t, allocationId)
		movedToChallengePool := alloc.MovedToChallenge

		_, err = utils.CancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, "Error cancelling allocation", strings.Join(output, "\n"))

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		alloc = utils.GetAllocation(t, allocationId)
		require.Equal(t, alloc.MovedToChallenge, movedToChallengePool, "MovedToChallenge should not change")

		// sleep for 5 minutes
		time.Sleep(3 * time.Minute)

		rewards := getTotalAllocationChallengeRewards(t, allocationId)

		totalBlobberChallengereward := int64(0)
		for _, v := range rewards {
			totalBlobberChallengereward += int64(v.(float64))
		}

		require.Greater(t, movedToChallengePool, totalBlobberChallengereward, "Total Blobber Challenge Reward should be less than MovedToChallenge")

		// Cancellation Rewards
		allocCancellationRewards, err := getAllocationCancellationReward(t, allocationId, blobberListString)
		if err != nil {
			return
		}

		blobber1CancellationReward := allocCancellationRewards[0]
		blobber2CancellationReward := allocCancellationRewards[1]

		totalExpectedCancellationReward := sizeInGB(int64(allocSize)*2) * 1000000000 * 0.2

		t.Log("totalExpectedCancellationReward", totalExpectedCancellationReward)

		t.Log("blobber1CancellationReward", blobber1CancellationReward)
		t.Log("blobber2CancellationReward", blobber2CancellationReward)

		require.InEpsilon(t, totalExpectedCancellationReward, float64(blobber1CancellationReward+blobber2CancellationReward), 0.05, "Total Cancellation Reward should be equal to total expected cancellation reward")
		require.InEpsilon(t, blobber1CancellationReward, blobber2CancellationReward, 0.05, "Blobber 1 Cancellation Reward should be equal to total expected cancellation reward")
	})

	t.RunWithTimeout("Create + Upload + Upgrade equal read price 0.1", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)

		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
			"expire": "5m",
		}))
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocationId, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "Error getting allocation ID", strings.Join(output, "\n"))

		alloc := utils.GetAllocation(t, allocationId)
		movedToChallengePool := alloc.MovedToChallenge

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 2 * MB
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

		time.Sleep(1 * time.Minute)

		alloc = utils.GetAllocation(t, allocationId)
		require.Greater(t, alloc.MovedToChallenge, movedToChallengePool, "MovedToChallenge should increase")
		movedToChallengePool = alloc.MovedToChallenge

		for count, intialBlobberInfo := range blobberDetailList {

			_, err := utils.ExecuteFaucetWithTokensForWallet(t, "wallet/blobber"+strconv.Itoa(count+1), configPath, 99)
			require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "read_price": utils.IntToZCN(intialBlobberInfo.Terms.Read_price + 1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallet/blobber"+strconv.Itoa(count+1), utils.CreateParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "write_price": utils.IntToZCN(intialBlobberInfo.Terms.Write_price + 1e9)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		_, err = utils.UpdateAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"size":       100 * MB,
		}), true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))

		alloc = utils.GetAllocation(t, allocationId)
		require.Equal(t, alloc.MovedToChallenge, movedToChallengePool, "MovedToChallenge should not change")

		// sleep for 5 minutes
		time.Sleep(6 * time.Minute)

		rewards := getTotalAllocationChallengeRewards(t, allocationId)

		totalBlobberChallengereward := int64(0)
		for _, v := range rewards {
			totalBlobberChallengereward += int64(v.(float64))
		}

		require.Equal(t, movedToChallengePool, totalBlobberChallengereward, "Total Blobber Challenge reward should be 0")

	})

	t.RunWithTimeout("External Party Upgrades Allocation", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
			"expire": "5m",
		}))
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocationId, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "Error getting allocation ID", strings.Join(output, "\n"))

		// Uploading 10% of allocation
		remotepath := "/dir/"
		filesize := 2 * MB
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

		time.Sleep(30 * time.Second)

		alloc := utils.GetAllocation(t, allocationId)
		movedToChallengePool := alloc.MovedToChallenge

		// register a new wallet
		nonAllocationOwnerWallet := "newwallet"
		output, err = utils.CreateWalletForName(t, configPath, nonAllocationOwnerWallet)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))
		_, err = utils.ExecuteFaucetWithTokensForWallet(t, nonAllocationOwnerWallet, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		// Setting allocation to third party extendable
		params := utils.CreateParams(map[string]interface{}{
			"allocation":                 allocationId,
			"set_third_party_extendable": nil,
		})
		output, err = utils.UpdateAllocation(t, configPath, params, true)

		// Updating allocation with new wallet
		_, err = utils.UpdateAllocationWithWallet(t, nonAllocationOwnerWallet, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"size":       100 * MB,
		}), true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		// sleep for 5 minutes
		time.Sleep(5 * time.Minute)

		rewards := getTotalAllocationChallengeRewards(t, allocationId)

		totalBlobberChallengereward := int64(0)
		for _, v := range rewards {
			totalBlobberChallengereward += int64(v.(float64))
		}

		require.Equal(t, movedToChallengePool, totalBlobberChallengereward, "Total Blobber Challenge reward should be equal to MovedToChallenge")

	})

}

func TestAddOrReplaceBlobberAllocationRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	prevBlock := utils.GetLatestFinalizedBlock(t)

	t.Log("prevBlock", prevBlock)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	var blobberDetailList []climodel.BlobberDetails
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	err = json.Unmarshal([]byte(output[0]), &blobberDetailList)
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

	stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
		1, 1, 1, 1, 1, 1,
	}, 1)

	t.RunSequentiallyWithTimeout("Add Blobber to Increase Parity", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		utils.ExecuteFaucetWithTokensForWallet(t, blobberOwnerWallet, configPath, 9)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)

		allocSize := 1 * GB

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"data":   1,
			"tokens": 99,
			"parity": 1,
			"expire": "10m",
		})
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocation := utils.GetAllocation(t, allocationId)

		var allocationBlobbers []string

		for _, blobber := range allocation.Blobbers {
			allocationBlobbers = append(allocationBlobbers, blobber.ID)
		}

		newBlobberID := ""

		for _, blobber := range blobberList {
			if !contains(allocationBlobbers, blobber.Id) {
				newBlobberID = blobber.Id
				allocationBlobbers = append(allocationBlobbers, newBlobberID)
				break
			}
		}

		params := utils.CreateParams(map[string]interface{}{
			"allocation":                 allocationId,
			"set_third_party_extendable": nil,
			"add_blobber":                newBlobberID,
		})

		output, err = utils.UpdateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.2 * GB
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

		// Challenge Rewards
		time.Sleep(12 * time.Minute)
		blobberRewards := getAllocationChallengeRewards(t, allocationId)

		require.Equal(t, 3, len(blobberRewards), "All 3 blobber should get the rewards")

		avgBlobberReward := 0
		for _, v := range blobberRewards {
			avgBlobberReward += int(v.(float64))
		}

		avgBlobberReward = avgBlobberReward / len(blobberRewards)

		for k, v := range blobberRewards {
			require.Containsf(t, allocationBlobbers, k, "blobber id not found in allocation blobber list")
			if v.(float64) == 0 {
				require.InEpsilon(t, avgBlobberReward, int(v.(float64)), 0.05, "blobber reward is not in range")
			}
		}

		// Cancellation Rewards
		allocCancellationRewards, err := getAllocationCancellationReward(t, allocationId, blobberListString)
		if err != nil {
			return
		}

		blobber1CancellationReward := allocCancellationRewards[0]
		blobber2CancellationReward := allocCancellationRewards[1]

		totalExpectedCancellationReward := sizeInGB(int64(allocSize)*2) * 1000000000 * 0.2

		t.Log("totalExpectedCancellationReward", totalExpectedCancellationReward)

		t.Log("blobber1CancellationReward", blobber1CancellationReward)
		t.Log("blobber2CancellationReward", blobber2CancellationReward)

		require.InEpsilon(t, totalExpectedCancellationReward, float64(blobber1CancellationReward+blobber2CancellationReward), 0.05, "Total Cancellation Reward should be equal to total expected cancellation reward")
		require.InEpsilon(t, blobber1CancellationReward, blobber2CancellationReward, 0.05, "Blobber 1 Cancellation Reward should be equal to total expected cancellation reward")

	})

	t.RunSequentiallyWithTimeout("Replace Blobber", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		utils.ExecuteFaucetWithTokensForWallet(t, blobberOwnerWallet, configPath, 9)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)

		allocSize := 1 * GB

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"data":   1,
			"tokens": 99,
			"parity": 1,
			"expire": "10m",
		})
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocation := utils.GetAllocation(t, allocationId)

		var allocationBlobbers []string

		for _, blobber := range allocation.Blobbers {
			allocationBlobbers = append(allocationBlobbers, blobber.ID)
		}

		newBlobberID := ""

		for _, blobber := range blobberList {
			if !contains(allocationBlobbers, blobber.Id) {
				newBlobberID = blobber.Id
				allocationBlobbers = append(allocationBlobbers, newBlobberID)
				break
			}
		}

		output, err = utils.UpdateAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation":     allocationId,
			"add_blobber":    newBlobberID,
			"remove_blobber": allocationBlobbers[0],
		}), true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))

		// remove allocationBlobbers[0] from allocationBlobbers
		allocationBlobbers = allocationBlobbers[1:]

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.2 * GB
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

		time.Sleep(12 * time.Minute)

		// Challenge Rewards
		blobberRewards := getAllocationChallengeRewards(t, allocationId)
		require.Equal(t, 2, len(blobberRewards), "Only 2 blobber should get the rewards")

		avgBlobberReward := 0
		for _, v := range blobberRewards {
			avgBlobberReward += int(v.(float64))
		}

		avgBlobberReward = avgBlobberReward / len(blobberRewards)

		for k, v := range blobberRewards {
			require.Containsf(t, allocationBlobbers, k, "blobber id not found in allocation blobber list")
			if v.(float64) == 0 {
				require.InEpsilon(t, avgBlobberReward, int(v.(float64)), 0.05, "blobber reward is not in range")
			}
		}

		// Cancellation Rewards
		allocCancellationRewards, err := getAllocationCancellationReward(t, allocationId, blobberListString)
		if err != nil {
			return
		}

		blobber1CancellationReward := allocCancellationRewards[0]
		blobber2CancellationReward := allocCancellationRewards[1]

		totalExpectedCancellationReward := sizeInGB(int64(allocSize)*2) * 1000000000 * 0.2

		t.Log("totalExpectedCancellationReward", totalExpectedCancellationReward)

		t.Log("blobber1CancellationReward", blobber1CancellationReward)
		t.Log("blobber2CancellationReward", blobber2CancellationReward)

		require.InEpsilon(t, totalExpectedCancellationReward, float64(blobber1CancellationReward+blobber2CancellationReward), 0.05, "Total Cancellation Reward should be equal to total expected cancellation reward")
		require.InEpsilon(t, blobber1CancellationReward, blobber2CancellationReward, 0.05, "Blobber 1 Cancellation Reward should be equal to total expected cancellation reward")

	})

}

func getAllocationCancellationReward(t *test.SystemTest, allocationID string, blobberList []string) ([]int64, error) {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl+"/v1/screst/"+StorageScAddress+"/cancellation-rewards?allocation_id=?", allocationID)

	t.Log("URL : ", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var allocationCancellationRewards map[string]ProviderAllocationRewards
	err = json.Unmarshal(body, &allocationCancellationRewards)
	if err != nil {
		return nil, err
	}

	var result []int64

	for _, blobber := range blobberList {
		result = append(result, allocationCancellationRewards[blobber].Total)
	}

	return result, nil
}

func getAllocationChallengeRewards(t *test.SystemTest, allocationID string) map[string]interface{} {
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-challenge-rewards?allocation_id=" + allocationID)

	t.Log("URL : ", url)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Error getting allocation challenge rewards: %v", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatalf("Error closing allocation challenge rewards: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading allocation challenge rewards: %v", err)
	}

	var allocationChallengeRewards map[string]interface{}
	err = json.Unmarshal(body, &allocationChallengeRewards)
	if err != nil {
		t.Fatalf("Error unmarshalling allocation challenge rewards: %v", err)
	}

	t.Log("allocationChallengeRewards", allocationChallengeRewards)

	blobberRewards := allocationChallengeRewards["blobber_rewards"].(map[string]interface{})

	return blobberRewards
}

func getTotalAllocationChallengeRewards(t *test.SystemTest, allocationID string) map[string]interface{} {
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-challenge-rewards?allocation_id=" + allocationID)

	t.Log("URL : ", url)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Error getting allocation challenge rewards: %v", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatalf("Error closing allocation challenge rewards: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading allocation challenge rewards: %v", err)
	}

	var allocationChallengeRewards map[string]interface{}
	err = json.Unmarshal(body, &allocationChallengeRewards)
	if err != nil {
		t.Fatalf("Error unmarshalling allocation challenge rewards: %v", err)
	}

	t.Log("allocationChallengeRewards", allocationChallengeRewards)

	challengeRewards := allocationChallengeRewards["blobber_rewards"].(map[string]interface{})

	for i, j := range allocationChallengeRewards["validator_rewards"].(map[string]interface{}) {
		challengeRewards[i] = j
	}

	return challengeRewards
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
