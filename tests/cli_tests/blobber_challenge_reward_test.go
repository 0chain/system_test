package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlobberChallengeRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	setupWalletWithCustomTokens(t, configPath, 9.0)

	var blobberList []climodel.BlobberInfo
	output, err := listBlobbers(t, configPath, "--json")
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
	output, err = listValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	var validatorListString []string
	for _, validator := range validatorList {
		validatorListString = append(validatorListString, validator.ID)
	}

	fmt.Println("Blobber List: ", blobberListString)
	fmt.Println("Validator List: ", validatorListString)

	t.RunSequentiallyWithTimeout("Case 1 : Client Uploads 10% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		//t.Skip()
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		// Creating Allocation

		output := setupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		output, err = uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		allocation := getAllocation(t, allocationId)

		fmt.Println(allocation.MovedToChallenge)

		challenges, _ := getAllChallenges(t, allocationId)

		totalExpectedReward := float64(allocation.MovedToChallenge)

		totalReward := 0.0
		blobber1TotalReward := 0.0
		blobber2TotalReward := 0.0
		blobber1DelegatesTotalReward := 0.0
		blobber2DelegatesTotalReward := 0.0
		validator1TotalReward := 0.0
		validator2TotalReward := 0.0
		validator1DelegatesTotalReward := 0.0
		validator2DelegatesTotalReward := 0.0

		for _, challenge := range challenges {

			var isBlobber1 bool
			if challenge.BlobberID == blobberList[0].Id {
				isBlobber1 = true
			}

			blobberReward := 0.0
			blobberDelegateReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegateReward := 0.0
			validator2DelegateReward := 0.0

			challengeRewards, err := getChallengeRewards(t, challenge.ChallengeID)

			if err != nil {
				fmt.Println(err)
			}
			require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

			// check if challengeReward.BlobberRewards is empty and if yes continue
			if len(challengeRewards.BlobberRewards) == 0 {
				continue
			}

			blobberChallengeRewards := challengeRewards.BlobberRewards
			validatorChallengeRewards := challengeRewards.ValidatorRewards
			blobberDelegateChallengeRewards := challengeRewards.BlobberDelegateRewards
			validatorDelegateChallengeRewards := challengeRewards.ValidatorDelegateRewards

			blobberReward += blobberChallengeRewards[0].Amount
			if isBlobber1 {
				blobber1TotalReward += blobberChallengeRewards[0].Amount
			} else {
				blobber2TotalReward += blobberChallengeRewards[0].Amount
			}

			for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
				blobberDelegateReward += blobberDelegateChallengeReward.Amount
				if isBlobber1 {
					blobber1DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				} else {
					blobber2DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				}
			}

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorList[0].ID {
					validator1Reward += validatorChallengeReward.Amount
					validator1TotalReward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorList[1].ID {
					validator2Reward += validatorChallengeReward.Amount
					validator2TotalReward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorList[0].ID {
					validator1DelegateReward += validatorDelegateChallengeReward.Amount
					validator1DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorList[1].ID {
					validator2DelegateReward += validatorDelegateChallengeReward.Amount
					validator2DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				}
			}

			blobberTotalReward := blobberReward + blobberDelegateReward
			validatorsTotalReward := validator1Reward + validator2Reward + validator1DelegateReward + validator2DelegateReward
			totalChallengeReward := blobberTotalReward + validatorsTotalReward

			// check if blobber got 97.5% of the total challenge reward with 5% error margin
			require.InDelta(t, blobberTotalReward, totalChallengeReward*0.975, totalChallengeReward*0.05, "Blobber Reward is not 97.5% of total challenge reward")
			// check if validators got 2.5% of the total challenge reward with 5% error margin
			require.InDelta(t, validatorsTotalReward, totalChallengeReward*0.025, totalChallengeReward*0.05, "Validators Reward is not 2.5% of total challenge reward")

			require.LessOrEqual(t, math.Abs(validator1Reward+validator1DelegateReward-validator2Reward-validator2DelegateReward), float64(3), "Validator 1 and Validator 2 rewards are not equal ")
		}

		totalReward = blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		fmt.Println("Total Reward: ", totalReward)
		fmt.Println("Total Expected Reward: ", totalExpectedReward)
		fmt.Println("Blobber 1 Total Reward: ", blobber1TotalReward)
		fmt.Println("Blobber 2 Total Reward: ", blobber2TotalReward)
		fmt.Println("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		fmt.Println("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		fmt.Println("Validator 1 Total Reward: ", validator1TotalReward)
		fmt.Println("Validator 2 Total Reward: ", validator2TotalReward)
		fmt.Println("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		fmt.Println("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, (blobber1TotalReward+blobber1DelegatesTotalReward)/(blobber2TotalReward+blobber2DelegatesTotalReward), 1, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Case 2 : Client Uploads 30% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		//t.Skip()
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// Creating Allocation

		output := setupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 0.3 * GB
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

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		allocation := getAllocation(t, allocationId)

		fmt.Println(allocation.MovedToChallenge)

		challenges, _ := getAllChallenges(t, allocationId)

		totalExpectedReward := float64(allocation.MovedToChallenge)

		totalReward := 0.0
		blobber1TotalReward := 0.0
		blobber2TotalReward := 0.0
		blobber1DelegatesTotalReward := 0.0
		blobber2DelegatesTotalReward := 0.0
		validator1TotalReward := 0.0
		validator2TotalReward := 0.0
		validator1DelegatesTotalReward := 0.0
		validator2DelegatesTotalReward := 0.0

		for _, challenge := range challenges {

			var isBlobber1 bool
			if challenge.BlobberID == blobberList[0].Id {
				isBlobber1 = true
			}

			blobberReward := 0.0
			blobberDelegateReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegateReward := 0.0
			validator2DelegateReward := 0.0

			challengeRewards, err := getChallengeRewards(t, challenge.ChallengeID)

			if err != nil {
				fmt.Println(err)
			}
			require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

			// check if challengeReward.BlobberRewards is empty and if yes continue
			if len(challengeRewards.BlobberRewards) == 0 {
				continue
			}

			blobberChallengeRewards := challengeRewards.BlobberRewards
			validatorChallengeRewards := challengeRewards.ValidatorRewards
			blobberDelegateChallengeRewards := challengeRewards.BlobberDelegateRewards
			validatorDelegateChallengeRewards := challengeRewards.ValidatorDelegateRewards

			blobberReward += blobberChallengeRewards[0].Amount
			if isBlobber1 {
				blobber1TotalReward += blobberChallengeRewards[0].Amount
			} else {
				blobber2TotalReward += blobberChallengeRewards[0].Amount
			}

			for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
				blobberDelegateReward += blobberDelegateChallengeReward.Amount
				if isBlobber1 {
					blobber1DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				} else {
					blobber2DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				}
			}

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorList[0].ID {
					validator1Reward += validatorChallengeReward.Amount
					validator1TotalReward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorList[1].ID {
					validator2Reward += validatorChallengeReward.Amount
					validator2TotalReward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorList[0].ID {
					validator1DelegateReward += validatorDelegateChallengeReward.Amount
					validator1DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorList[1].ID {
					validator2DelegateReward += validatorDelegateChallengeReward.Amount
					validator2DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				}
			}

			blobberTotalReward := blobberReward + blobberDelegateReward
			validatorsTotalReward := validator1Reward + validator2Reward + validator1DelegateReward + validator2DelegateReward
			totalChallengeReward := blobberTotalReward + validatorsTotalReward

			// check if blobber got 97.5% of the total challenge reward with 5% error margin
			require.InDelta(t, blobberTotalReward, totalChallengeReward*0.975, totalChallengeReward*0.05, "Blobber Reward is not 97.5% of total challenge reward")
			// check if validators got 2.5% of the total challenge reward with 5% error margin
			require.InDelta(t, validatorsTotalReward, totalChallengeReward*0.025, totalChallengeReward*0.05, "Validators Reward is not 2.5% of total challenge reward")

			require.LessOrEqual(t, math.Abs(validator1Reward+validator1DelegateReward-validator2Reward-validator2DelegateReward), float64(3), "Validator 1 and Validator 2 rewards are not equal ")
		}

		totalReward = blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		fmt.Println("Total Reward: ", totalReward)
		fmt.Println("Total Expected Reward: ", totalExpectedReward)
		fmt.Println("Blobber 1 Total Reward: ", blobber1TotalReward)
		fmt.Println("Blobber 2 Total Reward: ", blobber2TotalReward)
		fmt.Println("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		fmt.Println("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		fmt.Println("Validator 1 Total Reward: ", validator1TotalReward)
		fmt.Println("Validator 2 Total Reward: ", validator2TotalReward)
		fmt.Println("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		fmt.Println("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, (blobber1TotalReward+blobber1DelegatesTotalReward)/(blobber2TotalReward+blobber2DelegatesTotalReward), 1, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Case 3 : Client Uploads 10% of Allocation and 1 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 2, 1, 2,
		}, 1)

		// Creating Allocation

		output := setupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

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

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		allocation := getAllocation(t, allocationId)

		fmt.Println(allocation.MovedToChallenge)

		challenges, _ := getAllChallenges(t, allocationId)

		totalExpectedReward := float64(allocation.MovedToChallenge)

		totalReward := 0.0
		blobber1TotalReward := 0.0
		blobber2TotalReward := 0.0
		blobber1DelegatesTotalReward := 0.0
		blobber2DelegatesTotalReward := 0.0
		validator1TotalReward := 0.0
		validator2TotalReward := 0.0
		validator1DelegatesTotalReward := 0.0
		validator2DelegatesTotalReward := 0.0

		for _, challenge := range challenges {

			var isBlobber1 bool
			if challenge.BlobberID == blobberList[0].Id {
				isBlobber1 = true
			}

			blobberReward := 0.0
			blobberDelegateReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegateReward := 0.0
			validator2DelegateReward := 0.0

			challengeRewards, err := getChallengeRewards(t, challenge.ChallengeID)

			if err != nil {
				fmt.Println(err)
			}
			require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

			// check if challengeReward.BlobberRewards is empty and if yes continue
			if len(challengeRewards.BlobberRewards) == 0 {
				continue
			}

			blobberChallengeRewards := challengeRewards.BlobberRewards
			validatorChallengeRewards := challengeRewards.ValidatorRewards
			blobberDelegateChallengeRewards := challengeRewards.BlobberDelegateRewards
			validatorDelegateChallengeRewards := challengeRewards.ValidatorDelegateRewards

			blobberReward += blobberChallengeRewards[0].Amount
			if isBlobber1 {
				blobber1TotalReward += blobberChallengeRewards[0].Amount
			} else {
				blobber2TotalReward += blobberChallengeRewards[0].Amount
			}

			for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
				blobberDelegateReward += blobberDelegateChallengeReward.Amount
				if isBlobber1 {
					blobber1DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				} else {
					blobber2DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				}
			}

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorList[0].ID {
					validator1Reward += validatorChallengeReward.Amount
					validator1TotalReward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorList[1].ID {
					validator2Reward += validatorChallengeReward.Amount
					validator2TotalReward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorList[0].ID {
					validator1DelegateReward += validatorDelegateChallengeReward.Amount
					validator1DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorList[1].ID {
					validator2DelegateReward += validatorDelegateChallengeReward.Amount
					validator2DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				}
			}

			blobberTotalReward := blobberReward + blobberDelegateReward
			validatorsTotalReward := validator1Reward + validator2Reward + validator1DelegateReward + validator2DelegateReward
			totalChallengeReward := blobberTotalReward + validatorsTotalReward

			// check if blobber got 97.5% of the total challenge reward with 5% error margin
			require.InDelta(t, blobberTotalReward, totalChallengeReward*0.975, totalChallengeReward*0.05, "Blobber Reward is not 97.5% of total challenge reward")
			// check if validators got 2.5% of the total challenge reward with 5% error margin
			require.InDelta(t, validatorsTotalReward, totalChallengeReward*0.025, totalChallengeReward*0.05, "Validators Reward is not 2.5% of total challenge reward")

			require.LessOrEqual(t, math.Abs(validator1Reward+validator1DelegateReward-validator2Reward-validator2DelegateReward), float64(3), "Validator 1 and Validator 2 rewards are not equal ")
		}

		totalReward = blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		fmt.Println("Total Reward: ", totalReward)
		fmt.Println("Total Expected Reward: ", totalExpectedReward)
		fmt.Println("Blobber 1 Total Reward: ", blobber1TotalReward)
		fmt.Println("Blobber 2 Total Reward: ", blobber2TotalReward)
		fmt.Println("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		fmt.Println("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		fmt.Println("Validator 1 Total Reward: ", validator1TotalReward)
		fmt.Println("Validator 2 Total Reward: ", validator2TotalReward)
		fmt.Println("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		fmt.Println("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, (blobber1TotalReward+blobber1DelegatesTotalReward)/(blobber2TotalReward+blobber2DelegatesTotalReward), 1, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Case 4 : Client Uploads 10% of Allocation and 2 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		// Delegate Wallets
		b1D1Wallet, _ := getWalletForName(t, configPath, blobber1Delegate1Wallet)
		b2D1Wallet, _ := getWalletForName(t, configPath, blobber2Delegate1Wallet)
		v1D1Wallet, _ := getWalletForName(t, configPath, validator1Delegate1Wallet)
		v1D2Wallet, _ := getWalletForName(t, configPath, validator1Delegate2Wallet)
		v2D1Wallet, _ := getWalletForName(t, configPath, validator2Delegate1Wallet)

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1, 1, 1, 1, 1,
		}, 2)

		// Creating Allocation

		output := setupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

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

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		allocation := getAllocation(t, allocationId)

		fmt.Println(allocation.MovedToChallenge)

		challenges, _ := getAllChallenges(t, allocationId)

		totalExpectedReward := float64(allocation.MovedToChallenge)

		totalReward := 0.0
		blobber1TotalReward := 0.0
		blobber2TotalReward := 0.0
		blobber1DelegatesTotalReward := 0.0
		blobber2DelegatesTotalReward := 0.0
		validator1TotalReward := 0.0
		validator2TotalReward := 0.0
		validator1DelegatesTotalReward := 0.0
		validator2DelegatesTotalReward := 0.0

		blobber1Delegate1TotalReward := 0.0
		blobber1Delegate2TotalReward := 0.0
		blobber2Delegate1TotalReward := 0.0
		blobber2Delegate2TotalReward := 0.0
		validator1Delegate1TotalReward := 0.0
		validator1Delegate2TotalReward := 0.0
		validator2Delegate1TotalReward := 0.0
		validator2Delegate2TotalReward := 0.0

		for _, challenge := range challenges {
			var isBlobber1 bool
			if challenge.BlobberID == blobberListString[0] {
				isBlobber1 = true
			}

			blobberReward := 0.0
			blobberDelegate1Reward := 0.0
			blobberDelegate2Reward := 0.0
			blobberTotalReward := 0.0

			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1Delegate1Reward := 0.0
			validator1Delegate2Reward := 0.0
			validator2Delegate1Reward := 0.0
			validator2Delegate2Reward := 0.0
			validatorsTotalReward := 0.0

			totalChallengeReward := 0.0

			challengeRewards, err := getChallengeRewards(t, challenge.ChallengeID)
			if err != nil {
				fmt.Println(err)
			}
			require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

			// check if challengeReward.BlobberRewards is empty and if yes continue
			if len(challengeRewards.BlobberRewards) == 0 {
				continue
			}

			blobberChallengeRewards := challengeRewards.BlobberRewards
			validatorChallengeRewards := challengeRewards.ValidatorRewards
			blobberDelegatesChallengeRewards := challengeRewards.BlobberDelegateRewards
			validatorDelegatesChallengeRewards := challengeRewards.ValidatorDelegateRewards

			// parsing data for blobber challenge rewards
			for _, blobberChallengeReward := range blobberChallengeRewards {
				blobberReward += blobberChallengeReward.Amount

				if isBlobber1 {
					blobber1TotalReward += blobberChallengeReward.Amount
				} else {
					blobber2TotalReward += blobberChallengeReward.Amount
				}
			}

			// parsing data for blobber delegate rewards
			for _, blobberDelegateChallengeReward := range blobberDelegatesChallengeRewards {

				if isBlobber1 {
					if blobberDelegateChallengeReward.PoolId == b1D1Wallet.ClientID {
						blobberDelegate1Reward += blobberDelegateChallengeReward.Amount
						blobber1Delegate1TotalReward += blobberDelegateChallengeReward.Amount
					} else {
						blobberDelegate2Reward += blobberDelegateChallengeReward.Amount
						blobber1Delegate2TotalReward += blobberDelegateChallengeReward.Amount
					}

					blobber1DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				} else {
					if blobberDelegateChallengeReward.PoolId == b2D1Wallet.ClientID {
						blobberDelegate1Reward += blobberDelegateChallengeReward.Amount
						blobber2Delegate1TotalReward += blobberDelegateChallengeReward.Amount
					} else {
						blobberDelegate2Reward += blobberDelegateChallengeReward.Amount
						blobber2Delegate2TotalReward += blobberDelegateChallengeReward.Amount
					}

					blobber2DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				}
			}

			blobberTotalReward = blobberReward + blobberDelegate1Reward + blobberDelegate2Reward

			// parsing data for validator challenge rewards
			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
					validator1TotalReward += validatorChallengeReward.Amount
				} else {
					validator2Reward += validatorChallengeReward.Amount
					validator2TotalReward += validatorChallengeReward.Amount
				}
			}

			// parsing data for validator delegate rewards
			for _, validatorDelegateChallengeReward := range validatorDelegatesChallengeRewards {
				if validatorDelegateChallengeReward.PoolId == v1D1Wallet.ClientID {
					validator1Delegate1Reward += validatorDelegateChallengeReward.Amount
					validator1Delegate1TotalReward += validatorDelegateChallengeReward.Amount
					validator1DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.PoolId == v1D2Wallet.ClientID {
					validator1Delegate2Reward += validatorDelegateChallengeReward.Amount
					validator1Delegate2TotalReward += validatorDelegateChallengeReward.Amount
					validator1DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.PoolId == v2D1Wallet.ClientID {
					validator2Delegate1Reward += validatorDelegateChallengeReward.Amount
					validator2Delegate1TotalReward += validatorDelegateChallengeReward.Amount
					validator2DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else {
					validator2Delegate2Reward += validatorDelegateChallengeReward.Amount
					validator2Delegate2TotalReward += validatorDelegateChallengeReward.Amount
					validator2DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				}
			}

			validatorsTotalReward = validator1Reward + validator2Reward + validator1Delegate1Reward + validator1Delegate2Reward + validator2Delegate1Reward + validator2Delegate2Reward

			totalChallengeReward = blobberTotalReward + validatorsTotalReward

			// check if blobber got 97.5% of the total challenge reward with 5% error margin
			require.InDelta(t, blobberTotalReward, totalChallengeReward*0.975, totalChallengeReward*0.05, "Blobber Reward is not 97.5% of total challenge reward")
			// check if validators got 2.5% of the total challenge reward with 5% error margin
			require.InDelta(t, validatorsTotalReward, totalChallengeReward*0.025, totalChallengeReward*0.05, "Validators Reward is not 2.5% of total challenge reward")

			// check if blobber delegates got the same amount of reward with 5% error margin
			//require.InDelta(t, blobberDelegate1Reward, blobberDelegate2Reward, blobberDelegate1Reward*0.05, "Blobber delegates reward is not correct", strings.Join(output, "\n"))
			require.LessOrEqual(t, math.Abs(blobberDelegate1Reward-blobberDelegate2Reward), float64(3), "Blobber Delegate rewards are not equal")

			//// check both validators got the same amount of reward with 5% error margin
			//require.InDelta(t, validator1Reward, validator2Reward, validator1Reward*0.05, "Validators reward is not correct", strings.Join(output, "\n"))
			//// check both validators delegates got the same amount of reward with 5% error margin
			//require.InDelta(t, validator1Delegate1Reward, validator1Delegate2Reward, validator1Delegate1Reward*0.05, "Validators reward is not correct", strings.Join(output, "\n"))
			//require.InDelta(t, validator2Delegate1Reward, validator2Delegate2Reward, validator2Delegate1Reward*0.05, "Validators reward is not correct", strings.Join(output, "\n"))

			// check if both validators got the same amount of reward with 5% error margin
			require.LessOrEqual(t, math.Abs(validator1Reward+validator1Delegate1Reward+validator1Delegate2Reward-validator2Reward-validator2Delegate1Reward-validator2Delegate2Reward), float64(3), "Validators rewards are not equal")
			// check if both validators delegates got the same amount of reward with 5% error margin
			require.LessOrEqual(t, math.Abs(validator1Delegate1Reward-validator1Delegate2Reward), float64(3), "Validators Delegate rewards are not equal")
			require.LessOrEqual(t, math.Abs(validator2Delegate1Reward-validator2Delegate2Reward), float64(3), "Validators Delegate rewards are not equal")

		}

		totalReward = blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		fmt.Println("Total Reward: ", totalReward)
		fmt.Println("Total Expected Reward: ", totalExpectedReward)
		fmt.Println("Blobber 1 Total Reward: ", blobber1TotalReward)
		fmt.Println("Blobber 2 Total Reward: ", blobber2TotalReward)
		fmt.Println("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		fmt.Println("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		fmt.Println("Validator 1 Total Reward: ", validator1TotalReward)
		fmt.Println("Validator 2 Total Reward: ", validator2TotalReward)
		fmt.Println("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		fmt.Println("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		fmt.Println("Blobber 1 Delegate 1 Total Reward: ", blobber1Delegate1TotalReward)
		fmt.Println("Blobber 1 Delegate 2 Total Reward: ", blobber1Delegate2TotalReward)
		fmt.Println("Blobber 2 Delegate 1 Total Reward: ", blobber2Delegate1TotalReward)
		fmt.Println("Blobber 2 Delegate 2 Total Reward: ", blobber2Delegate2TotalReward)
		fmt.Println("Validator 1 Delegate 1 Total Reward: ", validator1Delegate1TotalReward)
		fmt.Println("Validator 1 Delegate 2 Total Reward: ", validator1Delegate2TotalReward)
		fmt.Println("Validator 2 Delegate 1 Total Reward: ", validator2Delegate1TotalReward)
		fmt.Println("Validator 2 Delegate 2 Total Reward: ", validator2Delegate2TotalReward)

		// check if total reward is equal to expected reward with 5% error margin
		require.InDeltaf(t, totalReward, totalExpectedReward, totalExpectedReward*0.05, "Total Reward is not equal to expected reward")
		// check if blobber 1 and blobber 2 got the same amount of reward with 5% error margin
		require.InDeltaf(t, blobber1TotalReward, blobber2TotalReward, blobber1TotalReward*0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		// check if blobber 1 and blobber 2 delegates got the same amount of reward with 5% error margin
		require.InDeltaf(t, blobber1DelegatesTotalReward, blobber2DelegatesTotalReward, blobber1DelegatesTotalReward*0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		// check if validator 1 and validator 2 got the same amount of reward with 5% error margin
		require.InDeltaf(t, validator1TotalReward, validator2TotalReward, validator1TotalReward*0.05, "Validator 1 and Validator 2 rewards are not equal")
		// check if validator 1 and validator 2 delegates got the same amount of reward with 5% error margin
		require.InDeltaf(t, validator1DelegatesTotalReward, validator2DelegatesTotalReward, validator1DelegatesTotalReward*0.05, "Validator 1 and Validator 2 delegate rewards are not equal")

		// check if both blobber delegates got the same amount of reward with 5% error margin
		require.InDeltaf(t, blobber1Delegate1TotalReward, blobber1Delegate2TotalReward, blobber1Delegate1TotalReward*0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not equal")
		require.InDeltaf(t, blobber2Delegate1TotalReward, blobber2Delegate2TotalReward, blobber2Delegate1TotalReward*0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not equal")

		//require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")
		//
		//require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		//require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		//require.InEpsilon(t, validator1TotalReward/validator2TotalReward, 1, 0.05, "Validator 1 and Validator 2 rewards are not equal")
		//require.InEpsilon(t, validator1DelegatesTotalReward/validator2DelegatesTotalReward, 1, 0.05, "Validator 1 and Validator 2 delegate rewards are not equal")
		//
		//require.InEpsilon(t, blobber1Delegate1TotalReward/blobber1Delegate2TotalReward, 1, 0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not equal")
		//require.InEpsilon(t, blobber2Delegate1TotalReward/blobber2Delegate2TotalReward, 1, 0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not equal")
		//require.InEpsilon(t, validator1Delegate1TotalReward/validator1Delegate2TotalReward, 1, 0.05, "Validator 1 Delegate 1 and Validator 1 Delegate 2 rewards are not equal")
		//require.InEpsilon(t, validator2Delegate1TotalReward/validator2Delegate2TotalReward, 1, 0.05, "Validator 2 Delegate 1 and Validator 2 Delegate 2 rewards are not equal")

	})

	t.RunSequentiallyWithTimeout("Case 5 : Client Uploads 10% of Allocation and 2 delegate each (unequal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators

		// Delegate Wallets
		b1D1Wallet, _ := getWalletForName(t, configPath, blobber1Delegate1Wallet)
		b2D1Wallet, _ := getWalletForName(t, configPath, blobber2Delegate1Wallet)
		v1D1Wallet, _ := getWalletForName(t, configPath, validator1Delegate1Wallet)
		v1D2Wallet, _ := getWalletForName(t, configPath, validator1Delegate2Wallet)
		v2D1Wallet, _ := getWalletForName(t, configPath, validator2Delegate1Wallet)

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1, 2, 2, 2, 2,
		}, 2)

		// Creating Allocation

		output := setupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

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

		// sleep for 10 minutes
		time.Sleep(10 * time.Minute)

		allocation := getAllocation(t, allocationId)

		fmt.Println(allocation.MovedToChallenge)

		challenges, _ := getAllChallenges(t, allocationId)

		totalExpectedReward := float64(allocation.MovedToChallenge)

		totalReward := 0.0
		blobber1TotalReward := 0.0
		blobber2TotalReward := 0.0
		blobber1DelegatesTotalReward := 0.0
		blobber2DelegatesTotalReward := 0.0
		validator1TotalReward := 0.0
		validator2TotalReward := 0.0
		validator1DelegatesTotalReward := 0.0
		validator2DelegatesTotalReward := 0.0

		blobber1Delegate1TotalReward := 0.0
		blobber1Delegate2TotalReward := 0.0
		blobber2Delegate1TotalReward := 0.0
		blobber2Delegate2TotalReward := 0.0
		validator1Delegate1TotalReward := 0.0
		validator1Delegate2TotalReward := 0.0
		validator2Delegate1TotalReward := 0.0
		validator2Delegate2TotalReward := 0.0

		for _, challenge := range challenges {
			var isBlobber1 bool
			if challenge.BlobberID == blobberListString[0] {
				isBlobber1 = true
			}

			blobberReward := 0.0
			blobberDelegate1Reward := 0.0
			blobberDelegate2Reward := 0.0
			blobberTotalReward := 0.0

			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1Delegate1Reward := 0.0
			validator1Delegate2Reward := 0.0
			validator2Delegate1Reward := 0.0
			validator2Delegate2Reward := 0.0
			validatorsTotalReward := 0.0

			totalChallengeReward := 0.0

			challengeRewards, err := getChallengeRewards(t, challenge.ChallengeID)
			if err != nil {
				fmt.Println(err)
			}
			require.Nil(t, err, "Error getting challenge rewards", strings.Join(output, "\n"))

			// check if challengeReward.BlobberRewards is empty and if yes continue
			if len(challengeRewards.BlobberRewards) == 0 {
				continue
			}

			blobberChallengeRewards := challengeRewards.BlobberRewards
			validatorChallengeRewards := challengeRewards.ValidatorRewards
			blobberDelegatesChallengeRewards := challengeRewards.BlobberDelegateRewards
			validatorDelegatesChallengeRewards := challengeRewards.ValidatorDelegateRewards

			// parsing data for blobber challenge rewards
			for _, blobberChallengeReward := range blobberChallengeRewards {
				blobberReward += blobberChallengeReward.Amount

				if isBlobber1 {
					blobber1TotalReward += blobberChallengeReward.Amount
				} else {
					blobber2TotalReward += blobberChallengeReward.Amount
				}
			}

			// parsing data for blobber delegate rewards
			for _, blobberDelegateChallengeReward := range blobberDelegatesChallengeRewards {

				if isBlobber1 {
					if blobberDelegateChallengeReward.PoolId == b1D1Wallet.ClientID {
						blobberDelegate1Reward += blobberDelegateChallengeReward.Amount
						blobber1Delegate1TotalReward += blobberDelegateChallengeReward.Amount
					} else {
						blobberDelegate2Reward += blobberDelegateChallengeReward.Amount
						blobber1Delegate2TotalReward += blobberDelegateChallengeReward.Amount
					}

					blobber1DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				} else {
					if blobberDelegateChallengeReward.PoolId == b2D1Wallet.ClientID {
						blobberDelegate1Reward += blobberDelegateChallengeReward.Amount
						blobber2Delegate1TotalReward += blobberDelegateChallengeReward.Amount
					} else {
						blobberDelegate2Reward += blobberDelegateChallengeReward.Amount
						blobber2Delegate2TotalReward += blobberDelegateChallengeReward.Amount
					}

					blobber2DelegatesTotalReward += blobberDelegateChallengeReward.Amount
				}
			}

			blobberTotalReward = blobberReward + blobberDelegate1Reward + blobberDelegate2Reward

			// parsing data for validator challenge rewards
			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
					validator1TotalReward += validatorChallengeReward.Amount
				} else {
					validator2Reward += validatorChallengeReward.Amount
					validator2TotalReward += validatorChallengeReward.Amount
				}
			}

			// parsing data for validator delegate rewards
			for _, validatorDelegateChallengeReward := range validatorDelegatesChallengeRewards {
				if validatorDelegateChallengeReward.PoolId == v1D1Wallet.ClientID {
					validator1Delegate1Reward += validatorDelegateChallengeReward.Amount
					validator1Delegate1TotalReward += validatorDelegateChallengeReward.Amount
					validator1DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.PoolId == v1D2Wallet.ClientID {
					validator1Delegate2Reward += validatorDelegateChallengeReward.Amount
					validator1Delegate2TotalReward += validatorDelegateChallengeReward.Amount
					validator1DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.PoolId == v2D1Wallet.ClientID {
					validator2Delegate1Reward += validatorDelegateChallengeReward.Amount
					validator2Delegate1TotalReward += validatorDelegateChallengeReward.Amount
					validator2DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				} else {
					validator2Delegate2Reward += validatorDelegateChallengeReward.Amount
					validator2Delegate2TotalReward += validatorDelegateChallengeReward.Amount
					validator2DelegatesTotalReward += validatorDelegateChallengeReward.Amount
				}
			}

			validatorsTotalReward = validator1Reward + validator2Reward + validator1Delegate1Reward + validator1Delegate2Reward + validator2Delegate1Reward + validator2Delegate2Reward

			totalChallengeReward = blobberTotalReward + validatorsTotalReward

			// check if blobber got 97.5% of the total challenge reward with 5% error margin
			require.InDelta(t, blobberTotalReward, totalChallengeReward*0.975, totalChallengeReward*0.05, "Blobber Reward is not 97.5% of total challenge reward")
			// check if validators got 2.5% of the total challenge reward with 5% error margin
			require.InDelta(t, validatorsTotalReward, totalChallengeReward*0.025, totalChallengeReward*0.05, "Validators Reward is not 2.5% of total challenge reward")

			// check if blobber delegates got the same amount of reward with 5% error margin
			//require.InDelta(t, blobberDelegate1Reward, blobberDelegate2Reward, blobberDelegate1Reward*0.05, "Blobber delegates reward is not correct", strings.Join(output, "\n"))
			require.LessOrEqual(t, math.Abs(blobberDelegate1Reward-blobberDelegate2Reward), float64(3), "Blobber Delegate rewards are not equal")

			//// check both validators got the same amount of reward with 5% error margin
			//require.InDelta(t, validator1Reward, validator2Reward, validator1Reward*0.05, "Validators reward is not correct", strings.Join(output, "\n"))
			//// check both validators delegates got the same amount of reward with 5% error margin
			//require.InDelta(t, validator1Delegate1Reward, validator1Delegate2Reward, validator1Delegate1Reward*0.05, "Validators reward is not correct", strings.Join(output, "\n"))
			//require.InDelta(t, validator2Delegate1Reward, validator2Delegate2Reward, validator2Delegate1Reward*0.05, "Validators reward is not correct", strings.Join(output, "\n"))

			// check if both validators got the same amount of reward with 5% error margin
			require.LessOrEqual(t, math.Abs((validator1Reward+validator1Delegate1Reward+validator1Delegate2Reward)*2-(validator2Reward-validator2Delegate1Reward-validator2Delegate2Reward)), float64(3), "Validators rewards are not in correct ratio")
			// check if both validators delegates got the same amount of reward with 5% error margin
			require.LessOrEqual(t, math.Abs(validator1Delegate1Reward*2-validator1Delegate2Reward), float64(3), "Validators Delegate rewards are not in correct ratio")
			require.LessOrEqual(t, math.Abs(validator2Delegate1Reward*2-validator2Delegate2Reward), float64(3), "Validators Delegate rewards are not in correct ratio")

		}

		totalReward = blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		fmt.Println("Total Reward: ", totalReward)
		fmt.Println("Total Expected Reward: ", totalExpectedReward)
		fmt.Println("Blobber 1 Total Reward: ", blobber1TotalReward)
		fmt.Println("Blobber 2 Total Reward: ", blobber2TotalReward)
		fmt.Println("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		fmt.Println("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		fmt.Println("Validator 1 Total Reward: ", validator1TotalReward)
		fmt.Println("Validator 2 Total Reward: ", validator2TotalReward)
		fmt.Println("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		fmt.Println("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		fmt.Println("Blobber 1 Delegate 1 Total Reward: ", blobber1Delegate1TotalReward)
		fmt.Println("Blobber 1 Delegate 2 Total Reward: ", blobber1Delegate2TotalReward)
		fmt.Println("Blobber 2 Delegate 1 Total Reward: ", blobber2Delegate1TotalReward)
		fmt.Println("Blobber 2 Delegate 2 Total Reward: ", blobber2Delegate2TotalReward)
		fmt.Println("Validator 1 Delegate 1 Total Reward: ", validator1Delegate1TotalReward)
		fmt.Println("Validator 1 Delegate 2 Total Reward: ", validator1Delegate2TotalReward)
		fmt.Println("Validator 2 Delegate 1 Total Reward: ", validator2Delegate1TotalReward)
		fmt.Println("Validator 2 Delegate 2 Total Reward: ", validator2Delegate2TotalReward)

		// check if total reward is equal to expected reward with 5% error margin
		require.InDeltaf(t, totalReward, totalExpectedReward, totalExpectedReward*0.05, "Total Reward is not equal to expected reward")
		// check if blobber 1 and blobber 2 got the same amount of reward with 5% error margin
		require.InDeltaf(t, blobber1TotalReward, blobber2TotalReward, blobber1TotalReward*0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		// check if blobber 1 and blobber 2 delegates got the same amount of reward with 5% error margin
		require.InDeltaf(t, blobber1DelegatesTotalReward, blobber2DelegatesTotalReward, blobber1DelegatesTotalReward*0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		// check if validator 1 and validator 2 got the same amount of reward with 5% error margin
		require.InDeltaf(t, validator1TotalReward, validator2TotalReward, validator1TotalReward*0.05, "Validator 1 and Validator 2 rewards are not equal")
		// check if validator 1 and validator 2 delegates got the same amount of reward with 5% error margin
		require.InDeltaf(t, validator1DelegatesTotalReward, validator2DelegatesTotalReward, validator1DelegatesTotalReward*0.05, "Validator 1 and Validator 2 delegate rewards are not equal")

		// check if 2nd blobber delegate getting twice the reward of 1st blobber delegate
		require.InDeltaf(t, blobber1Delegate1TotalReward*2, blobber1Delegate2TotalReward, blobber1Delegate2TotalReward*0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not in correct ratio")
		require.InDeltaf(t, blobber2Delegate1TotalReward*2, blobber2Delegate2TotalReward, blobber2Delegate2TotalReward*0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not in correct ratio")

		//require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")
		//
		//require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		//require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		//require.InEpsilon(t, validator1TotalReward/validator2TotalReward, 1, 0.05, "Validator 1 and Validator 2 rewards are not equal")
		//require.InEpsilon(t, validator1DelegatesTotalReward/validator2DelegatesTotalReward, 1, 0.05, "Validator 1 and Validator 2 delegate rewards are not equal")
		//
		//require.InEpsilon(t, blobber1Delegate1TotalReward/blobber1Delegate2TotalReward, 1, 0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not equal")
		//require.InEpsilon(t, blobber2Delegate1TotalReward/blobber2Delegate2TotalReward, 1, 0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not equal")
		//require.InEpsilon(t, validator1Delegate1TotalReward/validator1Delegate2TotalReward, 1, 0.05, "Validator 1 Delegate 1 and Validator 1 Delegate 2 rewards are not equal")
		//require.InEpsilon(t, validator2Delegate1TotalReward/validator2Delegate2TotalReward, 1, 0.05, "Validator 2 Delegate 1 and Validator 2 Delegate 2 rewards are not equal")

	})

}

// TODO : teardown function

func stakeTokensToBlobbersAndValidators(t *test.SystemTest, blobbers []climodel.BlobberInfo, validators []climodel.Validator, configPath string, tokens []float64, numDelegates int) {
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
			_, err := executeFaucetWithTokensForWallet(t, blobberDelegates[idx], configPath, tokens[tIdx]+5)
			if err != nil {
				fmt.Println(err)
				return
			}

			// stake tokens
			_, err = stakeTokensForWallet(t, configPath, blobberDelegates[idx], createParams(map[string]interface{}{
				"blobber_id": blobber.Id,
				"tokens":     tokens[tIdx],
			}), true)
			if err != nil {
				fmt.Println(err)
				return
			}

			idx++
			tIdx++
		}
	}

	idx = 0

	for i := 0; i < numDelegates; i++ {
		for _, validator := range validators {
			// add balance to delegate wallet
			_, err := executeFaucetWithTokensForWallet(t, validatorDelegates[idx], configPath, tokens[tIdx]+5)
			if err != nil {
				fmt.Println(err)
				return
			}

			// stake tokens
			_, err = stakeTokensForWallet(t, configPath, validatorDelegates[idx], createParams(map[string]interface{}{
				"validator_id": validator.ID,
				"tokens":       tokens[tIdx],
			}), true)
			if err != nil {
				fmt.Println(err)
				return
			}

			idx++
			tIdx++
		}
	}
}

func unstakeTokensForBlobbersAndValidators(t *test.SystemTest, blobbers []climodel.BlobberInfo, validators []climodel.Validator, configPath string, numDelegates int, options ...bool) {
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
			// unstake tokens
			_, err := unstakeTokensForWallet(t, configPath, blobberDelegates[idx], createParams(map[string]interface{}{
				"blobber_id": blobber.Id,
			}))
			if err != nil {
				fmt.Println(err)
				return
			}

			idx++
		}
	}

	idx = 0

	for i := 0; i < numDelegates; i++ {

		for _, validator := range validators {
			// unstake tokens
			_, err := unstakeTokensForWallet(t, configPath, validatorDelegates[idx], createParams(map[string]interface{}{
				"validator_id": validator.ID,
			}))
			if err != nil {
				fmt.Println(err)
				return
			}

			idx++
		}
	}
}

func getAllChallenges(t *test.SystemTest, allocationID string) ([]Challenge, error) {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := getSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/all-challenges?allocation_id=" + allocationID)

	var result []Challenge

	res, _ := http.Get(url)

	fmt.Println(res.Body)

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

type Challenge struct {
	ChallengeID    string           `json:"challenge_id"`
	CreatedAt      common.Timestamp `json:"created_at"`
	AllocationID   string           `json:"allocation_id"`
	BlobberID      string           `json:"blobber_id"`
	ValidatorsID   string           `json:"validators_id"`
	Seed           int64            `json:"seed"`
	AllocationRoot string           `json:"allocation_root"`
	Responded      bool             `json:"responded"`
	Passed         bool             `json:"passed"`
	RoundResponded int64            `json:"round_responded"`
	ExpiredN       int              `json:"expired_n"`
}

func getChallengeRewards(t *test.SystemTest, challengeID string) (*ChallengeRewards, error) {

	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := getSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/challenge-rewards?challenge_id=" + challengeID)

	var result *ChallengeRewards

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

type ChallengeRewards struct {
	BlobberDelegateRewards []struct {
		ID          int       `json:"ID"`
		CreatedAt   time.Time `json:"CreatedAt"`
		UpdatedAt   time.Time `json:"UpdatedAt"`
		Amount      float64   `json:"amount"`
		BlockNumber int       `json:"block_number"`
		PoolId      string    `json:"pool_id"`
		ProviderId  string    `json:"provider_id"`
		RewardType  int       `json:"reward_type"`
		ChallengeId string    `json:"challenge_id"`
	} `json:"blobber_delegate_rewards"`
	BlobberRewards []struct {
		ID          int       `json:"ID"`
		CreatedAt   time.Time `json:"CreatedAt"`
		UpdatedAt   time.Time `json:"UpdatedAt"`
		Amount      float64   `json:"amount"`
		BlockNumber int       `json:"block_number"`
		ProviderId  string    `json:"provider_id"`
		RewardType  int       `json:"reward_type"`
		ChallengeId string    `json:"challenge_id"`
	} `json:"blobber_rewards"`
	ValidatorDelegateRewards []struct {
		ID          int       `json:"ID"`
		CreatedAt   time.Time `json:"CreatedAt"`
		UpdatedAt   time.Time `json:"UpdatedAt"`
		Amount      float64   `json:"amount"`
		BlockNumber int       `json:"block_number"`
		PoolId      string    `json:"pool_id"`
		ProviderId  string    `json:"provider_id"`
		RewardType  int       `json:"reward_type"`
		ChallengeId string    `json:"challenge_id"`
	} `json:"validator_delegate_rewards"`
	ValidatorRewards []struct {
		ID          int       `json:"ID"`
		CreatedAt   time.Time `json:"CreatedAt"`
		UpdatedAt   time.Time `json:"UpdatedAt"`
		Amount      float64   `json:"amount"`
		BlockNumber int       `json:"block_number"`
		ProviderId  string    `json:"provider_id"`
		RewardType  int       `json:"reward_type"`
		ChallengeId string    `json:"challenge_id"`
	} `json:"validator_rewards"`
}
