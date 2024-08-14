package tokenomics_tests

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/tests/cli_tests"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	t.Run("Replace blobber in allocation, should work", func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
	})

	t.Run("Check token accounting of a blobber replacing in allocation, should work", func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		alloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, alloc)
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

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		updatedAlloc := utils.GetAllocation(t, allocationID)
		require.NotNil(t, updatedAlloc)

		var newReplaceeBlobberStake int64
		for _, blobber := range updatedAlloc.Blobbers {
			if blobber.ID == addBlobberID {
				newReplaceeBlobberStake = blobber.TotalStake
			}
		}

		require.Equal(t, prevReplaceeBlobberStake, newReplaceeBlobberStake, "Stake should be transferred from old blobber to new")
	})

	t.RunSequentially("Replace blobber with same price should work", func(t *test.SystemTest) {
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

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Ensure the blobber price is reset after the test
		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")
		}()
	})

	t.RunSequentially("Replace blobber with 0.5x price should work", func(t *test.SystemTest) {
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

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			// Ensure the blobber price is reset after the test
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")
		}()
	})

	t.RunSequentially("Replace blobber with 2x price should work", func(t *test.SystemTest) {
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

		addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, addBlobberID, addBlobberUrl)
		require.Nil(t, err, "Unable to generate auth ticket for add blobber")

		params := createParams(map[string]interface{}{
			"allocation":              allocationID,
			"add_blobber":             addBlobberID,
			"add_blobber_auth_ticket": addBlobberAuthTicket,
			"remove_blobber":          blobberToRemove,
		})

		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Ensure the blobber price is reset after the test
		defer func() {
			err = updateBlobberPrice(t, configPath, addBlobberID, originalPrice)
			require.Nil(t, err, "Error resetting blobber price after test")
		}()
	})

	//Repair file tests to be added later.
	//t.RunSequentiallyWithTimeout("Replace blobber in allocation with repair should work", 90*time.Second, func(t *test.SystemTest) {
	//	fileSize := int64(1024)
	//
	//	allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)
	//
	//	filename := utils.GenerateRandomTestFileName(t)
	//	err := utils.CreateFileWithSize(filename, fileSize)
	//	require.Nil(t, err)
	//
	//	remotePath := "/file" + filename
	//
	//	uploadParams := map[string]interface{}{
	//		"allocation": allocationID,
	//		"remotepath": remotePath,
	//		"localpath":  filename,
	//	}
	//	output, err := utils.UploadFile(t, configPath, uploadParams, true)
	//	require.Nil(t, err, strings.Join(output, "\n"))
	//
	//	wd, _ := os.Getwd()
	//	walletFile := filepath.Join(wd, "config", utils.EscapedTestName(t)+"_wallet.json")
	//	configFile := filepath.Join(wd, "config", configPath)
	//
	//	addBlobberID, addBlobberUrl, err := cli_tests.GetBlobberIdAndUrlNotPartOfAllocation(walletFile, configFile, allocationID)
	//	require.Nil(t, err)
	//
	//	addBlobberAuthTicket, err := utils.GetBlobberAuthTicketWithId(t, addBlobberID, addBlobberUrl)
	//	require.Nil(t, err, "Unable to generate auth ticket for add blobber")
	//
	//	params := createParams(map[string]interface{}{
	//		"allocation":              allocationID,
	//		"add_blobber":             addBlobberID,
	//		"add_blobber_auth_ticket": addBlobberAuthTicket,
	//		"remove_blobber":          blobberToRemove,
	//	})
	//
	//	output, err = updateAllocation(t, configPath, params, true)
	//	require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
	//	utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
	//	utils.AssertOutputMatchesAllocationRegex(t, repairCompletednRegex, output[len(output)-1])
	//
	//	fref, err := cli_tests.VerifyFileRefFromBlobber(walletFile, configFile, allocationID, addBlobberID, remotePath)
	//	require.Nil(t, err)
	//	require.NotNil(t, fref)
	//})

	t.Run("Replace blobber with the same one in allocation, shouldn't work", func(t *test.SystemTest) {
		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		blobberAuthTickets, _ := utils.GenerateBlobberAuthTickets(t)
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

		blobberAuthTickets, _ := utils.GenerateBlobberAuthTickets(t)

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

func setupAllocationAndGetRandomBlobber(t *test.SystemTest, cliConfigFilename string) (string, string) {
	allocationID := setupAllocation(t, cliConfigFilename)

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
		t.Log("Error updating blobber pricen " + strings.Join(output, "\n"))
		return err
	}

	t.Log("Updated blobber price:", strings.Join(output, "\n"))
	return nil
}
