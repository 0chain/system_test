package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func TestAllocationRewards(testSetup *testing.T) {
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

	t.RunSequentiallyWithTimeout("Create + Upload + Upgrade equal read price 0.1", 1*time.Hour, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidatorsForWallet(t, blobberListString, validatorListString, configPath, utils.EscapedTestName(t), []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
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
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		time.Sleep(1 * time.Minute)

		alloc = utils.GetAllocation(t, allocationId)
		require.Greater(t, alloc.MovedToChallenge, movedToChallengePool, "MovedToChallenge should increase")
		movedToChallengePool = alloc.MovedToChallenge

		for _, intialBlobberInfo := range blobberDetailList {
			_, err := utils.ExecuteFaucetWithTokensForWallet(t, "wallets/blobber_owner", configPath, 99)
			require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallets/blobber_owner", utils.CreateParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "read_price": utils.IntToZCN(intialBlobberInfo.Terms.ReadPrice * 10)}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallets/blobber_owner", utils.CreateParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "write_price": utils.IntToZCN(intialBlobberInfo.Terms.WritePrice * 10)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}

		_, err = utils.UpdateAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"size":       100 * MB,
		}), true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))

		// sleep for 10 minutes
		time.Sleep(11 * time.Minute)

		alloc = utils.GetAllocation(t, allocationId)
		require.Equal(t, true, alloc.Finalized, "Allocation should be finalized : ", alloc.ExpirationDate)
		require.Greater(t, alloc.MovedToChallenge, movedToChallengePool, "MovedToChallenge should not change")

		rewards := getTotalAllocationChallengeRewards(t, allocationId)

		totalBlobberChallengereward := int64(0)
		for _, v := range rewards {
			totalBlobberChallengereward += int64(v.(float64))
		}

		require.Equal(t, alloc.MovedToChallenge-alloc.MovedBack, totalBlobberChallengereward, "Total Blobber Challenge reward should not change")

		t.Log("Collecting rewards for blobbers count : ", len(alloc.Blobbers))

		for _, blobber := range alloc.Blobbers {
			t.Log("collecting rewards for blobber", blobber.ID)
			collectAndVerifyRewardsForWallet(t, blobber.ID, utils.EscapedTestName(t))
		}
	})

	t.RunSequentiallyWithTimeout("Create + Upload + Cancel equal read price 0.1", 1*time.Hour, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidatorsForWallet(t, blobberListString, validatorListString, configPath, utils.EscapedTestName(t), []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		allocSize := 10 * MB

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   allocSize,
			"data":   1,
			"lock":   2,
			"parity": 1,
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
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		time.Sleep(2 * time.Minute)

		alloc := utils.GetAllocation(t, allocationId)
		beforeExpiry := alloc.ExpirationDate
		beforeMovedToChallenge := alloc.MovedToChallenge

		_, err = utils.CancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, "Error canceling allocation", strings.Join(output, "\n"))

		// sleep for 30 seconds
		time.Sleep(30 * time.Second)

		alloc = utils.GetAllocation(t, allocationId)
		afterExpiry := alloc.ExpirationDate

		rewards := getTotalAllocationChallengeRewards(t, allocationId)
		totalBlobberChallengereward := int64(0)
		for _, v := range rewards {
			totalBlobberChallengereward += int64(v.(float64))
		}

		require.Equal(t, alloc.MovedToChallenge, beforeMovedToChallenge, "MovedToChallenge should not change")

		expectedChallengeRewards := float64(beforeMovedToChallenge) * (float64(afterExpiry-alloc.StartTime) / float64(beforeExpiry-alloc.StartTime))

		require.Equal(t, alloc.MovedToChallenge-alloc.MovedBack, totalBlobberChallengereward, "Total Blobber Challenge Reward should be less than MovedToChallenge")
		require.InEpsilon(t, int64(expectedChallengeRewards), totalBlobberChallengereward, 0.1, "Expected challenge rewards should be equal to actual challenge rewards")

		// cancelation Rewards
		allocCancelationRewards, err := getAllocationCancellationReward(t, allocationId, blobberListString)
		require.Nil(t, err, "Error getting allocation cancelation rewards", strings.Join(output, "\n"))

		blobber1cancelationReward := allocCancelationRewards[0]
		blobber2cancelationReward := allocCancelationRewards[1]

		totalExpectedcancelationReward := sizeInGB(int64(allocSize)) * 2 * 10000000000 * 0.2
		t.Log("totalExpectedcancelationReward", totalExpectedcancelationReward, "MovedToChallenge", alloc.MovedToChallenge, "MovedBack", alloc.MovedBack)

		totalExpectedcancelationReward -= float64(alloc.MovedToChallenge - alloc.MovedBack)

		t.Log("totalExpectedcancelationReward", totalExpectedcancelationReward)

		t.Log("blobber1cancelationReward", blobber1cancelationReward)
		t.Log("blobber2cancelationReward", blobber2cancelationReward)

		require.InEpsilon(t, totalExpectedcancelationReward, float64(blobber1cancelationReward+blobber2cancelationReward), 0.05, "Total cancelation Reward should be equal to total expected cancelation reward")
		require.Equal(t, blobber1cancelationReward, blobber1cancelationReward, "Blobber 1 cancelation Reward should be equal to total expected cancelation reward")
		require.Equal(t, blobber1cancelationReward, blobber2cancelationReward, "Blobber 2 cancelation Reward should be equal to total expected cancelation reward")

		t.Log("Collecting rewards for blobbers count : ", len(alloc.Blobbers))

		for _, blobber := range alloc.Blobbers {
			t.Log("collecting rewards for blobber", blobber.ID)
			collectAndVerifyRewardsForWallet(t, blobber.ID, utils.EscapedTestName(t))
		}
	})

	t.RunSequentiallyWithTimeout("External Party Upgrades Allocation", 1*time.Hour, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidatorsForWallet(t, blobberListString, validatorListString, configPath, utils.EscapedTestName(t), []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
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
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		time.Sleep(30 * time.Second)

		alloc := utils.GetAllocation(t, allocationId)
		movedToChallengePool := alloc.MovedToChallenge

		// Setting allocation to third party extendable
		params := utils.CreateParams(map[string]interface{}{
			"allocation":                 allocationId,
			"set_third_party_extendable": nil,
		})
		output, err = utils.UpdateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))

		// register a new wallet
		nonAllocationOwnerWallet := "newwallet"
		output, err = utils.CreateWalletForName(t, configPath, nonAllocationOwnerWallet)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))
		_, err = utils.ExecuteFaucetWithTokensForWallet(t, nonAllocationOwnerWallet, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		// Updating allocation with new wallet
		_, err = utils.UpdateAllocationWithWallet(t, nonAllocationOwnerWallet, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"size":       100 * MB,
		}), true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		// sleep for 5 minutes
		time.Sleep(11 * time.Minute)

		alloc = utils.GetAllocation(t, allocationId)
		require.Greater(t, alloc.MovedToChallenge, movedToChallengePool, "MovedToChallenge should increase")
		require.Equal(t, true, alloc.Finalized, "Allocation should be finalized : ", alloc.ExpirationDate)

		rewards := getTotalAllocationChallengeRewards(t, allocationId)

		totalBlobberChallengereward := int64(0)
		for _, v := range rewards {
			totalBlobberChallengereward += int64(v.(float64))
		}

		require.InEpsilon(t, alloc.MovedToChallenge-alloc.MovedBack, totalBlobberChallengereward, 0.10, "Total Blobber Challenge reward should be equal to MovedToChallenge")

		t.Log("Collecting rewards for blobbers count : ", len(alloc.Blobbers))

		for _, blobber := range alloc.Blobbers {
			t.Log("collecting rewards for blobber", blobber.ID)
			collectAndVerifyRewardsForWallet(t, blobber.ID, utils.EscapedTestName(t))
		}
	})
}

func TestAddOrReplaceBlobberAllocationRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
		"time_unit": "10m",
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))

	prevBlock := utils.GetLatestFinalizedBlock(t)

	t.Log("prevBlock", prevBlock)

	output, err = utils.CreateWallet(t, configPath)
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

	t.RunSequentiallyWithTimeout("Add Blobber to Increase Parity", 1*time.Hour, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidatorsForWallet(t, blobberListString, validatorListString, configPath, utils.EscapedTestName(t), []float64{
			1, 1, 1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)

		allocSize := 1 * GB

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"data":   1,
			"tokens": 99,
			"parity": 1,
		})
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocation := utils.GetAllocation(t, allocationId)

		var allocationBlobbers []string

		for _, blobber := range allocation.Blobbers {
			allocationBlobbers = append(allocationBlobbers, blobber.ID)
		}

		newBlobberID := ""

		for _, blobber := range blobberList {
			if !stringListContains(allocationBlobbers, blobber.Id) {
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
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// Challenge Rewards
		time.Sleep(10 * time.Minute)
		blobberRewards := getAllocationChallengeRewards(t, allocationId)

		require.Equal(t, 3, len(blobberRewards), "All 3 blobber should get the rewards")

		avgBlobberReward := 0
		for _, v := range blobberRewards {
			avgBlobberReward += int(v.(float64))
		}

		avgBlobberReward /= len(blobberRewards)

		for k, v := range blobberRewards {
			require.Containsf(t, allocationBlobbers, k, "blobber id not found in allocation blobber list")
			if v.(float64) == 0 {
				require.InEpsilon(t, avgBlobberReward, int(v.(float64)), 0.05, "blobber reward is not in range")
			}
		}

		// cancelation Rewards
		alloccancelationRewards, err := getAllocationCancellationReward(t, allocationId, blobberListString)
		require.Nil(t, err, "Error getting allocation cancelation rewards", strings.Join(output, "\n"))

		blobber1cancelationReward := alloccancelationRewards[0]
		blobber2cancelationReward := alloccancelationRewards[1]
		blobber3cancelationReward := alloccancelationRewards[2]

		totalExpectedcancelationReward := sizeInGB(int64(allocSize)*3) * 1000000000 * 0.2

		t.Log("totalExpectedcancelationReward", totalExpectedcancelationReward)

		allocation = utils.GetAllocation(t, allocationId)
		totalExpectedcancelationReward -= float64(allocation.MovedToChallenge - allocation.MovedBack)

		t.Log("totalExpectedcancelationReward", totalExpectedcancelationReward)

		t.Log("blobber1cancelationReward", blobber1cancelationReward)
		t.Log("blobber2cancelationReward", blobber2cancelationReward)
		t.Log("blobber3cancelationReward", blobber3cancelationReward)

		require.InEpsilon(t, totalExpectedcancelationReward, float64(blobber1cancelationReward+blobber2cancelationReward+blobber3cancelationReward), 0.05, "Total cancelation Reward should be equal to total expected cancelation reward")
		require.InEpsilon(t, blobber1cancelationReward, blobber2cancelationReward, 0.05, "Blobber 1 cancelation Reward should be equal to blobber 2 reward")
		require.InEpsilon(t, blobber1cancelationReward, blobber3cancelationReward, 0.05, "Blobber 2 cancelation Reward should be equal to blobber 3 reward")

		for _, blobber := range allocation.Blobbers {
			t.Log("collecting rewards for blobber", blobber.ID)
			collectAndVerifyRewardsForWallet(t, blobber.ID, utils.EscapedTestName(t))
		}
	})

	t.RunSequentiallyWithTimeout("Replace Blobber", 1*time.Hour, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidatorsForWallet(t, blobberListString, validatorListString, configPath, utils.EscapedTestName(t), []float64{
			1, 1, 1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))
		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)
		allocSize := 1 * GB // 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"data":   1,
			"tokens": 99,
			"parity": 1,
		})
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocation := utils.GetAllocation(t, allocationId)

		var allocationBlobbers []string

		for _, blobber := range allocation.Blobbers {
			allocationBlobbers = append(allocationBlobbers, blobber.ID)
		}

		newBlobberID := ""

		for _, blobber := range blobberList {
			if !stringListContains(allocationBlobbers, blobber.Id) {
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

		avgBlobberReward /= len(blobberRewards)

		for k, v := range blobberRewards {
			require.Containsf(t, allocationBlobbers, k, "blobber id not found in allocation blobber list")
			if v.(float64) == 0 {
				require.InEpsilon(t, avgBlobberReward, int(v.(float64)), 0.05, "blobber reward is not in range")
			}
		}

		// cancelation Rewards
		alloccancelationRewards, err := getAllocationCancellationReward(t, allocationId, blobberListString)
		require.Nil(t, err, "Error getting allocation cancelation rewards", strings.Join(output, "\n"))

		// Replaced blobber
		blobber1cancelationReward := alloccancelationRewards[0]
		expectedReplacedBlobberCancellationCharge := sizeInGB(int64(allocSize)) * 1000000000 * 0.2
		t.Log("expectedcancelationReward", expectedReplacedBlobberCancellationCharge)
		require.InEpsilon(t, expectedReplacedBlobberCancellationCharge, float64(blobber1cancelationReward), 0.05, "Replaced blobber cancellation charge Reward should be equal to total expected cancelation reward")

		blobber2cancelationReward := alloccancelationRewards[1]
		blobber3cancelationReward := alloccancelationRewards[2]
		totalExpectedcancelationReward := sizeInGB(int64(allocSize)*2) * 1000000000 * 0.2
		t.Log("totalExpectedcancelationReward", totalExpectedcancelationReward)

		allocation = utils.GetAllocation(t, allocationId)
		totalExpectedcancelationReward -= float64(allocation.MovedToChallenge - allocation.MovedBack)

		t.Log("totalExpectedcancelationReward", totalExpectedcancelationReward)

		t.Log("blobber1cancelationReward", blobber1cancelationReward)
		t.Log("blobber2cancelationReward", blobber2cancelationReward)
		t.Log("blobber3cancelationReward", blobber3cancelationReward)
		require.InEpsilon(t, totalExpectedcancelationReward, float64(blobber2cancelationReward+blobber3cancelationReward), 0.05, "Total cancelation Reward should be equal to total expected cancelation reward")
		require.InEpsilon(t, blobber2cancelationReward, blobber3cancelationReward, 0.05, "Blobber 1 cancelation Reward should be equal to total expected cancelation reward")

		for _, blobber := range allocation.Blobbers {
			t.Log("collecting rewards for blobber", blobber.ID)
			collectAndVerifyRewardsForWallet(t, blobber.ID, utils.EscapedTestName(t))
		}
	})
}

func getAllocationCancellationReward(t *test.SystemTest, allocationID string, blobberList []string) ([]int64, error) {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/cancellation-rewards?" + "allocation_id=" + allocationID)

	t.Log("URL : ", url)

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var allocationcancelationRewards map[string]ProviderAllocationRewards
	err = json.Unmarshal(body, &allocationcancelationRewards)
	if err != nil {
		return nil, err
	}

	var result []int64

	for _, blobber := range blobberList {
		result = append(result, allocationcancelationRewards[blobber].Total)
	}

	return result, nil
}

func getAllocationChallengeRewards(t *test.SystemTest, allocationID string) map[string]interface{} {
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-challenge-rewards?allocation_id=" + allocationID)

	t.Log("URL : ", url)

	resp, err := http.Get(url) //nolint:gosec
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

	resp, err := http.Get(url) //nolint:gosec
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

func stringListContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func collectAndVerifyRewardsForWallet(t *test.SystemTest, blobberID, wallet string) {
	modelWallet, err := utils.GetWalletForName(t, configPath, wallet)
	require.Nil(t, err, "Get wallet failed")

	balanceBefore := utils.GetBalanceFromSharders(t, modelWallet.ClientID)
	log.Println("balanceBefore", balanceBefore)

	output, err := utils.StakePoolInfo(t, configPath, utils.CreateParams(map[string]interface{}{
		"blobber_id": blobberID,
		"json":       "",
	}))
	require.Nil(t, err, "error getting stake pool info")
	require.Len(t, output, 1)
	stakePoolAfter := climodel.StakePoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &stakePoolAfter)
	require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
	require.NotEmpty(t, stakePoolAfter)

	rewards := int64(0)
	for _, poolDelegateInfo := range stakePoolAfter.Delegate {
		if poolDelegateInfo.DelegateID == modelWallet.ClientID {
			rewards = poolDelegateInfo.TotalReward
			break
		}
	}
	require.Greater(t, rewards, int64(0))
	t.Logf("reward tokens: %v", rewards)

	output, err = utils.CollectRewardsForWallet(t, configPath, utils.CreateParams(map[string]interface{}{
		"provider_type": "blobber",
		"provider_id":   blobberID,
	}), wallet, true)
	require.Nil(t, err, "Error collecting rewards", strings.Join(output, "\n"))

	balanceAfter := utils.GetBalanceFromSharders(t, modelWallet.ClientID)
	require.Nil(t, err, "Error getting balance", balanceAfter)

	require.GreaterOrEqual(t, balanceAfter+100000000, balanceBefore+rewards, "Balance should increase after collecting rewards")
}

func stakeTokensToBlobbersAndValidatorsForWallet(t *test.SystemTest, blobbers, validators []string, configPath, wallet string, tokens []float64, numDelegates int) {
	tIdx := 0

	for i := 0; i < numDelegates; i++ {
		for _, blobber := range blobbers { // add balance to delegate wallet
			_, err := utils.ExecuteFaucetWithTokensForWallet(t, wallet, configPath, tokens[tIdx]+1)
			require.Nil(t, err, "Error executing faucet")

			t.Log("Staking tokens for blobber: ", blobber)

			// stake tokens
			_, err = utils.StakeTokensForWallet(t, configPath, wallet, utils.CreateParams(map[string]interface{}{
				"blobber_id": blobber,
				"tokens":     tokens[tIdx],
			}), true)
			require.Nil(t, err, "Error staking tokens")

			tIdx++
		}
	}

	for i := 0; i < numDelegates; i++ {
		for _, validator := range validators {
			// add balance to delegate wallet
			_, err := utils.ExecuteFaucetWithTokensForWallet(t, wallet, configPath, tokens[tIdx]+1)
			require.Nil(t, err, "Error executing faucet")

			// stake tokens
			_, err = utils.StakeTokensForWallet(t, configPath, wallet, utils.CreateParams(map[string]interface{}{
				"validator_id": validator,
				"tokens":       tokens[tIdx],
			}), true)
			require.Nil(t, err, "Error staking tokens")

			tIdx++
		}
	}
}

//nolint:deadcode,unused
func unstakeTokensForBlobbersAndValidatorsForWallet(t *test.SystemTest, blobbers, validators []string, configPath, wallet string, numDelegates int) {
	for i := 0; i < numDelegates; i++ {
		for _, blobber := range blobbers {
			t.Log("Unstaking tokens for blobber: ", blobber)
			// unstake tokens
			_, err := utils.UnstakeTokensForWallet(t, configPath, wallet, utils.CreateParams(map[string]interface{}{
				"blobber_id": blobber,
			}))
			require.Nil(t, err, "Error unstaking tokens")
		}
	}

	for i := 0; i < numDelegates; i++ {
		for _, validator := range validators {
			t.Log("Unstaking tokens for validator: ", validator)
			// unstake tokens
			_, err := utils.UnstakeTokensForWallet(t, configPath, wallet, utils.CreateParams(map[string]interface{}{
				"validator_id": validator,
			}))
			require.Nil(t, err, "Error unstaking tokens")
		}
	}
}

func sizeInGB(size int64) float64 {
	return float64(size) / float64(GB)
}
