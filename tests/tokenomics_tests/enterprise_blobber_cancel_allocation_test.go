package tokenomics_tests

import (
	"fmt"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/cli_tests"
	"os"
	"path/filepath"
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
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.TestSetup("set storage config to use time_unit as 10 minutes", func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))
	})

	t.Cleanup(func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "1h",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))
	})

	// Make sure you check refund amount in every test

	// 1. Change time_unit to 10 minutes. Create allocation. Wait for 7 minutes. Cancel allocation. Check refund amount.
	// 2. Change time_unit to 10 minutes. Create allocation. Wait for 7 minutes. Update the allocation duration in one test case, in 2nd test case update size. Check refund amount.
	// 3. Change time_unit to 10 minutes. Create allocation. Wait for 7 minutes. Add blobber to allocation. Cancel allocation. Check refund amount.
	// 4. Same process for replace blobber with 2x price. Check refund amount.

	t.RunWithTimeout("Cancel allocation after waiting for 7 minutes check refund amount.", time.Minute*15, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
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
		require.Equal(t, beforeBalance-amountTotalLockedToAlloc-1e8, afterBalance, "Balance should be locked to allocation") // 1e8 is transaction fee
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
		realCostOfAlloc := costOfAlloc(beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedRefund := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		t.Logf("Time unit in seconds: %v", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %v", durationOfUsedInSeconds)
		t.Logf("Real cost of allocation: %v", realCostOfAlloc)
		t.Logf("Expected payment to blobbers: %v", expectedPaymentToBlobbers)
		t.Logf("Expected refund: %v", expectedRefund)
		t.Logf("Before balance: %v", beforeBalance)
		t.Logf("After balance: %v", afterBalance)

		require.InEpsilon(t, beforeBalance+expectedRefund-1e8, afterBalance, 0.01, "Refund should be credited to client balance after cancel allocation") // 1e8 is transaction fee

		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")
	})

	t.RunWithTimeout("Cancel allocation after updating duration check refund amount.", time.Minute*15, func(t *test.SystemTest) {
		// Setup: Change time_unit to 10 minutes
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create a wallet
		output, err = utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet", strings.Join(output, "\n"))

		// Execute faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		// Generate blobber auth tickets
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		balanceBeforeCreatingAllocation, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching wallet balance")

		// Create allocation
		params := map[string]interface{}{"size": "10000", "lock": "5", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")
		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))

		// Wait for 7 minutes
		t.Log("Waiting for 7 minutes ....")
		//time.Sleep(7 * time.Minute)

		balanceAfterCreatingAllocation, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching wallet balance")

		// Update the allocation duration
		updateAllocationParams := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err = utils.UpdateAllocation(t, configPath, updateAllocationParams, true)
		require.Nil(t, err, "Updating allocation duration failed", strings.Join(output, "\n"))

		balanceBefore, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching wallet balance", err)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is canceled and check refund
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		balanceAfter, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching wallet balance", err)
		refundAmount := balanceAfter - balanceBefore

		expectedRefund := balanceBeforeCreatingAllocation - balanceAfterCreatingAllocation
		require.Nil(t, err, "Error getting allocation cost", strings.Join(allocOutput, "\n"))

		require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
	})
	t.RunWithTimeout("Cancel allocation after adding blobber check the refund amount.", time.Minute*15, func(t *test.SystemTest) {
		// Setup: Change time_unit to 10 minutes
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet %v", strings.Join(output, "\n"))

		// Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create an allocation
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": "10000", "lock": "5", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocationForWallet(t, utils.EscapedTestName(t), configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")
		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error getting allocation ID")

		// Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		t.Log("Waiting for 7 minutes")
		//time.Sleep(7 * time.Minute) // Wait for 7 minutes

		// Retrieve a new blobber ID to add to the allocation
		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		newBlobberID, newBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err, "Unable to get blobber not part of allocation")

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, newBlobberID, newBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for adding blobber")

		updateAllocationParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             newBlobberID,
			"add_blobber_auth_ticket": blobberAuthTicket,
		})
		output, err = utils.UpdateAllocation(t, configPath, updateAllocationParams, true)
		require.Nil(t, err, "Adding blobber failed", strings.Join(output, "\n"))

		// Get the allocation details before cancellation
		allocationBefore := utils.GetAllocation(t, allocationID)
		beforeBalance, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching balance before cancel", beforeBalance)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is canceled and check refund
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		afterBalance, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching balance after cancel", afterBalance)

		refundAmount := beforeBalance - afterBalance
		expectedRefund := calculateExpectedRefund(allocationBefore, beforeBalance, afterBalance)

		require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
	})

	t.RunWithTimeout("Cancel allocation after adding a blobber with 2x amount check refund amount", time.Minute*15, func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet %v", strings.Join(output, "\n"))
		// Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		// Setup: Change time_unit to 10 minutes
		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create an allocation
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": "10000", "lock": "5", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")
		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error getting allocation ID")

		// Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		t.Log("Waiting for 7 minutes")
		//time.Sleep(7 * time.Minute)

		// Replace a blobber with another blobber with a higher price
		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		removeBlobber, err := cli_tests.GetRandomBlobber(walletFile, configFile, allocationID, addBlobberID)
		require.Nil(t, err)

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for replace blobber")

		updateAllocationParams := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"remove_blobber":          removeBlobber,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": blobberAuthTicket,
		})
		output, err = utils.UpdateAllocation(t, configPath, updateAllocationParams, true)
		require.Nil(t, err, "Replacing blobber failed", strings.Join(output, "\n"))

		// Get the allocation details before cancellation
		allocationBefore := utils.GetAllocation(t, allocationID)
		beforeBalance, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching balance before cancel", beforeBalance)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is canceled and check refund
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		afterBalance, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching balance after cancel", afterBalance)

		refundAmount := beforeBalance - afterBalance
		expectedRefund := calculateExpectedRefund(allocationBefore, beforeBalance, afterBalance)

		require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
	})

	t.Run("Cancel allocation immediately should work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet", strings.Join(output, "\n"))

		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
		require.Nil(t, err, "Error getting wallet with name %v")

		t.Log("Client id is" + wallet.ClientID)

		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		params := map[string]interface{}{"size": "10000", "lock": "5", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}

		output, err = utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")

		allocationID, err := utils.GetAllocationID(strings.Join(output, "\n"))
		require.Nil(t, err, "Error getting allocation id")

		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.NoError(t, err, "cancel allocation failed but should succeed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])
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

func calculateExpectedRefund(before climodel.Allocation, beforeBalance, afterBalance float64) interface{} {
	return beforeBalance - afterBalance
}

func cancelAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Canceling allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-cancel --allocation %s --silent "+
			"--wallet %s --configDir ./config --config %s",
		allocationID,
		utils.EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
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
