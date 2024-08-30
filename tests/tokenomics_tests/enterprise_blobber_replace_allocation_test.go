package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/cli_tests"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
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

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error creating configuration wallet", strings.Join(output, "\n"))

	var blobbersList []climodel.Blobber
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error fetching blobbers %v", strings.Join(output, "\n"))
	require.Len(t, output, 1, "Error wrong json format", strings.Join(output, "\n"))

	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobbersList)

	require.Nil(t, err, "Error decoding blobbers json")

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

		var blobbers []climodel.Blobber
		output, err = utils.ListBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobberes", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Error invalid json length", strings.Join(output, "\n"))

		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobbers)
		require.Nil(t, err, "Eerror decoding blobbers", strings.Join(output, "\n"))

		for _, blobber := range blobbers {
			if blobber.Terms.WritePrice != 1e9 {
				err := updateBlobberPrice(t, configPath, blobber.ID, 1e9)
				require.Nil(t, err, "Error resetting blobber write prices")
			}
		}
	})

	t.RunSequentiallyWithTimeout("Check token accounting of a blobber replacing in allocation, should work", time.Minute*15, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		// Set up allocation and get a random blobber to remove
		params := map[string]interface{}{
			"size":   1 * GB,
			"data":   2,
			"parity": 2,
			"lock":   "0.2",
		}

		allocationID := utils.SetupEnterpriseAllocation(t, configPath, params)

		// Fetch allocation details before replacement
		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		// Initial amount locked to allocation (0.2 ZCN)
		amountTotalLockedToAlloc := int64(2e9) // 0.2 ZCN

		waitForTimeInMinutesWhileLogging(t, 5)

		// Get the blobber to remove (first blobber in allocation)
		blobberToRemove := beforeAlloc.BlobberDetails[0].BlobberID

		// Calculate the total stake of the blobber to remove
		var prevReplaceeBlobberStake int64
		for _, blobber := range beforeAlloc.Blobbers {
			if blobber.ID == blobberToRemove {
				prevReplaceeBlobberStake = blobber.TotalStake
			}
		}

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		// Get new blobber details and generate authorization ticket
		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "Error no blobber details found")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		// Update allocation to replace blobber
		updateParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, updateParams, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Fetch updated allocation details
		updatedAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, updatedAlloc)

		// Validate that the write pool balance matches the expected value
		require.InEpsilon(t, amountTotalLockedToAlloc, updatedAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		txn, err := getTransacionFromSingleSharder(t, updatedAlloc.Tx)
		require.Nil(t, err, "Error getting update allocation txn from sharders")

		// Query blobber rewards
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobberToRemove, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		// Calculate expected payment to replaced blobber
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := int64(txn.CreationDate) - updatedAlloc.StartTime
		expectedPaymentToReplacedBlobber := 5e8 * durationOfUsedInSeconds / timeUnitInSeconds

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToReplacedBlobber)

		// Validate enterprise rewards and write pool balance
		require.InEpsilon(t, expectedPaymentToReplacedBlobber, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match after blobber replacement")

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
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		// Setup allocation with initial parameters
		params := map[string]interface{}{
			"size":   1 * GB,
			"data":   2,
			"parity": 2,
			"lock":   "0.2",
		}

		allocationID := utils.SetupEnterpriseAllocation(t, configPath, params)

		// Fetch allocation details before replacement
		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		// Wait for 5 minutes to simulate usage
		waitForTimeInMinutesWhileLogging(t, 5)

		// Initial amount locked to allocation (0.2 ZCN)
		amountTotalLockedToAlloc := int64(2e9) // 0.2 ZCN

		// Get the blobber to remove (first blobber in allocation)
		blobberToRemove := beforeAlloc.BlobberDetails[0].BlobberID

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		// Get details of a new blobber not part of the current allocation
		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "No blobber details found")

		// Get the original write price of the blobber
		originalPrice := currentBlobberDetails.Terms.WritePrice

		// Update the blobber price to the same value (no change)
		err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
		require.Nil(t, err, "Error updating blobber price")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		// Prepare parameters for allocation update
		updateParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		// Perform the allocation update to replace the blobber
		output, err := updateAllocation(t, configPath, updateParams, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Fetch the updated allocation details
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, afterAlloc, "Updated allocation should not be nil")

		// Log the expected and actual write pool balances
		t.Logf("Expected write pool %d actual write pool %d", amountTotalLockedToAlloc, afterAlloc.WritePool)

		// Check that the write pool balance matches the expected value
		require.InEpsilon(t, amountTotalLockedToAlloc, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		// Retrieve the transaction details for further analysis
		txn, err := getTransacionFromSingleSharder(t, afterAlloc.Tx)
		require.Nil(t, err, "Error getting transaction from sharder")

		// Query blobber rewards to validate the enterprise blobber rewards calculation
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobberToRemove, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		// Calculate the expected payment to the replaced blobber based on the time used and the same blobber price
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := int64(txn.CreationDate) - afterAlloc.StartTime
		expectedPaymentToReplacedBlobber := 5e8 * durationOfUsedInSeconds / timeUnitInSeconds

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToReplacedBlobber)

		// Validate the enterprise rewards after blobber replacement
		require.InEpsilon(t, expectedPaymentToReplacedBlobber, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match after blobber replacement")

		// Reset the blobber price after the test and clean up
		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")

			output, err = cancelAllocation(t, configPath, allocationID, true)
			require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
			require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allocation fail", strings.Join(output, "\n"))
		}()
	})

	t.RunSequentiallyWithTimeout("Replace blobber with 0.5x price should work", time.Minute*15, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		params := map[string]interface{}{
			"size":   1 * GB,
			"data":   2,
			"parity": 2,
			"lock":   "0.2",
		}

		allocationID := utils.SetupEnterpriseAllocation(t, configPath, params)

		// allocaiton
		alloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, alloc, "Error fetching allocation")

		blobberToRemove := alloc.BlobberDetails[0].BlobberID

		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		waitForTimeInMinutesWhileLogging(t, 5)

		amountTotalLockedToAlloc := int64(1e9) * 2

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "Error no blobber details found")

		halfPrice := currentBlobberDetails.Terms.WritePrice / 2
		amountTotalLockedToAlloc += 0.05 * 1e10

		// Update blobber price
		originalPrice := currentBlobberDetails.Terms.WritePrice
		err = updateBlobberPrice(t, configPath, addBlobberID, halfPrice)
		require.Nil(t, err, "Error updating blobber price")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		updateParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
			"lock":                    "0.05",
		})

		output, err := updateAllocation(t, configPath, updateParams, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, afterAlloc, "Updated allocation should not be nil")

		t.Logf("Expected write pool %d actual write pool %d", amountTotalLockedToAlloc, afterAlloc.WritePool)

		require.InEpsilon(t, amountTotalLockedToAlloc, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		txn, err := getTransacionFromSingleSharder(t, afterAlloc.Tx)
		require.Nil(t, err, "Error getting transaction from sharder")

		// Query blobber rewards
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobberToRemove, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		// Verify blobber rewards calculation
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := int64(txn.CreationDate) - afterAlloc.StartTime
		expectedPaymentToReplacedBlobber := 5e8 * durationOfUsedInSeconds / timeUnitInSeconds

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToReplacedBlobber)

		// Validate the enterprise rewards after blobber replacement
		require.InEpsilon(t, expectedPaymentToReplacedBlobber, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match after blobber replacement")

		// Reset blobber price
		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")

			output, err = cancelAllocation(t, configPath, allocationID, true)
			require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
			require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
		}()
	})

	t.RunSequentiallyWithTimeout("Replace blobber with 2x price should work", time.Minute*15, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		// Setup allocation with initial parameters
		params := map[string]interface{}{
			"size":   1 * GB,
			"data":   2,
			"parity": 2,
			"lock":   "0.2",
		}

		allocationID := utils.SetupEnterpriseAllocation(t, configPath, params)

		// Fetch allocation details before replacement
		beforeAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, beforeAlloc, "Allocation should not be nil")

		// Wait for 5 minutes to simulate usage
		waitForTimeInMinutesWhileLogging(t, 5)

		// Initial amount locked to allocation (0.2 ZCN)
		amountTotalLockedToAlloc := int64(2e9) // 0.2 ZCN

		// Get the blobber to remove (first blobber in allocation)
		alloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, alloc, "Error fetching allocation")
		blobberToRemove := alloc.BlobberDetails[0].BlobberID

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		// Get details of a new blobber not part of the current allocation
		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		currentBlobberDetails, err := utils.GetBlobberDetails(t, configPath, addBlobberID)
		require.Nil(t, err, "Error fetching blobber details")
		require.NotNil(t, currentBlobberDetails, "No blobber details found")

		// Calculate double the current blobber's write price
		doublePrice := currentBlobberDetails.Terms.WritePrice * 2

		// Increment the amount locked by 0.1 ZCN
		amountTotalLockedToAlloc += int64(0.1 * 1e10) // 0.1 ZCN

		// Update the blobber price to double of its original price
		originalPrice := currentBlobberDetails.Terms.WritePrice
		err = updateBlobberPrice(t, configPath, addBlobberID, doublePrice)
		require.Nil(t, err, "Error updating blobber price")

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		// Prepare parameters for allocation update
		updateParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
			"lock":                    "0.1",
		})

		// Perform the allocation update to replace the blobber
		output, err := updateAllocation(t, configPath, updateParams, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Fetch the updated allocation details
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, afterAlloc, "Updated allocation should not be nil")

		// Log the expected and actual write pool balances
		t.Logf("Expected write pool %d actual write pool %d", amountTotalLockedToAlloc, afterAlloc.WritePool)

		// Check that the write pool balance matches the expected value
		require.InEpsilon(t, amountTotalLockedToAlloc, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match after blobber replacement")

		// Retrieve the transaction details for further analysis
		txn, err := getTransacionFromSingleSharder(t, afterAlloc.Tx)
		require.Nil(t, err, "Error getting transaction from sharder")

		// Query blobber rewards to validate the enterprise blobber rewards calculation
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobberToRemove, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		// Calculate the expected payment to the replaced blobber based on the time used and the double blobber price
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := int64(txn.CreationDate) - afterAlloc.StartTime
		expectedPaymentToReplacedBlobber := 5e8 * durationOfUsedInSeconds / timeUnitInSeconds

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToReplacedBlobber)

		// Validate the enterprise rewards after blobber replacement
		require.InEpsilon(t, expectedPaymentToReplacedBlobber, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match after blobber replacement")

		// Reset the blobber price after the test and clean up
		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")

			output, err = cancelAllocation(t, configPath, allocationID, true)
			require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
			require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allocation fail", strings.Join(output, "\n"))
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

func setupAllocationAndGetRandomBlobber(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) (allocationId, randomBlobberId string) {
	utils.SetupWalletWithCustomTokens(t, cliConfigFilename, 10)

	options := map[string]interface{}{
		"data":   2,
		"parity": 2,
		"size":   1 * GB,
		"lock":   "0.2",
	}

	for _, params := range extraParams {
		for k, v := range params {
			options[k] = v
		}
	}

	allocationID := utils.SetupEnterpriseAllocation(t, cliConfigFilename, options)

	wd, _ := os.Getwd()
	walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
	configFile := filepath.Join(wd, "config", cliConfigFilename)

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
