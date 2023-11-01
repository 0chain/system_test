package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var reAuthToken = regexp.MustCompile(`^Auth token :(.*)$`)

func TestListFileSystem(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("No Files in Allocation Should Work")

	t.Parallel()

	t.TestSetup("Create tmp dir", func() {
		// Create a folder to keep all the generated files to be uploaded
		err := os.MkdirAll("tmp", os.ModePerm)
		require.Nil(t, err)
	})

	t.Run("No Files in Allocation Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("List Files in Root Directory Should Work", func(t *test.SystemTest) {
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
		}), true)
		require.Nil(t, err, "List files failed", strings.Join(output, "\n"))

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
	t.Run("List Encrypted Files Should Work", func(t *test.SystemTest) {
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
		}, true)
		require.Nil(t, err, "upload failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", fname)
		require.Equal(t, expected, output[1], strings.Join(output, "\n"))

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "List files failed", strings.Join(output, "\n"))

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

	t.Run("List Files and Check Lookup Hash Should Work", func(t *test.SystemTest) {
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
		}), true)
		require.Nil(t, err, "List files failed", strings.Join(output, "\n"))
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

	t.Run("List Files in a Directory Should Work", func(t *test.SystemTest) {
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
		}), true)
		require.Nil(t, err, "List files failed", strings.Join(output, "\n"))
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

	t.Run("List Files in Nested Directory Should Work", func(t *test.SystemTest) {
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
		}), true)
		require.Nil(t, err, "List file failed due to error ", strings.Join(output, "\n"))
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

	t.Run("List a File Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		// First Upload a file to the a directory
		filesize := int64(2)
		remotepath := "/test_file/"
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remotepathOfFile := filepath.Join(remotepath, fname)

		// Then list the files and check
		output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepathOfFile,
		}), true)
		require.Nil(t, err, "List files failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var listResults []climodel.ListFileResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
		require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

		require.Len(t, listResults, 1)
		result := listResults[0]

		require.Equal(t, fname, result.Name)
		require.Equal(t, remotepathOfFile, result.Path)
		require.Equal(t, filesize, result.ActualSize)
		require.Equal(t, "f", result.Type)
	})

	//FIXME: POSSIBLE BUG: Can't use lookuphash on self-owned wallet with remotepath doesn't work
	t.Run("List Files Using Lookup Hash and RemotePath Should Work", func(t *test.SystemTest) {
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
		}), true)
		require.Nil(t, err, "List file failed due to error ", strings.Join(output, "\n"))
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

	t.Run("List Files in Shared Directory Should Work", func(t *test.SystemTest) {
		var authTicket, filename string

		filesize := int64(10)
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
			require.Nil(t, err, "share folder failed", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})
		fname := filepath.Base(filename)

		// Just create a wallet so that we can work further
		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		// Listing contents using auth-ticket: should work
		output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"json":       "",
		}), true)
		require.Nil(t, err, "List file failed due to error ", strings.Join(output, "\n"))
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

	t.Run("List a Shared File Should Work", func(t *test.SystemTest) {
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocation(t, configPath)
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			require.NotEqual(t, "", filename)
			remotePathOfFile := filepath.Join(remotepath, filepath.Base(filename))

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotePathOfFile,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, "share folder failed", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})
		fname := filepath.Base(filename)

		// Just create a wallet so that we can work further
		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		// Listing contents using auth-ticket: should work
		output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"json":       "",
		}), true)
		require.Nil(t, err, "List file failed due to error ", strings.Join(output, "\n"))
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

	t.Run("List Shared Files Using Lookup Hash Should Work", func(t *test.SystemTest) {
		var authTicket, filename, lookupHash string

		filesize := int64(2)
		remotepath := "/"
		numFiles := 3

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Folder from Another Wallet", func(t *test.SystemTest) {
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

		// Just create a wallet so that we can work further
		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		// Listing contents of allocationID: should work
		output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"lookuphash": lookupHash,
			"json":       "",
		}), true)
		require.Nil(t, err, "list files using auth ticket [%v] failed: [%v]", authTicket, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var listResults []climodel.ListFileResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
		require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))
		require.Len(t, listResults, 3)
	})

	t.Run("List All Files Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		generateFileAndUpload(t, allocationID, "/", int64(10))
		generateFileAndUpload(t, allocationID, "/", int64(10))
		generateFileAndUpload(t, allocationID, "/dir/", int64(10))
		generateFileAndUpload(t, allocationID, "/dir/", int64(10))

		output, err := listAllFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}), true)
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

	t.Run("No Parameter Should Fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		output, err := listFilesInAllocation(t, configPath, "", false)
		require.NotNil(t, err,
			"List files with no parameter failed due to error", err,
			strings.Join(output, "\n"))

		require.Greater(t, len(output), 0)

		require.Equal(t,
			"Error: remotepath / authticket flag is missing",
			output[len(output)-1],
			strings.Join(output, "\n"))
	})

	t.Run("List Files in Other's Allocation Should Fail", func(t *test.SystemTest) { //todo: too slow
		var otherAllocationID string
		allocationID := setupAllocation(t, configPath)

		filesize := int64(10)
		remotepath := "/"

		t.Run("Get Other Allocation ID", func(t *test.SystemTest) {
			otherAllocationID = setupAllocation(t, configPath)

			generateFileAndUpload(t, otherAllocationID, remotepath, 1)

			// Listing contents of otherAllocationID: should work
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": otherAllocationID,
				"json":       "",
				"remotepath": remotepath,
			}), true)
			require.Nil(t, err)
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
		}), true)
		require.Nil(t, err, "List file failed due to error ", strings.Join(output, "\n"))

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
		}), false)

		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, strings.Join(output, "\n"), "error from server list response:", strings.Join(output, "\n"))
	})

	t.Run("List All Files Should Work On An Empty Allocation", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		output, err := listAllFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}), true)
		require.Nil(t, err, "list files failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var listResults []climodel.ListFileResult

		err = json.Unmarshal([]byte(output[0]), &listResults)
		require.Nil(t, err, "list files failed", strings.Join(output, "\n"))
		require.Empty(t, listResults)
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
	buffer := make([]byte, size)
	rand.Read(buffer) //nolint:gosec,revive
	return os.WriteFile(name, buffer, os.ModePerm)
}

func generateRandomTestFileName(t *test.SystemTest) string {
	path := strings.TrimSuffix(os.TempDir(), string(os.PathSeparator))

	randomFilename := cliutils.RandomAlphaNumericString(10)
	return fmt.Sprintf("%s%s%s_test.txt", path, string(os.PathSeparator), randomFilename)
}

func shareFolderInAllocation(t *test.SystemTest, cliConfigFilename, param string) ([]string, error) {
	return shareFolderInAllocationForWallet(t, escapedTestName(t), cliConfigFilename, param)
}

func shareFolderInAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string) ([]string, error) {
	t.Logf("Sharing file/folder...")
	cmd := fmt.Sprintf(
		"./zbox share %s --silent --wallet %s --configDir ./config --config %s",
		param,
		wallet+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func listFilesInAllocation(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return listFilesInAllocationForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func listFilesInAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 10*time.Second) // TODO replace with poller
	t.Logf("Listing individual file in allocation...")
	cmd := fmt.Sprintf(
		"./zbox list %s --silent --wallet %s --configDir ./config --config %s",
		param,
		wallet+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func listAllFilesInAllocation(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 10*time.Second) // TODO replace with poller
	t.Logf("Listing all files in allocation...")
	cmd := fmt.Sprintf(
		"./zbox list-all %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
