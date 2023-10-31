package tokenomics_tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func TestClientThrottling(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	var blobberDetailList []climodel.BlobberDetails
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	err = json.Unmarshal([]byte(output[0]), &blobberDetailList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var blobberListString []string
	for _, blobber := range blobberList {
		blobberListString = append(blobberListString, blobber.Id)
	}

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	var validatorListString []string
	for _, validator := range validatorList {
		validatorListString = append(validatorListString, validator.ID)
	}

	stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
		1, 1, 1, 1,
	}, 1)

	t.RunSequentiallyWithTimeout("Exceeding upload limits should blacklist user on blobber", 10*time.Minute, func(t *test.SystemTest) {
		_, err = utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
		})

		var successfullyUploadedFileName string

		for i := 0; i < 2; i++ {
			remotepath := "/dir/"
			filesize := 64 * KB
			filename := utils.GenerateRandomTestFileName(t)

			err = utils.CreateFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			_, err = utils.UploadFile(t, configPath, map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, false)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

			successfullyUploadedFileName = filename
		}

		time.Sleep(2 * time.Minute) // Wait for blacklist worker to run

		remotepath := "/dir/"
		filesize := 64 * KB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		_, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, false)
		require.NotNil(t, err, "File upload is expected to fail")

		err = os.Remove(successfullyUploadedFileName)
		require.Nil(t, err)

		_, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(successfullyUploadedFileName),
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), false)
		require.NotNil(t, err, "File download is expected to fail")
	})

	t.RunSequentiallyWithTimeout("Exceeding download limits should blacklist user on blobber", 10*time.Minute, func(t *test.SystemTest) {
		_, err = utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
		})

		remotepath := "/dir/"
		filesize := 64 * KB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		_, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, false)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		time.Sleep(2 * time.Minute) // Wait for blacklist worker to run

		for i := 0; i < 2; i++ {
			_, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  os.TempDir() + string(os.PathSeparator),
			}), false)
			require.Nil(t, err, "File download is expected to succeed")
		}

		_, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), false)
		require.NotNil(t, err, "File download is expected to fail")
	})

	t.RunSequentiallyWithTimeout("File upload should fail on exceeding max number of files", 10*time.Minute, func(t *test.SystemTest) {
		output, err := utils.CreateWalletForName(t, configPath, "client_wallet_2")
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

		for i := 0; i < 3; i++ {
			_, err = utils.UploadFile(t, configPath, map[string]interface{}{
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, false)
			require.Nil(t, err, "File upload is expected to succeed")
		}

		_, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, false)
		require.NotNil(t, err, "File upload is expected to fail as we already uploaded max files allowed")
	})
}
