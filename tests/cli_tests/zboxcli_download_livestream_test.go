package cli_tests

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

	KillFFMPEG()

	defer KillFFMPEG()

	t.Run("Downloading youtube feed to allocation should work", func(t *testing.T) {
		t.Parallel()
		feed, ok := getFeed()

		if !ok {
			t.Skipf("No live feed available right now")
		}

		walletOwner := escapedTestName(t) + "_wallet"

		_ = initialiseTest(t, walletOwner, true)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 0.5,
		}))
		t.Log(output)
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		_, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"tokens": 0.5,
		}), true)
		require.Nil(t, err, "error")

		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream"

		localfolderForUpload := filepath.Join(os.TempDir(), escapedTestName(t)+"_upload")
		err = os.MkdirAll(localfolderForUpload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localfolderForUpload)
		localpathForUpload := localfolderForUpload + "/up.m3u8"

		localfolderForDownload := filepath.Join(os.TempDir(), escapedTestName(t)+"_download")
		err = os.MkdirAll(localfolderForDownload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localfolderForDownload)
		localpathForDownload := localfolderForDownload + "/down.m3u8"

		defer os.RemoveAll(localfolderForUpload)
		defer os.RemoveAll(localfolderForDownload)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		errUploadChan := make(chan error, 1)
		errDownloadChan := make(chan error, 1)
		signalForDownload := make(chan bool, 1)
		signalForUpload := make(chan bool, 1)

		go startUploadFeed1(wg, errUploadChan, signalForDownload, signalForUpload, t, "feed", configPath, localfolderForUpload, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload,
			"feed":       feed,
		}))

		select {
		case s := <-signalForDownload:
			if s == true {
				break
			}
		}

		go startDownloadFeed(wg, errDownloadChan, t, signalForUpload, configPath, localfolderForDownload, createParams(map[string]interface{}{
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
		close(signalForUpload)
		close(signalForDownload)

		require.Nil(t, err, "error in killing download command")

		hashmap := make(map[string]string)

		err = filepath.Walk(localfolderForUpload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			if !info.IsDir() {
				file, err := os.Open(path)

				if err != nil {
					return err
				}

				hash := md5.New()
				_, err = io.Copy(hash, file)

				if err != nil {
					return err
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "ts" {

					num, err := strconv.Atoi(extension[0][2:])

					if err != nil {
						return err
					}

					num = num + 1

					newStr := fmt.Sprintf("up%d.ts", num)
					hashmap[newStr] = hex.EncodeToString(hash.Sum(nil))
				}

				file.Close()
				hash.Reset()
			}

			return nil
		})
		require.Nil(t, err, "error in traversing locally created upload files")

		count_m3u8 := 0
		count_ts := 0
		err = filepath.Walk(localfolderForDownload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}

			if !info.IsDir() {

				file, err := os.Open(path)

				if err != nil {
					return err
				}

				hash := md5.New()
				_, err = io.Copy(hash, file)

				if err != nil {
					log.Println(err)
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "m3u8" {
					count_m3u8 += 1
					return nil
				} else if extension[len(extension)-1] == "ts" {
					/*
						if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts > 0 {
							t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
							t.Log(count_ts)
							return errors.New(".ts file is not matched with the original one!")
						}
					*/
					count_ts += 1

				}
			}
			return nil
		})
		require.Nil(t, err, "error in traversing locally created .m3u8 and .ts files!")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created!")
		require.Greater(t, count_ts, 0, "at least one .ts file should be created!")

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

	t.Run("Downloading local webcam feed to allocation", func(t *testing.T) {
		//t.Parallel()

		walletOwner := escapedTestName(t) + "_wallet"

		_ = initialiseTest(t, walletOwner, true)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 0.5,
		}))
		t.Log(output)
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		_, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"tokens": 0.5,
		}), true)
		require.Nil(t, err, "error")

		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream"

		localfolderForUpload := filepath.Join(os.TempDir(), escapedTestName(t)+"_upload")
		err = os.MkdirAll(localfolderForUpload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localfolderForUpload)
		localpathForUpload := localfolderForUpload + "/up.m3u8"

		localfolderForDownload := filepath.Join(os.TempDir(), escapedTestName(t)+"_download")
		err = os.MkdirAll(localfolderForDownload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localfolderForDownload)
		localpathForDownload := localfolderForDownload + "/down.m3u8"

		defer os.RemoveAll(localfolderForUpload)
		defer os.RemoveAll(localfolderForDownload)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		errUploadChan := make(chan error, 1)
		errDownloadChan := make(chan error, 1)
		signalForDownload := make(chan bool, 1)
		signalForUpload := make(chan bool, 1)

		go startUploadFeed1(wg, errUploadChan, signalForDownload, signalForUpload, t, "stream", configPath, localfolderForUpload, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload,
		}))

		select {
		case s := <-signalForDownload:
			if s == true {
				break
			}
		}

		go startDownloadFeed(wg, errDownloadChan, t, signalForUpload, configPath, localfolderForDownload, createParams(map[string]interface{}{
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
		close(signalForUpload)
		close(signalForDownload)

		require.Nil(t, err, "error in killing download command")

		hashmap := make(map[string]string)

		err = filepath.Walk(localfolderForUpload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)

				if err != nil {
					return err
				}

				hash := md5.New()
				_, err = io.Copy(hash, file)

				if err != nil {
					return err
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "ts" {

					num, err := strconv.Atoi(extension[0][2:])

					if err != nil {
						return err
					}

					num = num + 1

					newStr := fmt.Sprintf("up%d.ts", num)
					hashmap[newStr] = hex.EncodeToString(hash.Sum(nil))
				}

				file.Close()
				hash.Reset()
			}

			return nil
		})
		require.Nil(t, err, "error in traversing locally created upload files")

		count_m3u8 := 0
		count_ts := 0
		err = filepath.Walk(localfolderForDownload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}

			if !info.IsDir() {

				file, err := os.Open(path)

				if err != nil {
					log.Println(err)
				}

				hash := md5.New()
				_, err = io.Copy(hash, file)

				if err != nil {
					log.Println(err)
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "m3u8" {
					count_m3u8 += 1
					return nil
				} else if extension[len(extension)-1] == "ts" {
					/*
						if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts > 0 {
							t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
							return errors.New(".ts file is not matched with the original one!")
						}
					*/
					count_ts += 1

				}
			}
			return nil
		})
		require.Nil(t, err, "error in traversing locally created .m3u8 and .ts files!")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created!")
		require.Greater(t, count_ts, 0, "at least one .ts file should be created!")

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

	t.Run("Downloading feed to allocation with delay flag", func(t *testing.T) {
		t.Parallel()
		feed, ok := getFeed()

		if !ok {
			t.Skipf("No live feed available right now")
		}

		walletOwner := escapedTestName(t) + "_wallet"

		_ = initialiseTest(t, walletOwner, true)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 0.5,
		}))
		t.Log(output)
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		_, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"tokens": 0.5,
		}), true)
		require.Nil(t, err, "error")

		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream"

		localfolderForUpload := filepath.Join(os.TempDir(), escapedTestName(t)+"_upload")

		err = os.MkdirAll(localfolderForUpload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localfolderForUpload)
		localpathForUpload := localfolderForUpload + "/up.m3u8"

		localfolderForDownload := filepath.Join(os.TempDir(), escapedTestName(t)+"_download")
		err = os.MkdirAll(localfolderForDownload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localfolderForDownload)
		localpathForDownload := localfolderForDownload + "/down.m3u8"

		defer os.RemoveAll(localfolderForUpload)
		defer os.RemoveAll(localfolderForDownload)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		errUploadChan := make(chan error, 1)
		errDownloadChan := make(chan error, 1)
		signalForDownload := make(chan bool, 1)
		signalForUpload := make(chan bool, 1)

		go startUploadFeed1(wg, errUploadChan, signalForDownload, signalForUpload, t, "feed", configPath, localfolderForUpload, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload,
			"feed":       feed,
		}))

		select {
		case s := <-signalForDownload:
			if s == true {
				break
			}
		}

		go startDownloadFeed(wg, errDownloadChan, t, signalForUpload, configPath, localfolderForDownload, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload,
			"delay":      10,
		}))

		wg.Wait()

		err = <-errUploadChan

		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		err = <-errDownloadChan
		require.Nil(t, err, "error in killing download command")

		close(errDownloadChan)
		close(errUploadChan)
		close(signalForUpload)
		close(signalForDownload)

		require.Nil(t, err, "error in killing download command")

		hashmap := make(map[string]string)

		err = filepath.Walk(localfolderForUpload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)

				if err != nil {
					return err
				}

				hash := md5.New()
				_, err = io.Copy(hash, file)

				if err != nil {
					return err
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "ts" {

					num, err := strconv.Atoi(extension[0][2:])

					if err != nil {
						return err
					}

					num = num + 1

					newStr := fmt.Sprintf("up%d.ts", num)
					hashmap[newStr] = hex.EncodeToString(hash.Sum(nil))
				}

				file.Close()
				hash.Reset()
			}

			return nil
		})
		require.Nil(t, err, "error in traversing locally created upload files")

		count_m3u8 := 0
		count_ts := 0
		err = filepath.Walk(localfolderForDownload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			if !info.IsDir() {

				file, err := os.Open(path)

				if err != nil {
					log.Println(err)
				}

				hash := md5.New()
				_, err = io.Copy(hash, file)

				if err != nil {
					log.Println(err)
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "m3u8" {
					count_m3u8 += 1
					return nil
				} else if extension[len(extension)-1] == "ts" {
					/*
						if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts > 0 {
							t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
							return errors.New(".ts file is not matched with the original one!")
						}
					*/
					count_ts += 1

				}
			}
			return nil
		})
		require.Nil(t, err, "error in traversing locally created .m3u8 and .ts files!")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created!")
		require.Greater(t, count_ts, 0, "at least one .ts file should be created!")

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

}

func startUploadFeed1(wg *sync.WaitGroup, errChan chan error, signalForDownload chan bool, signalForUpload chan bool, t *testing.T, command, cliConfigFilename, localFolder, params string) {
	defer wg.Done()

	t.Logf("Starting upload of live stream to zbox...")
	commandString := fmt.Sprintf("./zbox %s %s --silent --delay 10 --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, command, params)

	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)
	require.Nil(t, err, "error in uploading a live feed")

	ready := waitTsFilesReady(t, localFolder, 3)
	if !ready {
		defer cmd.Process.Kill() //nolint: errcheck

		errChan <- errors.New("download video files is timeout")
		return
	}

	signalForDownload <- true

	select {
	case k := <-signalForUpload:
		if k == true {
			break
		}
	}

	// Kills upload process as well as it's child processes
	err = cmd.Process.Kill()
	errChan <- err
}

func startDownloadFeed(wg *sync.WaitGroup, errChan chan error, t *testing.T, signalForUpload chan bool, cliConfigFilename, localFolder, params string) {

	defer wg.Done()

	t.Logf("Starting download of live stream from zbox.")

	commandString := fmt.Sprintf("./zbox download --live %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, params)

	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)

	require.Nil(t, err, "error in downloading a live feed")

	ready := waitTsFilesReady(t, localFolder, 3)

	if !ready {
		defer cmd.Process.Kill() //nolint: errcheck

		errChan <- errors.New("download video files is timeout")
		return
	}
	signalForUpload <- true

	err = cmd.Process.Kill()

	errChan <- err
}

func waitTsFilesReady(t *testing.T, localFolder string, target int) bool {
	// Need atleast 3-4 .ts files downloaded and generated by youtube-dl and ffmpeg
	ctx, cf := context.WithTimeout(context.TODO(), 5*time.Minute)
	defer cf()

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

			if c >= target {
				return true
			}
		}
	}
}

var feedMutex sync.Mutex
var feeds = []string{
	"https://www.youtube.com/watch?v=GfMA7VPkDYw",
	"https://www.youtube.com/watch?v=IggdFxXbnsA",
	"https://youtu.be/OBExtHgg-js",
	"https://youtu.be/EBvFX1NP4WM",
	"https://www.twitch.tv/videos/1628184971",
	"https://www.twitch.tv/videos/1626720273",
	"https://veoh.com/watch/v1422286563pBZAwaK",
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
