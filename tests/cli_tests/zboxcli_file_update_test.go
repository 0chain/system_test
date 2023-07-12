package cli_tests

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileUpdate(testSetup *testing.T) {
	//todo: very slow executions observed
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("update file with thumbnail")

	t.Parallel()

	t.Run("update file with thumbnail", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		thumbnailFile, thumbnailSize := updateFileWithThumbnail(t, allocationID, "/"+filepath.Base(localFilePath), localFilePath, int64(filesize))
		os.Remove(thumbnailFile) //nolint: errcheck

		downloadThumbnailDir := thumbnailFile + "down"
		defer os.RemoveAll(downloadThumbnailDir) //nolint: errcheck
		remotepath += filepath.Base(localFilePath)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  downloadThumbnailDir,
			"thumbnail":  true,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		localThumbnail := filepath.Join(downloadThumbnailDir, filepath.Base(remotepath))
		stats, err := os.Stat(localThumbnail)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, thumbnailSize, int(stats.Size()))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File Update with same size - Users should not be charged, blobber should not be paid", func(t *test.SystemTest) {
		// Logic: Upload a 1 MB file, get the write pool info. Update said file with another file
		// of size 1 MB. Get write pool info and check nothing has been deducted.

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "5",
			"size": 4 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		fileSize := int64(math.Floor(1 * MB))

		// Upload 1 MB file
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)

		cliutils.Wait(t, 30*time.Second)

		// initial write pool
		initialAllocation := getAllocation(t, allocationID)

		// Update with same size
		remotepath := "/" + filepath.Base(localpath)
		updateFileWithRandomlyGeneratedData(t, allocationID, remotepath, fileSize)

		// Get expected upload cost
		output, _ = getUploadCostInUnit(t, configPath, allocationID, localpath)

		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost is given in "per 720 hours", we need 1 hour
		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := expectedUploadCostInZCN / ((2 + 2) * 720)

		finalAllocation := getAllocation(t, allocationID)

		actualCost := initialAllocation.WritePool - finalAllocation.WritePool
		require.True(t, actualCost == 0 || intToZCN(actualCost) == actualExpectedUploadCostInZCN)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update thumbnail of uploaded file", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		thumbnail := escapedTestName(t) + "thumbnail.png"
		generateThumbnail(t, thumbnail) //nolint

		localFilePath := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{"thumbnailpath": thumbnail})

		os.Remove(thumbnail)     //nolint: errcheck
		os.Remove(localFilePath) //nolint: errcheck
		downloadDir, _ := filepath.Split(localFilePath)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  downloadDir,
			"thumbnail":  true,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		// Update with new thumbnail
		newThumbnail, newThumbnailSize := updateFileWithThumbnail(t, allocationID, "/"+filepath.Base(localFilePath), localFilePath, int64(filesize))

		os.Remove(newThumbnail)  //nolint: errcheck
		os.Remove(localFilePath) //nolint: errcheck

		downloadNewThumbnailDir := newThumbnail + "down"
		defer os.RemoveAll(downloadNewThumbnailDir) //nolint: errcheck

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  downloadNewThumbnailDir,
			"thumbnail":  true,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		localNewThumbnail := filepath.Join(downloadNewThumbnailDir, filepath.Base(remotepath))
		stat, err := os.Stat(localNewThumbnail)
		require.Nil(t, err)
		require.Equal(t, newThumbnailSize, int(stat.Size()))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update with another file of same size should work", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + filepath.Base(localFilePath),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, filesize, meta.ActualFileSize, "file size should be same as uploaded")

		updateFileWithRandomlyGeneratedData(t, allocationID, "/"+filepath.Base(localFilePath), int64(filesize))
		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + filepath.Base(localFilePath),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, filesize, meta.ActualFileSize)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update with another file of bigger size should work", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + filepath.Base(localFilePath),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var meta climodel.FileMetaResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, filesize, meta.ActualFileSize, "file size should be same as uploaded")

		newFileSize := int64(1.5 * MB)
		updateFileWithRandomlyGeneratedData(t, allocationID, "/"+filepath.Base(localFilePath), int64(newFileSize))

		output, err = getFileMeta(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath + filepath.Base(localFilePath),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, newFileSize, meta.ActualFileSize)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update non-encrypted file with encrypted file should work", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

		isEncrypted := strings.Split(output[2], "|")[8]
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

		isEncrypted = strings.Split(output[2], "|")[8]
		require.Equal(t, "YES", strings.TrimSpace(isEncrypted))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update encrypted file with non-encrypted file should work", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

		isEncrypted := strings.Split(output[2], "|")[8]
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

		yes := strings.Split(output[2], "|")[8]
		require.Equal(t, "NO", strings.TrimSpace(yes))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update encrypted file with encrypted file should work", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{"encrypt": true})

		params := createParams(map[string]interface{}{"allocation": allocationID, "remotepath": "/"})
		output, err := listFilesInAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		isEncrypted := strings.Split(output[2], "|")[8]
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

		yes := strings.Split(output[2], "|")[8]
		require.Equal(t, "YES", strings.Trim(yes, " "))
		filename = strings.Split(output[2], "|")[1]
		require.Equal(t, filepath.Base(localFilePath), strings.TrimSpace(filename))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update file that does not exists should fail", func(t *test.SystemTest) {
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
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "file_meta_error: Error getting the file meta data from blobbers")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update with another file of size larger than allocation should fail", func(t *test.SystemTest) {
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
		require.True(t, strings.Contains(strings.Join(output, "\n"), "alloc: no enough space left in allocation"), strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update with allocation update file option forbidden should fail", func(t *test.SystemTest) {
		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB, "forbid_update": nil})

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
		require.True(t, strings.Contains(strings.Join(output, "\n"), "this options for this file is not permitted for this allocation"), strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})
}

func generateThumbnail(t *test.SystemTest, localpath string) int {
	//nolint
	thumbnailBytes, _ := base64.StdEncoding.DecodeString(`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`)
	err := os.WriteFile(localpath, thumbnailBytes, os.ModePerm)
	require.Nil(t, err, "failed to generate thumbnail", err)

	return len(thumbnailBytes)
}

// nolint
func updateFileWithThumbnail(t *test.SystemTest, allocationID, remotePath, localpath string, size int64) (string, int) {
	thumbnail := escapedTestName(t) + "thumbnail.png"

	thumbnailSize := generateThumbnail(t, thumbnail)

	output, err := updateFile(t, configPath, map[string]interface{}{
		"allocation":    allocationID,
		"remotepath":    remotePath,
		"localpath":     localpath,
		"thumbnailpath": thumbnail,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)
	require.True(t, strings.HasPrefix(output[1], "Status completed callback.") && strings.HasSuffix(output[1], "Name = "+filepath.Base(localpath)))
	return thumbnail, thumbnailSize
}
