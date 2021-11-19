package cli_tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileUpdate(t *testing.T) {
	t.Parallel()

	t.Run("update with another file of same size should work", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{"allocation": allocationID, "remotepath": remotepath + filepath.Base(localFilePath)}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		actualSize, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(output[2], "|")[4]), 64)
		require.Nil(t, err)
		require.Equal(t, 0.5*MB, actualSize, "file size should be same as uploaded")

		updateFileWithRandomlyGeneratedData(t, allocationID, "/"+filepath.Base(localFilePath), int64(filesize))

		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{"allocation": allocationID, "remotepath": remotepath + filepath.Base(localFilePath)}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		actualSize, err = strconv.ParseFloat(strings.TrimSpace(strings.Split(output[2], "|")[4]), 64)
		require.Nil(t, err)
		require.Equal(t, 0.5*MB, actualSize)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update with another file of bigger size should work", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{"allocation": allocationID, "remotepath": remotepath + filepath.Base(localFilePath)}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		actualSize, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(output[2], "|")[4]), 64)
		require.Nil(t, err)
		require.Equal(t, 0.5*MB, actualSize, "file size should be same as uploaded")

		newFileSize := int64(1.5 * MB)
		updateFileWithRandomlyGeneratedData(t, allocationID, "/"+filepath.Base(localFilePath), int64(newFileSize))

		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{"allocation": allocationID, "remotepath": remotepath + filepath.Base(localFilePath)}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		actualSize, err = strconv.ParseFloat(strings.TrimSpace(strings.Split(output[2], "|")[4]), 64)
		require.Nil(t, err)
		require.Equal(t, 1.5*MB, actualSize)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update existing file with commit metadata should work", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		updateFileWithCommit(t, allocationID, "/"+filepath.Base(localFilePath), localFilePath)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"commit":     true,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(localFilePath),
		)
		require.Equal(t, expected, output[1])

		match := reCommitResponse.FindStringSubmatch(output[2])
		require.Len(t, match, 2)

		var commitResp climodel.CommitResponse
		err = json.Unmarshal([]byte(match[1]), &commitResp)
		require.Nil(t, err)

		require.Equal(t, "application/octet-stream", commitResp.MetaData.MimeType)
		require.Equal(t, filesize, commitResp.MetaData.Size)
		require.Equal(t, filepath.Base(localFilePath), commitResp.MetaData.Name)
		require.Equal(t, remotepath+filepath.Base(localFilePath), commitResp.MetaData.Path)
		require.Equal(t, "", commitResp.MetaData.EncryptedKey)
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(localFilePath))

		originalFileChecksum := generateChecksum(t, localFilePath)
		require.Equal(t, originalFileChecksum, downloadedFileChecksum)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update file that does not exists should fail", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		localfile := generateRandomTestFileName(t)
		err := createFileWithSize(localfile, filesize)
		require.Nil(t, err)

		output, err := updateFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(localfile),
			"localpath":  localfile,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update non-encrypted file with encrypted file should work", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		localfile := generateRandomTestFileName(t)
		err := createFileWithSize(localfile, int64(filesize))
		require.Nil(t, err)

		params := createParams(map[string]interface{}{"allocation": allocationID, "remotepath": "/"})
		output, err := listFilesInAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		isEncrypted := strings.Split(output[2], "|")[6]
		require.Equal(t, "NO", strings.TrimSpace(isEncrypted))

		// update with encrypted file
		output, err = updateFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(localFilePath),
			"localpath":  localfile,
			"encrypt":    true,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		params = createParams(map[string]interface{}{"allocation": allocationID, "remotepath": "/"})
		output, err = listFilesInAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		isEncrypted = strings.Split(output[2], "|")[6]
		require.Equal(t, "YES", strings.TrimSpace(isEncrypted))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update encrypted file with non-encrypted file should work", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{"encrypt": true})

		localfile := generateRandomTestFileName(t)
		err := createFileWithSize(localfile, int64(filesize))
		require.Nil(t, err)

		params := createParams(map[string]interface{}{"allocation": allocationID, "remotepath": "/"})
		output, err := listFilesInAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		isEncrypted := strings.Split(output[2], "|")[6]
		require.Equal(t, "YES", strings.TrimSpace(isEncrypted))

		// update with encrypted file
		output, err = updateFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(localFilePath),
			"localpath":  localfile,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		params = createParams(map[string]interface{}{"allocation": allocationID, "remotepath": "/"})
		output, err = listFilesInAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		yes := strings.Split(output[2], "|")[6]
		require.Equal(t, "NO", strings.TrimSpace(yes))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update encrypted file with encrypted file should work", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{"encrypt": true})

		params := createParams(map[string]interface{}{"allocation": allocationID, "remotepath": "/"})
		output, err := listFilesInAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		isEncrypted := strings.Split(output[2], "|")[6]
		require.Equal(t, "YES", strings.TrimSpace(isEncrypted))
		filename := strings.Split(output[2], "|")[1]
		require.Equal(t, filepath.Base(localFilePath), strings.TrimSpace(filename))

		localfile := generateRandomTestFileName(t)
		err = createFileWithSize(localfile, int64(filesize))
		require.Nil(t, err)

		// update with encrypted file
		output, err = updateFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(localFilePath),
			"localpath":  localfile,
			"encrypt":    true,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		params = createParams(map[string]interface{}{"allocation": allocationID, "remotepath": "/"})
		output, err = listFilesInAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		yes := strings.Split(output[2], "|")[6]
		require.Equal(t, "YES", strings.Trim(yes, " "))
		filename = strings.Split(output[2], "|")[1]
		require.Equal(t, filepath.Base(localFilePath), strings.TrimSpace(filename))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update with another file of size larger than allocation should fail", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 1 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		newFileSize := 2 * MB
		localfile := generateRandomTestFileName(t)
		err := createFileWithSize(localfile, int64(newFileSize))
		require.Nil(t, err)

		output, err := updateFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(localFilePath),
			"localpath":  localfile,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		createAllocationTestTeardown(t, allocationID)
	})
}

func updateFileWithThumbnailURL(t *testing.T, thumbnailURL, allocationID, remotePath, localpath string, size int64) string {
	thumbnail := "upload_thumbnail_test.png"
	output, err := cliutils.RunCommandWithoutRetry(fmt.Sprintf("wget %s -O %s", thumbnailURL, thumbnail))
	require.Nil(t, err, "Failed to download thumbnail png file: ", strings.Join(output, "\n"))

	output, err = updateFile(t, configPath, map[string]interface{}{
		"allocation":    allocationID,
		"remotepath":    remotePath,
		"localpath":     localpath,
		"thumbnailpath": thumbnail,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)
	require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(localpath), output[1])
	return thumbnail
}

func updateFileWithCommit(t *testing.T, allocationID, remotePath, localpath string) {
	output, err := updateFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remotePath,
		"localpath":  localpath,
		"commit":     true,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 3)
	require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(localpath), output[1])
}
