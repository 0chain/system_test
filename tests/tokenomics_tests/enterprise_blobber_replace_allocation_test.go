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
	"time"
)

func TestReplaceEnterpriseBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// Change time unit to 10 minutes

	// Try replacing blobber with 2x price, 0.5x price and same price. Check cost in all scenarios.
	t.Parallel()

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

	t.RunSequentiallyWithTimeout("Replace blobber in allocation with repair should work", 90*time.Second, func(t *test.SystemTest) {
		fileSize := int64(1024)

		allocationID, blobberToRemove := setupAllocationAndGetRandomBlobber(t, configPath)

		filename := utils.GenerateRandomTestFileName(t)
		err := utils.CreateFileWithSize(filename, fileSize)
		require.Nil(t, err)

		remotePath := "/file" + filename

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  filename,
		}
		output, err := utils.UploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))

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

		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation", strings.Join(output, "\n"))
		utils.AssertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		utils.AssertOutputMatchesAllocationRegex(t, repairCompletednRegex, output[len(output)-1])

		fref, err := cli_tests.VerifyFileRefFromBlobber(walletFile, configFile, allocationID, addBlobberID, remotePath)
		require.Nil(t, err)
		require.NotNil(t, fref)
	})

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
		require.Contains(t, strings.Join(output, "\n"), "blobber already exists in allocation")
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
		require.Contains(t, strings.Join(output, "\n"), "invalid configuration for given blobber ID")
	})

}

func setupAllocationAndGetRandomBlobber(t *test.SystemTest, cliConfigFilename string) (string, string) {
	allocationID := setupAllocation(t, cliConfigFilename)

	randomBlobber, err := cli_tests.GetRandomBlobber(utils.EscapedTestName(t), cliConfigFilename, allocationID, "")
	require.Nil(t, err, "Error getting random blobber")

	return allocationID, randomBlobber
}
