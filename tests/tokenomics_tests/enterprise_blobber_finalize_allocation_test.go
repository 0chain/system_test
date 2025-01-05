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
	finalizeAllocationRegex = regexp.MustCompile(`^Allocation finalized with txId : [a-f0-9]{64}$`)
)

func TestFinalizeEnterpriseAllocation(testSetup *testing.T) {
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

	t.RunWithTimeout("Finalize allocation after waiting for 7 minutes check finalization and balance.", time.Minute*20, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))
		require.Nil(t, err, "Error getting wallet")
		require.Equal(t, wallet.ClientID, wallet.ClientID, "Error getting wallet with name %v")

		beforeBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Create allocation
		amountTotalLockedToAlloc := int64(2e9)
		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
		params := map[string]interface{}{"size": 1 * GB, "lock": "0.2", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")

		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id %v", strings.Join(allocOutput, "\n"))

		beforeAlloc := utils.GetAllocation(t, allocationID)

		afterBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)
		require.Equal(t, beforeBalance-amountTotalLockedToAlloc-1e8, afterBalance, "Balance should be locked to allocation") // 1e8 is transaction fee
		beforeBalance = afterBalance

		t.Log("Waiting for 11 minutes ....")
		waitForTimeInMinutesWhileLogging(t, 7)

		// Finalize the allocation
		output, err := finalizeAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Finalize allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is finalized and check balance
		utils.AssertOutputMatchesAllocationRegex(t, finalizeAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.True(t, afterAlloc.Finalized, "Allocation should be finalized")
		require.Equal(t, int64(0), afterAlloc.WritePool, "Write pool balance should be 0")

		afterBalance = utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Calculate expected finalization amount
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds

		t.Logf("Time unit in seconds: %v", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %v", durationOfUsedInSeconds)
		t.Logf("Real cost of allocation: %v", realCostOfAlloc)
		t.Logf("Expected payment to blobbers: %v", expectedPaymentToBlobbers)
		t.Logf("Before balance: %v", beforeBalance)
		t.Logf("After balance: %v", afterBalance)

		require.InEpsilon(t, beforeBalance-expectedPaymentToBlobbers-1e8, afterBalance-beforeAlloc.WritePool, 0.01, "Finalization should correctly debit client balance") // 1e8 is transaction fee

		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")
	})

	t.RunWithTimeout("Finalize allocation after updating duration and check finalization and balance.", time.Minute*25, func(t *test.SystemTest) {
		// Setup, wallet creation, and initial balance retrieval
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		wallet, err := utils.GetWalletForName(t, configPath, utils.EscapedTestName(t))

		require.Nil(t, err, "Error getting wallet")
		beforeBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Create allocation
		amountTotalLockedToAlloc := int64(1e9 * 2)

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		params := map[string]interface{}{"data": 3, "parity": 3, "size": 1 * GB, "lock": "0.2", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}

		allocOutput, err := utils.CreateNewEnterpriseAllocation(t, configPath, createParams(params))
		require.Nil(t, err, "Error creating allocation")

		allocationID, err := utils.GetAllocationID(strings.Join(allocOutput, "\n"))
		require.Nil(t, err, "Error fetching allocation id")

		beforeAlloc := utils.GetAllocation(t, allocationID)

		waitForTimeInMinutesWhileLogging(t, 5)

		afterBalance := utils.GetBalanceFromSharders(t, wallet.ClientID)

		require.Equal(t, beforeBalance-amountTotalLockedToAlloc-1e8, afterBalance, "Balance should be locked to allocation")

		requiredWpBalance := 1e9 * 2
		beforeBalance = afterBalance

		// Update the allocation duration
		updateAllocationParams := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
			"lock":       float64(requiredWpBalance) / 1e10,
		})
		output, err := utils.UpdateAllocation(t, configPath, updateAllocationParams, true)
		require.Nil(t, err, "Updating allocation duration failed", strings.Join(output, "\n"))

		waitForTimeInMinutesWhileLogging(t, 7)

		// Finalize the allocation
		output, err = finalizeAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Finalize allocation failed", strings.Join(output, "\n"))

		// Validate that the allocation is finalized and check balance
		utils.AssertOutputMatchesAllocationRegex(t, finalizeAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.True(t, afterAlloc.Finalized, "Allocation should be finalized")
		require.Equal(t, int64(0), afterAlloc.WritePool, "Write pool balance should be 0")

		afterBalance = utils.GetBalanceFromSharders(t, wallet.ClientID)

		// Calculate expected finalization amount
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime
		realCostOfAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfAlloc * durationOfUsedInSeconds / timeUnitInSeconds

		t.Logf("Time unit in seconds: %v", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %v", durationOfUsedInSeconds)
		t.Logf("Real cost of allocation: %v", realCostOfAlloc)
		t.Logf("Expected payment to blobbers: %v", expectedPaymentToBlobbers)
		t.Logf("Before balance: %v", beforeBalance)
		t.Logf("After balance: %v", afterBalance)

		require.InEpsilon(t, beforeBalance-expectedPaymentToBlobbers-1e8, afterBalance-beforeAlloc.WritePool, 0.01, "Finalization should correctly debit client balance") // 1e8 is transaction fee

		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")
	})

	t.Run("Finalize Non-Expired Allocation Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet", strings.Join(output, "\n"))

		allocationID := utils.SetupEnterpriseAllocation(t, configPath)

		output, err = finalizeAllocation(t, configPath, allocationID, false)
		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: allocation is not expired yet", output[0])
	})

	t.Run("Finalize Other's Allocation Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWalletForName(t, configPath, utils.EscapedTestName(t)+"_other")
		require.Nil(t, err, "Unable to create wallet", strings.Join(output, "\n"))

		var otherAllocationID = utils.SetupEnterpriseAllocationWithWallet(t, utils.EscapedTestName(t)+"_other", configPath)

		// create wallet
		_, err = utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet")

		// Then try updating with otherAllocationID: should not work
		output, err = finalizeAllocation(t, configPath, otherAllocationID, false)

		// Error should not be nil since finalize is not working
		require.NotNil(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: not allowed, unknown finalization initiator", output[len(output)-1])
	})

	t.Run("No allocation param should fail", func(t *test.SystemTest) {
		_, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err)

		cmd := fmt.Sprintf(
			"./zbox alloc-fini --silent "+
				"--wallet %s --configDir ./config --config %s",
			utils.EscapedTestName(t)+"_wallet.json",
			configPath,
		)

		output, err := cliutils.RunCommandWithoutRetry(cmd)
		require.Error(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: allocation flag is missing", output[len(output)-1])
	})
}

func finalizeAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Finalizing allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-fini --allocation %s --silent "+
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
