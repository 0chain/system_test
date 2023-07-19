package cli_tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
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
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = createWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		// Set read price to 0 on all blobbers
		newReadPrice := 0
		for _, blobber := range blobberList {
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.ID, "read_price": newReadPrice}))
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "blobber settings updated successfully", output[0])
		}

		// init enough tokens to blobber owner wallet to issue txns
		_, err = executeFaucetWithTokensForWallet(t, blobberOwnerWallet, configPath, 100)
		require.NoError(t, err)
	})

	// revert read prices irrespective of test results
	t.Cleanup(func() {
		for _, blobber := range blobberList {
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.ID, "read_price": intToZCN(blobber.Terms.Read_price)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}
	})

	t.RunSequentially("Free reads should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		_ = setupWallet(t, configPath)

		// Create an allocation
		options := map[string]interface{}{"size": 1 * MB, "lock": "0.5"}
		output, err = createNewAllocation(t, configPath, createParams(options))
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
			// Create an allocation
			options := map[string]interface{}{"size": 1 * MB, "lock": "0.5"}
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
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		// Download file using auth-ticket: should work
		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
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
