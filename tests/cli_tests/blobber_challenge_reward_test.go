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

	//t.RunWithTimeout("Test Staking", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
	//	stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, false)
	//	//unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath)
	//
	//	fmt.Println(configPath)
	//
	//	output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
	//		"blobber_id": blobberList[0].Id,
	//		"json":       "",
	//	}))
	//
	//	fmt.Println(output)
	//
	//})
	//
	//t.Skip()

	t.RunWithTimeout("Read All Challenge Data", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		t.Skip()

		allocationId := "495b5fbab854e31d46c713d2f4d849bc7ab994f590c31a9298f94a238d6b2d67"

		var allocation climodel.Allocation

		//allocation = getAllocation(t, allocationId)
		//
		//fmt.Println(allocation.MovedToChallenge)

		allocation.MovedToChallenge = 54340

		challenges, _ := getAllChallenges(allocationId)

		totalExpectedReward := allocation.MovedToChallenge

		totalReward := 0.0
		blobber1TotalReward := 0.0
		blobber2TotalReward := 0.0
		blobber1DelegatesTotalReward := 0.0
		blobber2DelegatesTotalReward := 0.0
		validator1TotalReward := 0.0
		validator2TotalReward := 0.0
		validator1DelegatesTotalReward := 0.0
		validator2DelegatesTotalReward := 0.0

		count := 0

		for _, challenge := range challenges {
			fmt.Println(count)
			count++
			blobber1Reward := 0.0
			blobber2Reward := 0.0
			blobber1DelegatesReward := 0.0
			blobber2DelegatesReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegatesReward := 0.0
			validator2DelegatesReward := 0.0

			challengeRewards, err := getChallengeRewards(challenge.ChallengeID)

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

			if blobberChallengeRewards[0].ProviderId == blobberListString[0] {
				blobber1Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber1DelegatesReward += blobberDelegateChallengeReward.Amount
				}

			} else if blobberChallengeRewards[0].ProviderId == blobberListString[1] {
				blobber2Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber2DelegatesReward += blobberDelegateChallengeReward.Amount
				}
			}

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorListString[1] {
					validator2Reward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorListString[0] {
					validator1DelegatesReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorListString[1] {
					validator2DelegatesReward += validatorDelegateChallengeReward.Amount
				}
			}

			totalBlobberReward := blobber1Reward + blobber2Reward + blobber1DelegatesReward + blobber2DelegatesReward
			totalValidatorReward := validator1Reward + validator2Reward + validator1DelegatesReward + validator2DelegatesReward

			//fmt.Println("Total Blobber Reward: ", totalBlobberReward)
			//fmt.Println("Total Validator Reward: ", totalValidatorReward)
			fmt.Println(validator1DelegatesReward, validator2DelegatesReward)

			// check if the ratio is correct between blobber and validator rewards
			require.InEpsilon(t, totalBlobberReward*0.025, totalValidatorReward, 1, "Blobber Validator Rewards ratio is not 2.5%")
			// check if both validators have same rewards
			require.InEpsilon(t, validator1Reward+1, validator2Reward+1, (validator1Reward+validator2Reward+2)*0.05, "Validator 1 and Validator 2 rewards are not equal")
			//require.InEpsilon(t, validator1DelegatesReward+1, validator2DelegatesReward+1, (validator1DelegatesReward+validator2DelegatesReward+2)*0.05, "Validator 1 Delegate and Validator 2 Delegate rewards are not equal")

			blobber1TotalReward += blobber1Reward
			blobber2TotalReward += blobber2Reward
			blobber1DelegatesTotalReward += blobber1DelegatesReward
			blobber2DelegatesTotalReward += blobber2DelegatesReward
			validator1TotalReward += validator1Reward
			validator2TotalReward += validator2Reward
			validator1DelegatesTotalReward += validator1DelegatesReward
			validator2DelegatesTotalReward += validator2DelegatesReward
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

		//require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, (blobber1TotalReward+blobber1DelegatesTotalReward)/(blobber2TotalReward+blobber2DelegatesTotalReward), 1, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")
		require.InEpsilon(t, (validator1TotalReward+validator1DelegatesTotalReward)/(validator2TotalReward+validator2DelegatesTotalReward), 1, 0.05, "Validator 1 Total and Validator 2 Total rewards are not equal")
	})

	//t.Skip()

	t.RunWithTimeout("Read all data with multiple delegates", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		t.Skip()

		allocationId := "3bd18f410140468ba9395c00c48b57b1352b6448d802a989c1ec6a3c9c640252"

		//allocation := getAllocation(t, allocationId)
		//
		//fmt.Println(allocation.MovedToChallenge)

		challenges, _ := getAllChallenges(allocationId)

		//totalExpectedReward := float64(allocation.MovedToChallenge)

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
			blobber1Reward := 0.0
			blobber2Reward := 0.0
			blobber1DelegatesReward := 0.0
			blobber2DelegatesReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegatesReward := 0.0
			validator2DelegatesReward := 0.0

			challengeRewards, err := getChallengeRewards(challenge.ChallengeID)

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

			if blobberChallengeRewards[0].ProviderId == blobberListString[0] {
				blobber1Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber1DelegatesReward += blobberDelegateChallengeReward.Amount
				}

			} else if blobberChallengeRewards[0].ProviderId == blobberListString[1] {
				blobber2Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber2DelegatesReward += blobberDelegateChallengeReward.Amount
				}
			}

			// blobber's both delegates should get equal rewards
			blobber1Delegate1Reward := blobberDelegateChallengeRewards[0].Amount
			blobber1Delegate2Reward := blobberDelegateChallengeRewards[1].Amount
			fmt.Println("BB : ", blobber1Reward, blobber2Reward)
			fmt.Println("B : ", blobber1Delegate1Reward, blobber1Delegate2Reward)
			//require.InEpsilon(t, blobber1Delegate1Reward+1, blobber1Delegate2Reward+1, 1, "Blobber 1 and Blobber 2 delegate rewards are not equal")

			// validator's both delegates should get equal rewards
			validator1Delegate1Reward := validatorDelegateChallengeRewards[0].Amount
			validator1Delegate2Reward := validatorDelegateChallengeRewards[1].Amount
			validator2Delegate1Reward := validatorDelegateChallengeRewards[2].Amount
			validator2Delegate2Reward := validatorDelegateChallengeRewards[3].Amount

			fmt.Println("VV : ", validator1Reward, validator2Reward)
			fmt.Println("V : ", validator1Delegate1Reward, validator1Delegate2Reward, validator2Delegate1Reward, validator2Delegate2Reward)
			//require.InEpsilon(t, validator1Delegate1Reward+1, validator1Delegate2Reward+1, 1, "Validator 1 and Validator 2 delegate rewards are not equal")

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorListString[1] {
					validator2Reward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorListString[0] {
					validator1DelegatesReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorListString[1] {
					validator2DelegatesReward += validatorDelegateChallengeReward.Amount
				}
			}

			totalBlobberReward := blobber1Reward + blobber2Reward + blobber1DelegatesReward + blobber2DelegatesReward
			totalValidatorReward := validator1Reward + validator2Reward + validator1DelegatesReward + validator2DelegatesReward

			//fmt.Println("Total Blobber Reward: ", totalBlobberReward)
			//fmt.Println("Total Validator Reward: ", totalValidatorReward)
			//fmt.Println(validator1DelegatesReward, validator2DelegatesReward)

			// check if the ratio is correct between blobber and validator rewards
			require.InEpsilon(t, totalBlobberReward*0.025, totalValidatorReward, 1, "Blobber Validator Rewards ratio is not 2.5%")
			// check if both validators have same rewards
			require.InEpsilon(t, validator1Reward+1, validator2Reward+1, 1, "Validator 1 and Validator 2 rewards are not equal")
			require.InEpsilon(t, validator1DelegatesReward+1, validator2DelegatesReward+1, 1, "Validator 1 Delegate and Validator 2 Delegate rewards are not equal")

			blobber1TotalReward += blobber1Reward
			blobber2TotalReward += blobber2Reward
			blobber1DelegatesTotalReward += blobber1DelegatesReward
			blobber2DelegatesTotalReward += blobber2DelegatesReward
			validator1TotalReward += validator1Reward
			validator2TotalReward += validator2Reward
			validator1DelegatesTotalReward += validator1DelegatesReward
			validator2DelegatesTotalReward += validator2DelegatesReward
		}

		totalReward = blobber1TotalReward + blobber2TotalReward + blobber1DelegatesTotalReward + blobber2DelegatesTotalReward + validator1TotalReward + validator2TotalReward + validator1DelegatesTotalReward + validator2DelegatesTotalReward

		fmt.Println("Total Reward: ", totalReward)
		//fmt.Println("Total Expected Reward: ", totalExpectedReward)
		fmt.Println("Blobber 1 Total Reward: ", blobber1TotalReward)
		fmt.Println("Blobber 2 Total Reward: ", blobber2TotalReward)
		fmt.Println("Blobber 1 Delegates Total Reward: ", blobber1DelegatesTotalReward)
		fmt.Println("Blobber 2 Delegates Total Reward: ", blobber2DelegatesTotalReward)
		fmt.Println("Validator 1 Total Reward: ", validator1TotalReward)
		fmt.Println("Validator 2 Total Reward: ", validator2TotalReward)
		fmt.Println("Validator 1 Delegates Total Reward: ", validator1DelegatesTotalReward)
		fmt.Println("Validator 2 Delegates Total Reward: ", validator2DelegatesTotalReward)

		//require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")

		require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		require.InEpsilon(t, (blobber1TotalReward+blobber1DelegatesTotalReward)/(blobber2TotalReward+blobber2DelegatesTotalReward), 1, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")
		require.InEpsilon(t, (validator1TotalReward+validator1DelegatesTotalReward)/(validator2TotalReward+validator2DelegatesTotalReward), 1, 0.05, "Validator 1 Total and Validator 2 Total rewards are not equal")
	})

	//t.Skip()

	t.RunSequentiallyWithTimeout("Case 1 : Client Uploads 10% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		//t.Skip()
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)

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

		challenges, _ := getAllChallenges(allocationId)

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
			blobber1Reward := 0.0
			blobber2Reward := 0.0
			blobber1DelegatesReward := 0.0
			blobber2DelegatesReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegatesReward := 0.0
			validator2DelegatesReward := 0.0

			challengeRewards, err := getChallengeRewards(challenge.ChallengeID)

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

			if blobberChallengeRewards[0].ProviderId == blobberListString[0] {
				blobber1Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber1DelegatesReward += blobberDelegateChallengeReward.Amount
				}

			} else if blobberChallengeRewards[0].ProviderId == blobberListString[1] {
				blobber2Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber2DelegatesReward += blobberDelegateChallengeReward.Amount
				}
			}

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorListString[1] {
					validator2Reward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorListString[0] {
					validator1DelegatesReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorListString[1] {
					validator2DelegatesReward += validatorDelegateChallengeReward.Amount
				}
			}

			totalBlobberReward := blobber1Reward + blobber2Reward + blobber1DelegatesReward + blobber2DelegatesReward
			totalValidatorReward := validator1Reward + validator2Reward + validator1DelegatesReward + validator2DelegatesReward
			totalValidator1Reward := validator1Reward + validator1DelegatesReward
			//totalValidator2Reward := validator2Reward + validator2DelegatesReward

			//fmt.Println("Total Blobber Reward: ", totalBlobberReward)
			//fmt.Println("Total Validator Reward: ", totalValidatorReward)
			//fmt.Println(validator1DelegatesReward, validator2DelegatesReward)

			// check if the ratio is correct between blobber and validator rewards
			require.InEpsilon(t, totalBlobberReward*0.025, totalValidatorReward, 1, "Blobber Validator Rewards ratio is not 2.5%")
			// check if both validators have same rewards
			if totalValidatorReward != 0 {
				require.InEpsilon(t, (totalValidator1Reward*2)/totalValidatorReward, 1, 0.05, "Validator 1 and Validator 2 rewards are not equal")
				require.LessOrEqual(t, math.Abs(validator2DelegatesReward-validator1DelegatesReward), float64(3), "Validator 1 reward is not less than Validator 2 reward")
			}

			blobber1TotalReward += blobber1Reward
			blobber2TotalReward += blobber2Reward
			blobber1DelegatesTotalReward += blobber1DelegatesReward
			blobber2DelegatesTotalReward += blobber2DelegatesReward
			validator1TotalReward += validator1Reward
			validator2TotalReward += validator2Reward
			validator1DelegatesTotalReward += validator1DelegatesReward
			validator2DelegatesTotalReward += validator2DelegatesReward
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
		require.InEpsilon(t, (validator1TotalReward+validator1DelegatesTotalReward)/(validator2TotalReward+validator2DelegatesTotalReward), 1, 0.05, "Validator 1 Total and Validator 2 Total rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath)
	})

	//t.Skip()

	t.RunSequentiallyWithTimeout("Case 2 : Client Uploads 30% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		//t.Skip()
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)

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

		challenges, _ := getAllChallenges(allocationId)

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
			blobber1Reward := 0.0
			blobber2Reward := 0.0
			blobber1DelegatesReward := 0.0
			blobber2DelegatesReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegatesReward := 0.0
			validator2DelegatesReward := 0.0

			challengeRewards, err := getChallengeRewards(challenge.ChallengeID)

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

			if blobberChallengeRewards[0].ProviderId == blobberListString[0] {
				blobber1Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber1DelegatesReward += blobberDelegateChallengeReward.Amount
				}

			} else if blobberChallengeRewards[0].ProviderId == blobberListString[1] {
				blobber2Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber2DelegatesReward += blobberDelegateChallengeReward.Amount
				}
			}

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorListString[1] {
					validator2Reward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorListString[0] {
					validator1DelegatesReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorListString[1] {
					validator2DelegatesReward += validatorDelegateChallengeReward.Amount
				}
			}

			totalBlobberReward := blobber1Reward + blobber2Reward + blobber1DelegatesReward + blobber2DelegatesReward
			totalValidatorReward := validator1Reward + validator2Reward + validator1DelegatesReward + validator2DelegatesReward
			totalValidator1Reward := validator1Reward + validator1DelegatesReward
			//totalValidator2Reward := validator2Reward + validator2DelegatesReward

			//fmt.Println("Total Blobber Reward: ", totalBlobberReward)
			//fmt.Println("Total Validator Reward: ", totalValidatorReward)
			//fmt.Println(validator1DelegatesReward, validator2DelegatesReward)

			// check if the ratio is correct between blobber and validator rewards
			require.InEpsilon(t, totalBlobberReward*0.025, totalValidatorReward, 1, "Blobber Validator Rewards ratio is not 2.5%")
			// check if both validators have same rewards
			if totalValidatorReward != 0 {
				require.InEpsilon(t, (totalValidator1Reward*2)/totalValidatorReward, 1, 0.05, "Validator 1 and Validator 2 rewards are not equal")
				require.LessOrEqual(t, math.Abs(validator2DelegatesReward-validator1DelegatesReward), float64(3), "Validator 1 reward is not less than Validator 2 reward")
			}

			blobber1TotalReward += blobber1Reward
			blobber2TotalReward += blobber2Reward
			blobber1DelegatesTotalReward += blobber1DelegatesReward
			blobber2DelegatesTotalReward += blobber2DelegatesReward
			validator1TotalReward += validator1Reward
			validator2TotalReward += validator2Reward
			validator1DelegatesTotalReward += validator1DelegatesReward
			validator2DelegatesTotalReward += validator2DelegatesReward
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
		require.InEpsilon(t, (validator1TotalReward+validator1DelegatesTotalReward)/(validator2TotalReward+validator2DelegatesTotalReward), 1, 0.05, "Validator 1 Total and Validator 2 Total rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath)
	})

	t.RunSequentiallyWithTimeout("Case 3 : Client Uploads 10% of Allocation and 1 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, false)

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

		challenges, _ := getAllChallenges(allocationId)

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
			blobber1Reward := 0.0
			blobber2Reward := 0.0
			blobber1DelegatesReward := 0.0
			blobber2DelegatesReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegatesReward := 0.0
			validator2DelegatesReward := 0.0

			challengeRewards, err := getChallengeRewards(challenge.ChallengeID)

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

			if blobberChallengeRewards[0].ProviderId == blobberListString[0] {
				blobber1Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber1DelegatesReward += blobberDelegateChallengeReward.Amount
				}

			} else if blobberChallengeRewards[0].ProviderId == blobberListString[1] {
				blobber2Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber2DelegatesReward += blobberDelegateChallengeReward.Amount
				}
			}

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorListString[1] {
					validator2Reward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorListString[0] {
					validator1DelegatesReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorListString[1] {
					validator2DelegatesReward += validatorDelegateChallengeReward.Amount
				}
			}

			totalBlobberReward := blobber1Reward + blobber2Reward + blobber1DelegatesReward + blobber2DelegatesReward
			totalValidatorReward := validator1Reward + validator2Reward + validator1DelegatesReward + validator2DelegatesReward
			totalValidator1Reward := validator1Reward + validator1DelegatesReward
			//totalValidator2Reward := validator2Reward + validator2DelegatesReward

			//fmt.Println("Total Blobber Reward: ", totalBlobberReward)
			//fmt.Println("Total Validator Reward: ", totalValidatorReward)
			//fmt.Println(validator1DelegatesReward, validator2DelegatesReward)

			// check if the ratio is correct between blobber and validator rewards
			require.InEpsilon(t, totalBlobberReward*0.025, totalValidatorReward, 1, "Blobber Validator Rewards ratio is not 2.5%")
			// check if both validators have same rewards
			if totalValidatorReward != 0 {
				require.InEpsilon(t, (totalValidator1Reward*2)/totalValidatorReward, 1, 0.05, "Validator 1 and Validator 2 rewards are not equal")
				require.LessOrEqual(t, math.Abs(validator2DelegatesReward-validator1DelegatesReward), float64(3), "Validator 1 reward is not less than Validator 2 reward")
			}

			blobber1TotalReward += blobber1Reward
			blobber2TotalReward += blobber2Reward
			blobber1DelegatesTotalReward += blobber1DelegatesReward
			blobber2DelegatesTotalReward += blobber2DelegatesReward
			validator1TotalReward += validator1Reward
			validator2TotalReward += validator2Reward
			validator1DelegatesTotalReward += validator1DelegatesReward
			validator2DelegatesTotalReward += validator2DelegatesReward
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
		require.InEpsilon(t, (validator1TotalReward+validator1DelegatesTotalReward)/(validator2TotalReward+validator2DelegatesTotalReward), 1, 0.05, "Validator 1 Total and Validator 2 Total rewards are not equal")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath)
	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 4 : Client Uploads 10% of Allocation and 2 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, false)
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, false)

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

		challenges, _ := getAllChallenges(allocationId)

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
			blobber1Reward := 0.0
			blobber2Reward := 0.0
			blobber1DelegatesReward := 0.0
			blobber2DelegatesReward := 0.0
			validator1Reward := 0.0
			validator2Reward := 0.0
			validator1DelegatesReward := 0.0
			validator2DelegatesReward := 0.0

			challengeRewards, err := getChallengeRewards(challenge.ChallengeID)

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

			if blobberChallengeRewards[0].ProviderId == blobberListString[0] {
				blobber1Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber1DelegatesReward += blobberDelegateChallengeReward.Amount
				}

			} else if blobberChallengeRewards[0].ProviderId == blobberListString[1] {
				blobber2Reward += blobberChallengeRewards[0].Amount

				for _, blobberDelegateChallengeReward := range blobberDelegateChallengeRewards {
					blobber2DelegatesReward += blobberDelegateChallengeReward.Amount
				}
			}

			// blobber's both delegates should get equal rewards
			blobber1Delegate1Reward := blobberDelegateChallengeRewards[0].Amount
			blobber1Delegate2Reward := blobberDelegateChallengeRewards[1].Amount
			fmt.Println("B : ", blobber1Delegate1Reward, blobber1Delegate2Reward)
			//require.InEpsilon(t, blobber1Delegate1Reward+1, blobber1Delegate2Reward+1, 1, "Blobber 1 and Blobber 2 delegate rewards are not equal")

			// validator's both delegates should get equal rewards
			validator1Delegate1Reward := validatorDelegateChallengeRewards[0].Amount
			validator1Delegate2Reward := validatorDelegateChallengeRewards[1].Amount
			fmt.Println("V : ", validator1Delegate1Reward, validator1Delegate2Reward)
			//require.InEpsilon(t, validator1Delegate1Reward+1, validator1Delegate2Reward+1, 1, "Validator 1 and Validator 2 delegate rewards are not equal")

			for _, validatorChallengeReward := range validatorChallengeRewards {
				if validatorChallengeReward.ProviderId == validatorListString[0] {
					validator1Reward += validatorChallengeReward.Amount
				} else if validatorChallengeReward.ProviderId == validatorListString[1] {
					validator2Reward += validatorChallengeReward.Amount
				}
			}

			for _, validatorDelegateChallengeReward := range validatorDelegateChallengeRewards {
				if validatorDelegateChallengeReward.ProviderId == validatorListString[0] {
					validator1DelegatesReward += validatorDelegateChallengeReward.Amount
				} else if validatorDelegateChallengeReward.ProviderId == validatorListString[1] {
					validator2DelegatesReward += validatorDelegateChallengeReward.Amount
				}
			}

			//totalBlobberReward := blobber1Reward + blobber2Reward + blobber1DelegatesReward + blobber2DelegatesReward
			//totalValidatorReward := validator1Reward + validator2Reward + validator1DelegatesReward + validator2DelegatesReward

			//fmt.Println("Total Blobber Reward: ", totalBlobberReward)
			//fmt.Println("Total Validator Reward: ", totalValidatorReward)
			//fmt.Println(validator1DelegatesReward, validator2DelegatesReward)

			// check if the ratio is correct between blobber and validator rewards
			//require.InEpsilon(t, totalBlobberReward*0.025, totalValidatorReward, 1, "Blobber Validator Rewards ratio is not 2.5%")
			//// check if both validators have same rewards
			//require.InEpsilon(t, validator1Reward+1, validator2Reward+1, 1, "Validator 1 and Validator 2 rewards are not equal")
			//require.InEpsilon(t, validator1DelegatesReward+1, validator2DelegatesReward+1, 1, "Validator 1 Delegate and Validator 2 Delegate rewards are not equal")

			blobber1TotalReward += blobber1Reward
			blobber2TotalReward += blobber2Reward
			blobber1DelegatesTotalReward += blobber1DelegatesReward
			blobber2DelegatesTotalReward += blobber2DelegatesReward
			validator1TotalReward += validator1Reward
			validator2TotalReward += validator2Reward
			validator1DelegatesTotalReward += validator1DelegatesReward
			validator2DelegatesTotalReward += validator2DelegatesReward
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

		//require.InEpsilon(t, totalReward/totalExpectedReward, 1, 0.05, "Total Reward is not equal to expected reward")
		//
		//require.InEpsilon(t, blobber1TotalReward/blobber2TotalReward, 1, 0.05, "Blobber 1 and Blobber 2 rewards are not equal")
		//require.InEpsilon(t, blobber1DelegatesTotalReward/blobber2DelegatesTotalReward, 1, 0.05, "Blobber 1 and Blobber 2 delegate rewards are not equal")
		//require.InEpsilon(t, (blobber1TotalReward+blobber1DelegatesTotalReward)/(blobber2TotalReward+blobber2DelegatesTotalReward), 1, 0.05, "Blobber 1 Total and Blobber 2 Total rewards are not equal")
		//require.InEpsilon(t, (validator1TotalReward+validator1DelegatesTotalReward)/(validator2TotalReward+validator2DelegatesTotalReward), 1, 0.05, "Validator 1 Total and Validator 2 Total rewards are not equal")
	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 5 : Client Uploads 10% of Allocation and 2 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, false)
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, false)

		// Creating Allocation

		output := setupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "50m",
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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		challenges, _ := getAllChallenges(allocationId)

		totalExpectedRewardFor1GBForOneYear := 1000000000
		totalExpectedReward := totalExpectedRewardFor1GBForOneYear * 1 / (365 * 24 * 12 * 10 * 2) // (days * hours * minutes * uploadPercentage * 2)

		totalReward := 0

		var blobberChallengeRewards map[string]int
		blobberChallengeRewards = make(map[string]int)

		var validatorChallengeRewards map[string]int
		validatorChallengeRewards = make(map[string]int)

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			// Check how much reward the blobber is getting and compare it to the reward of the validator
			// it should be in the ratio of blobber vs validator is 0.975:0.025

			blobberChallengeReward := 0
			validatorChallengeReward := 0

			for _, i := range response["rps"].([]interface{}) {
				// check if provider_id in i is in the blobber list or not
				// if it is in the blobber list then the reward should be 0.975 * challengeReward
				// if it is not in the blobber list then the reward should be 0.025 * challengeReward
				providerId := i.(map[string]interface{})["provider_id"].(string)
				providerReward := int(i.(map[string]interface{})["amount"].(float64))

				if contains(blobberListString, providerId) {
					blobberChallengeReward += providerReward

					// if the provider is a blobber then add the reward to the blobberChallengeRewards map
					if _, ok := blobberChallengeRewards[providerId]; ok {
						blobberChallengeRewards[providerId] += providerReward
					} else {
						blobberChallengeRewards[providerId] = providerReward
					}
				} else {
					validatorChallengeReward += providerReward

					// if the provider is a validator then add the reward to the validatorChallengeRewards map
					if _, ok := validatorChallengeRewards[providerId]; ok {
						validatorChallengeRewards[providerId] += providerReward
					} else {
						validatorChallengeRewards[providerId] = providerReward
					}
				}
			}

			require.Equal(t, int(0.975*float64(challengeReward)), blobberChallengeReward, "Blobber reward is not equal to expected reward")
			require.Equal(t, int(0.025*float64(challengeReward)), validatorChallengeReward, "Validator reward is not equal to expected reward")

			totalReward += challengeReward
		}

		require.Equal(t, totalExpectedReward, totalReward, "Total reward is not equal to expected reward")
		require.Equal(t, float64(blobberChallengeRewards[blobberListString[0]])*3.3, float64(blobberChallengeRewards[blobberListString[1]])*6.7, "Blobber 1 and Blobber 2 rewards are not equal")
		require.Equal(t, validatorChallengeRewards[validatorListString[0]], validatorChallengeRewards[validatorListString[1]], "Validator 1 and Validator 2 rewards are not equal")
	})

	t.RunSequentiallyWithTimeout("Case 6 : Client Uploads 10% of Allocation and 1 delegate each (equal stake) with Uploading in starting and in the middle", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)

		// Creating Allocation

		output := setupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "50m",
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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		// Uploading 10% of allocation

		remotepath = "/dir/"
		filesize = 50 * MB
		filename = generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		time.Sleep(2 * time.Minute)

		challenges, _ := getAllChallenges(allocationId)

		totalExpectedRewardFor1GBForOneYear := 1000000000
		totalExpectedReward := totalExpectedRewardFor1GBForOneYear * 1 / (365 * 24 * 12 * 10 * 2) // (days * hours * minutes * uploadPercentage * 2)

		totalReward := 0

		var blobberChallengeRewards map[string]int
		blobberChallengeRewards = make(map[string]int)

		var validatorChallengeRewards map[string]int
		validatorChallengeRewards = make(map[string]int)

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			// Check how much reward the blobber is getting and compare it to the reward of the validator
			// it should be in the ratio of blobber vs validator is 0.975:0.025

			blobberChallengeReward := 0
			validatorChallengeReward := 0

			for _, i := range response["rps"].([]interface{}) {
				// check if provider_id in i is in the blobber list or not
				// if it is in the blobber list then the reward should be 0.975 * challengeReward
				// if it is not in the blobber list then the reward should be 0.025 * challengeReward
				providerId := i.(map[string]interface{})["provider_id"].(string)
				providerReward := int(i.(map[string]interface{})["amount"].(float64))

				if contains(blobberListString, providerId) {
					blobberChallengeReward += providerReward

					// if the provider is a blobber then add the reward to the blobberChallengeRewards map
					if _, ok := blobberChallengeRewards[providerId]; ok {
						blobberChallengeRewards[providerId] += providerReward
					} else {
						blobberChallengeRewards[providerId] = providerReward
					}
				} else {
					validatorChallengeReward += providerReward

					// if the provider is a validator then add the reward to the validatorChallengeRewards map
					if _, ok := validatorChallengeRewards[providerId]; ok {
						validatorChallengeRewards[providerId] += providerReward
					} else {
						validatorChallengeRewards[providerId] = providerReward
					}
				}
			}

			require.Equal(t, int(0.975*float64(challengeReward)), blobberChallengeReward, "Blobber reward is not equal to expected reward")
			require.Equal(t, int(0.025*float64(challengeReward)), validatorChallengeReward, "Validator reward is not equal to expected reward")

			totalReward += challengeReward
		}

		require.Equal(t, totalExpectedReward, totalReward, "Total reward is not equal to expected reward")
		require.Equal(t, blobberChallengeRewards[blobberListString[0]], blobberChallengeRewards[blobberListString[1]], "Blobber 1 and Blobber 2 rewards are not equal")
		require.Equal(t, validatorChallengeRewards[validatorListString[0]], validatorChallengeRewards[validatorListString[1]], "Validator 1 and Validator 2 rewards are not equal")
	})

}

// TODO : teardown function

func stakeTokensToBlobbersAndValidators(t *test.SystemTest, blobbers []climodel.BlobberInfo, validators []climodel.Validator, configPath string, equal bool) {

	count := 1

	for _, blobber := range blobbers {
		setupWalletWithCustomTokens(t, configPath, 9)
		_, err := stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     count,
		}), true)
		if err != nil {
			fmt.Println(err)
			return
		}

		if !equal {
			count++
		}
	}

	count = 1

	for _, validator := range validators {
		setupWalletWithCustomTokens(t, configPath, 9)

		_, err := stakeTokens(t, configPath, createParams(map[string]interface{}{
			"validator_id": validator.ID,
			"tokens":       count,
		}), true)
		if err != nil {
			fmt.Println(err)
		}

		if !equal {
			count++
		}
	}
}

func unstakeTokensForBlobbersAndValidators(t *test.SystemTest, blobbers []climodel.BlobberInfo, validators []climodel.Validator, configPath string) {

	for _, blobber := range blobbers {
		setupWalletWithCustomTokens(t, configPath, 9)
		_, err := unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
		}))
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for _, validator := range validators {
		setupWalletWithCustomTokens(t, configPath, 9)

		_, err := unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"validator_id": validator.ID,
		}))
		if err != nil {
			fmt.Println(err)
		}
	}
}

func getAllChallenges(allocationID string) ([]Challenge, error) {

	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/all-challenges?allocation_id=" + allocationID

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

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func getChallengeRewards(challengeID string) (*ChallengeRewards, error) {

	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/challenge-rewards?challenge_id=" + challengeID

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

	//fmt.Println("Printing Result : ", result)

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
