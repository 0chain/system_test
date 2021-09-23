package cli_tests

import (
	"encoding/json"
	"fmt"
	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"path/filepath"
	"strings"
	"testing"
)

func pretty(data interface{}) {
	bts, _ := json.MarshalIndent(data, "", "\t")
	fmt.Println(string(bts))
}

func TestFileMetadata(t *testing.T) {

	// Scenarios //

	// get file metadata on an empty allocation
	t.Run("Get File Meta on Empty Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0], strings.Join(output, "\n"))
	})

	// get folder metadata in empty directory
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

	// get folder metadata in non-empty directory
	t.Run("Get Folder Meta in Empty Directory Should Work", func(t *testing.T) {
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

		var meta cli_model.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "d", meta.Type)
		require.Equal(t, "/", meta.Path)
		require.Equal(t, "/", meta.Name)
		require.Equal(t, 0, meta.Size)
	})

	// get file metadata in the root directory
	t.Run("Get File Meta in Root Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

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

		var meta cli_model.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
	})

	// get file metadata in a sub directory
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

		var meta cli_model.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, remotepath+fname, meta.Path)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
	})

	// get file metadata by auth-ticket lookuphash
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

		var meta cli_model.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, "f", meta.Type)
		require.Equal(t, fname, meta.Name)
		require.Equal(t, filesize, meta.Size)
	})

	// get file metadata of another wallet's file
	t.Run("Get File Meta on Another Wallet File Should Fail", func(t *testing.T) {

	})

	// supply both file path and lookup hash to list file
	t.Run("Get File Meta by Path and Lookup Hash Should Work", func(t *testing.T) {

	})

	// supply no params
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
	return cli_utils.RunCommand(cmd)
}
