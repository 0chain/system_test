package cli_tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestResumeDownload(testSetup *testing.T) {

	t := test.NewSystemTest(testSetup)

	t.Run("Download File from Root Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(500 * MB)
		filesize := int64(200 * MB)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)
		originalFileChecksum := generateChecksum(t, filename)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath + filepath.Base(filename),
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		localDownloadFolder := strings.TrimSuffix(os.TempDir(), string(os.PathSeparator))

		cmd, err := startDownloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  localDownloadFolder,
		}), true)
		require.Nil(t, err, "Download failed to start")

		cliutils.Wait(t, 5*time.Second)

	})
}

func startDownloadFile(t *test.SystemTest, cliConfigFilename, param string, retry bool) (*exec.Cmd, error) {
	return startDownloadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func startDownloadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) (*exec.Cmd, error) {
	cliutils.Wait(t, 15*time.Second) // TODO replace with pollers
	t.Logf("Downloading file...")
	cmd := fmt.Sprintf(
		"./zbox download %s --silent --wallet %s --configDir ./config --config %s",
		param,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.StartCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.StartCommandWithoutRetry(cmd)
	}
}
