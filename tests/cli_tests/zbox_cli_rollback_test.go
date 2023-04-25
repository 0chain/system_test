package cli_tests

import (
	"fmt"
	"os"
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

	t.RunSequentially("rollback allocation", func(t *test.SystemTest) {

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

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

}

func rollbackAllocation(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {

	t.Log("Rollback allocation")

	cmd := fmt.Sprintf("./zbox rollback %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)

	return cliutils.RunCommand(t, cmd, 3, time.Second*2)

}
