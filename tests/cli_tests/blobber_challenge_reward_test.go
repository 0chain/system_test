package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
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

	t.RunWithTimeout("Case 1 : Client Uploads 10% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
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

	t.RunWithTimeout("Case 2 : Client Uploads 30% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
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
		filesize := 150 * MB
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
		totalExpectedReward := totalExpectedRewardFor1GBForOneYear * 3 / (365 * 24 * 12 * 10 * 2) // (days * hours * minutes * uploadPercentage * 2)

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

	t.RunWithTimeout("Case 3 : Client Uploads 10% of Allocation and 1 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators
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

	t.RunWithTimeout("Case 4 : Client Uploads 10% of Allocation and 2 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true)
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

		challenges, _ := getAllChallenges(allocationId)

		totalExpectedRewardFor1GBForOneYear := 1000000000
		totalExpectedReward := totalExpectedRewardFor1GBForOneYear * 3 / (365 * 24 * 12 * 10 * 2) // (days * hours * minutes * uploadPercentage * 2)

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

	t.RunWithTimeout("Case 5 : Client Uploads 10% of Allocation and 2 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
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

	t.RunWithTimeout("Case 6 : Client Uploads 10% of Allocation and 1 delegate each (equal stake) with Uploading in starting and in the middle", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
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
