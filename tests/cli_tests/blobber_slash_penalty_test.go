package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlobberSlashPenalty(testSetup *testing.T) {
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

	t.RunSequentiallyWithTimeout("Test Cancel Allocation After Expiry Rewards when client uploads 10% of allocation", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// check allocation remaining time
		allocation := getAllocation(t, allocationId)
		remainingTime := allocation.ExpirationDate - time.Now().Unix()

		// sleep for half of the remaining time
		time.Sleep(time.Duration(remainingTime/2) * time.Second)

		// 2. Kill a blobber
		_, err = killBlobber(t, configPath, createParams(map[string]interface{}{
			"id": blobberList[1].Id,
		}), true)
		require.Nil(t, err, "error killing blobber", strings.Join(output, "\n"))

		// 3. Sleep for the remaining time
		time.Sleep(time.Duration(remainingTime/2) * time.Second)

		allocation = getAllocation(t, allocationId)

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

			fmt.Println("Challenge ID: ", challenge.ChallengeID)
			fmt.Println("Blobber Reward: ", blobberReward)
			fmt.Println("Blobber Delegate Reward: ", blobberDelegateReward)
			fmt.Println("Validator 1 Reward: ", validator1Reward)
			fmt.Println("Validator 2 Reward: ", validator2Reward)
			fmt.Println("Validator 1 Delegate Reward: ", validator1DelegateReward)
			fmt.Println("Validator 2 Delegate Reward: ", validator2DelegateReward)
			fmt.Println("Total Challenge Reward: ", totalChallengeReward)
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

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

}

func killBlobber(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("kill blobber...")
	cmd := fmt.Sprintf("./zbox kill-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, scOwnerWallet, cliConfigFilename)

	fmt.Println(cmd)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getTotalChallengeRewardByProviderID(t *test.SystemTest, providerID string) int {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := getSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/transactions")
	res, _ := http.Get(url)
	body, _ := io.ReadAll(res.Body)

	var response map[string]interface{}

	if err := json.Unmarshal(body, &response); err != nil {
		panic(err)
	}

	return response["sum"].(int)
}
