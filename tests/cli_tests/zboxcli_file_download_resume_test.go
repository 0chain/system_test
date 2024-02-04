package cli_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestResumeDownload(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunWithTimeout("Resume download should work", 5*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(600 * MB)
		filesize := int64(300 * MB)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
			"data":   3,
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
			"allocation":  allocationID,
			"localpath":   filename,
			"remotepath":  remotepath + filepath.Base(filename),
			"chunknumber": 100,
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)
		defer func() {
			os.Remove(filename) //nolint: errcheck
		}()

		cmd, err := startDownloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation":      allocationID,
			"remotepath":      remotepath + filepath.Base(filename),
			"localpath":       filename,
			"blockspermarker": 100,
		}), false)
		require.Nil(t, err, "Download failed to start")

		idr, err := os.UserHomeDir()
		require.Nil(t, err)
		idr = filepath.Join(idr, ".zcn")
		hash := fnv.New64a()
		hash.Write([]byte(remotepath + filepath.Base(filename)))
		progressID := filepath.Join(idr, "download", allocationID[:8]+"_"+strconv.FormatUint(hash.Sum64(), 36))

		// Wait till more than 20% of the file is downloaded and send interrupt signal to command
		downloaded, dp := waitPartialDownloadAndInterrupt(t, cmd, filename, progressID, filesize)
		require.True(t, downloaded)

		// Allow command to stop
		time.Sleep(5 * time.Second)
		partialDownloadedBytes := int64(dp.LastWrittenBlock * 64 * KB * 3)
		percentDownloaded := float64(partialDownloadedBytes) / float64(filesize) * 100
		t.Logf("Partially downloaded %.2f%% of the file: %v / %v\n", percentDownloaded, partialDownloadedBytes, filesize)
		require.Greater(t, partialDownloadedBytes, int64(0))
		require.Less(t, partialDownloadedBytes, filesize)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation":      allocationID,
			"remotepath":      remotepath + filepath.Base(filename),
			"localpath":       filename,
			"blockspermarker": 100,
		}), true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		outputStatus := strings.Fields(output[0])
		actualDownloadedBytes, err := strconv.ParseInt(outputStatus[len(outputStatus)-5], 10, 64) // This gets the 5th element from the end
		require.Nil(t, err)

		t.Log("Bytes downloaded after resuming:", actualDownloadedBytes)
		require.InEpsilon(t, filesize-partialDownloadedBytes, actualDownloadedBytes, 0.005,
			fmt.Sprintf("Actual bytes downloaded after resume %v does not equal to expected amount of bytes %v",
				actualDownloadedBytes, filesize-partialDownloadedBytes))

		downloadedFileChecksum := generateChecksum(t, filename)
		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})
}

func startDownloadFile(t *test.SystemTest, cliConfigFilename, param string, retry bool) (*exec.Cmd, error) {
	return startDownloadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func startDownloadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) (*exec.Cmd, error) {
	t.Log("Downloading file...")
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

func waitPartialDownloadAndInterrupt(t *test.SystemTest, cmd *exec.Cmd, filename string, progressID string, filesize int64) (bool, sdk.DownloadProgress) {
	t.Log("Waiting till file is partially downloaded...")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	dp := sdk.DownloadProgress{}
	for {
		select {
		case <-ctx.Done():
			t.Log("Timeout waiting for partial download")
			return false, dp
		case <-time.After(2 * time.Second):
			buf, err := os.ReadFile(progressID)
			if err != nil {
				t.Log("Error reading download progress file:", err)
				continue
			}
			err = json.Unmarshal(buf, &dp)
			if err != nil {
				continue
			}
			if dp.LastWrittenBlock > 0 {
				//Send interrupt signal to command
				err := cmd.Process.Signal(os.Interrupt)
				require.Nil(t, err)
				t.Log("Interrupt signal sent to download command")
				return true, dp
			}
		}
	}
}
