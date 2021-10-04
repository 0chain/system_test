package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var reAuthToken = regexp.MustCompile(`^Auth token :(.*)$`)

func TestListFileSystem(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Run("Success Scenarios", func(t *testing.T) {
		t.Parallel()

		t.Run("No Files in Allocation Should Work", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath)

			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": "/",
				"json":       "",
			}))
			require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "null", output[0], strings.Join(output, "\n"))
		})

		t.Run("List Files in Root Directory Should Work", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath)

			// First Upload a file to the root directory
			filesize := int64(10)
			remotepath := "/"
			filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
			fname := filepath.Base(filename)

			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))

			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, 1)
			result := listResults[0]

			require.Equal(t, fname, result.Name)
			require.Equal(t, remotepath+fname, result.Path)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)
			require.Equal(t, "", result.EncryptionKey)
		})

		//FIXME: POSSIBLE BUG: Encrypted file require much more space
		t.Run("List Encrypted Files Should Work", func(t *testing.T) {
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

			output, err := uploadFile(t, configPath, map[string]interface{}{
				"allocation": allocationID,
				"localpath":  filename,
				"remotepath": remotepath + filepath.Base(filename),
				"encrypt":    "",
			})
			require.Nil(t, err, "upload failed", strings.Join(output, "\n"))
			require.Len(t, output, 2)

			expected := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", fname)
			require.Equal(t, expected, output[1], strings.Join(output, "\n"))

			output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))

			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, 1)
			result := listResults[0]

			require.Equal(t, fname, result.Name)
			require.Equal(t, remotepath+fname, result.Path)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)
			require.NotEqual(t, "", result.EncryptionKey)
		})

		t.Run("List Files and Check Lookup Hash Should Work", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath)

			filesize := int64(2)
			remotepath := "/"

			// First Upload a file to the a directory
			filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

			h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
			lookupHash := fmt.Sprintf("%x", h)

			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, 1)
			result := listResults[0]

			require.Equal(t, lookupHash, result.LookupHash)
			require.Equal(t, filepath.Base(filename), result.Name)
			require.Equal(t, remotepath+filepath.Base(filename), result.Path)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)
		})

		t.Run("List Files in a Directory Should Work", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath)

			// First Upload a file to the a directory
			filesize := int64(2)
			remotepath := "/test_file/"
			filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
			fname := filepath.Base(filename)

			// Then list the files and check
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, 1)
			result := listResults[0]

			require.Equal(t, fname, result.Name)
			require.Equal(t, remotepath+fname, result.Path)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)
		})

		t.Run("List Files in Nested Directory Should Work", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath)

			// First Upload a file to the a directory
			filesize := int64(2)
			remotepath := "/nested/test_file/"
			filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
			fname := filepath.Base(filename)

			// Then list the files and check
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, "List file failed due to error ", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, 1)
			result := listResults[0]

			require.Equal(t, fname, result.Name)
			require.Equal(t, remotepath+fname, result.Path)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)
		})

		//FIXME: POSSIBLE BUG: Can't use lookuphash on self-owned wallet with remotepath doesn't work
		t.Run("List Files Using Lookup Hash and RemotePath Should Work", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath)

			filesize := int64(2)
			remotepath := "/"
			numFiles := 2

			// First Upload a file to the a directory
			filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

			// Upload other files (no need to keep track of their IDs)
			for i := 0; i < numFiles-1; i++ {
				generateFileAndUpload(t, allocationID, remotepath, filesize)
			}

			h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
			lookupHash := fmt.Sprintf("%x", h)

			// Then list the files and check
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"remotepath": remotepath,
				"lookuphash": lookupHash,
			}))
			require.Nil(t, err, "List file failed due to error ", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, numFiles)
			var result climodel.ListFileResult
			for _, res := range listResults {
				if res.LookupHash == lookupHash {
					result = res
				}
			}

			require.Equal(t, filepath.Base(filename), result.Name)
			require.Equal(t, remotepath+filepath.Base(filename), result.Path)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)
		})

		t.Run("List Shared Files Should Work", func(t *testing.T) {
			t.Parallel()

			var authTicket, filename string

			filesize := int64(10)
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
				require.Nil(t, err, "share folder failed", strings.Join(output, "\n"))
				require.Len(t, output, 1)

				authTicket, err = extractAuthToken(output[0])
				require.Nil(t, err, "extract auth token failed")
				require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
			})
			fname := filepath.Base(filename)

			// Just register a wallet so that we can work further
			_, err := registerWallet(t, configPath)
			require.Nil(t, err)

			// Listing contents using auth-ticket: should work
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"authticket": authTicket,
				"json":       "",
			}))
			require.Nil(t, err, "List file failed due to error ", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, 1)
			result := listResults[0]

			require.Equal(t, fname, result.Name)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)
		})

		//FIXME: POSSIBLE BUG: Listing shared files with lookuphash doesn't list any files
		t.Run("List Shared Files Using Lookup Hash Should Work", func(t *testing.T) {
			t.Parallel()

			var authTicket, filename, lookupHash string

			filesize := int64(2)
			remotepath := "/"
			numFiles := 3

			// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
			t.Run("Share Folder from Another Wallet", func(t *testing.T) {
				allocationID := setupAllocation(t, configPath)
				filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
				require.NotEqual(t, "", filename)

				// Upload other files (no need to keep track of their IDs)
				for i := 0; i < numFiles-1; i++ {
					generateFileAndUpload(t, allocationID, remotepath, filesize)
				}

				shareParam := createParams(map[string]interface{}{
					"allocation": allocationID,
					"remotepath": remotepath,
				})

				output, err := shareFolderInAllocation(t, configPath, shareParam)
				require.Nil(t, err, "share failed", strings.Join(output, "\n"))
				require.Len(t, output, 1)

				authTicket, err = extractAuthToken(output[0])
				require.Nil(t, err, "extract auth token failed", authTicket)
				require.NotEqual(t, "", authTicket)

				h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
				lookupHash = fmt.Sprintf("%x", h)
				require.NotEqual(t, "", lookupHash, "Lookup Hash: ", lookupHash)
			})

			// Just register a wallet so that we can work further
			_, err := registerWallet(t, configPath)
			require.Nil(t, err, err)

			// Listing contents of allocationID: should work
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"authticket": authTicket,
				"lookuphash": lookupHash,
				"json":       "",
			}))
			require.Nil(t, err, "list files using auth ticket [%v] failed: [%v]", authTicket, strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, "null", output[0], strings.Join(output, "\n"))
		})

		t.Run("List All Files Should Work", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath)

			generateFileAndUpload(t, allocationID, "/", int64(10))
			generateFileAndUpload(t, allocationID, "/", int64(10))
			generateFileAndUpload(t, allocationID, "/dir/", int64(10))
			generateFileAndUpload(t, allocationID, "/dir/", int64(10))

			output, err := listAllFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
			}))
			require.Nil(t, err, "list files failed", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			totalFiles := 4
			totalFolders := 1
			expectedTotalEntries := totalFolders + totalFiles
			require.Len(t, listResults, expectedTotalEntries, "number of files from output [%v] do not mach expected", output)

			var numFile, numFolder int
			for _, lr := range listResults {
				if lr.Type == "f" {
					numFile++
				} else if lr.Type == "d" {
					numFolder++
				}
			}
			require.Equal(t, totalFiles, numFile)
			require.Equal(t, totalFolders, numFolder)
		})
	})

	t.Run("Failure Scenarios", func(t *testing.T) {
		t.Parallel()

		t.Run("No Parameter Should Fail", func(t *testing.T) {
			t.Parallel()

			output, err := listFilesInAllocation(t, configPath, "")
			require.NotNil(t, err,
				"List files with no parameter failed due to error", err,
				strings.Join(output, "\n"))

			require.Equal(t,
				"Error: remotepath / authticket flag is missing",
				output[len(output)-1],
				strings.Join(output, "\n"))
		})

		//FIXME: POSSIBLE BUG: Listing contents of another wallet's allocation doesn't throw
		// any errors. Good thing is that the contents are not shown.
		t.Run("List Files in Other's Wallet Should Fail", func(t *testing.T) {
			t.Parallel()

			var otherAllocationID string
			allocationID := setupAllocation(t, configPath)

			filesize := int64(10)
			remotepath := "/"

			t.Run("Get Other Allocation ID", func(t *testing.T) {
				otherAllocationID = setupAllocation(t, configPath)

				generateFileAndUpload(t, otherAllocationID, remotepath, 1)

				// Listing contents of otherAllocationID: should work
				output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
					"allocation": otherAllocationID,
					"json":       "",
					"remotepath": remotepath,
				}))
				require.Nil(t, err, err)
				require.Len(t, output, 1)
				require.NotEqual(t, "null", output[0], strings.Join(output, "\n"))
			})

			filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
			fname := filepath.Base(filename)

			// Listing contents of allocationID: should work
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, "List file failed due to error ", err, strings.Join(output, "\n"))

			require.Len(t, output, 1)

			var listResults []climodel.ListFileResult
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
			require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

			require.Len(t, listResults, 1)
			result := listResults[0]

			require.Equal(t, fname, result.Name)
			require.Equal(t, remotepath+fname, result.Path)
			require.Equal(t, filesize, result.ActualSize)
			require.Equal(t, "f", result.Type)

			// Listing contents of otherAllocationID: should not work
			output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": otherAllocationID,
				"json":       "",
				"remotepath": remotepath,
			}))
			require.Nil(t, err, err)
			require.Len(t, output, 1)
			require.Equal(t, "null", output[0], strings.Join(output, "\n"))
		})
	})
}

func extractAuthToken(str string) (string, error) {
	match := reAuthToken.FindStringSubmatch(str)
	if len(match) > 1 {
		return match[1], nil
	}
	return "", errors.New("auth token did not match")
}

func createFileWithSize(name string, size int64) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}

	if err := f.Truncate(size); err != nil {
		return err
	}

	return nil
}

func generateRandomTestFileName(t *testing.T) string {
	path, err := filepath.Abs("tmp")
	require.Nil(t, err)

	//FIXME: POSSIBLE BUG: when the name of the file is too long, the upload
	// consensus fails. So we are generating files with random (but short)
	// name here.
	nBig := cliutils.RandomAlphaNumericString(10)
	return fmt.Sprintf("%s/%s_test.txt", path, nBig)
}

func shareFolderInAllocation(t *testing.T, cliConfigFilename, param string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox share %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(cmd)
}

func listFilesInAllocation(t *testing.T, cliConfigFilename, param string) ([]string, error) {
	time.Sleep(15 * time.Second) // TODO replace with poller
	cmd := fmt.Sprintf(
		"./zbox list %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(cmd)
}

func listAllFilesInAllocation(t *testing.T, cliConfigFilename, param string) ([]string, error) {
	time.Sleep(15 * time.Second) // TODO replace with poller
	cmd := fmt.Sprintf(
		"./zbox list-all %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(cmd)
}
