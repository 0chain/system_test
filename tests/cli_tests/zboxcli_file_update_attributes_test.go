package cli_tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileUpdateAttributes(t *testing.T) {
	t.Parallel()

	t.Run("update file attribtes of existing file", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":              allocationID,
			"remotepath":              remotePath,
			"localpath":               file,
			"attr-who-pays-for-reads": "owner",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = updateFileAttributes(t, configPath, map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         remotePath,
			"who-pays-for-reads": "3rd_party",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "attributes updated", output[0])
	})

	t.Run("update attributes of existing file (no change)", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":              allocationID,
			"remotepath":              remotePath,
			"localpath":               file,
			"attr-who-pays-for-reads": "owner",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = updateFileAttributes(t, configPath, map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         remotePath,
			"who-pays-for-reads": "owner",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "no changes", output[0])
	})

	t.Run("update file attributes with commit param", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":              allocationID,
			"remotepath":              remotePath,
			"localpath":               file,
			"attr-who-pays-for-reads": "owner",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = updateFileAttributes(t, configPath, map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         remotePath,
			"who-pays-for-reads": "3rd_party",
			"commit":             "",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)
		require.Equal(t, "attributes updated", output[0])

		match := reCommitResponse.FindStringSubmatch(output[2])
		require.Len(t, match, 2)

		var commitResp climodel.CommitResponse
		err = json.Unmarshal([]byte(match[1]), &commitResp)
		require.Nil(t, err)

		require.Equal(t, "application/octet-stream", commitResp.MetaData.MimeType)
		require.Equal(t, fileSize, commitResp.MetaData.Size)
		require.Equal(t, filepath.Base(filename), commitResp.MetaData.Name)
		require.Equal(t, remotePath, commitResp.MetaData.Path)
		require.Equal(t, "", commitResp.MetaData.EncryptedKey)
	})

	t.Run("update attributes of non-existing file should fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := updateFileAttributes(t, configPath, map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         "/child/nonexisting.txt",
			"who-pays-for-reads": "3rd_party",
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "fetching the metadata: file_meta_error: Error getting the file meta data from blobbers", output[0])
	})

	t.Run("update file attributes from someone else's allocation should fail", func(t *testing.T) {
		t.Parallel()

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err := registerWalletForName(configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		// update file attributes
		output, err = updateFileAttributesForWallet(t, configPath, nonAllocOwnerWallet, map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         remotePath,
			"who-pays-for-reads": "3rd_party",
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "fetching the metadata: file_meta_error: Error getting the file meta data from blobbers", output[0])
	})

	t.Run("update file attributes with no allocation param should fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = updateFileAttributes(t, configPath, map[string]interface{}{
			"remotepath":         remotePath,
			"who-pays-for-reads": "3rd_party",
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing 'allocation' flag", output[0])
	})

	t.Run("update file attributes with no remotepath param should fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = updateFileAttributes(t, configPath, map[string]interface{}{
			"allocation":         allocationID,
			"who-pays-for-reads": "3rd_party",
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing 'remotepath' flag", output[0])
	})

	t.Run("update file attributes with no who-pays-for-reads param", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = updateFileAttributes(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "no changes", output[0])
	})

	// curator scenarios?
	// collaborator scenarios?
}

func updateFileAttributes(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	return updateFileAttributesForWallet(t, cliConfigFilename, escapedTestName(t), param)
}

func updateFileAttributesForWallet(t *testing.T, cliConfigFilename, wallet string, param map[string]interface{}) ([]string, error) {
	t.Logf("Updating file attributes...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox update-attributes %s --silent --wallet %s --configDir ./config --config %s",
		p,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithRetry(t, cmd, 3, time.Second*20)
}
