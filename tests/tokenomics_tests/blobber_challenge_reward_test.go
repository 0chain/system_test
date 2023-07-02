package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlobberChallengeRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

	var blobberList []climodel.BlobberInfo
	output, err := utils.ListBlobbers(t, configPath, "--json")
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

	t.Log("Blobber List: ", blobberListString)
	t.Log("Validator List: ", validatorListString)

	blobber1 := blobberListString[0]
	blobber2 := blobberListString[1]
	validator1 := validatorListString[0]
	validator2 := validatorListString[1]

	// Delegate Wallets
	b1D1Wallet, _ := utils.GetWalletForName(t, configPath, blobber1Delegate1Wallet)
	b1D2Wallet, _ := utils.GetWalletForName(t, configPath, blobber1Delegate2Wallet)
	b2D1Wallet, _ := utils.GetWalletForName(t, configPath, blobber2Delegate1Wallet)
	b2D2Wallet, _ := utils.GetWalletForName(t, configPath, blobber2Delegate2Wallet)
	v1D1Wallet, _ := utils.GetWalletForName(t, configPath, validator1Delegate1Wallet)
	v1D2Wallet, _ := utils.GetWalletForName(t, configPath, validator1Delegate2Wallet)
	v2D1Wallet, _ := utils.GetWalletForName(t, configPath, validator2Delegate1Wallet)
	v2D2Wallet, _ := utils.GetWalletForName(t, configPath, validator2Delegate2Wallet)

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		// Creating Allocation

		output := utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// sleep for 10 minutes
		time.Sleep(5 * time.Minute)

		allocation := utils.GetAllocation(t, allocationId)

		t.Log("Moved to Challenge", allocation.MovedToChallenge)

		totalExpectedReward := allocation.MovedToChallenge

		challengeRewards, err := getAllAllocationChallengeRewards(t, allocationId)
		require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

		blobber1TotalReward := challengeRewards[blobber1].Amount
		blobber2TotalReward := challengeRewards[blobber2].Amount
		blobber1DelegatesTotalReward := challengeRewards[blobber1].DelegateRewards[b1D1Wallet.ClientID]
		blobber2DelegatesTotalReward := challengeRewards[blobber2].DelegateRewards[b2D1Wallet.ClientID]
		validator1TotalReward := challengeRewards[validator1].Amount
		validator2TotalReward := challengeRewards[validator2].Amount
		validator1DelegatesTotalReward := challengeRewards[validator1].DelegateRewards[v1D1Wallet.ClientID]
		validator2DelegatesTotalReward := challengeRewards[validator2].DelegateRewards[v2D1Wallet.ClientID]

		totalReward := blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		t.Log("Total Reward: ", totalReward)
		t.Log("Total Expected Reward: ", totalExpectedReward)
		t.Log("Blobber 1 Total Reward: ", blobber1TotalReward)
		t.Log("Blobber 2 Total Reward: ", blobber2TotalReward)
		t.Log("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		t.Log("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		t.Log("Validator 1 Total Reward: ", validator1TotalReward)
		t.Log("Validator 2 Total Reward: ", validator2TotalReward)
		t.Log("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		t.Log("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		require.InEpsilon(t, totalReward, totalExpectedReward, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward, blobber2TotalReward, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward, blobber2DelegatesTotalReward, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, blobber1TotalReward+blobber1DelegatesTotalReward, blobber2TotalReward+blobber2DelegatesTotalReward, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)
	})

	t.RunSequentiallyWithTimeout("Client Uploads 30% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.3 * GB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		// Creating Allocation

		output := utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// sleep for 10 minutes
		time.Sleep(5 * time.Minute)

		allocation := utils.GetAllocation(t, allocationId)

		t.Log("Moved to Challenge", allocation.MovedToChallenge)

		totalExpectedReward := allocation.MovedToChallenge

		challengeRewards, err := getAllAllocationChallengeRewards(t, allocationId)
		require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

		blobber1TotalReward := challengeRewards[blobber1].Amount
		blobber2TotalReward := challengeRewards[blobber2].Amount
		blobber1DelegatesTotalReward := challengeRewards[blobber1].DelegateRewards[b1D1Wallet.ClientID]
		blobber2DelegatesTotalReward := challengeRewards[blobber2].DelegateRewards[b2D1Wallet.ClientID]
		validator1TotalReward := challengeRewards[validator1].Amount
		validator2TotalReward := challengeRewards[validator2].Amount
		validator1DelegatesTotalReward := challengeRewards[validator1].DelegateRewards[v1D1Wallet.ClientID]
		validator2DelegatesTotalReward := challengeRewards[validator2].DelegateRewards[v2D1Wallet.ClientID]

		totalReward := blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		t.Log("Total Reward: ", totalReward)
		t.Log("Total Expected Reward: ", totalExpectedReward)
		t.Log("Blobber 1 Total Reward: ", blobber1TotalReward)
		t.Log("Blobber 2 Total Reward: ", blobber2TotalReward)
		t.Log("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		t.Log("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		t.Log("Validator 1 Total Reward: ", validator1TotalReward)
		t.Log("Validator 2 Total Reward: ", validator2TotalReward)
		t.Log("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		t.Log("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		require.InEpsilon(t, totalReward, totalExpectedReward, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward, blobber2TotalReward, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward, blobber2DelegatesTotalReward, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, blobber1TotalReward+blobber1DelegatesTotalReward, blobber2TotalReward+blobber2DelegatesTotalReward, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 1 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 2, 1, 2,
		}, 1)

		// Creating Allocation

		output := utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
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

		// sleep for 10 minutes
		time.Sleep(5 * time.Minute)

		allocation := utils.GetAllocation(t, allocationId)

		t.Log(allocation.MovedToChallenge)

		totalExpectedReward := allocation.MovedToChallenge

		challengeRewards, err := getAllAllocationChallengeRewards(t, allocationId)
		require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

		blobber1TotalReward := challengeRewards[blobber1].Amount
		blobber2TotalReward := challengeRewards[blobber2].Amount
		blobber1DelegatesTotalReward := challengeRewards[blobber1].DelegateRewards[b1D1Wallet.ClientID]
		blobber2DelegatesTotalReward := challengeRewards[blobber2].DelegateRewards[b2D1Wallet.ClientID]
		validator1TotalReward := challengeRewards[validator1].Amount
		validator2TotalReward := challengeRewards[validator2].Amount
		validator1DelegatesTotalReward := challengeRewards[validator1].DelegateRewards[v1D1Wallet.ClientID]
		validator2DelegatesTotalReward := challengeRewards[validator2].DelegateRewards[v2D1Wallet.ClientID]

		totalReward := blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		t.Log("Total Reward: ", totalReward)
		t.Log("Total Expected Reward: ", totalExpectedReward)
		t.Log("Blobber 1 Total Reward: ", blobber1TotalReward)
		t.Log("Blobber 2 Total Reward: ", blobber2TotalReward)
		t.Log("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		t.Log("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		t.Log("Validator 1 Total Reward: ", validator1TotalReward)
		t.Log("Validator 2 Total Reward: ", validator2TotalReward)
		t.Log("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		t.Log("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		require.InEpsilon(t, totalReward, totalExpectedReward, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward, blobber2TotalReward, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward, blobber2DelegatesTotalReward, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, blobber1TotalReward+blobber1DelegatesTotalReward, blobber2TotalReward+blobber2DelegatesTotalReward, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 1)

	})

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 2 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1, 1, 1, 1, 1,
		}, 2)

		// Creating Allocation

		output := utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
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

		// sleep for 10 minutes
		time.Sleep(5 * time.Minute)

		allocation := utils.GetAllocation(t, allocationId)

		t.Log(allocation.MovedToChallenge)

		totalExpectedReward := float64(allocation.MovedToChallenge)

		challengeRewards, err := getAllAllocationChallengeRewards(t, allocationId)
		require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

		blobber1Delegate1TotalReward := challengeRewards[blobber1].DelegateRewards[b1D1Wallet.ClientID]
		blobber1Delegate2TotalReward := challengeRewards[blobber1].DelegateRewards[b1D2Wallet.ClientID]
		blobber2Delegate1TotalReward := challengeRewards[blobber2].DelegateRewards[b2D1Wallet.ClientID]
		blobber2Delegate2TotalReward := challengeRewards[blobber2].DelegateRewards[b2D2Wallet.ClientID]
		validator1Delegate1TotalReward := challengeRewards[validator1].DelegateRewards[v1D1Wallet.ClientID]
		validator1Delegate2TotalReward := challengeRewards[validator1].DelegateRewards[v1D2Wallet.ClientID]
		validator2Delegate1TotalReward := challengeRewards[validator2].DelegateRewards[v2D1Wallet.ClientID]
		validator2Delegate2TotalReward := challengeRewards[validator2].DelegateRewards[v2D2Wallet.ClientID]

		blobber1TotalReward := challengeRewards[blobber1].Amount
		blobber2TotalReward := challengeRewards[blobber2].Amount
		blobber1DelegatesTotalReward := blobber1Delegate1TotalReward + blobber1Delegate2TotalReward
		blobber2DelegatesTotalReward := blobber2Delegate1TotalReward + blobber2Delegate2TotalReward
		validator1TotalReward := challengeRewards[validator1].Amount
		validator2TotalReward := challengeRewards[validator2].Amount
		validator1DelegatesTotalReward := validator1Delegate1TotalReward + validator1Delegate2TotalReward
		validator2DelegatesTotalReward := validator2Delegate1TotalReward + validator2Delegate2TotalReward

		totalReward := blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		t.Log("Total Reward: ", totalReward)
		t.Log("Total Expected Reward: ", totalExpectedReward)
		t.Log("Blobber 1 Total Reward: ", blobber1TotalReward)
		t.Log("Blobber 2 Total Reward: ", blobber2TotalReward)
		t.Log("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		t.Log("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		t.Log("Validator 1 Total Reward: ", validator1TotalReward)
		t.Log("Validator 2 Total Reward: ", validator2TotalReward)
		t.Log("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		t.Log("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		t.Log("Blobber 1 Delegate 1 Total Reward: ", blobber1Delegate1TotalReward)
		t.Log("Blobber 1 Delegate 2 Total Reward: ", blobber1Delegate2TotalReward)
		t.Log("Blobber 2 Delegate 1 Total Reward: ", blobber2Delegate1TotalReward)
		t.Log("Blobber 2 Delegate 2 Total Reward: ", blobber2Delegate2TotalReward)
		t.Log("Validator 1 Delegate 1 Total Reward: ", validator1Delegate1TotalReward)
		t.Log("Validator 1 Delegate 2 Total Reward: ", validator1Delegate2TotalReward)
		t.Log("Validator 2 Delegate 1 Total Reward: ", validator2Delegate1TotalReward)
		t.Log("Validator 2 Delegate 2 Total Reward: ", validator2Delegate2TotalReward)

		// check if total reward is equal to expected reward with 5% error margin
		require.InEpsilon(t, totalReward, totalExpectedReward, 0.05, "Total Reward is not equal to expected reward")
		// check if blobber 1 and blobber 2 got the same amount of reward with 5% error margin
		require.InEpsilon(t, blobber1TotalReward, blobber2TotalReward, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		// check if blobber 1 and blobber 2 delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, blobber1DelegatesTotalReward, blobber2DelegatesTotalReward, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		// check if validator 1 and validator 2 got the same amount of reward with 5% error margin
		require.InEpsilon(t, validator1TotalReward, validator2TotalReward, 0.05, "Validator 1 and Validator 2 rewards are not equal")
		// check if validator 1 and validator 2 delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, validator1DelegatesTotalReward, validator2DelegatesTotalReward, 0.05, "Validator 1 and Validator 2 delegate rewards are not equal")

		// check if both blobber delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, blobber1Delegate1TotalReward, blobber1Delegate2TotalReward, 0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not equal")
		require.InEpsilon(t, blobber2Delegate1TotalReward, blobber2Delegate2TotalReward, 0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not equal")

		// check if both validator delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, validator1Delegate1TotalReward, validator1Delegate2TotalReward, 0.05, "Validator 1 Delegate 1 and Validator 1 Delegate 2 rewards are not equal")
		require.InEpsilon(t, validator2Delegate1TotalReward, validator2Delegate2TotalReward, 0.05, "Validator 2 Delegate 1 and Validator 2 Delegate 2 rewards are not equal")

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 2)

	})

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 2 delegate each (unequal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		// Delegate Wallets
		b1D1Wallet, _ := utils.GetWalletForName(t, configPath, blobber1Delegate1Wallet)
		b2D1Wallet, _ := utils.GetWalletForName(t, configPath, blobber2Delegate1Wallet)
		v1D1Wallet, _ := utils.GetWalletForName(t, configPath, validator1Delegate1Wallet)
		v1D2Wallet, _ := utils.GetWalletForName(t, configPath, validator1Delegate2Wallet)
		v2D1Wallet, _ := utils.GetWalletForName(t, configPath, validator2Delegate1Wallet)

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 2, 2, 1, 1, 2, 2,
		}, 2)

		// Creating Allocation

		output := utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
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

		// sleep for 10 minutes
		time.Sleep(5 * time.Minute)

		allocation := utils.GetAllocation(t, allocationId)

		t.Log(allocation.MovedToChallenge)

		totalExpectedReward := float64(allocation.MovedToChallenge)

		challengeRewards, err := getAllAllocationChallengeRewards(t, allocationId)
		require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

		blobber1Delegate1TotalReward := challengeRewards[blobber1].DelegateRewards[b1D1Wallet.ClientID]
		blobber1Delegate2TotalReward := challengeRewards[blobber1].DelegateRewards[b1D2Wallet.ClientID]
		blobber2Delegate1TotalReward := challengeRewards[blobber2].DelegateRewards[b2D1Wallet.ClientID]
		blobber2Delegate2TotalReward := challengeRewards[blobber2].DelegateRewards[b2D2Wallet.ClientID]
		validator1Delegate1TotalReward := challengeRewards[validator1].DelegateRewards[v1D1Wallet.ClientID]
		validator1Delegate2TotalReward := challengeRewards[validator1].DelegateRewards[v1D2Wallet.ClientID]
		validator2Delegate1TotalReward := challengeRewards[validator2].DelegateRewards[v2D1Wallet.ClientID]
		validator2Delegate2TotalReward := challengeRewards[validator2].DelegateRewards[v2D2Wallet.ClientID]

		blobber1TotalReward := challengeRewards[blobber1].Amount
		blobber2TotalReward := challengeRewards[blobber2].Amount
		blobber1DelegatesTotalReward := blobber1Delegate1TotalReward + blobber1Delegate2TotalReward
		blobber2DelegatesTotalReward := blobber2Delegate1TotalReward + blobber2Delegate2TotalReward
		validator1TotalReward := challengeRewards[validator1].Amount
		validator2TotalReward := challengeRewards[validator2].Amount
		validator1DelegatesTotalReward := validator1Delegate1TotalReward + validator1Delegate2TotalReward
		validator2DelegatesTotalReward := validator2Delegate1TotalReward + validator2Delegate2TotalReward

		totalReward := blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		t.Log("Total Reward: ", totalReward)
		t.Log("Total Expected Reward: ", totalExpectedReward)
		t.Log("Blobber 1 Total Reward: ", blobber1TotalReward)
		t.Log("Blobber 2 Total Reward: ", blobber2TotalReward)
		t.Log("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		t.Log("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		t.Log("Validator 1 Total Reward: ", validator1TotalReward)
		t.Log("Validator 2 Total Reward: ", validator2TotalReward)
		t.Log("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		t.Log("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		t.Log("Blobber 1 Delegate 1 Total Reward: ", blobber1Delegate1TotalReward)
		t.Log("Blobber 1 Delegate 2 Total Reward: ", blobber1Delegate2TotalReward)
		t.Log("Blobber 2 Delegate 1 Total Reward: ", blobber2Delegate1TotalReward)
		t.Log("Blobber 2 Delegate 2 Total Reward: ", blobber2Delegate2TotalReward)
		t.Log("Validator 1 Delegate 1 Total Reward: ", validator1Delegate1TotalReward)
		t.Log("Validator 1 Delegate 2 Total Reward: ", validator1Delegate2TotalReward)
		t.Log("Validator 2 Delegate 1 Total Reward: ", validator2Delegate1TotalReward)
		t.Log("Validator 2 Delegate 2 Total Reward: ", validator2Delegate2TotalReward)

		// check if total reward is equal to expected reward with 5% error margin
		require.InEpsilon(t, totalReward, totalExpectedReward, 0.05, "Total Reward is not equal to expected reward")
		// check if blobber 1 and blobber 2 got the same amount of reward with 5% error margin
		require.InEpsilon(t, blobber1TotalReward, blobber2TotalReward, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		// check if blobber 1 and blobber 2 delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, blobber1DelegatesTotalReward, blobber2DelegatesTotalReward, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		// check if validator 1 and validator 2 got the same amount of reward with 5% error margin
		require.InEpsilon(t, validator1TotalReward, validator2TotalReward, 0.05, "Validator 1 and Validator 2 rewards are not equal")
		// check if validator 1 and validator 2 delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, validator1DelegatesTotalReward, validator2DelegatesTotalReward, 0.05, "Validator 1 and Validator 2 delegate rewards are not equal")

		// check if both blobber delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, blobber1Delegate1TotalReward*2, blobber1Delegate2TotalReward, 0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not equal")
		require.InEpsilon(t, blobber2Delegate1TotalReward*2, blobber2Delegate2TotalReward, 0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not equal")

		// check if both validator delegates got the same amount of reward with 5% error margin
		require.InEpsilon(t, validator1Delegate1TotalReward*2, validator1Delegate2TotalReward, 0.05, "Validator 1 Delegate 1 and Validator 1 Delegate 2 rewards are not equal")
		require.InEpsilon(t, validator2Delegate1TotalReward*2, validator2Delegate2TotalReward, 0.05, "Validator 2 Delegate 1 and Validator 2 Delegate 2 rewards are not equal")

		tearDownRewardsTests(t, blobberListString, validatorListString, configPath, allocationId, 2)
	})
}

func stakeTokensToBlobbersAndValidators(t *test.SystemTest, blobbers []string, validators []string, configPath string, tokens []float64, numDelegates int) {
	var blobberDelegates []string
	var validatorDelegates []string

	blobberDelegates = append(blobberDelegates, blobber1Delegate1Wallet)
	blobberDelegates = append(blobberDelegates, blobber2Delegate1Wallet)
	blobberDelegates = append(blobberDelegates, blobber1Delegate2Wallet)
	blobberDelegates = append(blobberDelegates, blobber2Delegate2Wallet)

	validatorDelegates = append(validatorDelegates, validator1Delegate1Wallet)
	validatorDelegates = append(validatorDelegates, validator2Delegate1Wallet)
	validatorDelegates = append(validatorDelegates, validator1Delegate2Wallet)
	validatorDelegates = append(validatorDelegates, validator2Delegate2Wallet)

	idx := 0
	tIdx := 0

	for i := 0; i < numDelegates; i++ {
		for _, blobber := range blobbers {

			// add balance to delegate wallet
			_, err := utils.ExecuteFaucetWithTokensForWallet(t, blobberDelegates[idx], configPath, tokens[tIdx]+1)
			require.Nil(t, err, "Error executing faucet")

			t.Log("Staking tokens for blobber: ", blobber)

			// stake tokens
			_, err = utils.StakeTokensForWallet(t, configPath, blobberDelegates[idx], utils.CreateParams(map[string]interface{}{
				"blobber_id": blobber,
				"tokens":     tokens[tIdx],
			}), true)
			require.Nil(t, err, "Error staking tokens")

			idx++
			tIdx++

		}
	}

	idx = 0

	for i := 0; i < numDelegates; i++ {
		for _, validator := range validators {
			// add balance to delegate wallet
			_, err := utils.ExecuteFaucetWithTokensForWallet(t, validatorDelegates[idx], configPath, tokens[tIdx]+1)
			require.Nil(t, err, "Error executing faucet")

			// stake tokens
			_, err = utils.StakeTokensForWallet(t, configPath, validatorDelegates[idx], utils.CreateParams(map[string]interface{}{
				"validator_id": validator,
				"tokens":       tokens[tIdx],
			}), true)
			require.Nil(t, err, "Error staking tokens")

			idx++
			tIdx++

		}
	}
}

func unstakeTokensForBlobbersAndValidators(t *test.SystemTest, blobbers []string, validators []string, configPath string, numDelegates int) {
	var blobberDelegates []string
	var validatorDelegates []string

	blobberDelegates = append(blobberDelegates, blobber1Delegate1Wallet)
	blobberDelegates = append(blobberDelegates, blobber2Delegate1Wallet)
	blobberDelegates = append(blobberDelegates, blobber1Delegate2Wallet)
	blobberDelegates = append(blobberDelegates, blobber2Delegate2Wallet)

	validatorDelegates = append(validatorDelegates, validator1Delegate1Wallet)
	validatorDelegates = append(validatorDelegates, validator2Delegate1Wallet)
	validatorDelegates = append(validatorDelegates, validator1Delegate2Wallet)
	validatorDelegates = append(validatorDelegates, validator2Delegate2Wallet)

	idx := 0

	for i := 0; i < numDelegates; i++ {

		for _, blobber := range blobbers {
			t.Log("Unstaking tokens for blobber: ", blobber)
			// unstake tokens
			_, err := utils.UnstakeTokensForWallet(t, configPath, blobberDelegates[idx], utils.CreateParams(map[string]interface{}{
				"blobber_id": blobber,
			}))
			require.Nil(t, err, "Error unstaking tokens")

			idx++
		}
	}

	idx = 0

	for i := 0; i < numDelegates; i++ {

		for _, validator := range validators {
			t.Log("Unstaking tokens for validator: ", validator)
			// unstake tokens
			_, err := utils.UnstakeTokensForWallet(t, configPath, validatorDelegates[idx], utils.CreateParams(map[string]interface{}{
				"validator_id": validator,
			}))
			require.Nil(t, err, "Error unstaking tokens")

			idx++
		}
	}
}

type Challenge struct {
	ChallengeID    string           `json:"challenge_id"`
	CreatedAt      common.Timestamp `json:"created_at"`
	AllocationID   string           `json:"allocation_id"`
	BlobberID      string           `json:"blobber_id"`
	ValidatorsID   string           `json:"validators_id"`
	Seed           int64            `json:"seed"`
	AllocationRoot string           `json:"allocation_root"`
	Responded      int64            `json:"responded"`
	Passed         bool             `json:"passed"`
	RoundResponded int64            `json:"round_responded"`
	ExpiredN       int              `json:"expired_n"`
}

func getAllAllocationChallengeRewards(t *test.SystemTest, allocationID string) (map[string]ProviderAllocationRewards, error) {
	var result map[string]ProviderAllocationRewards

	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/alloc-challenge-rewards?allocation_id=" + allocationID)

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

type ProviderAllocationRewards struct {
	DelegateRewards map[string]int64 `json:"delegate_rewards"`
	Amount          int64            `json:"amount"`
	Total           int64            `json:"total"`
	ProviderType    int64            `json:"provider_type"`
}

func tearDownRewardsTests(t *test.SystemTest, blobberList []string, validatorList []string, configPath string, allocationID string, numDelegates int) {
	waitUntilAllocationIsFinalized(t, allocationID)
	unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, numDelegates)
}

func waitUntilAllocationIsFinalized(t *test.SystemTest, allocationID string) {
	for {
		allocation := utils.GetAllocation(t, allocationID)

		if allocation.Finalized == true {
			break
		}

		time.Sleep(5 * time.Second)
	}
}
