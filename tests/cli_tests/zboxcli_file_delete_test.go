package cli_tests

import (
	"encoding/base64"
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

		thumbnail := "upload_thumbnail_test.png"
		thumbnailPath := "./tmp/" + thumbnail

		//nolint
		thumbnailBytes, _ := base64.StdEncoding.DecodeString(`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`)
		err := os.WriteFile(thumbnailPath, thumbnailBytes, os.ModePerm)
		require.Nil(t, err, "failed to generate thumbnail", err)

		defer func() {
			os.Remove(thumbnailPath)
		}()

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, filesize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":    allocationID,
			"remotepath":    remotepath,
			"localpath":     filename,
			"thumbnailpath": thumbnailPath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		fname := filepath.Base(filename)
		remoteFilePath := filepath.Join(remotepath, fname)

		output, err = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), false)
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
