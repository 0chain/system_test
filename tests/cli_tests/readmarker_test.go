package cli_tests

import (
	"encoding/json"
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
	t.SetSmokeTests("After downloading a file, return a readmarker for each blobber used in download")

	t.Parallel()

	const blobbersRequiredForDownload = 3 // download needs (data shards) number of blobbers

	var sharderUrl string
	t.TestSetup("Create wallet and temp directories, get sharder url", func() {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))
		sharderUrl = getSharderUrl(t)

		err = os.MkdirAll("tmp", os.ModePerm)
		require.Nil(t, err)
	})

	t.Run("After downloading a file, return a readmarker for each blobber used in download", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotePath := "/dir/"

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
			"data":   3,
			"parity": 1,
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
		require.GreaterOrEqual(t, len(readMarkers), blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})

	t.Run("After downloading an encrypted file, return a readmarker for each blobber used in download", func(t *test.SystemTest) {
		allocSize := int64(10 * MB)
		filesize := int64(10)
		remotePath := "/"

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
			"data":   3,
			"parity": 1,
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

		sharderUrl := getSharderUrl(t)
		beforeCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.Zero(t, beforeCount.ReadMarkersCount, "non zero read-marker count before download")

		// Downloading encrypted file should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotePath + filepath.Base(filename),
			"localpath":  os.TempDir(),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationId, sharderUrl)
		require.GreaterOrEqual(t, len(readMarkers), blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})

	t.RunWithTimeout("After downloading a shared file, return a readmarker for each blobber used in download", 120*time.Second, func(t *test.SystemTest) {
		var authTicket, filename, allocationID string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID = setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
				"data":   3,
				"parity": 1,
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

		// Just create a wallet so that we can work further
		err := createWalletAndLockReadTokens(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationID, sharderUrl)
		require.GreaterOrEqual(t, len(readMarkers), blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationID, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})

	t.RunWithTimeout("After downloading a shared encrypted file, return a readmarker for each blobber used in download", 3*time.Minute, func(t *test.SystemTest) {
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"
		var allocationID string

		// create viewer wallet
		viewerWalletName := escapedTestName(t) + "_viewer"
		createWalletForNameAndLockReadTokens(t, configPath, viewerWalletName)

		viewerWallet, err := getWalletForName(t, configPath, viewerWalletName)
		require.Nil(t, err)
		require.NotNil(t, viewerWallet)

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID = setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
				"data":   3,
				"parity": 1,
			})
			filename = generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{
				"encrypt": "",
			})
			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation":          allocationID,
				"remotepath":          remotepath + filepath.Base(filename),
				"encryptionpublickey": viewerWallet.EncryptionPublicKey,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		file := "tmp/" + filepath.Base(filename)

		// Download file using auth-ticket: should work
		output, err := downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  file,
		}), true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[len(output)-1], StatusCompletedCB)
		require.Contains(t, output[len(output)-1], filepath.Base(filename))

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationID, sharderUrl)
		require.GreaterOrEqual(t, len(readMarkers), blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationID, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})

	t.RunWithTimeout("After downloading a shared encrypted file by lookuphash, return a readmarker for each blobber used in download", 120*time.Second, func(t *test.SystemTest) {
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"
		var allocationID string

		// create viewer wallet
		viewerWalletName := escapedTestName(t) + "_viewer"
		createWalletForNameAndLockReadTokens(t, configPath, viewerWalletName)

		viewerWallet, err := getWalletForName(t, configPath, viewerWalletName)
		require.Nil(t, err)
		require.NotNil(t, viewerWallet)

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID = setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
				"data":   3,
				"parity": 1,
			})
			filename = generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{
				"encrypt": "",
			})
			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation":          allocationID,
				"remotepath":          remotepath + filepath.Base(filename),
				"encryptionpublickey": viewerWallet.EncryptionPublicKey,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		file := "tmp/" + filepath.Base(filename)

		// Download file using auth-ticket and lookuphash: should work
		output, err := downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"lookuphash": GetReferenceLookup(allocationID, remotepath+filepath.Base(filename)),
			"localpath":  file,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationID, sharderUrl)
		require.GreaterOrEqual(t, len(readMarkers), blobbersRequiredForDownload)

		afterCount := CountReadMarkers(t, allocationID, sharderUrl)
		require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})

	t.Run("After downloading a file by blocks, return a readmarker for each blobber used in download", func(t *test.SystemTest) {
		// 1 block is of size 65536, we upload 20 blocks and download 1 block
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
			"data":   3,
			"parity": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(filename),
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		startBlock := 1
		endBlock := 6
		// Minimum Startblock value should be 1 (since gosdk subtracts 1 from start block, so 0 would lead to startblock being -1).
		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
			"endblock":   endBlock,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		time.Sleep(time.Second * 20)

		readMarkers := GetReadMarkers(t, allocationID, sharderUrl)
		require.GreaterOrEqual(t, len(readMarkers), blobbersRequiredForDownload)

		for _, rm := range readMarkers {
			require.Equal(t, int64(6), rm.ReadCounter)
		}

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
