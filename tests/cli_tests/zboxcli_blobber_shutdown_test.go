package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestShutDownBlobber(t *testing.T) {
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	blobbers := []climodel.BlobberInfo{}
	output, err = listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)
	err = json.Unmarshal([]byte(output[0]), &blobbers)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

	// Pick a random blobber to shutdown
	blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

	t.Run("Shutting down blobber by blobber's delegate wallet should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, strings.Join(output, "\n"))

		readPoolParams := createParams(map[string]interface{}{
			"tokens": 1,
		})
		output, err = readPoolLock(t, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"lock":   1,
			"data":   4,
			"parity": 1,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		filesize := int64(256)
		remotepath := "/"
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		output, err = shutdownBlobberForWallet(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
		}), blobberOwnerWallet)
		require.Nil(t, err)
		require.Len(t, output, 1)
		require.Equal(t, "shut down blobber", output[0])

		// FIXME: Uncomment
		// // blobber.IsShutDown should be true for > 25% sharders
		// statuses := blobberStatusFromEndpoint(t, blobber.Id)

		// var count int
		// totalSharders := 2
		// for _, status := range statuses {
		// 	if status.Status == apimodel.ShutDown {
		// 		count++
		// 	}
		// }
		// require.GreaterOrEqual(t, float64(count)/float64(totalSharders), 0.25)

		alloc := getAllocation(t, allocationID)
		require.Equal(t, 6, len(alloc.Blobbers))

		// Get the new Write-Pool info after upload
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		// TODO: Assert on writepool balance after https://github.com/0chain/0chain/pull/1373 is merged
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
		downloadedFileChecksum := generateChecksum(t, os.TempDir()+string(os.PathSeparator)+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("shutted down blobber should not be listed", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		blobbers := getBlobbersList(t)
		require.NotContains(t, blobbers, blobber)
	})

	t.Run("Should not be able to use shutted down blobber for new allocations", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Should throw error as one of the 6 blobbers is shutdown
		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"size":   1024,
			"lock":   1,
			"data":   5,
			"parity": 1,
		}))
		require.NotNil(t, err, "expected error but got none", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: not enough blobbers to honor the allocation", output[0])
	})
}

func shutdownBlobber(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return shutdownBlobberForWallet(t, cliConfigFilename, params, escapedTestName(t))
}

func shutdownBlobberForWallet(t *testing.T, cliConfigFilename, params, wallet string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox shut-down-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}

func blobberStatusFromEndpoint(t *testing.T, blobberId string) []apimodel.ProviderStatus {
	var statuses []apimodel.ProviderStatus

	output, err := getSharders(t, configPath)
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 1)
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]*climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	// Get base URL for API calls.
	sharderBaseURLs := getAllSharderBaseURLs(sharders)

	for _, sharderBaseURL := range sharderBaseURLs {
		res, err := http.Get(fmt.Sprintf(sharderBaseURL + "/v1/screst/" + storageSmartContractAddress +
			"/blobber-status" + "?id=" + blobberId))

		if err != nil || res.StatusCode < 200 || res.StatusCode >= 300 {
			continue
		}

		require.NotNil(t, res.Body, "Open challenges API response must not be nil")

		resBody, err := io.ReadAll(res.Body)
		defer res.Body.Close()

		require.Nil(t, err, "Error reading response body")
		var status apimodel.ProviderStatus
		err = json.Unmarshal(resBody, &status)
		require.Nil(t, err, "error unmarshalling response body")

		statuses = append(statuses, status)
	}
	return statuses
}
