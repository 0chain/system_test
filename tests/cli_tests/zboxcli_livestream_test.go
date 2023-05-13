package cli_tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestStreamUploadDownload(testSetup *testing.T) {
	testSetup.Skip("Flaky")
	t := test.NewSystemTest(testSetup)

	t.TestSetup("Kill FFMPEG", KillFFMPEG)
	defer KillFFMPEG()

	// Success scenarios
	t.RunSequentiallyWithTimeout("Uploading remote feed to allocation should work", 2*time.Minute, func(t *test.SystemTest) { // todo slow
		feed, ok := getFeed()

		if !ok {
			t.Skipf("No live feed available right now")
		}

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		os.RemoveAll(localpath)
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "feed", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
			"feed":       feed,
		}))
		require.Nil(t, err, fmt.Sprintf("startUploadFeed: %s", err))

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

	t.RunSequentiallyWithTimeout("Upload from feed with delay flag must work", 4*time.Minute, func(t *test.SystemTest) { // todo slow
		feed, ok := getFeed()

		if !ok {
			t.Skipf("No live feed available right now")
		}

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		os.RemoveAll(localpath)
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "feed", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
			"feed":       feed,
			"delay":      2,
		}))
		require.Nil(t, err, fmt.Sprintf("startUploadFeed: %s", err))

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

	t.RunSequentiallyWithTimeout("Upload from feed with a different chunknumber must work", 2*time.Minute, func(t *test.SystemTest) {
		feed, ok := getFeed()

		if !ok {
			t.Skipf("No live feed available right now")
		}

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		os.RemoveAll(localpath)
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
		require.Nil(t, err, fmt.Sprintf("startUploadFeed: %s", err))

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

	t.RunSequentiallyWithTimeout("Uploading local webcam feed to allocation should work", 60*time.Second, func(t *test.SystemTest) {
		t.Skip("github runner has no any audio/camera device to test this feature yet")

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		os.RemoveAll(localpath)
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "stream", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
		}))
		require.Nil(t, err, fmt.Sprintf("startUploadFeed: %s", err))

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

	t.RunSequentiallyWithTimeout("Uploading local webcam feed to allocation with delay specified should work", 60*time.Second, func(t *test.SystemTest) {
		t.Skip("github runner has no any audio/camera device to test this feature yet")

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		os.RemoveAll(localpath)
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "stream", localfolder, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpath,
			"delay":      10,
		}))
		require.Nil(t, err, fmt.Sprintf("startUploadFeed: %s", err))

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

	t.RunSequentiallyWithTimeout("Upload local webcam feed with a different chunknumber must work", 60*time.Second, func(t *test.SystemTest) {
		t.Skip("github runner has no any audio/camera device to test this feature yet")

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream.m3u8"
		localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
		localpath := filepath.Join(localfolder, "up.m3u8")
		os.RemoveAll(localpath)
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		err = startUploadFeed(t, configPath, "stream", localfolder, createParams(map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  remotepath,
			"localpath":   localpath,
			"chunknumber": 10,
		}))
		require.Nil(t, err, fmt.Sprintf("startUploadFeed: %s", err))

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

func startUploadFeed(t *test.SystemTest, cliConfigFilename, cmdName, localFolder, params string) error {
	t.Logf("Starting upload of live stream to zbox...")
	commandString := fmt.Sprintf("./zbox %s %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, cmdName, params)
	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)
	require.Nil(t, err, fmt.Sprintf("error in uploading a live feed: %s", err))

	ready := waitTsFilesReady(t, localFolder)
	if !ready {
		defer cmd.Process.Kill() //nolint: errcheck

		return errors.New("download video files is timeout")
	}

	// waiting for .ts files to upload
	cliutils.Wait(t, 30*time.Second)
	cmd.Process.Kill() //nolint: errcheck
	return nil
}

func waitTsFilesReady(t *test.SystemTest, localFolder string) bool {
	// Need atleast 3-4 .ts files downloaded and generated by youtube-dl and ffmpeg
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(1 * time.Second):
			files, _ := os.ReadDir(localFolder)
			c := 0
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".ts") {
					c++
				}
			}

			if c > 2 {
				return true
			}
		}
	}
}

var feedMutex sync.Mutex
var feeds = []string{
	`https://www.youtube.com/watch?v=GfMA7VPkDYw`,
	`https://www.youtube.com/watch?v=IggdFxXbnsA`,
	"https://youtu.be/OBExtHgg-js",
	"https://youtu.be/EBvFX1NP4WM",
	"https://www.twitch.tv/videos/1628184971",
	"https://www.twitch.tv/videos/1626720273",
	"https://veoh.com/watch/v17432006xmkFQTa",
	"https://odysee.com/@samuel_earp_artist:c/how-to-paint-a-landscape-in-gouache:8",
	"https://odysee.com/@fireship:6/how-to-never-write-bug:4",
}

func getFeed() (string, bool) {
	feedMutex.Lock()
	defer feedMutex.Unlock()
	n := len(feeds)

	i := rand.Intn(n) //nolint
	var m int
	for {
		if m >= n {
			return "", false
		}
		feed := feeds[i]

		resp, err := http.Get(feed) //nolint

		if resp != nil && resp.Body != nil {
			resp.Body.Close() //nolint
		}

		if err == nil && resp.StatusCode == http.StatusOK {
			return feed, true
		}

		i++

		if i >= n {
			i = 0
		}

		m++
	}
}

func KillFFMPEG() {
	if runtime.GOOS == "windows" {
		_ = exec.Command("Taskkill", "/IM", "ffmpeg.exe", "/F").Run()
	} else {
		_ = exec.Command("killall", "ffmpeg").Run()
	}
}
