package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/tests/cli_tests"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var (
	updateAllocationRegex = regexp.MustCompile(`^Allocation updated with txId : [a-f0-9]{64}$`)
	repairCompletednRegex = regexp.MustCompile(`Repair file completed, Total files repaired: {2}1`)
)

func costOfAlloc(alloc *climodel.Allocation) int64 {
	cost := float64(0)
	for _, blobber := range alloc.BlobberDetails {
		cost += (sizeInGB(alloc.Size) / float64(alloc.DataShards)) * float64(blobber.Terms.WritePrice)
	}

	return int64(cost)
}

func TestUpdateEnterpriseAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error creating configuration wallet", strings.Join(output, "\n"))

	var blobbersList []climodel.Blobber
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error fetching blobbers %v", strings.Join(output, "\n"))
	require.Len(t, output, 1, "Error wrong json format", strings.Join(output, "\n"))

	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobbersList)

	require.Nil(t, err, "Error decoding blobbers json")

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

		var blobbers []climodel.Blobber
		output, err = utils.ListBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobberes", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Error invalid json length", strings.Join(output, "\n"))

		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobbers)
		require.Nil(t, err, "Error decoding blobbers list", strings.Join(output, "\n"))

		for _, blobber := range blobbers {
			if blobber.Terms.WritePrice != 1e9 {
				err := updateBlobberPrice(t, configPath, blobber.ID, 1e9)
				require.Nil(t, err, "Error resetting blobber write prices")
			}
		}
	})

	t.RunSequentiallyWithTimeout("Blobber price change extend size of allocation", 25*time.Minute, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		amountTotalLockedToAlloc := int64(2e9) // 0.2ZCN
		allocationID := utils.SetupEnterpriseAllocation(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   3,
			"parity": 3,
			"lock":   0.2, // 2GB total size where write price per blobber is 0.1ZCN
		})

		beforeAlloc := utils.GetAllocation(t, allocationID)
		t.Logf("Update 1 Allocation %+v\n", beforeAlloc)

		blobber := beforeAlloc.BlobberDetails[0]
		t.Logf("Blobber old write price %d", blobber.Terms.WritePrice)
		t.Logf("Blobber new write price %d", 2*blobber.Terms.WritePrice)

		err := updateBlobberPrice(t, configPath, blobber.BlobberID, blobber.Terms.WritePrice*2)
		require.Nil(t, err, "Error updating blobber price")

		waitForTimeInMinutesWhileLogging(t, 5)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"lock":       0.5,
			"size":       1 * GB,
		})
		output, err := updateAllocation(t, configPath, params, true)

		amountTotalLockedToAlloc += 5e9 // 0.5ZCN

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		t.Logf("Update 1 immediate Allocation %+v\n", afterAlloc)

		require.Less(t, beforeAlloc.ExpirationDate, afterAlloc.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", beforeAlloc.ExpirationDate, "After:", afterAlloc.ExpirationDate),
		)

		// Verify write pool calculations
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime - timeUnitInSeconds

		realCostOfBeforeAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedWritePoolBalance := amountTotalLockedToAlloc - expectedPaymentToBlobbers
		amountTotalLockedToAlloc = expectedWritePoolBalance
		// Log all values
		t.Logf("Time unit in seconds: %d", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %d", durationOfUsedInSeconds)
		t.Logf("Expire time before: %d", beforeAlloc.ExpirationDate)
		t.Logf("Expire time after: %d", afterAlloc.ExpirationDate)
		t.Logf("Real cost of before alloc: %d", realCostOfBeforeAlloc)
		t.Logf("Expected payment to blobbers: %d", expectedPaymentToBlobbers)
		t.Logf("Expected write pool balance: %d", expectedWritePoolBalance)
		t.Logf("Write pool balance: %d", afterAlloc.WritePool)

		require.InEpsilon(t, expectedWritePoolBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		// Verify blobber rewards calculations
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Logf("Enterprise reward: %+v", enterpriseReward)
		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")

		afterUpdate1Alloc := utils.GetAllocation(t, allocationID)
		t.Logf("Update 1 Allocation %+v\n", afterUpdate1Alloc)

		waitForTimeInMinutesWhileLogging(t, 5)

		// Upgrade 2
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"lock":       0.5,
			"size":       1 * GB,
		})
		output, err = updateAllocation(t, configPath, params, true)
		amountTotalLockedToAlloc += 5e9 // 0.5ZCN

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc = utils.GetAllocation(t, allocationID)
		t.Logf("Update 2 Allocation %+v\n", afterAlloc)

		require.Less(t, afterUpdate1Alloc.ExpirationDate, afterAlloc.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", afterUpdate1Alloc.ExpirationDate, "After:", afterAlloc.ExpirationDate),
		)

		// Verify write pool calculations
		timeUnitInSeconds = int64(600) // 10 minutes
		afterFirstUpdateAllocStartTime := afterUpdate1Alloc.ExpirationDate - timeUnitInSeconds
		durationOfUsedInSeconds = afterAlloc.ExpirationDate - afterFirstUpdateAllocStartTime - timeUnitInSeconds // 50

		realCostOfBeforeAlloc = costOfAlloc(&afterUpdate1Alloc)
		expectedPaymentToBlobbersAfterFirstUpdate := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedPaymentToBlobbers += expectedPaymentToBlobbersAfterFirstUpdate
		expectedWritePoolBalance = amountTotalLockedToAlloc - expectedPaymentToBlobbersAfterFirstUpdate

		// Log all values
		t.Logf("Time unit in seconds: %d", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %d", durationOfUsedInSeconds)
		t.Logf("Expire time before: %d", beforeAlloc.ExpirationDate)
		t.Logf("Expire time after: %d", afterAlloc.ExpirationDate)
		t.Logf("Real cost of before alloc: %d", realCostOfBeforeAlloc)
		t.Logf("Expected payment to blobbers: %d", expectedPaymentToBlobbers)
		t.Logf("Expected write pool balance: %d", expectedWritePoolBalance)
		t.Logf("Write pool balance: %d", afterAlloc.WritePool)

		require.InEpsilon(t, expectedWritePoolBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		// Verify blobber rewards calculations
		rewardQuery = fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err = getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Logf("Enterprise reward: %+v", enterpriseReward)
		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")

		// reset blobber write price.
		err = updateBlobberPrice(t, configPath, blobber.BlobberID, blobber.Terms.WritePrice)
		require.Nil(t, err, "Error resting blobber prices to original")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.RunSequentiallyWithTimeout("Blobber price change extend duration of allocation", 25*time.Minute, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		amountTotalLockedToAlloc := int64(2e9) // 0.2ZCN
		allocationID := utils.SetupEnterpriseAllocation(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   3,
			"parity": 3,
			"lock":   0.2, // 2GB total size where write price per blobber is 0.1ZCN
		})

		beforeAlloc := utils.GetAllocation(t, allocationID)
		t.Logf("Before Allocation %+v\n", beforeAlloc)
		blobber := beforeAlloc.BlobberDetails[0]

		err := updateBlobberPrice(t, configPath, blobber.BlobberID, blobber.Terms.WritePrice*2)
		require.Nil(t, err, "Error updating blobber prices")

		waitForTimeInMinutesWhileLogging(t, 5)

		// First update
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
			"lock":       0.5,
		})
		output, err := updateAllocation(t, configPath, params, true)
		amountTotalLockedToAlloc += 5e9

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.Less(t, beforeAlloc.ExpirationDate, afterAlloc.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", beforeAlloc.ExpirationDate, "After:", afterAlloc.ExpirationDate),
		)

		// Verify write pool calculations
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime - timeUnitInSeconds

		realCostOfBeforeAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedWritePoolBalance := amountTotalLockedToAlloc - expectedPaymentToBlobbers
		amountTotalLockedToAlloc = expectedWritePoolBalance

		// Log all values
		t.Logf("Time unit in seconds: %d", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %d", durationOfUsedInSeconds)
		t.Logf("Expire time before: %d", beforeAlloc.ExpirationDate)
		t.Logf("Expire time after: %d", afterAlloc.ExpirationDate)
		t.Logf("Real cost of before alloc: %d", realCostOfBeforeAlloc)
		t.Logf("Expected payment to blobbers: %d", expectedPaymentToBlobbers)
		t.Logf("Expected write pool balance: %d", expectedWritePoolBalance)
		t.Logf("Write pool balance: %d", afterAlloc.WritePool)

		require.InEpsilon(t, expectedWritePoolBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		// Verify blobber rewards calculations
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Logf("Enterprise reward: %+v", enterpriseReward)
		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")

		afterUpdate1Alloc := utils.GetAllocation(t, allocationID)
		t.Logf("Update 1 Allocation %+v\n", afterUpdate1Alloc)

		waitForTimeInMinutesWhileLogging(t, 5)

		// Second update
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
			"lock":       0.5,
		})
		output, err = updateAllocation(t, configPath, params, true)
		amountTotalLockedToAlloc += 5e9

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc = utils.GetAllocation(t, allocationID)
		t.Logf("Update 2 Allocation %+v\n", afterAlloc)

		require.Less(t, afterUpdate1Alloc.ExpirationDate, afterAlloc.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", afterUpdate1Alloc.ExpirationDate, "After:", afterAlloc.ExpirationDate),
		)

		// Verify write pool calculations
		timeUnitInSeconds = int64(600) // 10 minutes
		afterFirstUpdateAllocStartTime := afterUpdate1Alloc.ExpirationDate - timeUnitInSeconds
		durationOfUsedInSeconds = afterAlloc.ExpirationDate - afterFirstUpdateAllocStartTime - timeUnitInSeconds // 50

		realCostOfBeforeAlloc = costOfAlloc(&afterUpdate1Alloc)
		expectedPaymentToBlobbersAfterSecondUpdate := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedPaymentToBlobbers += expectedPaymentToBlobbersAfterSecondUpdate
		expectedWritePoolBalance = amountTotalLockedToAlloc - expectedPaymentToBlobbersAfterSecondUpdate

		// Log all values
		t.Logf("Time unit in seconds: %d", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %d", durationOfUsedInSeconds)
		t.Logf("Expire time before: %d", afterUpdate1Alloc.ExpirationDate)
		t.Logf("Expire time after: %d", afterAlloc.ExpirationDate)
		t.Logf("Real cost of before alloc: %d", realCostOfBeforeAlloc)
		t.Logf("Expected payment to blobbers: %d", expectedPaymentToBlobbers)
		t.Logf("Expected write pool balance: %d", expectedWritePoolBalance)
		t.Logf("Write pool balance: %d", afterAlloc.WritePool)

		require.InEpsilon(t, expectedWritePoolBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		// Verify blobber rewards calculations
		rewardQuery = fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err = getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Logf("Enterprise reward: %+v", enterpriseReward)
		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")

		// Cleanup
		err = updateBlobberPrice(t, configPath, blobber.BlobberID, blobber.Terms.WritePrice)
		require.Nil(t, err, "Error updating blobber price")

		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Extend duration cost calculation", 15*time.Minute, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		amountTotalLockedToAlloc := int64(2e9) // 0.2ZCN
		allocationID := utils.SetupEnterpriseAllocation(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   3,
			"parity": 3,
			"lock":   0.2, // 2GB total size where write price per blobber is 0.1ZCN
		})

		beforeAlloc := utils.GetAllocation(t, allocationID)

		realCostOfBeforeAlloc := costOfAlloc(&beforeAlloc)

		waitForTimeInMinutesWhileLogging(t, 5)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
			"lock":       0.2, // Locking 0.2 ZCN as we can't calculate the accurate time of update, will validate numbers after updating the alloc
		})
		output, err := updateAllocation(t, configPath, params, true)
		amountTotalLockedToAlloc += 2e9 // 0.2ZCN

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.Less(t, beforeAlloc.ExpirationDate, afterAlloc.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", beforeAlloc.ExpirationDate, "After:", afterAlloc.ExpirationDate),
		)

		// Verify write pool calculations
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime - timeUnitInSeconds

		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedWritePoolBalance := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		// Log all values
		t.Logf("Time unit in seconds: %d", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %d", durationOfUsedInSeconds)
		t.Logf("Expire time before: %d", beforeAlloc.ExpirationDate)
		t.Logf("Expire time after: %d", afterAlloc.ExpirationDate)
		t.Logf("Real cost of before alloc: %d", realCostOfBeforeAlloc)
		t.Logf("Expected payment to blobbers: %d", expectedPaymentToBlobbers)
		t.Logf("Expected write pool balance: %d", expectedWritePoolBalance)
		t.Logf("Write pool balance: %d", afterAlloc.WritePool)

		require.InEpsilon(t, expectedWritePoolBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		// Verify blobber rewards calculations
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Upgrade size cost calculation", 15*time.Minute, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		amountTotalLockedToAlloc := int64(2e9) // 0.2ZCN
		allocationID := utils.SetupEnterpriseAllocation(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   3,
			"parity": 3,
			"lock":   0.2, // 2GB total size where write price per blobber is 0.1ZCN
		})

		beforeAlloc := utils.GetAllocation(t, allocationID)

		waitForTimeInMinutesWhileLogging(t, 5)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
			"lock":       0.4, // Locking 0.4 ZCN (double as we have new size) as we can't calculate the accurate time of update, will validate numbers after updating the alloc
			"size":       1 * GB,
		})
		output, err := updateAllocation(t, configPath, params, true)
		amountTotalLockedToAlloc += 4e9 // 0.4ZCN

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)
		require.Less(t, beforeAlloc.ExpirationDate, afterAlloc.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", beforeAlloc.ExpirationDate, "After:", afterAlloc.ExpirationDate),
		)

		// Verify write pool calculations
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := afterAlloc.ExpirationDate - beforeAlloc.StartTime - timeUnitInSeconds

		realCostOfBeforeAlloc := costOfAlloc(&beforeAlloc)
		expectedPaymentToBlobbers := realCostOfBeforeAlloc * durationOfUsedInSeconds / timeUnitInSeconds
		expectedWritePoolBalance := amountTotalLockedToAlloc - expectedPaymentToBlobbers

		// Log all values
		t.Logf("Time unit in seconds: %d", timeUnitInSeconds)
		t.Logf("Duration of used in seconds: %d", durationOfUsedInSeconds)
		t.Logf("Expire time before: %d", beforeAlloc.ExpirationDate)
		t.Logf("Expire time after: %d", afterAlloc.ExpirationDate)
		t.Logf("Real cost of before alloc: %d", realCostOfBeforeAlloc)
		t.Logf("Expected payment to blobbers: %d", expectedPaymentToBlobbers)
		t.Logf("Expected write pool balance: %d", expectedWritePoolBalance)
		t.Logf("Write pool balance: %d", afterAlloc.WritePool)

		require.InEpsilon(t, expectedWritePoolBalance, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		// Verify blobber rewards calculations
		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToBlobbers)

		require.InEpsilon(t, expectedPaymentToBlobbers, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Add blobber cost calculation", 15*time.Minute, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		amountTotalLockedToAlloc := int64(2e9) // 0.2ZCN
		allocationID := utils.SetupEnterpriseAllocation(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   2,
			"parity": 2,
			"lock":   0.2, // 2GB total size where write price per blobber is 0.1ZCN
		})

		waitForTimeInMinutesWhileLogging(t, 5)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)
		addBlobberID, addBlobberUrl, err := utils.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"lock":                    0.05, // Locking 0.05 ZCN (cost of single blobber)
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
		})
		output, err := updateAllocation(t, configPath, params, true)
		amountTotalLockedToAlloc += 0.05 * 1e10 // 0.25ZCN

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)

		require.InEpsilon(t, amountTotalLockedToAlloc, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		rewardQuery := fmt.Sprintf("allocation_id='%s' AND reward_type=%d", allocationID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		require.Equal(t, float64(0), enterpriseReward.TotalReward, "Enterprise blobber reward should be 0")
		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Replace blobber cost calculation", 15*time.Minute, func(t *test.SystemTest) {
		utils.SetupWalletWithCustomTokens(t, configPath, 10)

		amountTotalLockedToAlloc := int64(2e9) // 0.2ZCN
		allocationID := utils.SetupEnterpriseAllocation(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"data":   2,
			"parity": 2,
			"lock":   0.2, // 2GB total size where write price per blobber is 0.1ZCN
		})

		beforeAlloc := utils.GetAllocation(t, allocationID)

		waitForTimeInMinutesWhileLogging(t, 5)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)
		addBlobberID, addBlobberUrl, err := utils.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          beforeAlloc.BlobberDetails[0].BlobberID,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		afterAlloc := utils.GetAllocation(t, allocationID)

		require.InEpsilon(t, amountTotalLockedToAlloc, afterAlloc.WritePool, 0.01, "Write pool balance doesn't match")

		txn, err := getTransacionFromSingleSharder(t, afterAlloc.Tx)
		require.Nil(t, err)

		// Verify blobber rewards calculations
		timeUnitInSeconds := int64(600) // 10 minutes
		durationOfUsedInSeconds := int64(txn.CreationDate) - afterAlloc.StartTime

		expectedPaymentToReplacedBlobber := 5e8 * durationOfUsedInSeconds / timeUnitInSeconds

		rewardQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, beforeAlloc.BlobberDetails[0].BlobberID, EnterpriseBlobberReward)
		enterpriseReward, err := getQueryRewards(t, rewardQuery)
		require.Nil(t, err)

		t.Log("Enterprise reward: ", enterpriseReward.TotalReward, "Expected: ", expectedPaymentToReplacedBlobber)

		require.InEpsilon(t, expectedPaymentToReplacedBlobber, enterpriseReward.TotalReward, 0.01, "Enterprise blobber reward doesn't match")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Update Expiry Should Work", 15*time.Minute, func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath, map[string]interface{}{
			"lock": "0.1",
		})

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})

		time.Sleep(5 * time.Minute)

		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		ac := utils.GetAllocation(t, allocationID)
		require.Less(t, allocationBeforeUpdate.ExpirationDate, ac.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", allocationBeforeUpdate.ExpirationDate, "After:", ac.ExpirationDate),
		)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update Size Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(256)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation "+
			"due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, "After:", ac.Size),
		)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update All Parameters Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(2048)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Less(t, allocationBeforeUpdate.ExpirationDate, ac.ExpirationDate)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Update Allocation flags for forbid and allow file_options should succeed", 8*time.Minute, func(t *test.SystemTest) {
		_, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err)

		allocationID := setupAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc := utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<0))

		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_delete": nil,
		})
		t.Logf("forbidden delete")
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<1))

		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_update": nil,
		})
		t.Logf("forbidden update")
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<2))

		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_move": nil,
		})
		t.Logf("forbidden move")
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<3))

		t.Logf("forbidden copy")
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_copy": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<4))

		t.Logf("forbidden rename")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_rename": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<5))

		t.Logf("allow upload")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": false,
		})

		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(1), alloc.FileOptions)

		t.Logf("allow delete")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_delete": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(2), alloc.FileOptions&(1<<1))

		t.Logf("allow update")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_update": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(4), alloc.FileOptions&(1<<2))

		t.Logf("allow move")
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_move": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(8), alloc.FileOptions&(1<<3))

		t.Logf("allow copy")
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_copy": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(16), alloc.FileOptions&(1<<4))

		t.Logf("allow rename")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_rename": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(32), alloc.FileOptions&(1<<5))

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update allocation set_third_party_extendable flag should work", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		alloc := utils.GetAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update allocation expand by third party if third_party_extendable = true should succeed", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})

		output, err := updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Equal(t, output[0], "Error updating allocation:allocation_updating_failed: update allocation changes nothing")
		} else {
			require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		alloc := utils.GetAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := utils.EscapedTestName(t) + "_NON_OWNER"

		_, err = utils.CreateWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err)

		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2,
			"extend":     true,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))

		allocUpdated := utils.GetAllocation(t, allocationID)
		require.Equal(t, alloc.Size+2, allocUpdated.Size)

		require.Nil(t, err)
		require.Less(t, alloc.ExpirationDate, allocUpdated.ExpirationDate)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update allocation with add blobber should succeed", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet", strings.Join(output, "\n"))

		allocSize := int64(64 * KB * 2)

		allocationID := utils.SetupEnterpriseAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)
		blobberID, blobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, blobberID, blobberUrl)
		require.Nil(t, err, "Unable to generate add blobber auth ticket")

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
			"add_blobber":                blobberID,
			"add_blobber_auth_ticket":    blobberAuthTicket,
		})

		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update allocation with replace blobber should succeed", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet %v", strings.Join(output, "\n"))

		allocSize := int64(64 * KB * 2)
		fileSize := int64(1024)

		allocationID := utils.SetupEnterpriseAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := utils.GenerateRandomTestFileName(t)
		err = utils.CreateFileWithSize(filename, fileSize)
		require.Nil(t, err)

		remotePath := "/dir" + filename
		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)
		removeBlobber, err := cli_tests.GetRandomBlobber(walletFile, configFile, allocationID, addBlobberID)
		require.Nil(t, err)

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
			"add_blobber":                addBlobberID,
			"add_blobber_auth_ticket":    addBlobberAuthTicket,
			"remove_blobber":             removeBlobber,
		})

		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		utils.AssertOutputMatchesAllocationRegex(t, repairCompletednRegex, output[len(output)-1])
		fref, err := cli_tests.VerifyFileRefFromBlobber(walletFile, configFile, allocationID, addBlobberID, remotePath)
		require.Nil(t, err)
		require.NotNil(t, fref)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Run all update operations one by one", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)

		// Extend Allocation
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error extending allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Increase Allocation Size
		size := int64(2048)
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error increasing allocation size", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Set Third Party Extendable
		params = createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error setting third party extendable", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		// Add Blobber
		blobberID, blobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err, "Unable to get blobber not part of allocaiton")

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, blobberID, blobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params = createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             blobberID,
			"add_blobber_auth_ticket": blobberAuthTicket,
		})
		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error adding blobber", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Validate Final Allocation State
		alloc := utils.GetAllocation(t, allocationID)
		require.Greater(t, alloc.Size, allocationBeforeUpdate.Size)
		require.True(t, alloc.ThirdPartyExtendable)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Run all update operations at once", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)

		size := int64(2048)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		// Add Blobber
		blobberID, blobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err, "Unable to get blobber not part of allocaiton")

		blobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, configPath, blobberID, blobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		// Combine all update operations
		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"extend":                     true,
			"size":                       size,
			"set_third_party_extendable": nil,
			"add_blobber":                blobberID,
			"add_blobber_auth_ticket":    blobberAuthTicket,
		})

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation with all operations at once", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Validate Final Allocation State
		alloc := utils.GetAllocation(t, allocationID)
		require.Greater(t, alloc.Size, allocationBeforeUpdate.Size)
		require.True(t, alloc.ThirdPartyExtendable)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update Size beyond blobber capacity should fail", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)
		size := int64(1099511627776000)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "Could not update allocation "+
			"due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "doesn't have enough free space")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update Negative Size Should Fail", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(-256)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error updating allocation:allocation_updating_failed: allocation can't be reduced", output[0])

		alloc := utils.GetAllocation(t, allocationID)

		require.Equal(t, allocationBeforeUpdate.Size, alloc.Size)
		require.Equal(t, allocationBeforeUpdate.ExpirationDate, alloc.ExpirationDate)

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update Nothing Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: update allocation changes nothing", output[0])

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update Non-existent Allocation Should Fail", func(t *test.SystemTest) {
		_, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet")

		allocationID := "123abc"

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:couldnt_find_allocation: Couldn't find the allocation required for update", output[0])
	})

	t.RunWithTimeout("Update Other's Allocation Should Fail", 5*time.Minute, func(t *test.SystemTest) {
		myAllocationID := setupAllocation(t, configPath)

		targetWalletName := utils.EscapedTestName(t) + "_TARGET"
		_, err := utils.CreateWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err)

		size := int64(2048)

		params := createParams(map[string]interface{}{
			"allocation": myAllocationID,
			"size":       size,
		})

		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		params = createParams(map[string]interface{}{
			"allocation": myAllocationID,
			"size":       size,
		})
		output, err = updateAllocationWithWallet(t, targetWalletName, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error updating allocation:allocation_updating_failed: only owner can update the allocation", output[0])
	})

	t.Run("Update Mistake Size Parameter Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		size := "ab"

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at "+
			"least 1", strings.Join(output, "\n"))
		expected := fmt.Sprintf(
			`Error: invalid argument "%v" for "--size" flag: strconv.ParseInt: parsing "%v": invalid syntax`,
			size, size,
		)
		require.Equal(t, expected, output[0])

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Updating same file options twice should fail", func(w *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
			"forbid_delete": nil,
			"forbid_move":   nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
			"forbid_delete": nil,
			"forbid_move":   nil,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "changes nothing")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update allocation set_third_party_extendable flag should fail if third_party_extendable is already true", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		alloc := utils.GetAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		params = createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "changes nothing")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})

	t.Run("Update allocation expand by third party if third_party_extendable = false should fail", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       1,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		alloc := utils.GetAllocation(t, allocationID)
		require.False(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := utils.EscapedTestName(t) + "_NON_OWNER"

		_, err = utils.CreateWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err)

		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, true)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})
	t.RunWithTimeout("Update allocation any other action than expand by third party regardless of third_party_extendable should fail", 7*time.Minute, func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		alloc := utils.GetAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := utils.EscapedTestName(t) + "_NON_OWNER"

		_, err = utils.CreateWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "Error creating wallet for non allocaiton owner")

		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       -100,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		params = createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"forbid_upload":              nil,
			"forbid_update":              nil,
			"forbid_delete":              nil,
			"forbid_rename":              nil,
			"forbid_move":                nil,
			"forbid_copy":                nil,
			"set_third_party_extendable": nil,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		params = createParams(map[string]interface{}{
			"allocation":     allocationID,
			"add_blobber":    "new_blobber_id",
			"remove_blobber": "blobber_id",
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"lock":       100,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		updatedAlloc := utils.GetAllocation(t, allocationID)

		require.Equal(t, alloc.Size, updatedAlloc.Size)

		require.Equal(t, alloc.FileOptions, updatedAlloc.FileOptions)

		require.Equal(t, len(alloc.Blobbers), len(updatedAlloc.Blobbers))

		// Cleanup
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Unable to cancel allocation", strings.Join(output, "\n"))
		require.Regexp(t, cancelAllocationRegex, strings.Join(output, "\n"), "cancel allcoation fail", strings.Join(output, "\n"))
	})
}

func setupAndParseAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) (string, climodel.Allocation) {
	allocationID := setupAllocation(t, cliConfigFilename, extraParams...)

	allocations := parseListAllocations(t, cliConfigFilename)
	allocation, ok := allocations[allocationID]
	require.True(t, ok, "current allocation not found", allocationID, allocations)

	return allocationID, allocation
}

func parseListAllocations(t *test.SystemTest, cliConfigFilename string) map[string]climodel.Allocation {
	output, err := listAllocations(t, cliConfigFilename)
	require.Nil(t, err, "list allocations failed", err, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var allocations []*climodel.Allocation
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&allocations)
	require.Nil(t, err, "error deserializing JSON", err)

	allocationMap := make(map[string]climodel.Allocation)

	for _, ac := range allocations {
		allocationMap[ac.ID] = *ac
	}

	return allocationMap
}

func setupAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	return setupAllocationWithWallet(t, utils.EscapedTestName(t), cliConfigFilename, extraParams...)
}

func setupAllocationWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	output, err := utils.CreateWalletForName(t, configPath, walletName)
	require.Nil(t, err, "Error creating wallet", strings.Join(output, "\n"))

	output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
	require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

	blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)
	options := map[string]interface{}{"size": "10000000", "lock": "5", "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}

	for _, params := range extraParams {
		for k, v := range params {
			options[k] = v
		}
	}

	output, err = utils.CreateNewEnterpriseAllocation(t, cliConfigFilename, utils.CreateParams(options))
	require.NoError(t, err, "create new allocation failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	allocationID, err := utils.GetAllocationID(output[0])
	require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

	return allocationID
}
func getAllocationCost(str string) (float64, error) {
	allocationCostInOutput, err := strconv.ParseFloat(strings.Fields(str)[5], 64)
	if err != nil {
		return 0.0, err
	}

	unit := strings.Fields(str)[6]
	allocationCostInZCN := utils.UnitToZCN(allocationCostInOutput, unit)

	return allocationCostInZCN, nil
}

func createParams(params map[string]interface{}) string {
	var builder strings.Builder

	for k, v := range params {
		if v == nil {
			_, _ = builder.WriteString(fmt.Sprintf("--%s ", k))
		} else if reflect.TypeOf(v).String() == "bool" {
			_, _ = builder.WriteString(fmt.Sprintf("--%s=%v ", k, v))
		} else {
			_, _ = builder.WriteString(fmt.Sprintf("--%s %v ", k, v))
		}
	}
	return strings.TrimSpace(builder.String())
}

func updateAllocation(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return updateAllocationWithWallet(t, utils.EscapedTestName(t), cliConfigFilename, params, retry)
}

func updateAllocationWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Updating allocation...")
	cmd := fmt.Sprintf(
		"./zbox updateallocation %s --silent --wallet %s "+
			"--configDir ./config --config %s",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func listAllocations(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Listing allocations...")
	cmd := fmt.Sprintf(
		"./zbox listallocations --json --silent "+
			"--wallet %s --configDir ./config --config %s",
		utils.EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func getTransacionFromSingleSharder(t *test.SystemTest, hash string) (TransactionVerify, error) {
	sharderBaseUrl := utils.GetSharderUrl(t)
	requestURL := fmt.Sprintf("%s/v1/transaction/get/confirmation?hash=%s", sharderBaseUrl, hash)

	var result TransactionVerify

	res, _ := http.Get(requestURL) //nolint:gosec

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)

	body, _ := io.ReadAll(res.Body)

	err := json.Unmarshal(body, &result)
	if err != nil {
		return TransactionVerify{}, err
	}

	return result, nil
}

type TransactionVerify struct {
	Version           string `json:"version"`
	Hash              string `json:"hash"`
	BlockHash         string `json:"block_hash"`
	PreviousBlockHash string `json:"previous_block_hash"`
	Txn               struct {
		Hash              string `json:"hash"`
		Version           string `json:"version"`
		ClientId          string `json:"client_id"`
		PublicKey         string `json:"public_key"`
		ToClientId        string `json:"to_client_id"`
		ChainId           string `json:"chain_id"`
		TransactionData   string `json:"transaction_data"`
		TransactionValue  int    `json:"transaction_value"`
		Signature         string `json:"signature"`
		CreationDate      int    `json:"creation_date"`
		TransactionFee    int    `json:"transaction_fee"`
		TransactionNonce  int    `json:"transaction_nonce"`
		TransactionType   int    `json:"transaction_type"`
		TransactionOutput string `json:"transaction_output"`
		TxnOutputHash     string `json:"txn_output_hash"`
		TransactionStatus int    `json:"transaction_status"`
	} `json:"txn"`
	CreationDate      int    `json:"creation_date"`
	MinerId           string `json:"miner_id"`
	Round             int    `json:"round"`
	TransactionStatus int    `json:"transaction_status"`
	RoundRandomSeed   int64  `json:"round_random_seed"`
	StateChangesCount int    `json:"state_changes_count"`
	MerkleTreeRoot    string `json:"merkle_tree_root"`
	MerkleTreePath    struct {
		Nodes     []string `json:"nodes"`
		LeafIndex int      `json:"leaf_index"`
	} `json:"merkle_tree_path"`
	ReceiptMerkleTreeRoot string `json:"receipt_merkle_tree_root"`
	ReceiptMerkleTreePath struct {
		Nodes     []string `json:"nodes"`
		LeafIndex int      `json:"leaf_index"`
	} `json:"receipt_merkle_tree_path"`
}
