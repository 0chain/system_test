package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestReadMarker(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	const blobbersRequiredForDownload = 3 // download needs (data shards + 1) number of blobbers
	sharderUrl := getSharderUrl(t)

	t.RunWithTimeout("After downloading a file, return a readmarker for each blobber used in download", 80*time.Second, func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotePath := "/dir/"

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationId, remotePath, filesize)

		err := os.Remove(filename)
		require.Nil(t, err)

		beforeCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.Zero(t, beforeCount.ReadMarkersCount, "non zero read-marker count before download")

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotePath + filepath.Base(filename),
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationId, sharderUrl)
		require.Len(t, readMarkers, blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})

	t.RunWithTimeout("After downloading an encrypted file, return a readmarker for each blobber used in download", 80*time.Second, func(t *test.SystemTest) {
		allocSize := int64(10 * MB)
		filesize := int64(10)
		remotePath := "/"

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"localpath":  filename,
			"remotepath": remotePath + filepath.Base(filename),
			"encrypt":    "",
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		// Downloading encrypted file should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotePath + filepath.Base(filename),
			"localpath":  os.TempDir(),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		sharderUrl := getSharderUrl(t)
		beforeCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.Zero(t, beforeCount.ReadMarkersCount, "non zero read-marker count before download")

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotePath + filepath.Base(filename),
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationId, sharderUrl)
		require.Len(t, readMarkers, blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})

	t.RunWithTimeout("After downloading a shared file, return a readmarker for each blobber used in download", 80*time.Second, func(t *test.SystemTest) {
		var authTicket, filename, allocationID string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID = setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)

			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath + filepath.Base(filename),
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just register a wallet so that we can work further
		err := registerWalletAndLockReadTokens(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationID, sharderUrl)
		require.Len(t, readMarkers, blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationID, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})
}

func CountReadMarkers(t *test.SystemTest, allocationId, sharderBaseUrl string) *climodel.ReadMarkersCount {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + cliutils.StorageScAddress + "/count_readmarkers")
	params := map[string]string{
		"allocation_id": allocationId,
	}
	return cliutils.ApiGet[climodel.ReadMarkersCount](t, url, params)
}

func GetReadMarkers(t *test.SystemTest, allocationId, sharderBaseUrl string) []climodel.ReadMarker {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + cliutils.StorageScAddress + "/readmarkers")
	params := make(map[string]string)
	if len(allocationId) > 0 {
		params["allocation_id"] = allocationId
	}
	return cliutils.ApiGetList[climodel.ReadMarker](t, url, params, 0, 100)
}
