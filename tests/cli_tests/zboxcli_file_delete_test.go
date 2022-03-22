package cli_tests

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileDelete(t *testing.T) {
	t.Parallel()

	t.Run("delete existing file in root directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		remotepath := "/"
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

	t.Run("delete existing file in sub directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

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

	t.Run("delete existing file with encryption should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

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

	t.Run("delete shared file by owner should work", func(t *testing.T) {
		t.Parallel()

		collaboratorWalletName := escapedTestName(t) + "_collaborator"

		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Collaborator %s added successfully for the file %s", collaboratorWallet.ClientID, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		output, err = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
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

	t.Run("delete existing non-root directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

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
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete existing file with thumbnail should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

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

	t.Run("delete existing root directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

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

	t.Run("delete file that does not exist should fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		remotepath := "/"

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + "doesnotexist",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Delete failed. Delete failed: Success_rate", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("delete file by not supplying remotepath should fail", func(t *testing.T) {
		t.Parallel()

		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": "abc",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: remotepath flag is missing", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("delete file by not supplying allocation ID should fail", func(t *testing.T) {
		t.Parallel()

		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"remotepath": "/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: allocation flag is missing", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("delete existing file in root directory with wallet balance accounting", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		remotepath := "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getBalance(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 500.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

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

		output, err = getBalance(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 500.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("delete existing file in someone else's allocation should fail", func(t *testing.T) {
		t.Parallel()

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

	t.Run("delete shared file by collaborator should fail", func(t *testing.T) {
		t.Parallel()

		collaboratorWalletName := escapedTestName(t) + "_collaborator"

		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Collaborator %s added successfully for the file %s", collaboratorWallet.ClientID, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		output, err = deleteFile(t, collaboratorWalletName, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Delete failed. Delete failed: Success_rate", "Unexpected output", strings.Join(output, "\n"))

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		}), false)

		require.Nil(t, err, "List files after delete failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], remotepath, strings.Join(output, "\n"))
	})
}
