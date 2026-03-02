package cli_tests

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

// Fixed
func TestFileDelete(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("delete existing file in root directory should work")

	t.Parallel()
	t.Run("delete non root directory with multiple existing file present should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		const remotepath = "/"
		filesize := int64(1 * KB)
		filename1 := generateFileAndUpload(t, allocationID, remotepath, filesize)
		filename2 := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname1 := filepath.Base(filename1)
		remoteFilePath1 := path.Join(remotepath, fname1)
		fname2 := filepath.Base(filename2)
		remoteFilePath2 := path.Join(remotepath, fname2)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remotepath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath1,
			"json":       "",
		}), true)
		require.NotNil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Invalid path record not found")

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath2,
			"json":       "",
		}), true)
		require.NotNil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Invalid path record not found")
	})

	t.Run("delete non-root directory with No existing file should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)
		dirname := "/child"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", dirname), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"json":       "",
		}), true)
		require.NotNil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `error from server list response: {"code":"invalid_parameters","error":"invalid_parameters: Invalid path record not found"}`, output[0], strings.Join(output, "\n"))
	})

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

	t.Run("delete existing non-root directory with file present should work", func(t *test.SystemTest) {
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

	t.Run("delete existing root directory with files present should work", func(t *test.SystemTest) {
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
		createWallet(t)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": "abc",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: remotepath flag is missing", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("delete file by not supplying allocation ID should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"remotepath": "/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: allocation flag is missing", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("delete existing file in root directory with wallet balance accounting", func(t *test.SystemTest) {
		createWallet(t)

		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		remotepath := "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		// Measure balance AFTER allocation setup and upload (setupAllocation calls faucet internally)
		cliutils.Wait(t, 5*time.Second)
		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

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

		// Wait for delete transaction to finalize on chain before checking balance
		cliutils.Wait(t, 15*time.Second)

		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Delete operation should only cost a small transaction fee
		deleteCost := balanceBefore - balanceAfter
		require.GreaterOrEqual(t, deleteCost, 0.0, "Balance should not increase after delete")
		require.Less(t, deleteCost, 2.0, "Delete transaction fee should be reasonable")
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
		const allocSize int64 = 64 * KB * 2 * 2
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		const remotePathPrefix = "/"

		// Upload files sequentially to avoid nonce collisions during setup.
		remotePaths := make([]string, 0, 2)
		for i := 0; i < 2; i++ {
			fileName := filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
			remotePaths = append(remotePaths, filepath.Join(remotePathPrefix, fileName))
		}

		// Delete both files concurrently using multi-operation (single transaction, no nonce collision).
		err := MultiDelete(escapedTestName(t), configPath, allocationID, remotePaths)
		require.Nil(t, err, "multi-operation delete failed")

		for _, remotePath := range remotePaths {
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotePath,
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
