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

	t.RunWithTimeout("Resume download should work", 10*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(4096 * MB)
		filesize := int64(3000 * MB)
		remotepath := "/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"lock":   9,
			"data":   2,
			"parity": 1,
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
		progressID := filepath.Join(idr, "download", "d"+allocationID[:8]+"_"+strconv.FormatUint(hash.Sum64(), 36))

		// Wait till more than 20% of the file is downloaded and send interrupt signal to command
		downloaded, dp := waitPartialDownloadAndInterrupt(t, cmd, filename, progressID, filesize)
		require.True(t, downloaded, "Could not capture partial download state - download may have completed too fast for interrupt on this infrastructure")

		// Allow command to stop
		time.Sleep(5 * time.Second)
		partialDownloadedBytes := int64(dp.LastWrittenBlock * 64 * KB * 2)
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
		t.Log("Output status:", outputStatus)
		// Parse downloaded bytes from progress output.
		// Progress output is multiple lines concatenated: "0 / TOTAL ... N / TOTAL ... FINAL / TOTAL"
		// The last "N / TOTAL" pair before completion shows actual bytes downloaded.
		// Find the last numeric value that appears before a "/" separator and is > 0.
		var actualDownloadedBytes int64
		for i, field := range outputStatus {
			if v, parseErr := strconv.ParseInt(field, 10, 64); parseErr == nil && v > 0 {
				// Check if next field is "/" (this is a "downloaded / total" pair)
				if i+1 < len(outputStatus) && outputStatus[i+1] == "/" {
					actualDownloadedBytes = v
				}
			}
		}
		require.NotZero(t, actualDownloadedBytes, "Could not parse downloaded bytes from output: %v", outputStatus)

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

func waitPartialDownloadAndInterrupt(t *test.SystemTest, cmd *exec.Cmd, filename, progressID string, filesize int64) (bool, sdk.DownloadProgress) {
	t.Logf("Waiting till file is partially downloaded... (progress file: %s)", progressID)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dp := sdk.DownloadProgress{}
	for {
		select {
		case <-ctx.Done():
			t.Log("Timeout waiting for partial download")
			// List what files exist in the download directory for debugging
			downloadDir := filepath.Dir(progressID)
			if files, err := filepath.Glob(filepath.Join(downloadDir, "*")); err == nil {
				t.Logf("Files in %s: %v", downloadDir, files)
			}
			return false, dp
		case <-time.After(2 * time.Second):
			buf, err := os.ReadFile(progressID)
			if err != nil {
				// Only log every 10th attempt to avoid noise
				continue
			}
			err = json.Unmarshal(buf, &dp)
			if err != nil {
				continue
			}
			if dp.LastWrittenBlock > 0 {
				// Send interrupt signal to command
				err := cmd.Process.Signal(os.Interrupt)
				require.Nil(t, err)
				t.Log("Interrupt signal sent to download command")
				return true, dp
			}
		}
	}
}
