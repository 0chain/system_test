package tokenomics_tests

import (
	"fmt"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/cli_tests"
	"math"
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

	// Make sure you check refund amount in every test

	// 1. Change time_unit to 10 minutes. Create allocation. Wait for 7 minutes. Cancel allocation. Check refund amount.
	// 2. Change time_unit to 10 minutes. Create allocation. Wait for 7 minutes. Update the allocation duration in one test case, in 2nd test case update size. Check refund amount.
	// 3. Change time_unit to 10 minutes. Create allocation. Wait for 7 minutes. Add blobber to allocation. Cancel allocation. Check refund amount.
	// 4. Same process for replace blobber with 2x price. Check refund amount.
	t.RunWithTimeout("Cancel allocation after waiting for 7 minutes check refund amount.", time.Minute*15, func(t *test.SystemTest) {
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
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t)

		balanceBeforeCreatingAllocation, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching wallet balance")

		// Create allocation
		params := map[string]interface{}{"size": "10000", "lock": "5", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")

		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id %v", strings.Join(allocOutput, "\n"))

		//allocCost, err := getAllocationCost(strings.Join(allocOutput, "\n"))
		//require.Nil(t, err, "Error fetching allocation cost %v", strings.Join(allocOutput, "\n"))

		balanceBefore, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching wallet balance", err)

		// Wait for 7 minutes
		t.Log("Waiting for 7 minutes ....")
		//time.Sleep(7 * time.Minute)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is canceled and check refund
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		balanceAfter, err := utils.GetBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching wallet balance", err)
		refundAmount := balanceBefore - balanceAfter

		expectedRefund := balanceBeforeCreatingAllocation - balanceBefore

		require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
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
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t)

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
		refundAmount := balanceBefore - balanceAfter

		expectedRefund := balanceBeforeCreatingAllocation - balanceAfterCreatingAllocation
		require.Nil(t, err, "Error getting allocation cost", strings.Join(allocOutput, "\n"))

		require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
	})
	t.RunWithTimeout("Cancel allocation after adding blobber check the refund amount.", time.Minute*15, func(t *test.SystemTest) {
		// Setup: Change time_unit to 10 minutes
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet %v", strings.Join(output, "\n"))

		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create an allocation
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t)
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

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, newBlobberID, newBlobberUrl)
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

		// Setup: Change time_unit to 10 minutes
		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create an allocation
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t)
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

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, addBlobberID, addBlobberUrl)
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

		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t)

		var allocation climodel.Allocation

		params := map[string]interface{}{"size": "10000", "lock": "5", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}

		output, err = utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")

		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 100000000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		require.NotNil(t, allocation, "Allocation id is nil")

		allocationID := allocation.ID

		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.NoError(t, err, "cancel allocation failed but should succeed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])
	})

	t.Run("Cancel Other's Allocation Should Fail", func(t *test.SystemTest) {
		otherAllocationID := utils.SetupEnterpriseAllocationWithWallet(t, utils.EscapedTestName(t)+"_other", configPath)
		output, err := utils.ExecuteFaucetWithTokensForWallet(t, utils.EscapedTestName(t)+"_other_wallet.json", configPath, 1000)
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
	return math.Ceil(beforeBalance) - math.Ceil(afterBalance)
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
