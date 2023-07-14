package cli_tests

import (
	"context"
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

	t.RunWithTimeout("Resume download should work", 5*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(500 * MB)
		filesize := int64(400 * MB)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})
		defer func() {
			createAllocationTestTeardown(t, allocationID)
		}()

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

		cmd, err := startDownloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}), false)
		require.Nil(t, err, "Download failed to start")
		defer func() {
			os.Remove(filename) //nolint: errcheck
		}()

		// Wait till 20% of the file is downloaded
		downloaded := waitPartialDownload(t, filename, filesize)
		require.True(t, downloaded)

		// Send interrupt signal to command
		err = cmd.Process.Signal(os.Interrupt)
		require.Nil(t, err)

		// Allow command to stop
		time.Sleep(1 * time.Second)

		info, err := os.Stat(filename)
		require.Nil(t, err, "File was not partially downloaded")

		percentDownloaded := float64(info.Size()) / float64(filesize) * 100
		t.Logf("Partially downloaded %.2f%% of the file: %v / %v\n", percentDownloaded, info.Size(), filesize)

		require.Greater(t, info.Size(), int64(0))
		require.Less(t, info.Size(), filesize)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}), true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, filename)

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})
}

func startDownloadFile(t *test.SystemTest, cliConfigFilename, param string, retry bool) (*exec.Cmd, error) {
	return startDownloadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func startDownloadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) (*exec.Cmd, error) {
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

func waitPartialDownload(t *test.SystemTest, filename string, filesize int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Log("Timeout waiting for partial download")
			return false
		case <-time.After(1 * time.Second):
			info, err := os.Stat(filename)
			if err != nil {
				t.Logf("File is not yet created: %v", err)
				continue
			}
			if info.Size() > filesize/5 {
				t.Log("File is more than 20% downloaded")
				return true
			}
		}
	}
}
