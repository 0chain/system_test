package cli_tests

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
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

	feed, isStreamAvailable := checkYoutubeFeedAvailabiity()

	if !isStreamAvailable {
		t.Skipf("Youtube live streams are not available right now!")
	}

	t.Run("Downloading youtube feed to allocation should work", func(t *testing.T) {
		t.Parallel()

		_ = initialiseTest(t)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream"

		localfolderForUpload := filepath.Join(os.TempDir(), escapedTestName(t)+"_upload")
		localpathForUpload := localfolderForUpload
		err = os.MkdirAll(localpathForUpload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpathForUpload)

		localfolderForDownload := filepath.Join(os.TempDir(), escapedTestName(t)+"_download")
		localpathForDownload := localfolderForDownload
		err = os.MkdirAll(localpathForDownload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpathForDownload)

		defer os.RemoveAll(localfolderForUpload)
		defer os.RemoveAll(localfolderForDownload)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		errUploadChan := make(chan error, 1)

		errDownloadChan := make(chan error, 1)

		go startUploadFeed1(wg, errUploadChan, t, "feed", configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload + "/up.m3u8",
			"feed":       feed,
		}))
		cliutils.Wait(t, 20*time.Second)

		go startDownloadFeed(wg, errDownloadChan, t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload + "/down.m3u8",
		}))

		wg.Wait()

		err = <-errUploadChan

		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		err = <-errDownloadChan
		require.Nil(t, err, "error in killing download command")

		close(errDownloadChan)
		close(errUploadChan)

		wg.Wait()
		require.Nil(t, err, "error in killing download command")

		t.Logf("LOCAL PATH for UPLOAD : %s", localpathForUpload)

		hashmap := make(map[string]string)

		err = filepath.Walk(localfolderForUpload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			//t.Logf(info.Name())
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

		t.Logf("LOCAL FOLDER for DOWNLOAD : %s", localpathForDownload)
		count_m3u8 := 0
		count_ts := 0
		err = filepath.Walk(localfolderForDownload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			//t.Logf(info.Name())
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

					if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts < 1 {
						t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
						return errors.New(".ts file is not matched with the original one!")
					}
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

		t.Logf("%d\n", len(files))
		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
			//t.Logf("%s , %s , %s", file.Name, file.Path, file.Hash)
		}
	})

	t.Run("Downloading local webcam feed to allocation", func(t *testing.T) {
		t.Parallel()

		_ = initialiseTest(t)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream"

		localfolderForUpload := filepath.Join(os.TempDir(), escapedTestName(t)+"_upload")
		localpathForUpload := localfolderForUpload
		err = os.MkdirAll(localpathForUpload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpathForUpload)

		localfolderForDownload := filepath.Join(os.TempDir(), escapedTestName(t)+"_download")
		localpathForDownload := localfolderForDownload
		err = os.MkdirAll(localpathForDownload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpathForDownload)

		defer os.RemoveAll(localfolderForUpload)
		defer os.RemoveAll(localfolderForDownload)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		errUploadChan := make(chan error, 1)

		errDownloadChan := make(chan error, 1)

		go startUploadFeed1(wg, errUploadChan, t, "stream", configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload + "/up.m3u8",
		}))
		cliutils.Wait(t, 20*time.Second)

		go startDownloadFeed(wg, errDownloadChan, t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload + "/down.m3u8",
		}))

		wg.Wait()

		err = <-errUploadChan

		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		err = <-errDownloadChan
		require.Nil(t, err, "error in killing download command")

		close(errDownloadChan)
		close(errUploadChan)

		wg.Wait()
		require.Nil(t, err, "error in killing download command")

		t.Logf("LOCAL PATH for UPLOAD : %s", localpathForUpload)

		hashmap := make(map[string]string)

		err = filepath.Walk(localfolderForUpload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			//t.Logf(info.Name())
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

		t.Logf("LOCAL FOLDER for DOWNLOAD : %s", localpathForDownload)
		count_m3u8 := 0
		count_ts := 0
		err = filepath.Walk(localfolderForDownload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			//t.Logf(info.Name())
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
					if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts < 1 {
						t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
						return errors.New(".ts file is not matched with the original one!")
					}
					count_ts += 1
				}
			}
			return nil
		})
		require.Nil(t, err, "error in traversing locally created .m3u8 and .ts files!")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created!")
		require.GreaterOrEqual(t, count_ts, 1, "at least one .ts file should be created!")
		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats
		var data climodel.FileStats
		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data = range stats {
			t.Log(data.Name)
		}

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

		t.Logf("%d\n", len(files))
		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
		}
	})

	t.Run("Downloading feed to allocation with delay flag", func(t *testing.T) {
		t.Parallel()

		_ = initialiseTest(t)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock": 1,
		}))
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		remotepath := "/live/stream"

		localfolderForUpload := filepath.Join(os.TempDir(), escapedTestName(t)+"_upload")
		localpathForUpload := localfolderForUpload
		err = os.MkdirAll(localpathForUpload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpathForUpload)

		localfolderForDownload := filepath.Join(os.TempDir(), escapedTestName(t)+"_download")
		localpathForDownload := localfolderForDownload
		err = os.MkdirAll(localpathForDownload, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpathForDownload)

		defer os.RemoveAll(localfolderForUpload)
		defer os.RemoveAll(localfolderForDownload)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		errUploadChan := make(chan error, 1)

		errDownloadChan := make(chan error, 1)

		go startUploadFeed1(wg, errUploadChan, t, "feed", configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload + "/up.m3u8",
			"feed":       feed,
		}))
		cliutils.Wait(t, 20*time.Second)

		go startDownloadFeed(wg, errDownloadChan, t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload + "/down.m3u8",
			"delay":      1,
		}))

		wg.Wait()

		err = <-errUploadChan

		require.Nil(t, err, "error in killing upload command")
		KillFFMPEG()

		err = <-errDownloadChan
		require.Nil(t, err, "error in killing download command")

		close(errDownloadChan)
		close(errUploadChan)

		wg.Wait()
		require.Nil(t, err, "error in killing download command")

		t.Logf("LOCAL PATH for UPLOAD : %s", localpathForUpload)

		hashmap := make(map[string]string)

		err = filepath.Walk(localfolderForUpload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			//t.Logf(info.Name())
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

		t.Logf("LOCAL FOLDER for DOWNLOAD : %s", localpathForDownload)
		count_m3u8 := 0
		count_ts := 0
		err = filepath.Walk(localfolderForDownload, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				return err
			}
			//t.Logf(info.Name())
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
					if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts < 1 {
						t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
						return errors.New(".ts file is not matched with the original one!")
					}
					count_ts += 1
				}
			}
			return nil
		})
		require.Nil(t, err, "error in traversing locally created .m3u8 and .ts files!")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created!")
		require.GreaterOrEqual(t, count_ts, 1, "at least one .ts file should be created!")
		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats
		var data climodel.FileStats
		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data = range stats {
			t.Log(data.Name)
		}

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

		t.Logf("%d\n", len(files))
		for _, file := range files {
			require.Regexp(t, regexp.MustCompile(`up(\d+).ts`), file.Name, "files created locally must be found uploaded to allocation")
			//t.Logf("%s , %s , %s", file.Name, file.Path, file.Hash)
		}
	})
}

func startUploadFeed1(wg *sync.WaitGroup, errChan chan error, t *testing.T, cmdName, cliConfigFilename, params string) {

	defer wg.Done()

	t.Logf("Starting upload of live stream to zbox...")
	commandString := fmt.Sprintf("./zbox %s %s --silent --delay 10 --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, cmdName, params)

	t.Logf(commandString)

	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)
	require.Nil(t, err, "error in uploading a live feed")

	// Need atleast 3-4 .ts files uploaded
	cliutils.Wait(t, 180*time.Second)

	// Kills upload process as well as it's child processes
	err = cmd.Process.Kill()
	errChan <- err
}

func startDownloadFeed(wg *sync.WaitGroup, errChan chan error, t *testing.T, cliConfigFilename, params string) {

	defer wg.Done()

	t.Logf("Starting download of live stream from zbox.")

	commandString := fmt.Sprintf("./zbox download --live %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, params)

	t.Logf(commandString)

	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)

	require.Nil(t, err, "error in downloading a live feed")
	cliutils.Wait(t, 160*time.Second)

	err = cmd.Process.Kill()

	errChan <- err
}
