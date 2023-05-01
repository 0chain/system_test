package cli_tests

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
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
		allocationID := setupAllocation(t, configPath)
		createAllocationTestTeardown(t, allocationID)

		const remotepath = "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)
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

}

func rollbackAllocation(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {

	t.Log("Rollback allocation")

	cmd := fmt.Sprintf("./zbox rollback %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)

	return cliutils.RunCommand(t, cmd, 3, time.Second*2)

}
