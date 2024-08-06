package tokenomics_tests

import (
	"fmt"
	climodel "github.com/0chain/system_test/internal/cli/model"
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

		// Create an allocation
		allocationID := utils.SetupEnterpriseAllocation(t, configPath)

		//Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing facuet transaction", strings.Join(output, "\n"))

		t.Log("Waiting for 7 minutes")
		time.Sleep(7 * time.Minute) // Wait for 7 minutes

		// Get the allocation details before cancellation
		//allocBefore := utils.GetAllocation(t, allocationID)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is canceled and check refund
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		//allocAfter := utils.GetAllocation(t, allocationID)
		//refundAmount := allocBefore.WritePool - allocAfter.WritePool
		//TODO: create cost funciton
		//expectedRefund := calculateExpectedRefund(allocBefore, time.Minute*7)

		//require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
	})
	t.RunWithTimeout("Cancel allocation after updating duration check refund amount.", time.Minute*15, func(t *test.SystemTest) {
		// Setup: Change time_unit to 10 minutes
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create an allocation
		allocationID := utils.SetupEnterpriseAllocation(t, configPath)
		//Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing facuet transaction", strings.Join(output, "\n"))

		t.Log("Waiting for 7 minutes")
		time.Sleep(7 * time.Minute) // Wait for 7 minutes

		// Update the allocation duration
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err = utils.UpdateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Updating allocation duration failed", strings.Join(output, "\n"))

		// Get the allocation details before cancellation
		//allocBefore := utils.GetAllocation(t, allocationID)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is canceled and check refund
		utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		//TODO: create cost funciton
		//allocAfter := utils.GetAllocation(t, allocationID)
		//refundAmount := allocBefore.WritePool - allocAfter.WritePool
		//expectedRefund := calculateExpectedRefund(allocBefore, time.Minute*7)
		//
		//require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
	})
	t.RunWithTimeout("Cancel allocation after adding blobber check the refund amount.", time.Minute*15, func(t *test.SystemTest) {
		// Setup: Change time_unit to 10 minutes
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create an allocation
		allocationID := utils.SetupEnterpriseAllocation(t, configPath)

		//Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing facuet transaction", strings.Join(output, "\n"))

		t.Log("Waiting for 7 minutes")
		time.Sleep(7 * time.Minute) // Wait for 7 minutes

		// Add a new blobber
		newBlobberID := "new_blobber_id"
		params := createParams(map[string]interface{}{
			"allocation":  allocationID,
			"add_blobber": newBlobberID,
		})
		output, err = utils.UpdateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Adding blobber failed", strings.Join(output, "\n"))

		// Get the allocation details before cancellation
		//allocBefore := utils.GetAllocation(t, allocationID)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		//TODO: create cost funciton
		// Validate that the allocation is canceled and check refund
		//utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])
		//
		//allocAfter := utils.GetAllocation(t, allocationID)
		//refundAmount := allocBefore.WritePool - allocAfter.WritePool
		//expectedRefund := calculateExpectedRefund(allocBefore, time.Minute*7)

		//require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
	})
	t.RunWithTimeout("Cancel allocation after adding a blobber with 2x amount check refund amount", time.Minute*15, func(t *test.SystemTest) {
		// Setup: Change time_unit to 10 minutes
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "10m",
		}, true)
		require.Nil(t, err, "Error updating sc config", strings.Join(output, "\n"))

		// Create an allocation
		allocationID := utils.SetupEnterpriseAllocation(t, configPath)

		//Faucet transaction
		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing facuet transaction", strings.Join(output, "\n"))

		t.Log("Waiting for 7 minutes")
		time.Sleep(7 * time.Minute)

		// TODO: Replace a blobber with another blobber with a higher price
		oldBlobberID := "old_blobber_id"
		newBlobberID := "new_blobber_id"
		params := createParams(map[string]interface{}{
			"allocation":     allocationID,
			"remove_blobber": oldBlobberID,
			"add_blobber":    newBlobberID,
		})
		output, err = utils.UpdateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Replacing blobber failed", strings.Join(output, "\n"))

		// Get the allocation details before cancellation
		//allocBefore := utils.GetAllocation(t, allocationID)

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Cancel allocation failed", strings.Join(output, "\n"))

		//TODO: create cost funciton
		// Validate that the allocation is canceled and check refund
		//utils.AssertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])

		//allocAfter := utils.GetAllocation(t, allocationID)
		//refundAmount := allocBefore.WritePool - allocAfter.WritePool
		//expectedRefund := calculateExpectedRefund(allocBefore, time.Minute*7)

		//require.InEpsilon(t, expectedRefund, refundAmount, 0.05, "Refund amount is not as expected")
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

		output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		//TODO: add seperate allocation parsing here.
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
