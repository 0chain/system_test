package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestRollbackAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.RunSequentially("rollback allocation after updating a file should work", func(t *test.SystemTest) {
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   4 * MB,
			"tokens": 9,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		originalFileChecksum := generateChecksum(t, localFilePath)

		err := os.Remove(localFilePath)
		require.Nil(t, err)

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

		// rollback allocation

		output, err = rollbackAllocation(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		t.Log(strings.Join(output, "\n"))
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(localFilePath))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(localFilePath))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)

		createAllocationTestTeardown(t, allocationID)
	})

	t.RunSequentially("rollback allocation after deleting a file should work", func(t *test.SystemTest) {
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * MB,
			"tokens": 9,
		})
		createAllocationTestTeardown(t, allocationID)

		const remotepath = "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)
		time.Sleep(1 * time.Second)
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

		output, err = rollbackAllocation(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		t.Log(strings.Join(output, "\n"))
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.RunSequentially("rollback allocation after moving a file should work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destpath := "/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])
		time.Sleep(1 * time.Second)
		// move file
		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" moved"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been moved
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))

		// rollback
		output, err = rollbackAllocation(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		t.Log(strings.Join(output, "\n"))
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		files = []climodel.AllocationFile{}
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been moved
		foundAtSource = false
		foundAtDest = false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.False(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.RunSequentially("rollback allocation after renaming a file should work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destName := "new_" + filename
		destPath := "/child/" + destName

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])
		time.Sleep(1 * time.Second)
		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" renamed"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been renamed
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destPath {
				foundAtDest = true
				require.Equal(t, destName, f.Name, strings.Join(output, "\n"))
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))

		// rollback
		output, err = rollbackAllocation(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		t.Log(strings.Join(output, "\n"))
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		files = []climodel.AllocationFile{}
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been renamed
		foundAtSource = false
		foundAtDest = false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destPath {
				foundAtDest = true
				require.Equal(t, destName, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.False(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})
}

func rollbackAllocation(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Log("Rollback allocation")

	cmd := fmt.Sprintf("./zbox rollback %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)

	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}
