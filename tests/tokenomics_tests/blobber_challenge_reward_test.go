package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func TestBlobberChallengeRewards(testSetup *testing.T) {
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

	var blobberList []climodel.BlobberInfo
	output, err := utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 2)

	err = json.Unmarshal([]byte(output[1]), &blobberList)
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

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 2 delegate each (equal stake)", 100*time.Minute, func(t *test.SystemTest) {
		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 2)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1, 1, 1, 1, 1,
		}, 2)

		// Creating Allocation
		utils.SetupWalletWithCustomTokens(t, configPath, 9.0)
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForTwoDelegatesEach(t, allocationId, blobberListString, validatorListString, 0.1*GB, []int64{
			1, 1, 1, 1, 1, 1, 1, 1,
		})
	})

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 2 delegate each (unequal stake)", 100*time.Minute, func(t *test.SystemTest) {
		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 2)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 2, 2, 1, 1, 2, 2,
		}, 2)

		// Creating Allocation
		utils.SetupWalletWithCustomTokens(t, configPath, 9.0)
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForTwoDelegatesEach(t, allocationId, blobberListString, validatorListString, 0.1*GB, []int64{
			1, 1, 2, 2, 1, 1, 2, 2,
		})
	})

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 1 delegate each (equal stake)", 100*time.Minute, func(t *test.SystemTest) {
		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 0.1*GB, 1, 0)
	})

	t.RunSequentiallyWithTimeout("Client Uploads 30% of Allocation and 1 delegate each (equal stake)", 100*time.Minute, func(t *test.SystemTest) {
		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 0.3*GB, 1, 0)
	})

	t.RunSequentiallyWithTimeout("Client Uploads 10% of Allocation and 1 delegate each (unequal stake 2:1)", 100*time.Minute, func(t *test.SystemTest) {
		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1)
		})

		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 2, 1, 2,
		}, 1)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 0.1*GB, 1, 0)
	})

	t.RunSequentiallyWithTimeout("Client Uploads 20% of Allocation and delete 10% immediately and 1 delegate each (equal stake)", 100*time.Minute, func(t *test.SystemTest) {
		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 99,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 0.1*GB, 2, 1)
	})
}

func stakeTokensToBlobbersAndValidators(t *test.SystemTest, blobbers, validators []string, configPath string, tokens []float64, numDelegates int) {
	var blobberDelegates []string
	var validatorDelegates []string

	blobberDelegates = append(blobberDelegates, blobber1Delegate1Wallet, blobber2Delegate1Wallet, blobber1Delegate2Wallet, blobber2Delegate2Wallet)
	validatorDelegates = append(validatorDelegates, validator1Delegate1Wallet, validator2Delegate1Wallet, validator1Delegate2Wallet, validator2Delegate2Wallet)

	idx := 0
	tIdx := 0

	for i := 0; i < numDelegates; i++ {
		for _, blobber := range blobbers { // add balance to delegate wallet
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

func unstakeTokensForBlobbersAndValidators(t *test.SystemTest, blobbers, validators []string, configPath string, numDelegates int) {
	var blobberDelegates []string
	var validatorDelegates []string

	blobberDelegates = append(blobberDelegates, blobber1Delegate1Wallet, blobber2Delegate1Wallet, blobber1Delegate2Wallet, blobber2Delegate2Wallet)
	validatorDelegates = append(validatorDelegates, validator1Delegate1Wallet, validator2Delegate1Wallet, validator1Delegate2Wallet, validator2Delegate2Wallet)

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

	res, _ := http.Get(url) //nolint:gosec

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
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

func tearDownRewardsTests(t *test.SystemTest, blobberList, validatorList []string, configPath string, numDelegates int) {
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

func assertChallengeRewardsForOneDelegateEach(t *test.SystemTest, allocationId string, blobberListString, validatorListString []string, filesize float64, numFiles, numDeletes int) {
	t.Cleanup(func() {
		waitUntilAllocationIsFinalized(t, allocationId)
	})

	blobber1 := blobberListString[0]
	blobber2 := blobberListString[1]
	validator1 := validatorListString[0]
	validator2 := validatorListString[1]

	// Delegate Wallets
	b1D1Wallet, _ := utils.GetWalletForName(t, configPath, blobber1Delegate1Wallet)
	b2D1Wallet, _ := utils.GetWalletForName(t, configPath, blobber2Delegate1Wallet)
	v1D1Wallet, _ := utils.GetWalletForName(t, configPath, validator1Delegate1Wallet)
	v2D1Wallet, _ := utils.GetWalletForName(t, configPath, validator2Delegate1Wallet)

	var fileNames []string

	for i := 0; i < numFiles; i++ {
		remotepath := "/dir/"
		filename := utils.GenerateRandomTestFileName(t)

		err := utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		fileNames = append(fileNames, filename)
	}

	for i := 0; i < numDeletes; i++ {
		remotepath := "/dir/"
		filename := fileNames[numFiles-i-1]

		output, err := utils.DeleteFile(t, utils.EscapedTestName(t), utils.CreateParams(map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
		}), true)
		require.Nil(t, err, "error deleting file", strings.Join(output, "\n"))
	}

	// sleep for 10 minutes
	time.Sleep(10 * time.Minute)

	allocation := utils.GetAllocation(t, allocationId)

	t.Log("Moved to Challenge", allocation.MovedToChallenge)

	totalExpectedReward := allocation.MovedToChallenge - allocation.MovedBack

	challengeRewards, err := getAllAllocationChallengeRewards(t, allocationId)
	require.Nil(t, err, "Error getting challenge rewards")

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

	// check if validator 1 and validator 2 got the same amount of reward with 5% error margin
	require.InEpsilon(t, validator1TotalReward, validator2TotalReward, 0.05, "Validator 1 and Validator 2 rewards are not equal")
	// check if validator 1 and validator 2 delegates got the same amount of reward with 5% error margin
	require.InEpsilon(t, validator1DelegatesTotalReward, validator2DelegatesTotalReward, 0.05, "Validator 1 and Validator 2 delegate rewards are not equal")
}

func assertChallengeRewardsForTwoDelegatesEach(t *test.SystemTest, allocationId string, blobberListString, validatorListString []string, filesize float64, stakes []int64) {
	t.Cleanup(func() {
		waitUntilAllocationIsFinalized(t, allocationId)
	})

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

	remotepath := "/dir/"
	filename := utils.GenerateRandomTestFileName(t)

	err := utils.CreateFileWithSize(filename, int64(filesize))
	require.Nil(t, err)

	output, err := utils.UploadFile(t, configPath, map[string]interface{}{
		// fetch the latest block in the chain
		"allocation": allocationId,
		"remotepath": remotepath + filepath.Base(filename),
		"localpath":  filename,
	}, true)
	require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

	// sleep for 10 minutes
	time.Sleep(10 * time.Minute)

	allocation := utils.GetAllocation(t, allocationId)

	t.Log(allocation.MovedToChallenge)

	totalExpectedReward := float64(allocation.MovedToChallenge - allocation.MovedBack)

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
	require.InEpsilon(t, blobber1Delegate1TotalReward*stakes[2], blobber1Delegate2TotalReward*stakes[0], 0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not in correct proportion")
	require.InEpsilon(t, blobber2Delegate1TotalReward*stakes[3], blobber2Delegate2TotalReward*stakes[1], 0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not in correct proportion")

	// check if both validator delegates got the same amount of reward with 5% error margin
	require.InEpsilon(t, validator1Delegate1TotalReward*stakes[6], validator1Delegate2TotalReward*stakes[4], 0.05, "Validator 1 Delegate 1 and Validator 1 Delegate 2 rewards are not in correct proportion")
	require.InEpsilon(t, validator2Delegate1TotalReward*stakes[7], validator2Delegate2TotalReward*stakes[5], 0.05, "Validator 2 Delegate 1 and Validator 2 Delegate 2 rewards are not in correct proportion")
}

//
// func assertCR(t *test.SystemTest, alloc *climodel.Allocation) {
//	var (
//		expectedAllocationCost            float64
//		expectedCancellationCharge        float64
//		expectedWritePoolBalance          float64
//		expectedMovedToChallenge          float64
//		expectedChallengeRewards          float64
//		expectedBlobberChallengeRewards   float64
//		expectedValidatorChallengeRewards float64
//
//		allocCreatedAt                  int64
//		allocExpiredAt                  int64
//		actualCancellationCharge        float64
//		actualWritePoolBalance          float64
//		actualMovedToChallenge          float64
//		actualChallengeRewards          float64
//		actualBlobberChallengeRewards   float64
//		actualValidatorChallengeRewards float64
//		actualMinLockDemandReward       float64
//
//		totalBlockRewardPerRound float64
//		totalRounds              int64
//
//		expectedBlockReward float64
//		actualBlockReward   float64
//	)
//
//	// Calculating expected allocation cost
//	totalWritePrice := int64(0)
//	for _, blobber := range alloc.Blobbers {
//		expectedAllocationCost += float64(blobber.Terms.WritePrice) * sizeInGB(int64(allocSize/alloc.DataShards))
//		totalWritePrice += blobber.Terms.WritePrice
//	}
//
//	for _, blobber := range alloc.Blobbers {
//		challengeRewardQuery := fmt.Sprintf("allocation_id = '%s' AND provider_id = '%s' AND reward_type = %d", alloc.ID, blobber.ID, ChallengePassReward)
//
//		queryReward := getRewardByQuery(t, challengeRewardQuery)
//		actualChallengeRewardForBlobber := queryReward.TotalReward
//		totalDelegateReward := queryReward.TotalDelegateReward
//
//		// Updating total values
//		actualBlobberChallengeRewards += actualChallengeRewardForBlobber
//		actualChallengeRewards += actualChallengeRewardForBlobber
//
//		expectedChallengeRewardForBlobber := expectedBlobberChallengeRewards * writePriceWeight(blobber.Terms.WritePrice, totalWritePrice)
//
//		t.Log("Expected Challenge Reward: ", expectedChallengeRewardForBlobber)
//		t.Log("Actual Challenge Reward: ", actualChallengeRewardForBlobber)
//
//		require.InEpsilon(t, expectedChallengeRewardForBlobber, actualChallengeRewardForBlobber, standardErrorMargin, "Expected challenge reward for blobber is not equal to actual")
//		require.InEpsilon(t, queryReward.TotalReward*blobber.StakePoolSettings.ServiceCharge, queryReward.TotalProviderReward, standardErrorMargin, "Expected provider reward is not equal to actual")
//		require.InEpsilon(t, queryReward.TotalReward*(1.0-blobber.StakePoolSettings.ServiceCharge), queryReward.TotalDelegateReward, standardErrorMargin, "Expected delegate reward is not equal to actual")
//
//		// Compare Stakepool Rewards
//		blobberStakePools, err := sdk.GetStakePoolInfo(sdk.ProviderBlobber, blobber.ID)
//		require.NoError(t, err, "Error while getting blobber stake pool info")
//
//		totalStakePoolBalance := float64(blobberStakePools.Balance)
//
//		blobberDelegateRewardsQuery := fmt.Sprintf("allocation_id = '%s' AND provider_id = '%s' AND reward_type = %d", alloc.ID, blobber.ID, ChallengePassReward)
//		blobberDelegateRewards := getDelegateRewardByQuery(t, blobberDelegateRewardsQuery)
//
//		for _, blobberStakePool := range blobberStakePools.Delegate {
//			delegateID := blobberStakePool.DelegateID
//			delegateStakePoolBalance := float64(blobberStakePool.Balance)
//			delegateReward := float64(blobberDelegateRewards[string(delegateID)])
//
//			require.InEpsilon(t, delegateStakePoolBalance/totalStakePoolBalance, delegateReward/totalDelegateReward, standardErrorMargin, "Expected delegate reward is not in proportion to stake pool balance")
//		}
//	}
//}
//
// func getRewardByQuery(t *test.SystemTest, query string) *QueryRewardsResponse {
//	var result *QueryRewardsResponse
//
//	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
//	sharderBaseUrl := utils.GetSharderUrl(t)
//	queryRewardRequestURL := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/query-rewards?query=" + url.QueryEscape(query))
//
//	t.Log("Query Rewards URL: ", queryRewardRequestURL)
//
//	res, _ := http.Get(queryRewardRequestURL) //nolint:gosec
//
//	defer func(Body io.ReadCloser) {
//		err := Body.Close()
//		if err != nil {
//			return
//		}
//	}(res.Body)
//
//	body, _ := io.ReadAll(res.Body)
//
//	err := json.Unmarshal(body, &result)
//	if err != nil {
//		return nil
//	}
//
//	return result
//}
//
// func getDelegateRewardByQuery(t *test.SystemTest, query string) map[string]int64 {
//	var result map[string]int64
//
//	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
//	sharderBaseUrl := utils.GetSharderUrl(t)
//	queryRewardRequestURL := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/query-delegate-rewards?query=" + url.QueryEscape(query))
//
//	t.Log("Query Rewards URL: ", queryRewardRequestURL)
//
//	res, _ := http.Get(queryRewardRequestURL) //nolint:gosec
//
//	defer func(Body io.ReadCloser) {
//		err := Body.Close()
//		if err != nil {
//			return
//		}
//	}(res.Body)
//
//	body, _ := io.ReadAll(res.Body)
//
//	err := json.Unmarshal(body, &result)
//	if err != nil {
//		return nil
//	}
//
//	return result
//}
//
// type QueryRewardsResponse struct {
//	TotalProviderReward float64 `json:"total_provider_reward"`
//	TotalDelegateReward float64 `json:"total_delegate_reward"`
//	TotalReward         float64 `json:"total_reward"`
//}
//
//func writePriceWeight(writePrice, totalWritePrice int64) float64 {
//	return float64(writePrice) / float64(totalWritePrice)
//}
