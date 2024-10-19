package cli_tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestRepairSize(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("repair size should work", 5*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(512 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = text/plain. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		output, err = getRepairSize(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
		}, true)
		require.Nilf(t, err, "error getting repair size: %v", err)
		var rs model.RepairSize
		err = json.Unmarshal([]byte(output[0]), &rs)
		require.Nilf(t, err, "error unmarshal repair size: %v", err)
		require.Equal(t, uint64(0), rs.UploadSize, "upload size should be zero")
		require.Equal(t, uint64(0), rs.DownloadSize, "download size should be zero")
		// optional repairpath
		output, err = getRepairSize(t, configPath, map[string]interface{}{
			"allocation": allocationID,
		}, true)
		require.Nilf(t, err, "error getting repair size: %v", err)
		var rs2 model.RepairSize
		err = json.Unmarshal([]byte(output[0]), &rs2)
		require.Nilf(t, err, "error unmarshal repair size: %v", err)
		require.Equal(t, uint64(0), rs2.UploadSize, "upload size should be zero")
		require.Equal(t, uint64(0), rs2.DownloadSize, "download size should be zero")
	})
}

func getRepairSize(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return getRepairSizeForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func getRepairSizeForWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("getting Repair size...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox repair-size %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*40)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
