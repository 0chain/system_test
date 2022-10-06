package cli_tests

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestLivestreamDownload(t *testing.T) {
	t.Parallel()

	feed, isStreamAvailable := checkYoutubeFeedAvailabiity()

	if !isStreamAvailable {
		t.Skipf("Youtube live streams are not available right now!")
	}

	t.Run("Downloading youtube feed to allocation should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"

		localfolder := filepath.Join(os.TempDir()+"upload", escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)

		localfolderForDownload := filepath.Join(os.TempDir()+"download", escapedTestName(t))
		localpathForDownload := filepath.Join(localfolderForDownload, "down.m3u8")
		err = os.MkdirAll(localpathForDownload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpathForDownload)

		defer os.RemoveAll(localfolder)
		defer os.RemoveAll(localfolderForDownload)

		wg := new(sync.WaitGroup)
		wg.Add(2)

		errUploadChan := make(chan error, 10)
		errDownloadChan := make(chan error, 10)

		go helperUploadFeed(wg, errUploadChan, t, configPath, "feed", createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
			"feed":       feed,
		}))

		go startDownloadFeed(wg, errDownloadChan, t, configPath, "download --live", createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload,
		}))

		wg.Wait()

		err = <-errUploadChan
		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		err = <-errDownloadChan
		require.Nil(t, err, "error in killing download command")

		close(errDownloadChan)
		close(errUploadChan)

		count_m3u8 := 0
		err = filepath.Walk(localfolderForDownload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			t.Log(info.Name())
			extension := strings.Split(info.Name(), ".")
			if extension[len(extension)-1] == "m3u8" {
				count_m3u8 += 1
				return nil
			}
			return nil
		})
		require.Nil(t, err, "error in traversing locally created .m3u8 and .ts files!")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created!")

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error listing files in remotepath")
		require.Len(t, output, 1, "listing files should return an output")

		files := []climodel.ListFileResult{}
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "error in unmarshalling the response from the list files")

		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
		}
	})
	/*
		t.Run("Downloading local webcam feed to allocation", func(t *testing.T) {
		})
		t.Run("Download from the feed with delay flag", func(t *testing.T) {
		})
		t.Run("Download from the feed with blockspermarker flag", func(t *testing.T) {
		})
		t.Run("Download from the feed starting from a specific block number (--startblock)", func(t *testing.T) {
		})
		t.Run("Download from the feed till a specific block number (--endblock)", func(t *testing.T) {
		})
	*/
}

func helperUploadFeed(wg *sync.WaitGroup, errChan chan error, t *testing.T, cmdName string, cliConfigFilename string, params string) {

	defer wg.Done()

	err := startUploadFeed(t, cmdName, cliConfigFilename, params)

	errChan <- err
}

func startDownloadFeed(wg *sync.WaitGroup, errChan chan error, t *testing.T, cmdName, cliConfigFilename, params string) {

	defer wg.Done()

	t.Logf("Starting download of live stream from zbox.")

	commandString := fmt.Sprintf("./zbox %s %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, cmdName, params)
	t.Logf(commandString)
	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)

	require.Nil(t, err, "error in downloading a live feed")

	cliutils.Wait(t, 20*time.Second)

	err = cmd.Process.Kill()

	errChan <- err
}
