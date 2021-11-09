package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileUpdate(t *testing.T) {
	t.Parallel()

	t.Run("update with another file of same size", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Get write pool info before file update
		output := writePoolInfo(t, configPath)
		initialWritePool := []climodel.WritePoolInfo{}
		err := json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		require.InEpsilonf(t, 0.5, intToZCN(initialWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		newLocalFilePath := updateFileWithRandomlyGeneratedData(t, allocationID, "/"+filepath.Base(localFilePath), int64(filesize))
		cost, unit := uploadCostWithUnit(t, configPath, allocationID, newLocalFilePath)
		expectedUploadCostInZCN := unitToZCN(cost, unit)

		// Expected cost takes into account data+parity, so we divide by that
		expectedUploadCostPerEntity := (expectedUploadCostInZCN / (2 + 2))

		// Wait before fetching final write pool
		wait(t, time.Minute/2)

		// Get the new Write Pool info after update
		output = writePoolInfo(t, configPath)
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		require.InEpsilon(t, (0.5 - expectedUploadCostPerEntity), intToZCN(finalWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5-expectedUploadCostPerEntity, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		go createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update with another file of bigger size", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Get write pool info before file update
		output := writePoolInfo(t, configPath)
		initialWritePool := []climodel.WritePoolInfo{}
		err := json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		require.InEpsilonf(t, 0.5, intToZCN(initialWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		newFileSize := 5 * MB
		newLocalFilePath := updateFileWithRandomlyGeneratedData(t, allocationID, "/"+filepath.Base(localFilePath), int64(newFileSize))
		cost, unit := uploadCostWithUnit(t, configPath, allocationID, newLocalFilePath)
		expectedUploadCostInZCN := unitToZCN(cost, unit)

		// Expected cost takes into account data+parity, so we divide by that
		expectedUploadCostPerEntity := (expectedUploadCostInZCN / (2 + 2))

		// Wait before fetching final write pool
		wait(t, time.Minute/2)

		// Get the new Write Pool info after update
		output = writePoolInfo(t, configPath)
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		require.InEpsilon(t, (0.5 - expectedUploadCostPerEntity), intToZCN(finalWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5-expectedUploadCostPerEntity, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		go createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update file with thumbnail", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		thumbnailFile := updateFileWithThumbnail(t, allocationID, "/"+filepath.Base(localFilePath), localFilePath, int64(filesize))

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"thumbnail":  true,
		}))

		// BUG: File download of thumbnail not working
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		
		defer func() {
			// Delete the downloaded thumbnail file
			err = os.Remove(thumbnailFile)
			require.Nil(t, err)
		}()
		go createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update file with commit", func(t *testing.T) {
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
		}))
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

		go createAllocationTestTeardown(t, allocationID)
	})
}

func updateFileWithThumbnail(t *testing.T, allocationID string, remotePath string, localpath string, size int64) string {
	thumbnail := "upload_thumbnail_test.png"
	output, err := cliutils.RunCommand("wget https://en.wikipedia.org/static/images/project-logos/enwiki-2x.png -O " + thumbnail)
	require.Nil(t, err, "Failed to download thumbnail png file: ", strings.Join(output, "\n"))

	output, err = updateFile(t, configPath, map[string]interface{}{
		"allocation":    allocationID,
		"remotepath":    remotePath,
		"localpath":     localpath,
		"thumbnailpath": thumbnail,
	})
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)
	return thumbnail
}

func updateFileWithCommit(t *testing.T, allocationID string, remotePath string, localpath string) {
	output, err := updateFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remotePath,
		"localpath":  localpath,
		"commit":     true,
	})
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)
}
