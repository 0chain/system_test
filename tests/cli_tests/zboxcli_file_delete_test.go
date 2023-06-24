package cli_tests

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

func TestFileDelete(testSetup *testing.T) {
	//todo: slow operations
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("delete existing file in root directory should work")

	t.Parallel()

	t.Run("delete existing file in root directory should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		const remotepath = "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remoteFilePath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete existing file in sub directory should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/root/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remoteFilePath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete existing file with encryption should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/"

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 10)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  filename,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remoteFilePath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete existing non-root directory should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/root/"
		filesize := int64(1 * KB)
		generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remotepath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Invalid path record not found")
	})

	t.Run("delete existing file with thumbnail should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/"
		filesize := int64(1 * KB)

		thumbnail := escapedTestName(t) + "thumbnail.png"

		//nolint
		generateThumbnail(t, thumbnail)

		localFilePath := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{"thumbnailpath": thumbnail})
		//nolint: errcheck
		os.Remove(thumbnail)
		//nolint: errcheck
		os.Remove(localFilePath)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remotepath+filepath.Base(localFilePath)), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete existing root directory should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/"
		filesize := int64(1 * KB)
		generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remotepath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete file that does not exist should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/"

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + "doesnotexist",
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remotepath+"doesnotexist"), output[0])
	})

	t.Run("delete file by not supplying remotepath should fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": "abc",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: remotepath flag is missing", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("delete file by not supplying allocation ID should fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"remotepath": "/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: allocation flag is missing", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("delete existing file in root directory with wallet balance accounting", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 1.99, balance)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remoteFilePath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))

		balance, err = getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 1.99, balance)
	})

	t.Run("delete existing file in someone else's allocation should fail", func(t *test.SystemTest) {
		var allocationID, filename string
		remotepath := "/"
		filesize := int64(2)
		setupAllocation(t, configPath)

		allocationID = setupAllocationWithWallet(t, escapedTestName(t)+"_owner", configPath)
		filename = generateFileAndUploadForWallet(t, escapedTestName(t)+"_owner", allocationID, remotepath, filesize)
		require.NotEqual(t, "", filename)

		remoteFilePath := remotepath + filepath.Base(filename)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Delete failed")

		output, err = listFilesInAllocationForWallet(t, escapedTestName(t)+"_owner", configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], remotepath, strings.Join(output, "\n"))
	})

	t.Run("delete existing file with allocation delete file options forbidden should fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"data":          1,
			"parity":        1,
			"forbid_delete": nil,
		})

		remotepath := "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "this options for this file is not permitted for this allocation")

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), remoteFilePath, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Delete file concurrently in existing directory, should work", 5*time.Minute, func(t *test.SystemTest) { // TODO: slow
		const allocSize int64 = 2048
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"expire": "5m",
		})

		var fileNames [2]string

		const remotePathPrefix = "/"

		var outputList [2][]string
		var errorList [2]error
		var wg sync.WaitGroup

		for i, fileName := range fileNames {
			wg.Add(1)
			go func(currentFileName string, currentIndex int) {
				defer wg.Done()

				fileName := filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
				fileNames[currentIndex] = fileName

				remoteFilePath := filepath.Join(remotePathPrefix, fileName)

				op, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
					"allocation": allocationID,
					"remotepath": remoteFilePath,
				}), true)

				errorList[currentIndex] = err
				outputList[currentIndex] = op
			}(fileName, i)
		}

		wg.Wait()

		const expectedPattern = "%s deleted"

		for i := 0; i < 2; i++ {
			require.Nil(t, errorList[i], strings.Join(outputList[i], "\n"))
			require.Len(t, outputList, 2, strings.Join(outputList[i], "\n"))

			require.Equal(t, fmt.Sprintf(expectedPattern, fileNames[i]), filepath.Base(outputList[i][0]), "Output is not appropriate")
		}

		for i := 0; i < 2; i++ {
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": path.Join(remotePathPrefix, fileNames[i]),
				"json":       "",
			}), true)

			require.NotNil(t, err, strings.Join(output, "\n"))
			require.Contains(t, strings.Join(output, "\n"), "Invalid path record not found")
		}
	})

}

func deleteFile(t *test.SystemTest, walletName, params string, retry bool) ([]string, error) {
	t.Logf("Deleting file...")
	cmd := fmt.Sprintf(
		"./zbox delete %s --silent --wallet %s "+
			"--configDir ./config --config %s",
		params,
		walletName+"_wallet.json",
		configPath,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
