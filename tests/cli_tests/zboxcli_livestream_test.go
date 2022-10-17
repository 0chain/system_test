package cli_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestStreamUploadDownload(t *testing.T) {
	t.Parallel()
	// 24*7 lofi playlist that we will use to test --feed --sync flags
	KillFFMPEG()

	feed, isStreamAvailable := checkYoutubeFeedAvailabiity()

	if !isStreamAvailable {
		t.Skipf("No youtube live feed available right now")
	}

	// Success scenarios

	t.Run("Uploading youtube feed to allocation should work", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "feed", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
			"feed":       feed,
		}))
		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		// Check some .ts files and 1 .m3u8 file must have been created on localpath by youtube-dl
		count_m3u8 := 0
		err = filepath.Walk(localfolder,
			func(path string, info fs.FileInfo, err error) error {
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
		require.Nil(t, err, "error in traversing locally created .m3u8 or .ts files")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created")

		// Check all locally created files have been uploaded to allocation
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error listing files in remotepath")
		require.Len(t, output, 1, "listing files should return output")

		files := []climodel.ListFileResult{}
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "error unmarshalling the response from list files")

		// remotepath must have numbered .ts files
		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
		}
	})

	t.Run("Upload from feed with delay flag must work", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "feed", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
			"feed":       feed,
			"delay":      10,
		}))
		require.Nil(t, err, "error in killing upload command with delay flag")
		KillFFMPEG()

		// Check some .ts files and 1 .m3u8 file must have been created on localpath by youtube-dl
		count_m3u8 := 0
		err = filepath.Walk(localfolder,
			func(path string, info fs.FileInfo, err error) error {
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
		require.Nil(t, err, "error in traversing locally created .m3u8 or .ts files")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created")

		// Check all locally created files have been uploaded to allocation
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error listing files in remotepath")
		require.Len(t, output, 1, "listing files should return output")

		files := []climodel.ListFileResult{}
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "error unmarshalling the response from list files")

		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
		}
	})

	t.Run("Upload from feed with a different chunknumber must work", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "feed", localfolder, createParams(map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  remotepath,
			"localpath":   localpath,
			"feed":        feed,
			"chunknumber": 10,
		}))
		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		// Check some .ts files and 1 .m3u8 file must have been created on localpath by youtube-dl
		count_m3u8 := 0
		err = filepath.Walk(localfolder,
			func(path string, info fs.FileInfo, err error) error {
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
		require.Nil(t, err, "error in traversing locally created .m3u8 or .ts files")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created")

		// Check all locally created files have been uploaded to allocation
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error listing files in remotepath")
		require.Len(t, output, 1, "listing files should return output")

		files := []climodel.ListFileResult{}
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "error unmarshalling the response from list files")

		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
			// FIXME: Num of blocks must be equal to ceil(size/chunksize)
			// require.Equal(t, int64(file.NumBlocks), math.Ceil(float64(file.Size)/float64(chunksize)), "chunksize should be: ", chunksize)
		}
	})

	t.Run("Uploading local webcam feed to allocation should work", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "stream", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
		}))
		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		// Check some .ts files and 1 .m3u8 file must have been created on localpath by youtube-dl
		count_m3u8 := 0
		err = filepath.Walk(localfolder,
			func(path string, info fs.FileInfo, err error) error {
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
		require.Nil(t, err, "error in traversing locally created .m3u8 or .ts files")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created")

		// Check all locally created files have been uploaded to allocation
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error listing files in remotepath")
		require.Len(t, output, 1, "listing files should return output")

		files := []climodel.ListFileResult{}
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "error unmarshalling the response from list files")

		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
		}
	})

	t.Run("Uploading local webcam feed to allocation with delay specified should work", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "stream", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
			"delay":      10,
		}))
		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		// Check some .ts files and 1 .m3u8 file must have been created on localpath by youtube-dl
		count_m3u8 := 0
		err = filepath.Walk(localfolder,
			func(path string, info fs.FileInfo, err error) error {
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
		require.Nil(t, err, "error in traversing locally created .m3u8 or .ts files")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created")

		// Check all locally created files have been uploaded to allocation
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error listing files in remotepath")
		require.Len(t, output, 1, "listing files should return output")

		files := []climodel.ListFileResult{}
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "error unmarshalling the response from list files")

		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
		}
	})

	t.Run("Upload local webcam feed with a different chunknumber must work", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "stream", localfolder, createParams(map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  remotepath,
			"localpath":   localpath,
			"chunknumber": 10,
		}))
		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		// Check some .ts files and 1 .m3u8 file must have been created on localpath by youtube-dl
		count_m3u8 := 0
		err = filepath.Walk(localfolder,
			func(path string, info fs.FileInfo, err error) error {
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
		require.Nil(t, err, "error in traversing locally created .m3u8 or .ts files")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created")

		// Check all locally created files have been uploaded to allocation
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error listing files in remotepath")
		require.Len(t, output, 1, "listing files should return output")

		files := []climodel.ListFileResult{}
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "error unmarshalling the response from list files")

		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
			// FIXME: Num of blocks must be equal to ceil(size/chunksize)
			// require.Equal(t, int64(file.NumBlocks), math.Ceil(float64(file.Size)/float64(chunksize)), "chunksize should be: ", chunksize)
		}
	})

	// Failure Scenarios
	// FIXME: Disabled for now due to process hanging
}

func startUploadFeed(t *testing.T, cliConfigFilename, cmdName, localFolder, params string) error {
	t.Logf("Starting upload of live stream to zbox...")
	commandString := fmt.Sprintf("./zbox %s %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, cmdName, params)
	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)
	require.Nil(t, err, "error in uploading a live feed")

	// Need atleast 3-4 .ts files uploaded
	ctx, cf := context.WithTimeout(context.TODO(), 1*time.Minute)
	defer cf()

	var done bool
	for {

		select {
		case <-ctx.Done():
			done = true
			break
		case <-time.After(5 * time.Second):
			files, _ := os.ReadDir(localFolder)
			c := 0
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".ts") {
					c++
				}
			}

			if c > 2 {
				done = true
				break
			}
		}

		if done {
			break
		}

	}

	// Kills upload process as well as it's child processes
	err = cmd.Process.Kill()
	return err
}

func checkYoutubeFeedAvailabiity() (feed string, isStreamAvailable bool) {
	feed = ""
	const feed1 = `https://www.youtube.com/watch?v=Dx5qFachd3A`
	const feed2 = `https://www.youtube.com/watch?v=fuXfT4Rv_WM`
	const feed3 = `https://www.youtube.com/watch?v=oaSLqdnKniA`

	for i := 1; i < 3; i++ {
		var resp *http.Response
		var err error
		switch i {
		case 1:
			resp, err = http.Get(feed1)
			feed = feed1
		case 2:
			resp, err = http.Get(feed2)
			feed = feed2
		case 3:
			resp, err = http.Get(feed3)
			feed = feed3
		}
		if err == nil && resp.StatusCode == 200 {
			return feed, true
		}
	}
	return "", false
}

func KillFFMPEG() {
	if runtime.GOOS == "windows" {
		_ = exec.Command("Taskkill", "/IM", "ffmpeg.exe", "/F").Run()
	} else if runtime.GOOS == "linux" {
		_ = exec.Command("killall", "ffmpeg").Run()
	}
}
