package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

const StatusCompletedCBDD = "Status completed callback"

func TestDownloadDir(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Download Directory Should Work")
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	// Success Scenarios
	t.Run("Download Directory from Root Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/dir1"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		// originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Download Directory Concurrently Should Work for two Different Directory", 6*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(4096)
		filesize := int64(1024)
		remoteFilePaths := [2]string{"/dir1/", "/dir2/"}

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		fileNameOfFirstDirectory := generateFileAndUpload(t, allocationID, remoteFilePaths[0], filesize)
		fileNameOfSecondDirectory := generateFileAndUpload(t, allocationID, remoteFilePaths[1], filesize)

		// deleting uploaded file from /dir1 since we will be downloading it now
		err := os.Remove(fileNameOfFirstDirectory)
		require.Nil(t, err)

		// deleting uploaded file from /dir2 since we will be downloading it now
		err = os.Remove(fileNameOfSecondDirectory)
		require.Nil(t, err)

		var outputList [2][]string
		var errorList [2]error
		var wg sync.WaitGroup

		fileNames := [2]string{fileNameOfFirstDirectory, fileNameOfSecondDirectory}
		for index, fileName := range fileNames {
			wg.Add(1)
			go func(currentFileName string, currentIndex int) {
				defer wg.Done()
				op, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
					"allocation": allocationID,
					"remotepath": remoteFilePaths[currentIndex][:len(remoteFilePaths[currentIndex])-1],
					"localpath":  "tmp/",
				}), true)
				errorList[currentIndex] = err
				outputList[currentIndex] = op
			}(fileName, index)
		}

		wg.Wait()
		require.Nil(t, errorList[0], strings.Join(outputList[0], "\n"))
		require.Nil(t, errorList[1], strings.Join(outputList[1], "\n"))
	})

	t.Run("Download Directory from a Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/dir/dir1"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		// originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

	})

	t.Run("Download Directory from Nested Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/nested/dir/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		// originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Download Entire Directory Should Work but does not see blobber/issues/588", 3*time.Minute, func(t *test.SystemTest) { // todo: slow
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/nested/dir/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/dir",
		}), false)
		require.Error(t, err, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Download Directory From Shared Folder Should Work but does not see blobber/issues/588", 3*time.Minute, func(t *test.SystemTest) {
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/dirx"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Entire Folder from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocation(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			require.NotEqual(t, "", filename)
			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})
			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})
		// Just create a wallet so that we can work further
		createWallet(t)
		// Download file using auth-ticket: should work
		output, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/dir",
			"remotepath": "/dirx",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Download Shared File Should Work", 5*time.Minute, func(t *test.SystemTest) { // todo: too slow
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/dirx"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Directory from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocation(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)

			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just create a wallet so that we can work further
		createWallet(t)

		// Download file using auth-ticket: should work
		output, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Download  Directory having Encrypted files Should Work", 5*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(10 * MB)
		filesize := int64(10)
		remotepath := "/dirx"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath + filepath.Base(filename),
			"encrypt":    "",
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		// Downloading encrypted file should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  os.TempDir(),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})
}

func downloadDirectory(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return downloadDirectoryFromWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func downloadDirectoryFromWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 15*time.Second) // TODO replace with pollers
	t.Logf("Downloading file...")
	cmd := fmt.Sprintf(
		"./zbox downloaddir %s --silent --wallet %s --configDir ./config --config %s",
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
