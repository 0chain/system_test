package cli_tests

import (
	"crypto/md5" //nolint:gosec
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
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestLivestreamDownload(testSetup *testing.T) { // nolint cyclomatic complexity 48
	// testSetup.Skip("Flaky")
	t := test.NewSystemTest(testSetup)
	t.TestSetup("Kill FFMPEG", KillFFMPEG)

	defer KillFFMPEG()

	t.RunSequentiallyWithTimeout("Downloading youtube feed to allocation should work", 400*time.Second, func(t *test.SystemTest) {
		feed, ok := getFeed()

		if !ok {
			t.Skipf("No live feed available right now")
		}

		walletOwner := escapedTestName(t) + "_wallet"

		_ = initialiseTest(t, walletOwner, true)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))

		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		_, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"tokens": 1,
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

		err = startUploadAndDownloadFeed(t, "feed", configPath, localfolderForUpload, localfolderForDownload, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload,
			"feed":       feed,
		}), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload,
		}))
		require.Nil(t, err, "error in startUploadAndDownloadFeed")

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

				hash := md5.New() //nolint:gosec
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

					num++

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

				hash := md5.New() //nolint:gosec
				_, err = io.Copy(hash, file)

				if err != nil {
					log.Println(err)
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "m3u8" {
					count_m3u8 += 1
					return nil
				} else if extension[len(extension)-1] == "ts" {
					count_ts += 1
					if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts < 3 {
						t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
						t.Log(count_ts)
						return errors.New(".ts file is not matched with the original one!")
					}
					return nil
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
	}) // TODO: TOO LONG

	t.RunSequentiallyWithTimeout("Downloading local webcam feed to allocation", 60*time.Second, func(t *test.SystemTest) {
		t.Skip("github runner has no any audio/camera device to test this feature yet")
		walletOwner := escapedTestName(t) + "_wallet"

		_ = initialiseTest(t, walletOwner, true)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		t.Log(output)
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		_, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"tokens": 1,
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

		err = startUploadAndDownloadFeed(t, "stream", configPath, localfolderForUpload, localfolderForDownload, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload,
		}), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload,
		}))
		require.Nil(t, err, "error in startUploadAndDownloadFeed")

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

				hash := md5.New() //nolint:gosec
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

					num++

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

				hash := md5.New() //nolint:gosec
				_, err = io.Copy(hash, file)

				if err != nil {
					log.Println(err)
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "m3u8" {
					count_m3u8 += 1
					return nil
				} else if extension[len(extension)-1] == "ts" {
					count_ts += 1

					if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts < 3 {
						t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
						return errors.New(".ts file is not matched with the original one!")
					}
					return nil
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

	t.RunSequentiallyWithTimeout("Downloading feed to allocation with delay flag", 3*time.Minute, func(t *test.SystemTest) { // todo this is unacceptably slow
		feed, ok := getFeed()

		if !ok {
			t.Skipf("No live feed available right now")
		}

		walletOwner := escapedTestName(t) + "_wallet"

		_ = initialiseTest(t, walletOwner, true)

		output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   5,
			"expire": "1h",
			"size":   "10000",
		}))
		t.Log(output)
		require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))

		_, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"tokens": 1,
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

		err = startUploadAndDownloadFeed(t, "feed", configPath, localfolderForUpload, localfolderForDownload, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForUpload,
			"feed":       feed,
		}), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  localpathForDownload,
		}))
		require.Nil(t, err, "error in startUploadAndDownloadFeed")

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

				hash := md5.New() //nolint:gosec
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

					num++

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

				hash := md5.New() //nolint:gosec
				_, err = io.Copy(hash, file)

				if err != nil {
					log.Println(err)
				}

				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "m3u8" {
					count_m3u8 += 1
					return nil
				} else if extension[len(extension)-1] == "ts" {
					count_ts += 1
					if hashmap[info.Name()] != hex.EncodeToString(hash.Sum(nil)) && count_ts < 3 {
						t.Logf("HASH of UP :%s\nHASH of DOWN :%s\n \n", hashmap[info.Name()], hex.EncodeToString(hash.Sum(nil)))
						return errors.New(".ts file is not matched with the original one!")
					}
					return nil
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

func startUploadAndDownloadFeed(t *test.SystemTest, command, cliConfigFilename, localfolderForUpload, localfolderForDownload, uploadParams, downloadParams string) error {
	t.Logf("Starting upload of live stream to zbox...")
	commandString := fmt.Sprintf("./zbox %s %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, command, uploadParams)

	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)
	require.Nil(t, err, "error in uploading a live feed")

	ready := waitTsFilesReady(t, localfolderForUpload)
	if !ready {
		defer cmd.Process.Kill() //nolint: errcheck

		return errors.New("download video files is timeout")
	}
	err = startDownloadFeed(t, cliConfigFilename, localfolderForDownload, downloadParams)
	require.Nil(t, err, "error in startDownloadFeed")

	err = cmd.Process.Kill()
	require.Nil(t, err, "error in killing process")

	return nil
}

func startDownloadFeed(t *test.SystemTest, cliConfigFilename, localFolder, params string) error {
	t.Logf("Starting download of live stream from zbox.")

	commandString := fmt.Sprintf("./zbox download --live %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, params)

	cmd, err := cliutils.StartCommand(t, commandString, 3, 15*time.Second)

	require.Nil(t, err, "error in downloading the live feed")

	ready := waitTsFilesReady(t, localFolder)

	if !ready {
		defer cmd.Process.Kill() //nolint: errcheck

		return errors.New("download video files is timeout")
	}

	err = cmd.Process.Kill()

	return err
}
