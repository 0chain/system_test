package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAllocation(testSetup *testing.T) {
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

	// Free Allocation
	err = bls.Init(bls.CurveFp254BNb)
	require.NoError(t, err, "Error initializing BLS")

	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	assigner := escapedTestName(t) + "_ASSIGNER"

	// register SC owner wallet
	output, err = registerWalletForName(t, configPath, scOwnerWallet)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	// register assigner wallet
	output, err = registerWalletForName(t, configPath, assigner)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	assignerWallet := readWalletFile(t, "./config/"+assigner+"_wallet.json")

	// necessary cli call to generate wallet to avoid polluting logs of succeeding cli calls
	output, err = registerWallet(t, configPath)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	output, err = getStorageSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	//cfg, _ := keyValuePairStringToMap(output)

	// miners list
	output, err = getMiners(t, configPath)
	require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var miners climodel.NodeList
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
	require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))

	input := map[string]interface{}{
		"name":  assignerWallet.ClientID,
		"key":   assignerWallet.ClientKey,
		"limit": freeTokensIndividualLimit,
		"max":   freeTokensTotalLimit,
	}
	output, err = createFreeStorageAllocation(t, configPath, scOwnerWallet, createParams(input))
	require.NoError(t, err)
	t.Log(output)

	t.RunSequentiallyWithTimeout("Create + Upload + Cancel, equal read price 0.1", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = executeFaucetWithTokens(t, configPath, 9)

		allocSize := 10 * MB

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 2 * MB
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

		_, err = cancelAllocation(t, configPath, allocationId, true)
		if err != nil {
			fmt.Println("Error cancelling allocation", err)
		}

		// sleep for 10 minutes
		time.Sleep(2 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		fmt.Println("curBlock", curBlock)

		allocation := getAllocation(t, allocationId)

		// get all challenges
		challenges, _ := getAllChallenges(t, allocationId)

		fmt.Println(challenges)

		passedChallenges := 0

		for _, challenge := range challenges {
			if challenge.Passed {
				passedChallenges++
			}
			//require.True(t, challenge.Passed != true, "All Challenges should be passed")

			fmt.Println(challenge.CreatedAt, allocation.ExpirationDate)
		}

		failedChallenges := len(challenges) - passedChallenges

		fmt.Println("passedChallenges", passedChallenges)
		fmt.Println("failedChallenges", failedChallenges)

		require.Equal(t, 0, passedChallenges, "All Challenges should fail")

		// Cancellation Rewards
		allocCancellationRewards, err := getAllocationCancellationReward(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList)
		if err != nil {
			return
		}

		blobber1CancellationReward := allocCancellationRewards[0]
		blobber2CancellationReward := allocCancellationRewards[1]

		totalExpectedCancellationReward := sizeInGB(int64(allocSize)*2) * 1000000000 * 0.2

		fmt.Println("totalExpectedCancellationReward", totalExpectedCancellationReward)

		fmt.Println("blobber1CancellationReward", blobber1CancellationReward)
		fmt.Println("blobber2CancellationReward", blobber2CancellationReward)

		require.InEpsilon(t, totalExpectedCancellationReward, float64(blobber1CancellationReward+blobber2CancellationReward), 0.05, "Total Cancellation Reward should be equal to total expected cancellation reward")
		require.InEpsilon(t, blobber1CancellationReward, blobber2CancellationReward, 0.05, "Blobber 1 Cancellation Reward should be equal to total expected cancellation reward")

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Create + Upload + Upgrade, equal read price 0.1", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		recipient := escapedTestName(t)

		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		rawHash, err := hex.DecodeString(data)
		require.Nil(t, err, "failed to decode hex %s", data)
		require.NotNil(t, rawHash, "failed to decode hex %s", data)
		secretKey := crypto.ToSecretKey(t, assignerWallet)
		marker.Signature = crypto.Sign(t, string(rawHash), secretKey)

		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationForWallet(t, recipient, configPath, createParams(map[string]interface{}{"free_storage": markerFile, "size": 10 * MB}))
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")
		allocationId := strings.Fields(output[0])[2]

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 2 * MB
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

		_, err = updateAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"size":       100 * MB,
		}), true)
		if err != nil {
			fmt.Println("Error updating allocation", err)
		}

		// sleep for 6 minutes
		time.Sleep(6 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		fmt.Println("curBlock", curBlock)

		// get all challenges
		challenges, _ := getAllChallenges(t, allocationId)

		passedChallenges := 0

		for _, challenge := range challenges {
			if challenge.Passed {
				passedChallenges++
			}
			//require.True(t, challenge.Passed != true, "All Challenges should be passed")
		}

		failedChallenges := len(challenges) - passedChallenges

		fmt.Println("passedChallenges", passedChallenges)
		fmt.Println("failedChallenges", failedChallenges)

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("External Party Upgrades Allocation", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		t.Skip()
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 2 * MB
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

		//remoteFilepath := remotepath + filepath.Base(filename)
		//
		//output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
		//	"allocation": allocationId,
		//	"remotepath": remoteFilepath,
		//	"localpath":  os.TempDir() + string(os.PathSeparator),
		//}), true)
		//require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		//_, err = cancelAllocation(t, configPath, allocationId, true)
		//if err != nil {
		//	fmt.Println("Error cancelling allocation", err)
		//}

		// sleep for 10 minutes
		time.Sleep(2 * time.Minute)

		curBlock := getLatestFinalizedBlock(t)

		fmt.Println("curBlock", curBlock)

		// get all challenges
		challenges, _ := getAllChallenges(t, allocationId)

		passedChallenges := 0

		for _, challenge := range challenges {
			if challenge.Passed {
				passedChallenges++
			}
			//require.True(t, challenge.Passed != true, "All Challenges should be passed")
		}

		failedChallenges := len(challenges) - passedChallenges

		fmt.Println("passedChallenges", passedChallenges)
		fmt.Println("failedChallenges", failedChallenges)

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

}

func createFreeStorageAllocation(t *test.SystemTest, configFile, from, params string) ([]string, error) {
	t.Logf("Creating new free storage allocation...")
	return cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox add --silent %s --wallet %s --configDir ./config --config %s",
		params,
		from+"_wallet.json",
		configFile), 3, time.Second*5)
}

func getAllocationCancellationReward(t *test.SystemTest, startBlockNumber, endBlockNumber string, blobberList []climodel.BlobberInfo) ([]int64, error) {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := getSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/cancellation-rewards?start_block=" + startBlockNumber + "&end_block=" + endBlockNumber)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var allocationCancellationRewards AllocationCancellationRewards
	err = json.Unmarshal(body, &allocationCancellationRewards)
	if err != nil {
		return nil, err
	}

	blobber1TotalReward := int64(0)
	blobber2TotalReward := int64(0)

	for _, reward := range allocationCancellationRewards.DelegateRewards {
		if reward.ProviderId == blobberList[0].Id {
			blobber1TotalReward += int64(reward.Amount)
		} else if reward.ProviderId == blobberList[1].Id {
			blobber2TotalReward += int64(reward.Amount)
		}
	}

	for _, reward := range allocationCancellationRewards.ProviderRewards {
		if reward.ProviderId == blobberList[0].Id {
			blobber1TotalReward += int64(reward.Amount)
		} else if reward.ProviderId == blobberList[1].Id {
			blobber2TotalReward += int64(reward.Amount)
		}
	}

	return []int64{blobber1TotalReward, blobber2TotalReward}, nil
}

type AllocationCancellationRewards struct {
	DelegateRewards []struct {
		ID          int       `json:"ID"`
		CreatedAt   time.Time `json:"CreatedAt"`
		UpdatedAt   time.Time `json:"UpdatedAt"`
		Amount      int       `json:"amount"`
		BlockNumber int       `json:"block_number"`
		PoolId      string    `json:"pool_id"`
		ProviderId  string    `json:"provider_id"`
		RewardType  int       `json:"reward_type"`
		ChallengeId string    `json:"challenge_id"`
	} `json:"delegate_rewards"`
	ProviderRewards []struct {
		ID          int       `json:"ID"`
		CreatedAt   time.Time `json:"CreatedAt"`
		UpdatedAt   time.Time `json:"UpdatedAt"`
		Amount      int       `json:"amount"`
		BlockNumber int       `json:"block_number"`
		ProviderId  string    `json:"provider_id"`
		RewardType  int       `json:"reward_type"`
		ChallengeId string    `json:"challenge_id"`
	} `json:"provider_rewards"`
}
