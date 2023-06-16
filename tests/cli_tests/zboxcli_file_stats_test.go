package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestFileStats(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("get file stats in root directory should work")

	t.Parallel()

	t.TestSetup("Create tmp dir", func() {
		// Create a folder to keep all the generated files to be uploaded
		err := os.MkdirAll("tmp", os.ModePerm)
		require.Nil(t, err)
	})

	const chunksize = 64 * 1024

	t.Run("get file stats in root directory should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		filesize := int64(555)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, int64(1), data.NumOfUpdates)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			// FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}
	})

	t.Run("get file stats in sub directory should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		remotepath := "/dir/"
		filesize := int64(533)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, int64(1), data.NumOfUpdates)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			// FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}
	})

	t.Run("get file stats in nested sub directory should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		remotepath := "/nested/dir/"
		filesize := int64(523)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, int64(1), data.NumOfUpdates)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			// FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}
	})

	t.Run("get file stats on an empty allocation", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		remotepath := "/"

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, "", data.Name)
			require.Equal(t, "", data.Path)
			require.Equal(t, "", data.PathHash)
			require.Equal(t, int64(0), data.Size)
			require.Equal(t, int64(0), data.NumOfBlocks)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, int64(0), data.NumOfChallenges)
			require.Equal(t, int64(0), data.NumOfUpdates)
			require.Equal(t, "", data.WriteMarkerTxn)
			require.Equal(t, "", data.LastChallengeTxn)
			require.Equal(t, "", data.BlobberID)
			require.Equal(t, "", data.BlobberURL)
			require.Equal(t, false, data.BlockchainAware)
		}
	})

	t.Run("get file stats for a file that does not exists", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		absentFileName := "randomFileName.txt"

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": path.Join(remotepath, absentFileName),
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, "", data.Name)
			require.Equal(t, "", data.Path)
			require.Equal(t, "", data.PathHash)
			require.Equal(t, int64(0), data.Size)
			require.Equal(t, int64(0), data.NumOfBlocks)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, int64(0), data.NumOfChallenges)
			require.Equal(t, int64(0), data.NumOfUpdates)
			require.Equal(t, "", data.WriteMarkerTxn)
			require.Equal(t, "", data.LastChallengeTxn)
			require.Equal(t, "", data.BlobberID)
			require.Equal(t, "", data.BlobberURL)
			require.Equal(t, false, data.BlockchainAware)
		}
	})

	t.Run("get file stats for an allocation you dont own", func(t *test.SystemTest) {
		otherAllocationID := ""
		remotepath := "/"
		filesize := int64(533)
		remoteFilePath := ""

		t.Run("Get Other Allocation ID", func(t *test.SystemTest) {
			otherAllocationID = setupAllocation(t, configPath)

			filename := generateFileAndUpload(t, otherAllocationID, remotepath, filesize)
			fname := filepath.Base(filename)
			remoteFilePath = path.Join(remotepath, fname)

			// allocation should work for other wallet
			output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
				"allocation": otherAllocationID,
				"remotepath": remoteFilePath,
				"json":       "",
			}), true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			var stats map[string]climodel.FileStats
			err = json.Unmarshal([]byte(output[0]), &stats)
			require.Nil(t, err)

			for _, data := range stats {
				require.Equal(t, fname, data.Name)
				require.Equal(t, remoteFilePath, data.Path)
				require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(otherAllocationID+":"+remoteFilePath))), data.PathHash)
				require.Equal(t, int64(0), data.NumOfBlockDownloads)
				require.Equal(t, int64(1), data.NumOfUpdates)
				require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
				if data.WriteMarkerTxn == "" {
					require.Equal(t, false, data.BlockchainAware)
				} else {
					require.Equal(t, true, data.BlockchainAware)
				}

				// FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
				// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
			}
		})

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating own wallet failed", err, strings.Join(output, "\n"))

		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": otherAllocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		var stats map[string]climodel.FileStats
		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)
		require.Len(t, stats, 0)
	})

	t.Run("get file stats with no params supplied", func(t *test.SystemTest) {
		setupAllocation(t, configPath)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "getting stats without params should fail", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: allocation flag is missing", strings.Join(output, "\n"))
	})

	t.Run("get file stats with no allocation param supplied", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		filesize := int64(533)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"remotepath": remoteFilePath,
			"json":       "",
		}), false)
		require.NotNil(t, err, "getting stats without params should fail", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: allocation flag is missing", strings.Join(output, "\n"))
	})

	t.Run("get file stats with no remotepath param supplied", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
		}), false)
		require.NotNil(t, err, "getting stats without params should fail", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error: remotepath flag is missing", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("get file stats before and after update", 3*time.Minute, func(t *test.SystemTest) { //todo: too slow
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		remotepath := "/"
		filesize := int64(0.5 * MB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			require.Equal(t, int64(1), data.NumOfUpdates)
			require.Equal(t, int64(0.5*MB/2), data.Size)
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			// FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}

		// update size for the file
		updateFileWithRandomlyGeneratedData(t, allocationID, "/"+fname, int64(1*MB))

		cliutils.Wait(t, 2*time.Minute)
		// fetch file stats after update
		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			require.Equal(t, int64(2), data.NumOfUpdates, "the number of updates count should increment")
			require.Equal(t, int64(1*MB/2), data.Size)
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			// FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}
	})

	t.RunWithTimeout("get file stats before and after download", 7*time.Minute, func(t *test.SystemTest) { //todo: too slow
		allocSize := int64(2048)

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		remotepath := "/"
		filesize := int64(256)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			require.Equal(t, int64(1), data.NumOfUpdates)
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}
		}

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"localpath":  "tmp/",
		}), true)

		require.Nil(t, err, strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, StatusCompletedCB)
		require.Contains(t, aggregatedOutput, filepath.Base(filename))

		cliutils.Wait(t, 2*time.Minute)
		// get file stats after download
		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		var skippedBlobber int
		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			if data.NumOfBlockDownloads == 0 {
				skippedBlobber++
			} else {
				require.Equal(t, int64(1), data.NumOfBlockDownloads)
			}
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, int64(1), data.NumOfUpdates)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}
		}
		require.Equal(t, 2, skippedBlobber)
	})
}

func getFileStats(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	t.Logf("Getting file stats...")
	cmd := fmt.Sprintf(
		"./zbox stats %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
