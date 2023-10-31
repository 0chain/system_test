package tokenomics_tests

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	downloadLimits = 102400 // 100KB
	uploadLimits   = 102400 // 100KB
)

func TestClientThrottling(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	totalUploadedDataPerBlobber := 0
	totalDownloadedDataPerBlobber := 0
	totalFilesUploaded := 0

	t.RunSequentially("Upload and download limits should allow blocks less than limits", func(t *test.SystemTest) {
		// Max upload blocks set in config is 6400KB

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
		}))
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocationId, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "Error getting allocation ID", strings.Join(output, "\n"))

		remotepath := "/dir/"
		filesize := 1024
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
		totalUploadedDataPerBlobber += 1024
		totalFilesUploaded += 1

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		totalDownloadedDataPerBlobber += 1024
	})

	t.RunSequentially("Upload limits should not allow upload blocks more than limits", func(t *test.SystemTest) {
		// Max upload blocks set in config is 6400KB

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
		}))
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocationId, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "Error getting allocation ID", strings.Join(output, "\n"))

		remotepath := "/dir/"
		filesize := 1024 * 7
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.NotNil(t, err, "File upload is expected to fail")
	})

	t.RunSequentially("Upload and download limits should allow blocks less than limits", func(t *test.SystemTest) {
		// Max upload blocks set in config is 6400KB

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
		}))
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocationId, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "Error getting allocation ID", strings.Join(output, "\n"))

		remotepath := "/dir/"
		filesize := 1024
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
		totalUploadedDataPerBlobber += 1024
		totalFilesUploaded += 1

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		for totalDownloadedDataPerBlobber <= downloadLimits {
			output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remoteFilepath,
				"localpath":  os.TempDir() + string(os.PathSeparator),
			}), true)
			require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
			totalDownloadedDataPerBlobber += 1024
		}

		output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.NotNil(t, err, "File download is expected to fail")
	})

	t.RunSequentially("File upload should fail on exceeding max number of files", func(t *test.SystemTest) {
		// Max upload blocks set in config is 6400KB

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		output, err = utils.CreateNewAllocation(t, configPath, utils.CreateParams(map[string]interface{}{
			"size":   10 * MB,
			"data":   1,
			"lock":   2,
			"parity": 1,
		}))
		require.Nil(t, err, "Error creating allocation", strings.Join(output, "\n"))

		allocationId, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "Error getting allocation ID", strings.Join(output, "\n"))

		remotepath := "/dir/"
		filesize := 1024
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.NotNil(t, err, "File upload is expected to fail as we already uploaded max files allowed")
	})
}
