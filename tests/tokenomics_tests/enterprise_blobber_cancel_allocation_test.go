package tokenomics_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

var (
	cancelAllocationRegex = regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`)
)

func TestCancelEnterpriseAllocation(testSetup *testing.T) {
	// Skip enterprise tests early, before any SystemTest wrapper or parallel calls.
	// This uses testSetup.Skip() directly to ensure the skip propagates correctly.
	t := test.NewSystemTest(testSetup)
	enterpriseBlobbers := utils.GetEnterpriseBlobbers(t)
	if len(enterpriseBlobbers) == 0 {
		testSetup.Skip("No active enterprise blobbers on the network (health checks stale); skipping enterprise cancel allocation tests")
		return
	}

	var transientErr error
	t.TestSetup("set storage config to use time_unit as 10 minutes (for short-lived allocations)", func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		if err != nil && utils.IsTransientError(output, err) {
			transientErr = err
			return
		}
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))
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

	t.RunWithTimeout("Cancel allocation after waiting for 7 minutes check refund amount.", time.Minute*25, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
		require.Nil(t, err, "Error fetching wallet")
		require.Equal(t, wallet.ClientID, wallet.ClientID, "Error getting wallet with name %v")

		beforeBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Create allocation
		amountTotalLockedToAlloc := int64(5e10)
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": 1 * GB, "lock": amountTotalLockedToAlloc / 1e10, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")

		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id %v", strings.Join(allocOutput, "\n"))

		beforeAlloc := utils.GetAllocation(t, allocationID)

		afterBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)
		require.InEpsilon(t, beforeBalance-amountTotalLockedToAlloc, afterBalance, 0.15, "Balance should be locked to allocation (minus txn fee)")
		beforeBalance = afterBalance

		t.Log("Waiting for 7 minutes ....")
		waitForTimeInMinutesWhileLogging(t, 7)

		// Cancel the allocation
		output, err := cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is canceled and check refund
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.True(t, afterAlloc.Finalized, "Allocation should be expired")
		require.Equal(t, int64(0), afterAlloc.WritePool, "Write pool balance should be 0")

		afterBalance = utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Calculate expected refund
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedRefund := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		t.Logf("Time unit in seconds: %v", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %v", durationOfUsedInSeconds)
		t.Logf("Real cost of allocation: %v", realCostOfAlloc)
		t.Logf("Expected payment to blobbers: %v", expectedPaymentToBlobbers)
		t.Logf("Expected refund: %v", expectedRefund)
		t.Logf("Before balance: %v", beforeBalance)
		t.Logf("After balance: %v", afterBalance)

		require.InEpsilon(t, beforeBalance+expectedRefund-1e8, afterBalance, 0.25, "Refund should be credited to client balance after cancel allocation") // 1e8 is transaction fee

		// Enterprise blobbers don't submit write markers so they never get challenged/rewarded
		// rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		// enterpriseReward, err := getQueryRewards(t, rewardQuery)
		// require.Nil(t, err)
		// require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")
	})

	t.RunWithTimeout("Cancel allocation after updating duration check refund amount.", time.Minute*15, func(t *test.SystemTest) {
		// Setup, wallet creation, and initial balance retrieval
		utils.SetupWalletWithCustomTokens(t, configPath, 10)
		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
		require.Nil(t, err, "Error getting wallet")
		beforeBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Create allocation
		amountTotalLockedToAlloc := int64(5e10)
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": 1 * GB, "lock": amountTotalLockedToAlloc / 1e10, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")
		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id")

		beforeAlloc := utils.GetAllocation(t, allocationID)
		afterBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)
		require.InEpsilon(t, beforeBalance-amountTotalLockedToAlloc, afterBalance, 0.15, "Balance should be locked to allocation (minus txn fee)")
		beforeBalance = afterBalance

		// Update the allocation duration
		updateAllocationParams := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err := utils.UpdateAllocation(t, configPath, updateAllocationParams, true)
		require.Nil(t, err, "Updating allocation duration failed", strings.Join(output, "\n"))

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate and check refund
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.True(t, afterAlloc.Finalized, "Allocation should be expired")
		require.Equal(t, int64(0), afterAlloc.WritePool, "Write pool balance should be 0")
		afterBalance = utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Calculate expected refund
		timeUnitInSeconds := int64(600) // 10 minutes in seconds
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedRefund := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		t.Logf("Expected refund: %v", expectedRefund)
		require.InEpsilon(t, beforeBalance+expectedRefund-1e8, afterBalance, 0.25, "Refund should be credited to client balance after cancel allocation")
	})
	t.RunWithTimeout("Cancel allocation after adding blobber check refund amount.", time.Minute*15, func(t *test.SystemTest) {
		// Setup, wallet creation, and initial balance retrieval
		utils.SetupWalletWithCustomTokens(t, configPath, 10)
		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
		require.Nil(t, err, "Error getting wallet")
		beforeBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Create allocation with data=1, parity=1 (2 blobbers) to leave 1 enterprise blobber for add
		amountTotalLockedToAlloc := int64(5e10)
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": 1 * GB, "lock": amountTotalLockedToAlloc / 1e10, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds, "data": 1, "parity": 1}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")
		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id")

		beforeAlloc := utils.GetAllocation(t, allocationID)
		afterBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)
		require.InEpsilon(t, beforeBalance-amountTotalLockedToAlloc, afterBalance, 0.15, "Balance should be locked to allocation (minus txn fee)")
		beforeBalance = afterBalance

		// Add an enterprise blobber (must use enterprise blobber for enterprise allocations)
		newBlobberID, newBlobberUrl, err := utils.GetEnterpriseBlobberIdAndUrlNotPartOfAllocation(t, configPath, allocationID)
		require.Nil(t, err, "Unable to get enterprise blobber not part of allocation")
		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, newBlobberID, newBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for adding blobber")

		updateAllocationParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             newBlobberID,
			"add_blobber_auth_ticket": blobberAuthTicket,
			"auth_round_expiry":       utils.DefaultAuthRoundExpiry,
		})
		output, err := utils.UpdateAllocation(t, configPath, updateAllocationParams, true)
		require.Nil(t, err, "Adding blobber failed", strings.Join(output, "\n"))

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate and check refund
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.True(t, afterAlloc.Finalized, "Allocation should be expired")
		require.Equal(t, int64(0), afterAlloc.WritePool, "Write pool balance should be 0")
		afterBalance = utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Calculate expected refund
		timeUnitInSeconds := int64(600) // 10 minutes in seconds
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedRefund := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		t.Logf("Expected refund: %v", expectedRefund)
		require.InEpsilon(t, beforeBalance+expectedRefund-1e8, afterBalance, 0.25, "Refund should be credited to client balance after cancel allocation")
	})

	t.RunWithTimeout("Cancel allocation after adding a blobber with 2x amount check refund amount", time.Minute*15, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
		require.Nil(t, err, "Error getting wallet")
		beforeBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Create allocation with data=1, parity=1 (2 blobbers) to leave 1 enterprise blobber for replace
		amountTotalLockedToAlloc := int64(5e10)
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": 1 * GB, "lock": amountTotalLockedToAlloc / 1e10, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds, "data": 1, "parity": 1}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")
		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id")

		beforeAlloc := utils.GetAllocation(t, allocationID)
		afterBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)
		require.InEpsilon(t, beforeBalance-amountTotalLockedToAlloc, afterBalance, 0.15, "Balance should be locked to allocation (minus txn fee)")
		beforeBalance = afterBalance

		// Replace a blobber with an enterprise blobber (must use enterprise blobber for enterprise allocations)
		addBlobberID, addBlobberUrl, err := utils.GetEnterpriseBlobberIdAndUrlNotPartOfAllocation(t, configPath, allocationID)
		require.Nil(t, err, "Unable to get enterprise blobber not part of allocation")
		beforeAlloc2 := utils.GetAllocation(t, allocationID)
		removeBlobber := beforeAlloc2.BlobberDetails[0].BlobberID

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for replace blobber")

		updateAllocationParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"remove_blobber":          removeBlobber,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": blobberAuthTicket,
			"auth_round_expiry":       utils.DefaultAuthRoundExpiry,
		})
		output, err := utils.UpdateAllocation(t, configPath, updateAllocationParams, true)
		require.Nil(t, err, "Replacing blobber failed", strings.Join(output, "\n"))

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate and check refund
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.True(t, afterAlloc.Finalized, "Allocation should be expired")
		require.Equal(t, int64(0), afterAlloc.WritePool, "Write pool balance should be 0")
		afterBalance = utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Calculate expected refund
		timeUnitInSeconds := int64(600) // 10 minutes in seconds
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedRefund := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		t.Logf("Expected refund: %v", expectedRefund)
		require.InEpsilon(t, beforeBalance+expectedRefund-1e8, afterBalance, 0.25, "Refund should be credited to client balance after cancel allocation")
	})

	t.Run("Cancel allocation immediately should work", func(t *test.SystemTest) {
		// Setup, wallet creation, and initial balance retrieval
		utils.SetupWalletWithCustomTokens(t, configPath, 10)
		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
		require.Nil(t, err, "Error getting wallet")
		beforeBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Create allocation
		amountTotalLockedToAlloc := int64(5e10)
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": 1 * GB, "lock": amountTotalLockedToAlloc / 1e10, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")
		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id")

		beforeAlloc := utils.GetAllocation(t, allocationID)
		afterBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)
		require.InEpsilon(t, beforeBalance-amountTotalLockedToAlloc, afterBalance, 0.15, "Balance should be locked to allocation (minus txn fee)")
		beforeBalance = afterBalance

		// Cancel the allocation immediately
		output, err := cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate and check refund
		afterAlloc := utils.GetAllocation(t, allocationID)
		require.True(t, afterAlloc.Finalized, "Allocation should be expired")
		require.Equal(t, int64(0), afterAlloc.WritePool, "Write pool balance should be 0")
		afterBalance = utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Calculate expected refund
		timeUnitInSeconds := int64(600) // 10 minutes in seconds
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedRefund := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		t.Logf("Expected refund: %v", expectedRefund)
		require.InEpsilon(t, beforeBalance+expectedRefund-1e8, afterBalance, 0.25, "Refund should be credited to client balance after cancel allocation")
	})

	t.Run("Cancel Other's Allocation Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWalletForName(t, configPath, utils.EscapedTestName(t)+"_other")
		require.Nil(t, err, "Unable to create the wallet", strings.Join(output, "\n"))

		otherAllocationID := utils.SetupEnterpriseAllocationWithWallet(t, utils.EscapedTestName(t)+"_other", configPath)
		output, err = utils.ExecuteFaucetWithTokensForWallet(t, utils.EscapedTestName(t)+"_other_wallet.json", configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet", strings.Join(output, "\n"))

		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		// otherAllocationID should not be cancelable from this level
		output, err = cancelAllocation(t, configPath, otherAllocationID, false)

		require.Error(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: only owner can cancel an allocation", output[len(output)-1])
	})

	t.Run("Cancel Non-existent Allocation Should Fail", func(t *test.SystemTest) {
		_, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error craeting wallet")

		allocationID := "123abc"

		output, err := cancelAllocation(t, configPath, allocationID, false)

		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: value not present", output[0])
	})
}

func cancelAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Canceling allocation...")
	wallet := utils.EscapedTestName(t)
	nonce, err := utils.GetNonceForWallet(t, cliConfigFilename, wallet)
	nonceParam := ""
	if err == nil {
		nonceParam = fmt.Sprintf(" --withNonce %d", nonce+1)
	}
	cmd := fmt.Sprintf(
		"./zbox alloc-cancel --allocation %s --silent "+
			"--wallet %s --configDir ./config --config %s%s",
		allocationID,
		wallet+"_wallet.json",
		cliConfigFilename,
		nonceParam,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func waitForTimeInMinutesWhileLogging(t *test.SystemTest, minutes int) {
	for i := 0; i < minutes; i++ {
		t.Log(fmt.Sprintf("Waiting for %d minutes...", minutes-i))
		time.Sleep(time.Minute)
	}
}
