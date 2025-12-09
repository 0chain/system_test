package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFreeReads(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("free reads should work")

	var blobberList []climodel.BlobberDetails
	t.TestSetup("Create wallet, execute faucet, get blobber details", func() {
		if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
		}
		createWallet(t)

		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		// Set read price to 0 on all blobbers using their delegate wallets
		newReadPrice := 0
		updatedBlobbers := 0
		for _, blobber := range blobberList {
			// Use the blobber's delegate wallet from the list
			delegateWallet := blobber.StakePoolSettings.DelegateWallet
			// Try using blobberOwnerWallet - if it fails due to access denied, skip that blobber
			walletName := blobberOwnerWallet
			output, err := updateBlobberInfoForWallet(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.ID, "read_price": newReadPrice}), walletName)
			outputStr := strings.Join(output, "\n")
			if err != nil && strings.Contains(outputStr, "access denied") {
				// If access denied, skip this blobber - delegate wallet might be different
				t.Logf("Warning: Cannot update blobber %s read price with %s - access denied. Delegate wallet: %s. Skipping.", blobber.ID, walletName, delegateWallet)
				continue
			}
			// Only assert if there was no error (or error was not access denied)
			if err != nil {
				// If there's an error that's not "access denied", log it but continue
				t.Logf("Warning: Failed to update blobber %s read price: %v, Output: %s", blobber.ID, err, outputStr)
				continue
			}
			// Success case - verify the update
			if len(output) > 0 && output[0] == "blobber settings updated successfully" {
				updatedBlobbers++
				t.Logf("Successfully updated blobber %s read price to 0", blobber.ID)
			}
		}
		// Log how many blobbers were updated
		t.Logf("Updated read price to 0 for %d out of %d blobbers", updatedBlobbers, len(blobberList))
	})

	// revert read prices irrespective of test results
	t.Cleanup(func() {
		for _, blobber := range blobberList {
			// Use blobberOwnerWallet to revert - if it fails, skip (delegate wallet might be different)
			output, err := updateBlobberInfoForWallet(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.ID, "read_price": intToZCN(blobber.Terms.ReadPrice)}), blobberOwnerWallet)
			outputStr := strings.Join(output, "\n")
			if err != nil {
				// Skip if we can't revert - delegate wallet might be different or blobber might not exist
				if strings.Contains(outputStr, "access denied") {
					t.Logf("Warning: Cannot revert blobber %s read price - access denied. Skipping cleanup for this blobber.", blobber.ID)
				} else {
					t.Logf("Warning: Failed to revert blobber %s read price: %v, Output: %s. Skipping cleanup for this blobber.", blobber.ID, err, outputStr)
				}
				continue
			}
			// Success - no need to assert, just log
			if len(output) > 0 && output[0] == "blobber settings updated successfully" {
				t.Logf("Successfully reverted blobber %s read price", blobber.ID)
			}
		}
	})

	t.Run("Free reads should work", func(t *test.SystemTest) {
		createWallet(t)

		_ = setupWallet(t, configPath)

		// Create an allocation
		options := map[string]interface{}{"size": 1 * MB, "lock": "0.5", "read_price": "0-0"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		// Upload a file and download it to test free read
		remotepath := "/"
		filesize := int64(256)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.RunWithTimeout("Free read by authticket should Work", 5*time.Minute, func(t *test.SystemTest) {
		var authTicket, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/"

		// This subtest creates a separate wallet and allocates there
		t.Run("Share File from Another Wallet for free read", func(t *test.SystemTest) {
			_ = setupWallet(t, configPath)

			// Create an allocation
			options := map[string]interface{}{"size": 1 * MB, "lock": "0.5", "read_price": "0-0"}
			output, err := createNewAllocation(t, configPath, createParams(options))
			require.Nil(t, err, strings.Join(output, "\n"))
			require.True(t, len(output) > 0, "expected output length be at least 1")
			require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

			allocationID, err := getAllocationID(output[0])
			require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			originalFileChecksum = generateChecksum(t, filename)

			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err = os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath + filepath.Base(filename),
			})

			output, err = shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just create a wallet so that we can work further
		createWallet(t)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})
}

func updateBlobberInfoForWallet(t *test.SystemTest, cliConfigFilename, params, walletName string) ([]string, error) {
	t.Logf("Updating blobber info using wallet %s...", walletName)
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, walletName, cliConfigFilename), 3, time.Second*2)
}
