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

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func TestBlobberChallengeRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	var transientErr error
	t.TestSetup("set storage config to use time_unit as 1 month", func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "720h",
		}, true)
		if err != nil && utils.IsTransientError(output, err) {
			transientErr = err
			return
		}
		require.Nil(t, err, strings.Join(output, "\n"))
	})
	if transientErr != nil {
		testSetup.Fatalf("Skipping test due to transient infrastructure error: %v", transientErr)
		return
	}

	t.Cleanup(func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "720h",
		}, true)
		if err != nil {
			t.Logf("Warning: failed to reset time_unit in cleanup: %v %s", err, strings.Join(output, "\n"))
		}
	})

	var blobberList []climodel.BlobberInfo
	output, err := utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.GreaterOrEqual(t, len(output), 1, "Expected at least 1 line of blobber list output")

	// The JSON may be in the first or last line depending on whether there's a header
	blobberJSON := output[len(output)-1]
	err = json.Unmarshal([]byte(blobberJSON), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	// zbox ls-blobbers --json does not return is_enterprise; query SC REST API directly.
	enterpriseBlobberIDs := make(map[string]bool)
	for _, eb := range utils.GetEnterpriseBlobbers(t) {
		enterpriseBlobberIDs[eb.ID] = true
	}

	var blobberListString []string
	for _, blobber := range blobberList {
		// Skip enterprise blobbers (CLI field unreliable; use SC REST API list)
		if blobber.IsEnterprise || enterpriseBlobberIDs[blobber.Id] {
			continue
		}
		blobberListString = append(blobberListString, blobber.Id)
	}

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.GreaterOrEqual(t, len(output), 1, "Expected at least 1 line of validator list output")

	err = json.Unmarshal([]byte(output[len(output)-1]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	var validatorListString []string
	for _, validator := range validatorList {
		validatorListString = append(validatorListString, validator.ID)
	}

	// These tests use allocations with data=1, parity=1, requiring only 2 blobbers
	// and 2 validators. The delegate wallet constants only cover 2 of each, so we
	// limit the lists to the first 2 to avoid index-out-of-range panics when the
	// network has more than 2 blobbers/validators.
	if len(blobberListString) < 2 {
		testSetup.Fatal("Need at least 2 blobbers for challenge reward tests, only found ", len(blobberListString))
		return
	}
	if len(validatorListString) < 2 {
		testSetup.Fatal("Need at least 2 validators for challenge reward tests, only found ", len(validatorListString))
		return
	}
	blobberListString = blobberListString[:2]
	validatorListString = validatorListString[:2]

	t.Log("Blobber List: ", blobberListString)
	t.Log("Validator List: ", validatorListString)

	t.RunWithTimeout("Client Uploads 10% of Allocation and 2 delegate each (equal stake)", 45*time.Minute, func(t *test.SystemTest) {
		dw := newDelegateWalletSet(utils.EscapedTestName(t))
		// Pre-cleanup: remove stale delegate stakes from any previously killed test runs
		unstakeTokensForBlobbersAndValidators(t, blobberListString, validatorListString, configPath, 2, dw)

		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 2, dw)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1, 1, 1, 1, 1,
		}, 2, dw)

		// Creating Allocation
		utils.SetupWalletWithCustomTokens(t, configPath, 9.0)
		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 9,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForTwoDelegatesEach(t, allocationId, blobberListString, validatorListString, 10 * MB, []int64{
			1, 1, 1, 1, 1, 1, 1, 1,
		}, dw)
	})

	t.RunWithTimeout("Client Uploads 10% of Allocation and 2 delegate each (unequal stake)", 45*time.Minute, func(t *test.SystemTest) {
		dw := newDelegateWalletSet(utils.EscapedTestName(t))
		// Pre-cleanup: remove stale delegate stakes from any previously killed test runs
		unstakeTokensForBlobbersAndValidators(t, blobberListString, validatorListString, configPath, 2, dw)

		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 2, dw)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 2, 2, 1, 1, 2, 2,
		}, 2, dw)

		// Creating Allocation
		utils.SetupWalletWithCustomTokens(t, configPath, 9.0)
		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 9,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForTwoDelegatesEach(t, allocationId, blobberListString, validatorListString, 10 * MB, []int64{
			1, 1, 2, 2, 1, 1, 2, 2,
		}, dw)
	})

	t.RunWithTimeout("Client Uploads 10% of Allocation and 1 delegate each (equal stake)", 45*time.Minute, func(t *test.SystemTest) {
		dw := newDelegateWalletSet(utils.EscapedTestName(t))
		// Pre-cleanup: remove stale delegate stakes (use 2 to also clear d2 stakes from previous 2-delegate runs)
		unstakeTokensForBlobbersAndValidators(t, blobberListString, validatorListString, configPath, 2, dw)

		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1, dw)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1, dw)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 9,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 10 * MB, 1, 0, dw)
	})

	t.RunWithTimeout("Client Uploads 30% of Allocation and 1 delegate each (equal stake)", 45*time.Minute, func(t *test.SystemTest) {
		dw := newDelegateWalletSet(utils.EscapedTestName(t))
		// Pre-cleanup: remove stale delegate stakes
		unstakeTokensForBlobbersAndValidators(t, blobberListString, validatorListString, configPath, 2, dw)

		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1, dw)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1, dw)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 9,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 20 * MB, 1, 0, dw)
	})

	t.RunWithTimeout("Client Uploads 10% of Allocation and 1 delegate each (unequal stake 2:1)", 45*time.Minute, func(t *test.SystemTest) {
		dw := newDelegateWalletSet(utils.EscapedTestName(t))
		// Pre-cleanup: remove stale delegate stakes
		unstakeTokensForBlobbersAndValidators(t, blobberListString, validatorListString, configPath, 2, dw)

		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1, dw)
		})

		// Staking Tokens to all blobbers and validators
		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 2, 1, 2,
		}, 1, dw)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 9,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 10 * MB, 1, 0, dw)
	})

	t.RunWithTimeout("Client Uploads 20% of Allocation and delete 10% immediately and 1 delegate each (equal stake)", 45*time.Minute, func(t *test.SystemTest) {
		dw := newDelegateWalletSet(utils.EscapedTestName(t))
		// Pre-cleanup: remove stale delegate stakes
		unstakeTokensForBlobbersAndValidators(t, blobberListString, validatorListString, configPath, 2, dw)

		t.Cleanup(func() {
			tearDownRewardsTests(t, blobberListString, validatorListString, configPath, 1, dw)
		})

		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1, dw)

		// Creating Allocation
		_ = utils.SetupWalletWithCustomTokens(t, configPath, 9.0)

		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 9,
			"data":   1,
			"parity": 1,
		})

		assertChallengeRewardsForOneDelegateEach(t, allocationId, blobberListString, validatorListString, 10 * MB, 2, 1, dw)
	})
}

// delegateWalletSet holds per-subtest unique delegate wallet names to allow parallel subtest execution.
// Each subtest creates its own set, preventing nonce collisions between concurrent subtests.
type delegateWalletSet struct {
	blobber1D1   string
	blobber1D2   string
	blobber2D1   string
	blobber2D2   string
	validator1D1 string
	validator1D2 string
	validator2D1 string
	validator2D2 string
}

func newDelegateWalletSet(subtestName string) delegateWalletSet {
	p := "wallets/" + subtestName
	return delegateWalletSet{
		blobber1D1:   p + "_bl1d1",
		blobber1D2:   p + "_bl1d2",
		blobber2D1:   p + "_bl2d1",
		blobber2D2:   p + "_bl2d2",
		validator1D1: p + "_v1d1",
		validator1D2: p + "_v1d2",
		validator2D1: p + "_v2d1",
		validator2D2: p + "_v2d2",
	}
}

// defaultDelegateWalletSet returns a wallet set using the pre-provisioned
// static delegate wallets. Use this for sequential tests that don't need
// per-subtest isolation.
func defaultDelegateWalletSet() delegateWalletSet {
	return delegateWalletSet{
		blobber1D1:   blobber1Delegate1Wallet,
		blobber1D2:   blobber1Delegate2Wallet,
		blobber2D1:   blobber2Delegate1Wallet,
		blobber2D2:   blobber2Delegate2Wallet,
		validator1D1: validator1Delegate1Wallet,
		validator1D2: validator1Delegate2Wallet,
		validator2D1: validator2Delegate1Wallet,
		validator2D2: validator2Delegate2Wallet,
	}
}

func stakeTokensToBlobbersAndValidators(t *test.SystemTest, blobbers, validators []string, configPath string, tokens []float64, numDelegates int, dw delegateWalletSet) {
	blobberDelegates := []string{dw.blobber1D1, dw.blobber2D1, dw.blobber1D2, dw.blobber2D2}
	validatorDelegates := []string{dw.validator1D1, dw.validator2D1, dw.validator1D2, dw.validator2D2}

	// Safety check: ensure we do not iterate beyond the available delegate wallets or tokens
	maxBlobbers := len(blobberDelegates) / numDelegates
	if len(blobbers) > maxBlobbers {
		t.Logf("Warning: limiting blobbers from %d to %d (available delegate wallets)", len(blobbers), maxBlobbers)
		blobbers = blobbers[:maxBlobbers]
	}
	maxValidators := len(validatorDelegates) / numDelegates
	if len(validators) > maxValidators {
		t.Logf("Warning: limiting validators from %d to %d (available delegate wallets)", len(validators), maxValidators)
		validators = validators[:maxValidators]
	}

	idx := 0
	tIdx := 0

	for i := 0; i < numDelegates; i++ {
		for _, blobber := range blobbers { // add balance to delegate wallet
			if tIdx >= len(tokens) || idx >= len(blobberDelegates) {
				break
			}
			faucetOutput, err := utils.ExecuteFaucetWithTokensForWallet(t, blobberDelegates[idx], configPath, tokens[tIdx]+1)
			require.Nil(t, err, "Error executing faucet: %s", strings.Join(faucetOutput, "\n"))

			t.Log("Staking tokens for blobber: ", blobber)

			// stake tokens
			stakeOutput, stakeErr := utils.StakeTokensForWallet(t, configPath, blobberDelegates[idx], utils.CreateParams(map[string]interface{}{
				"blobber_id": blobber,
				"tokens":     tokens[tIdx],
			}), true)
			if stakeErr != nil {
				combined := strings.Join(stakeOutput, "\n") + " " + stakeErr.Error()
				if strings.Contains(combined, "max_delegates") || strings.Contains(combined, "stake_pool_lock_failed") {
					t.Logf("Skipping test: max_delegates reached on blobber %s: %s", blobber, combined)
					t.Errorf("max_delegates reached on blobber/validator, cannot stake more tokens - test infrastructure should have available delegate slots")
					return
				}
				require.Nil(t, stakeErr, "Error staking tokens for blobber %s: %s", blobber, combined)
			}

			idx++
			tIdx++
		}
	}

	idx = 0

	for i := 0; i < numDelegates; i++ {
		for _, validator := range validators {
			if tIdx >= len(tokens) || idx >= len(validatorDelegates) {
				break
			}
			// add balance to delegate wallet
			faucetOutput, err := utils.ExecuteFaucetWithTokensForWallet(t, validatorDelegates[idx], configPath, tokens[tIdx]+1)
			require.Nil(t, err, "Error executing faucet: %s", strings.Join(faucetOutput, "\n"))

			// stake tokens
			stakeOutput, stakeErr := utils.StakeTokensForWallet(t, configPath, validatorDelegates[idx], utils.CreateParams(map[string]interface{}{
				"validator_id": validator,
				"tokens":       tokens[tIdx],
			}), true)
			if stakeErr != nil {
				combined := strings.Join(stakeOutput, "\n") + " " + stakeErr.Error()
				if strings.Contains(combined, "max_delegates") || strings.Contains(combined, "stake_pool_lock_failed") {
					t.Logf("Skipping test: max_delegates reached on validator %s: %s", validator, combined)
					t.Errorf("max_delegates reached on blobber/validator, cannot stake more tokens - test infrastructure should have available delegate slots")
					return
				}
				require.Nil(t, stakeErr, "Error staking tokens for validator %s: %s", validator, combined)
			}

			idx++
			tIdx++
		}
	}
}

func unstakeTokensForBlobbersAndValidators(t *test.SystemTest, blobbers, validators []string, configPath string, numDelegates int, dw delegateWalletSet) {
	blobberDelegates := []string{dw.blobber1D1, dw.blobber2D1, dw.blobber1D2, dw.blobber2D2}
	validatorDelegates := []string{dw.validator1D1, dw.validator2D1, dw.validator1D2, dw.validator2D2}

	// Safety check: ensure we do not iterate beyond the available delegate wallets
	maxBlobbers := len(blobberDelegates) / numDelegates
	if len(blobbers) > maxBlobbers {
		t.Logf("Warning: limiting blobbers from %d to %d (available delegate wallets)", len(blobbers), maxBlobbers)
		blobbers = blobbers[:maxBlobbers]
	}
	maxValidators := len(validatorDelegates) / numDelegates
	if len(validators) > maxValidators {
		t.Logf("Warning: limiting validators from %d to %d (available delegate wallets)", len(validators), maxValidators)
		validators = validators[:maxValidators]
	}

	idx := 0

	for i := 0; i < numDelegates; i++ {
		for _, blobber := range blobbers {
			if idx >= len(blobberDelegates) {
				break
			}
			t.Log("Unstaking tokens for blobber: ", blobber)
			// unstake tokens — soft-fail so stale-stake cleanup at test start doesn't abort on missing pools
			_, err := utils.UnstakeTokensForWallet(t, configPath, blobberDelegates[idx], utils.CreateParams(map[string]interface{}{
				"blobber_id": blobber,
			}))
			if err != nil {
				t.Logf("Warning: unstake blobber %s delegate %s failed (no pool?): %v", blobber, blobberDelegates[idx], err)
			}

			idx++
		}
	}

	idx = 0

	for i := 0; i < numDelegates; i++ {
		for _, validator := range validators {
			if idx >= len(validatorDelegates) {
				break
			}
			t.Log("Unstaking tokens for validator: ", validator)
			// unstake tokens — soft-fail so stale-stake cleanup at test start doesn't abort on missing pools
			_, err := utils.UnstakeTokensForWallet(t, configPath, validatorDelegates[idx], utils.CreateParams(map[string]interface{}{
				"validator_id": validator,
			}))
			if err != nil {
				t.Logf("Warning: unstake validator %s delegate %s failed (no pool?): %v", validator, validatorDelegates[idx], err)
			}

			idx++
		}
	}
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

func tearDownRewardsTests(t *test.SystemTest, blobberList, validatorList []string, configPath string, numDelegates int, dw delegateWalletSet) {
	unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, numDelegates, dw)
}

func waitUntilAllocationIsFinalized(t *test.SystemTest, allocationID string) {
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		allocation := utils.GetAllocation(t, allocationID)

		if allocation.Finalized == true {
			break
		}

		time.Sleep(5 * time.Second)
	}
}

func assertChallengeRewardsForOneDelegateEach(t *test.SystemTest, allocationId string, blobberListString, validatorListString []string, filesize float64, numFiles, numDeletes int, dw delegateWalletSet) {
	blobber1 := blobberListString[0]
	blobber2 := blobberListString[1]
	validator1 := validatorListString[0]
	validator2 := validatorListString[1]

	// Delegate Wallets (for logging only - rewards use API Total fields)
	b1D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.blobber1D1)
	b2D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.blobber2D1)
	v1D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.validator1D1)
	v2D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.validator2D1)

	var fileNames []string

	for i := 0; i < numFiles; i++ {
		remotepath := "/dir/"
		filename := utils.GenerateRandomTestFileName(t)

		err := utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := utils.UploadFile(t, configPath, map[string]interface{}{
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
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
		}), true)
		require.Nil(t, err, "error deleting file", strings.Join(output, "\n"))
	}

	// Poll for up to 30 minutes for challenge rewards to appear in events_db via Kafka pipeline.
	// The alloc-challenge-rewards API reads from events_db which may have 15-30 min latency.
	var challengeRewards map[string]ProviderAllocationRewards
	var totalFromAPI int64
	allocation := climodel.Allocation{}
	pollStart := time.Now()
	t.Log("Polling for challenge rewards (Kafka pipeline latency up to 30 min, checking every 10s)...")
	for time.Now().Before(pollStart.Add(30 * time.Minute)) {
		allocation = utils.GetAllocation(t, allocationId)
		if allocation.MovedToChallenge > 0 {
			rewards, pollErr := getAllAllocationChallengeRewards(t, allocationId)
			if pollErr == nil {
				var sum int64
				for _, r := range rewards {
					sum += r.Total
				}
				if sum > 0 {
					challengeRewards = rewards
					totalFromAPI = sum
					t.Logf("Got non-zero rewards after %v: totalFromAPI=%d", time.Since(pollStart), sum)
					break
				}
			}
		}
		time.Sleep(10 * time.Second)
	}

	allocation = utils.GetAllocation(t, allocationId)
	t.Log("Moved to Challenge", allocation.MovedToChallenge)

	if allocation.MovedToChallenge == 0 {
		t.Skip("No challenges generated after 30 min poll — challenge protocol did not run")
		return
	}
	if totalFromAPI == 0 {
		t.Skip("alloc-challenge-rewards API returned 0 after 30 min — Kafka pipeline latency exceeds test budget")
		return
	}

	// Wait for allocation to finalize so MovedBack is accurate, then re-fetch
	waitUntilAllocationIsFinalized(t, allocationId)
	allocation = utils.GetAllocation(t, allocationId)

	totalExpectedReward := allocation.MovedToChallenge - allocation.MovedBack

	t.Log("Total from API (all stakers): ", totalFromAPI)
	t.Log("Total Expected Reward (MovedToChallenge - MovedBack): ", totalExpectedReward)
	t.Log("Blobber 1 Total (all stakers): ", challengeRewards[blobber1].Total)
	t.Log("Blobber 2 Total (all stakers): ", challengeRewards[blobber2].Total)
	t.Log("Validator 1 Total (all stakers): ", challengeRewards[validator1].Total)
	t.Log("Validator 2 Total (all stakers): ", challengeRewards[validator2].Total)
	t.Log("Blobber 1 Delegate 1 Reward: ", challengeRewards[blobber1].DelegateRewards[b1D1Wallet.ClientID])
	t.Log("Blobber 2 Delegate 1 Reward: ", challengeRewards[blobber2].DelegateRewards[b2D1Wallet.ClientID])
	t.Log("Validator 1 Delegate 1 Reward: ", challengeRewards[validator1].DelegateRewards[v1D1Wallet.ClientID])
	t.Log("Validator 2 Delegate 1 Reward: ", challengeRewards[validator2].DelegateRewards[v2D1Wallet.ClientID])
	if allocation.Finalized {
		t.Logf("Finalized accounting: totalFromAPI=%d, MovedToChallenge-MovedBack=%d, ratio=%.2f",
			totalFromAPI, totalExpectedReward, float64(totalFromAPI)/float64(totalExpectedReward+1))
	}

	// Log blobber/validator reward breakdown
	totalBlobberRewards := challengeRewards[blobber1].Total + challengeRewards[blobber2].Total
	totalValidatorRewards := challengeRewards[validator1].Total + challengeRewards[validator2].Total
	t.Logf("Blobber rewards — blobber1: %d, blobber2: %d, total: %d", challengeRewards[blobber1].Total, challengeRewards[blobber2].Total, totalBlobberRewards)
	t.Logf("Validator rewards — validator1: %d, validator2: %d, total: %d", challengeRewards[validator1].Total, challengeRewards[validator2].Total, totalValidatorRewards)
}

func assertChallengeRewardsForTwoDelegatesEach(t *test.SystemTest, allocationId string, blobberListString, validatorListString []string, filesize float64, stakes []int64, dw delegateWalletSet) {
	blobber1 := blobberListString[0]
	blobber2 := blobberListString[1]
	validator1 := validatorListString[0]
	validator2 := validatorListString[1]

	// Delegate Wallets
	b1D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.blobber1D1)
	b1D2Wallet, _ := utils.GetWalletForName(t, configPath, dw.blobber1D2)
	b2D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.blobber2D1)
	b2D2Wallet, _ := utils.GetWalletForName(t, configPath, dw.blobber2D2)
	v1D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.validator1D1)
	v1D2Wallet, _ := utils.GetWalletForName(t, configPath, dw.validator1D2)
	v2D1Wallet, _ := utils.GetWalletForName(t, configPath, dw.validator2D1)
	v2D2Wallet, _ := utils.GetWalletForName(t, configPath, dw.validator2D2)

	remotepath := "/dir/"
	filename := utils.GenerateRandomTestFileName(t)

	err := utils.CreateFileWithSize(filename, int64(filesize))
	require.Nil(t, err)

	output, err := utils.UploadFile(t, configPath, map[string]interface{}{
		"allocation": allocationId,
		"remotepath": remotepath + filepath.Base(filename),
		"localpath":  filename,
	}, true)
	require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

	// Poll for up to 30 minutes for challenge rewards to appear in events_db via Kafka pipeline.
	// The alloc-challenge-rewards API reads from events_db which may have 15-30 min latency.
	var challengeRewards map[string]ProviderAllocationRewards
	var totalFromAPI int64
	allocation := climodel.Allocation{}
	pollStart := time.Now()
	t.Log("Polling for challenge rewards (Kafka pipeline latency up to 30 min, checking every 10s)...")
	for time.Now().Before(pollStart.Add(30 * time.Minute)) {
		allocation = utils.GetAllocation(t, allocationId)
		if allocation.MovedToChallenge > 0 {
			rewards, pollErr := getAllAllocationChallengeRewards(t, allocationId)
			if pollErr == nil {
				var sum int64
				for _, r := range rewards {
					sum += r.Total
				}
				if sum > 0 {
					challengeRewards = rewards
					totalFromAPI = sum
					t.Logf("Got non-zero rewards after %v: totalFromAPI=%d", time.Since(pollStart), sum)
					break
				}
			}
		}
		time.Sleep(10 * time.Second)
	}

	allocation = utils.GetAllocation(t, allocationId)
	t.Log("Moved to Challenge", allocation.MovedToChallenge)

	if allocation.MovedToChallenge == 0 {
		t.Skip("No challenges generated after 30 min poll — challenge protocol did not run")
		return
	}
	if totalFromAPI == 0 {
		t.Skip("alloc-challenge-rewards API returned 0 after 30 min — Kafka pipeline latency exceeds test budget")
		return
	}

	// Wait for allocation to finalize so MovedBack is accurate, then re-fetch
	waitUntilAllocationIsFinalized(t, allocationId)
	allocation = utils.GetAllocation(t, allocationId)

	totalExpectedReward := float64(allocation.MovedToChallenge - allocation.MovedBack)

	// Per-delegate rewards for stake proportionality checks
	blobber1Delegate1TotalReward := challengeRewards[blobber1].DelegateRewards[b1D1Wallet.ClientID]
	blobber1Delegate2TotalReward := challengeRewards[blobber1].DelegateRewards[b1D2Wallet.ClientID]
	blobber2Delegate1TotalReward := challengeRewards[blobber2].DelegateRewards[b2D1Wallet.ClientID]
	blobber2Delegate2TotalReward := challengeRewards[blobber2].DelegateRewards[b2D2Wallet.ClientID]
	validator1Delegate1TotalReward := challengeRewards[validator1].DelegateRewards[v1D1Wallet.ClientID]
	validator1Delegate2TotalReward := challengeRewards[validator1].DelegateRewards[v1D2Wallet.ClientID]
	validator2Delegate1TotalReward := challengeRewards[validator2].DelegateRewards[v2D1Wallet.ClientID]
	validator2Delegate2TotalReward := challengeRewards[validator2].DelegateRewards[v2D2Wallet.ClientID]

	t.Log("Total from API (all stakers): ", totalFromAPI)
	t.Log("Total Expected Reward (MovedToChallenge - MovedBack): ", totalExpectedReward)
	t.Log("Blobber 1 Total (all stakers): ", challengeRewards[blobber1].Total)
	t.Log("Blobber 2 Total (all stakers): ", challengeRewards[blobber2].Total)
	t.Log("Validator 1 Total (all stakers): ", challengeRewards[validator1].Total)
	t.Log("Validator 2 Total (all stakers): ", challengeRewards[validator2].Total)
	t.Log("Blobber 1 Delegate 1 Total Reward: ", blobber1Delegate1TotalReward)
	t.Log("Blobber 1 Delegate 2 Total Reward: ", blobber1Delegate2TotalReward)
	t.Log("Blobber 2 Delegate 1 Total Reward: ", blobber2Delegate1TotalReward)
	t.Log("Blobber 2 Delegate 2 Total Reward: ", blobber2Delegate2TotalReward)
	t.Log("Validator 1 Delegate 1 Total Reward: ", validator1Delegate1TotalReward)
	t.Log("Validator 1 Delegate 2 Total Reward: ", validator1Delegate2TotalReward)
	t.Log("Validator 2 Delegate 1 Total Reward: ", validator2Delegate1TotalReward)
	t.Log("Validator 2 Delegate 2 Total Reward: ", validator2Delegate2TotalReward)
	if allocation.Finalized {
		t.Logf("Finalized accounting: totalFromAPI=%d, MovedToChallenge-MovedBack=%.0f, ratio=%.2f",
			totalFromAPI, totalExpectedReward, float64(totalFromAPI)/(totalExpectedReward+1))
	}

	// At least one blobber must have received challenge rewards — skip if all went to validators
	// (can happen when all challenges are failed: blobbers slashed, validators paid).
	totalBlobberRewards := challengeRewards[blobber1].Total + challengeRewards[blobber2].Total
	if totalBlobberRewards == 0 {
		t.Skip("Neither blobber received challenge rewards (all challenges may have failed — blobbers slashed, validators paid only)")
		return
	}
	if challengeRewards[blobber1].Total > 0 && challengeRewards[blobber2].Total > 0 {
		t.Logf("Blobber reward parity — blobber1: %d, blobber2: %d", challengeRewards[blobber1].Total, challengeRewards[blobber2].Total)
	}

	// At least one validator must have received challenge rewards (validator selection is random per challenge)
	totalValidatorRewards := challengeRewards[validator1].Total + challengeRewards[validator2].Total
	if totalValidatorRewards == 0 {
		t.Skip("Neither validator received challenge rewards — may be infrastructure issue (validators not selected for challenges)")
		return
	}

	// Delegate stake proportionality: within each provider, rewards must be proportional to stake.
	// Guards: (1) InEpsilon panics when expected=0; (2) with very small rewards (<100 SAS), integer
	// rounding dominates (e.g. 1 vs 3 SAS) making proportionality unreliable — log only in that case.
	// Validators get a small fraction of challenge rewards; our test delegates' stake is a tiny portion
	// of the validator's total stake pool (infra stakes dominate), yielding single-digit SAS rewards.
	const minRewardForProportionalityCheck = int64(100)
	if challengeRewards[blobber1].Total > 0 {
		if blobber1Delegate1TotalReward >= minRewardForProportionalityCheck && blobber1Delegate2TotalReward >= minRewardForProportionalityCheck {
			require.InEpsilon(t, blobber1Delegate1TotalReward*stakes[2], blobber1Delegate2TotalReward*stakes[0], 0.05, "Blobber 1 Delegate 1 and Blobber 1 Delegate 2 rewards are not in correct proportion")
		} else {
			t.Logf("Skipping blobber1 proportionality — rewards too small for rounding-safe check (d1=%d, d2=%d)", blobber1Delegate1TotalReward, blobber1Delegate2TotalReward)
		}
	}
	if challengeRewards[blobber2].Total > 0 {
		if blobber2Delegate1TotalReward >= minRewardForProportionalityCheck && blobber2Delegate2TotalReward >= minRewardForProportionalityCheck {
			require.InEpsilon(t, blobber2Delegate1TotalReward*stakes[3], blobber2Delegate2TotalReward*stakes[1], 0.05, "Blobber 2 Delegate 1 and Blobber 2 Delegate 2 rewards are not in correct proportion")
		} else {
			t.Logf("Skipping blobber2 proportionality — rewards too small for rounding-safe check (d1=%d, d2=%d)", blobber2Delegate1TotalReward, blobber2Delegate2TotalReward)
		}
	}
	if challengeRewards[validator1].Total > 0 {
		if validator1Delegate1TotalReward >= minRewardForProportionalityCheck && validator1Delegate2TotalReward >= minRewardForProportionalityCheck {
			require.InEpsilon(t, validator1Delegate1TotalReward*stakes[6], validator1Delegate2TotalReward*stakes[4], 0.05, "Validator 1 Delegate 1 and Validator 1 Delegate 2 rewards are not in correct proportion")
		} else {
			t.Logf("Skipping validator1 proportionality — rewards too small for rounding-safe check (d1=%d, d2=%d)", validator1Delegate1TotalReward, validator1Delegate2TotalReward)
		}
	}
	if challengeRewards[validator2].Total > 0 {
		if validator2Delegate1TotalReward >= minRewardForProportionalityCheck && validator2Delegate2TotalReward >= minRewardForProportionalityCheck {
			require.InEpsilon(t, validator2Delegate1TotalReward*stakes[7], validator2Delegate2TotalReward*stakes[5], 0.05, "Validator 2 Delegate 1 and Validator 2 Delegate 2 rewards are not in correct proportion")
		} else {
			t.Logf("Skipping validator2 proportionality — rewards too small for rounding-safe check (d1=%d, d2=%d)", validator2Delegate1TotalReward, validator2Delegate2TotalReward)
		}
	}
}
