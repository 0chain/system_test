package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestFileMetadata(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	// Success Scenarios

	t.Run("Get Folder Meta in Non-Empty Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		// Upload a sample file
		generateFileAndUpload(t, allocationID, "/", 4)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "d", meta.Type)
		require.Equal(t, "/", meta.Path)
		require.Equal(t, "/", meta.Name)
		require.Equal(t, int64(0), meta.Size)
	})

	t.Run("Get File Meta in Root Directory Should Work", func(t *testing.T) {
		t.Parallel()

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
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
		require.Equal(t, "", meta.EncryptedKey)
	})

	t.Run("Get File Meta in Sub Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/dir/"
		filesize := int64(4)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + fname,
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
	})

	t.Run("Get Shared File Meta by Auth Ticket and Lookup Hash Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename, lookupHash string

		filesize := int64(2)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Folder from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocation(t, configPath)
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			require.NotEqual(t, "", filename)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, err)
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, err)
			require.NotEqual(t, "", authTicket)

			h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
			lookupHash = fmt.Sprintf("%x", h)
			require.NotEqual(t, "", lookupHash)
		})
		fname := filepath.Base(filename)

		// Just register a wallet so that we can work further
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		// Listing contents of allocationID: should work
		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"lookuphash": lookupHash,
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
	})

	//FIXME: POSSIBLE BUG: Using lookuphash with remotepath causes no effects. lookuphash
	// is simply ignored
	t.Run("Get File Meta by Path and Lookup Hash Should Work", func(t *testing.T) {
		t.Parallel()

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
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)

		// Check with random lookuphash
		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + fname,
			"lookuphash": "ab12ok90",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
		require.Equal(t, "", meta.EncryptedKey)
	})

	t.Run("Get File Meta for Encrypted File Should Work", func(t *testing.T) {
		t.Parallel()

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

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath,
			"encrypt":    "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", fname)
		require.Equal(t, expected, output[1], strings.Join(output, "\n"))

		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + fname,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
		require.NotEqual(t, "", meta.EncryptedKey)
	})

	// Failure Scenarios

	t.Run("Get File Meta on Another Wallet File Should Fail", func(t *testing.T) {
		t.Parallel()

		var otherAllocationID, otherfile string
		allocationID := setupAllocation(t, configPath)

		filesize := int64(10)
		remotepath := "/"

		t.Run("Get Other Allocation ID", func(t *testing.T) {
			otherAllocationID = setupAllocation(t, configPath)

			otherfile = generateFileAndUpload(t, otherAllocationID, remotepath, 1)

			// Listing contents of otherAllocationID: should work
			output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
				"allocation": otherAllocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var meta climodel.FileMetaResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
			require.Nil(t, err, strings.Join(output, "\n"))

			require.Equal(t, "d", meta.Type)
			require.Equal(t, remotepath, meta.Path)
			require.Equal(t, remotepath, meta.Name)
			require.Equal(t, int64(0), meta.Size)

		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)

		// Listing contents of allocationID: should work
		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + fname,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)

		// Listing contents of otherAllocationID: should not work
		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": otherAllocationID,
			"json":       "",
			"remotepath": remotepath + filepath.Base(otherfile),
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "file_meta_error: Error getting the file meta data from blobbers", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get File Meta for Missing remotepath and authticket Should Fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get Folder Meta in Empty Directory Should Fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "file_meta_error: Error getting the file meta data from blobbers", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get File Meta by Lookup Hash Should Fail", func(t *testing.T) {
		t.Parallel()

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
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0], strings.Join(output, "\n"))
	})

	t.Run("Get File Meta Without Parameter Should Fail", func(t *testing.T) {
		t.Parallel()

		output, err := getFileMeta(t, configPath, "")
		require.NotNil(t, err, strings.Join(output, "\n"))

		require.Equal(t,
			"Error: remotepath / authticket flag is missing",
			output[len(output)-1],
			strings.Join(output, "\n"))
	})
}

func getFileMeta(t *testing.T, cliConfigFilename string, param string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox meta %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(cmd)
}
