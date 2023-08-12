package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestFileMetadata(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get Folder Meta in Non-Empty Directory Should Work")

	t.Parallel()

	t.TestSetup("Create tmp dir", func() {
		// Create a folder to keep all the generated files to be uploaded
		err := os.MkdirAll("tmp", os.ModePerm)
		require.Nil(t, err)
	})

	// Success Scenarios

	t.Run("Get Folder Meta in Non-Empty Directory Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		filesize := int64(4)

		// Upload a sample file
		generateFileAndUpload(t, allocationID, "/", filesize)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "d", meta.Type)
		require.Equal(t, "/", meta.Path)
		require.Equal(t, "/", meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)
	})

	t.Run("Get File Meta in Root Directory Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		remotepath := "/"
		filesize := int64(4)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + fname,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)
		require.Equal(t, "", meta.EncryptedKey)
	})

	t.Run("Get File Meta in Sub Directory Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		remotepath := "/dir/"
		filesize := int64(4)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + fname,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)
	})

	t.Run("Get Shared File Meta by Auth Ticket and Lookup Hash Should Work", func(t *test.SystemTest) {
		var authTicket, filename, lookupHash string

		filesize := int64(2)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Folder from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocation(t, configPath)
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			require.NotEqual(t, "", filename)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err)
			require.NotEqual(t, "", authTicket)

			h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
			lookupHash = fmt.Sprintf("%x", h)
			require.NotEqual(t, "", lookupHash)
		})
		fname := filepath.Base(filename)

		// Just create a wallet so that we can work further
		output, err := createWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		// Listing contents of allocationID: should work
		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"lookuphash": lookupHash,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)
	})

	// FIXME: POSSIBLE BUG: Using lookuphash with remotepath causes no effects. lookuphash
	// is simply ignored
	t.Run("Get File Meta by Path and Lookup Hash Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		filesize := int64(2)
		remotepath := "/"

		// First Upload a file to the a directory
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)

		h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
		lookupHash := fmt.Sprintf("%x", h)

		// Check with calculated lookuphash
		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + fname,
			"lookuphash": lookupHash,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)

		// Check with random lookuphash
		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + fname,
			"lookuphash": "ab12ok90",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)
		require.Equal(t, "", meta.EncryptedKey)
	})

	t.Run("Get File Meta for Encrypted File Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		// First Upload a file to the root directory
		filesize := int64(10)
		remotepath := "/"
		filename := generateRandomTestFileName(t)
		fname := filepath.Base(filename)

		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		expected := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", fname)
		require.Equal(t, expected, output[2], strings.Join(output, "\n"))

		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + fname,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)
		require.NotEqual(t, "", meta.EncryptedKey)
	})

	// Failure Scenarios

	t.Run("Get File Meta on Another Wallet File Should Fail", func(t *test.SystemTest) {
		var otherAllocationID, otherfile string
		allocationID := setupAllocation(t, configPath)

		filesize := int64(1)
		remotepath := "/"

		t.Run("Get Other Allocation ID", func(t *test.SystemTest) {
			otherAllocationID = setupAllocation(t, configPath)

			otherfile = generateFileAndUpload(t, otherAllocationID, remotepath, filesize)

			// Listing contents of otherAllocationID: should work
			output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
				"allocation": otherAllocationID,
				"json":       "",
				"remotepath": remotepath,
			}), true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var meta climodel.FileMetaResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
			require.Nil(t, err, strings.Join(output, "\n"))

			require.Equal(t, "d", meta.Type)
			require.Equal(t, remotepath, meta.Path)
			require.Equal(t, remotepath, meta.Name)
			require.Equal(t, filesize, meta.ActualFileSize)
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)

		// Listing contents of allocationID: should work
		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + fname,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.ActualFileSize)

		// Listing contents of otherAllocationID: should not work
		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": otherAllocationID,
			"json":       "",
			"remotepath": remotepath + filepath.Base(otherfile),
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "file_meta_error: Error getting the file meta data from blobbers", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get File Meta for Missing remotepath and authticket Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get Folder Meta in Empty Directory Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "file_meta_error: Error getting the file meta data from blobbers", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get File Meta by Lookup Hash Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		filesize := int64(2)
		remotepath := "/"

		// First Upload a file to the a directory
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
		lookupHash := fmt.Sprintf("%x", h)

		// Check with calculated lookuphash
		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"lookuphash": lookupHash,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get File Meta Without Parameter Should Fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		output, err := getFileMeta(t, configPath, "", false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)

		require.Equal(t,
			"Error: remotepath / authticket flag is missing",
			output[len(output)-1],
			strings.Join(output, "\n"))
	})
}

func getFileMeta(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return getFileMetaWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func getFileMetaWithWallet(t *test.SystemTest, walletName, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Getting file metadata...")
	cmd := fmt.Sprintf(
		"./zbox meta %s --silent --wallet %s --configDir ./config --config %s",
		param,
		walletName+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
