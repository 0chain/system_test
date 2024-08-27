package tokenomics_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/tests/cli_tests"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReplaceEnterpriseBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	var (
		replaceBlobberWithSameIdFailRegex  = `^Error updating allocation:allocation_updating_failed: cannot add blobber [a-f0-9]{64}, already in allocation$`
		replaceBlobberWithWrongIdFailRegex = `^Error updating allocation:allocation_updating_failed: can't get blobber (.+)$`
	)

	t.Parallel()

	t.TestSetup("set storage config to use time_unit as 10 minutes", func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.Cleanup(func() {
		// Reset the storage config time unit
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "1h",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.RunSequentiallyWithTimeout("Replace blobber with 0.5x price should work", time.Minute*15, func(t *test.SystemTest) {
		// Setup and fetch current allocation details
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		// Wait for 5 minutes after allocation creation to simulate usage
		waitForTimeInMinutesWhileLogging(t, 5)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "Error no blobber details found")

		halfPrice := currentBlobberDetails.Terms.WritePrice / 2

		// Update blobber price
		originalPrice := currentBlobberDetails.Terms.WritePrice
		err = updateBlobberPrice(t, configPath, addBlobberID, halfPrice)
		require.Nil(t, err, "Error updating blobber price")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, afterAlloc, "Updated allocation should not be nil")

		// Calculate expected payments and rewards
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfBeforeAlloc := costOfAlloc(beforeAlloc)
		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds

		// Query blobber rewards
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobberToRemove, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToBlobbers)

		// Validate the write pool balance after blobber replacement
		expectedWritePoolBalance := beforeAlloc.WritePool - expectedPaymentToBlobbers - int64(1e9/2)

		require.InEpsilon(t, expectedWritePoolBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		// Validate the enterprise rewards after blobber replacement
		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match after blobber replacement")

		// Reset blobber price
		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")
		}()
	})

	t.RunSequentiallyWithTimeout("Check token accounting of a blobber replacing in allocation, should work", time.Minute*15, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		// Set up allocation and get a random blobber to remove
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		// Wait for 5 minutes to simulate usage
		waitForTimeInMinutesWhileLogging(t, 5)

		// Fetch allocation details before replacement
		alloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, alloc)

		// Calculate the total stake of the blobber to remove
		var prevReplaceeBlobberStake int64
		for _, blobber := range alloc.Blobbers {
			if blobber.ID == blobberToRemove {
				prevReplaceeBlobberStake = blobber.TotalStake
			}
		}

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		// Update allocation to replace blobber
		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		waitForTimeInMinutesWhileLogging(t, 5)

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Fetch updated allocation details
		updatedAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, updatedAlloc)

		// Calculate expected payment to replace blobber
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := updatedAlloc.ExpirationDate - alloc.StartTime - timeUnitInSeconds
		realCostOfBeforeAlloc := costOfAlloc(alloc)
		expectedPaymentToReplacedBlobber := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds

		// Query blobber rewards
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobberToRemove, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToReplacedBlobber)

		// Validate enterprise rewards and write pool balance
		require.InEpsilon(t, expectedPaymentToReplacedBlobber, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match after blobber replacement")

		expectedWritePoolBalance := alloc.WritePool - expectedPaymentToReplacedBlobber
		require.InEpsilon(t, expectedWritePoolBalance, updatedAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		// Validate that stake is transferred from the old blobber to the new blobber
		var newReplaceeBlobberStake int64
		for _, blobber := range updatedAlloc.Blobbers {
			if blobber.ID == addBlobberID {
				newReplaceeBlobberStake = blobber.TotalStake
			}
		}

		require.Equal(t, prevReplaceeBlobberStake, newReplaceeBlobberStake, "Stake should be transferred from old blobber to new")
	})

	t.RunSequentiallyWithTimeout("Replace blobber with same price should work", time.Minute*15, func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "Error no blobber details found")

		originalPrice := currentBlobberDetails.Terms.WritePrice

		// Store the original price before changing it
		err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
		require.Nil(t, err, "Error updating blobber price")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		// Get the allocation details before replacing the blobber
		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		waitForTimeInMinutesWhileLogging(t, 5)

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Fetch the updated allocation details
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, afterAlloc, "Updated allocation should not be nil")

		// Calculate expected payments and refund
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime - timeUnitInSeconds
		realCostOfBeforeAlloc := costOfAlloc(beforeAlloc)
		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedWpBalance := beforeAlloc.WritePool - expectedPaymentToBlobbers

		query := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseRewards, err := getQueryRewards(t, query)
		require.Nil(t, err, "Error fetching blobber rewards from sharder query")

		t.Logf("Total blobber rewards: %f,  Expected payment to blobbers: %v", enterpriseRewards.TotalReward, expectedPaymentToBlobbers)
		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseRewards.TotalReward, 0.01, "Enterprise rewards not matching")

		// Validate the write pool balance after blobber replacement
		require.InEpsilon(t, expectedWpBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		// Ensure the blobber price is reset after the test
		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")
		}()
	})

	t.RunSequentiallyWithTimeout("Replace blobber with 0.5x price should work", time.Minute*15, func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "Error no blobber details found")

		halfPrice := currentBlobberDetails.Terms.WritePrice / 2

		// Store the original price before changing it
		originalPrice := currentBlobberDetails.Terms.WritePrice
		err = updateBlobberPrice(t, configPath, addBlobberID, halfPrice)
		require.Nil(t, err, "Error updating blobber price")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		// Get the allocation details before replacing the blobber
		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		waitForTimeInMinutesWhileLogging(t, 5)

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Fetch the updated allocation details
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, afterAlloc, "Updated allocation should not be nil")

		// Calculate expected payments and refund
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime - timeUnitInSeconds
		realCostOfBeforeAlloc := costOfAlloc(beforeAlloc)
		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedWpBalance := beforeAlloc.WritePool - expectedPaymentToBlobbers

		query := fmt.Sprintf("allocation_id='%s' AND reward_query=%d", allocationID, EnterpriseBlobberReward)
		enterpriseRewards, err := getQueryRewards(t, query)
		require.Nil(t, err, "Error fetching total blobber rewards")

		t.Logf("Total blobber rewards: %v", enterpriseRewards.TotalReward)
		t.Logf("Expected payment to blobbers: %v", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseRewards.TotalReward, 0.01, "Enterprise blobber rewards should match")

		// Validate the write pool balance after blobber replacement
		require.InEpsilon(t, expectedWpBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")
		}()
	})

	t.RunSequentiallyWithTimeout("Replace blobber with 2x price should work", time.Minute*15, func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "Error no blobber details found")

		doublePrice := currentBlobberDetails.Terms.WritePrice * 2

		// Store the original price before changing it
		originalPrice := currentBlobberDetails.Terms.WritePrice
		err = updateBlobberPrice(t, configPath, addBlobberID, doublePrice)
		require.Nil(t, err, "Error updating blobber price")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		// Get the allocation details before replacing the blobber
		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		waitForTimeInMinutesWhileLogging(t, 5)

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Fetch the updated allocation details
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, afterAlloc, "Updated allocation should not be nil")

		// Calculate expected payments and refund
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfBeforeAlloc := costOfAlloc(beforeAlloc)
		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedWpBalance := beforeAlloc.WritePool - expectedPaymentToBlobbers

		query := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseRewards, err := getQueryRewards(t, query)
		require.Nil(t, err, "Error fetching blobber rewards")

		t.Logf("Total blobber rewards: %+v", enterpriseRewards)
		t.Logf("Expected payment to blobbers: %v", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseRewards.TotalReward, 0.01, "Blobber rewards should match")

		// Validate the write pool balance after blobber replacement
		require.InEpsilon(t, expectedWpBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")
		}()
	})

	t.Run("Replace blobber with the same one in allocation, shouldn't work", func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		blobberAuthTickets, _ := utils.GenerateBlobberAuthTickets(t, configPath)
		addBlobberAuthTicket := blobberAuthTickets[0]

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             blobberToRemove,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, params, false)
		require.NotNil(t, err, "Expected error updating allocation but got none", strings.Join(output, "\n"))
		require.Regexp(t, replaceBlobberWithSameIdFailRegex, strings.Join(output, "\n"),
			"Error regex match fail update allocation for replace blobber with the same one")
	})

	t.Run("Replace blobber with incorrect blobber ID of an old blobber, shouldn't work", func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		incorrectBlobberID := "1234abc"

		blobberAuthTickets, _ := utils.GenerateBlobberAuthTickets(t, configPath)

		addBlobberAuthTicket := blobberAuthTickets[0]

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             incorrectBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, params, false)
		require.NotNil(t, err, "Expected error updating allocation but got none", strings.Join(output, "\n"))
		require.Regexp(t, replaceBlobberWithWrongIdFailRegex, strings.Join(output, "\n"),
			"Error regex match update allocation for replace blobber with incorrect blobber")
	})
}

func setupAllocationAndGetRandomBlobber(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) (string, string) {
	utils.SetupWalletWithCustomTokens(t, configPath, 10)

	lockAmountPassed := false
	faucetTokens := 10.0

	options := map[string]interface{}{
		"data":   2,
		"parity": 2,
		"size":   1 * GB,
		"lock":   "0.2",
	}

	for _, params := range extraParams {
		// Extract parameters unrelated to upload
		if tokenStr, ok := params["tokens"]; ok {
			token, err := strconv.ParseFloat(fmt.Sprintf("%v", tokenStr), 64)
			require.Nil(t, err)
			faucetTokens = token
			delete(params, "tokens")
		}

		if _, lockPassed := params["lock"]; lockPassed {
			lockAmountPassed = true
		}

		for k, v := range params {
			options[k] = v
		}
	}

	if !lockAmountPassed {
		options["lock"] = faucetTokens / 2
	}

	allocationID := utils.SetupEnterpriseAllocation(t, configPath, options)

	wd, _ := os.Getwd()
	walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
	configFile := filepath.Join(wd, "config", configPath)

	randomBlobber, err := cli_tests.GetRandomBlobber(walletFile, configFile, allocationID, "")
	require.Nil(t, err, "Error getting random blobber")

	return allocationID, randomBlobber
}

func updateBlobberPrice(t *test.SystemTest, configPath, blobberID string, newPrice int64) error {
	params := map[string]interface{}{
		"blobber_id":  blobberID,
		"write_price": utils.IntToZCN(newPrice),
	}
	output, err := utils.ExecuteFaucetWithTokensForWallet(t, "wallets/blobber_owner", configPath, 99)
	require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

	output, err = utils.UpdateBlobberInfoForWallet(t, configPath, "wallets/blobber_owner", utils.CreateParams(params))
	if err != nil {
		t.Log("Error updating blobber price: " + strings.Join(output, "\n"))
		return err
	}

	t.Log("Updated blobber price:", strings.Join(output, "\n"))
	return nil
}
