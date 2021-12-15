package cli_tests

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	feed := `https://www.youtube.com/watch?v=5qap5aO4i9A`

	t.Run("Uploading youtube feed to allocation should work", func(t *testing.T) {
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
		localfolder := filepath.Join(os.TempDir(), "stream-up")
		localpath := filepath.Join(localfolder, escapedTestName(t)+".m3u8")
		err = os.MkdirAll(localpath, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localpath)
		defer os.RemoveAll(localfolder)

		// Using exec.Command as we need the pid to kill this later
		cmd := exec.Command("./zbox", "upload", "--allocation", allocationID, "--remotepath", remotepath,
			"--localpath", localpath, "--feed", feed, "--sync", "--silent", "--wallet", escapedTestName(t)+"_wallet.json",
			"--configDir", "./config", "--config", configPath)
		cliutils.Setpgid(cmd)
		err = cmd.Start()
		require.Nil(t, err, "error in uploading a live feed")

		// Need atleast 3-4 .ts files uploaded
		cliutils.Wait(t, 30*time.Second)

		// Kills upload process as well as it's child processes
		cmd.Process.Kill()

		// Check some .ts files and 1 .m3u8 file must have been created on localpath by youtube-dl
		count_m3u8 := 0
		count_ts := 0
		ts_files := make([]string, 0)
		err = filepath.Walk(localfolder,
			func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				extension := strings.Split(info.Name(), ".")
				if extension[len(extension)-1] == "m3u8" {
					count_m3u8 = 1
					return nil
				} else if extension[len(extension)-1] == "ts" {
					ts_files = append(ts_files, info.Name())
					count_ts = 1
					return nil
				}
				return nil
			})
		require.Nil(t, err, "error in traversing locally created .m3u8 or .ts files")
		require.Equal(t, count_m3u8, 1, "exactly one .m3u8 file should be created")
		require.GreaterOrEqual(t, count_ts, 1, "atleast one .ts file should be created")

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

		for i, file := range files {
			require.Equal(t, file.Name, ts_files[i], "each file created locally must also be found uploaded to allocation")
		}

		// output, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
		// 	"allocation": allocationID,
		// 	"tokens":     1,
		// 	"duration":   "5m",
		// }), true)
		// require.Nil(t, err, "error locking readpool tokens", strings.Join(output, "\n"))
		// require.Len(t, output, 1)
		// require.Equal(t, "locked", output[0])

	})
}
