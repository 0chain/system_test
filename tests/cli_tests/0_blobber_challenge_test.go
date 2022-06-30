package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBlobberChallenge(t *testing.T) {
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	// Get sharder list.
	output, err = getSharders(t, configPath)
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 1)
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]*climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	// Get base URL for API calls.
	sharderBaseURLs := getAllSharderBaseURLs(sharders)
	require.Greater(t, len(sharderBaseURLs), 0, "No sharder URLs found.")

	blobberList := []climodel.BlobberInfo{}
	output, err = listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	t.Run("Uploading a file greater than 1 MB should generate randomized challenges", func(t *testing.T) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after an upload operation")
	})

	t.Run("Downloading a file greater than 1 MB should generate randomized challenges", func(t *testing.T) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after a move operation")
	})

	t.Run("Moving a file greater than 1 MB should generate randomized challenges", func(t *testing.T) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err, "error creating file")

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"destpath":   "/dest/",
		}, true)
		require.Nil(t, err, "error moving file", strings.Join(output, "\n"))

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after a move operation")
	})

	t.Run("Deleting a file greater than 1 MB should generate randomized challenges", func(t *testing.T) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err, "error creating file")

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

		output, err = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
		}), true)
		require.Nil(t, err, "error deleting file", strings.Join(output, "\n"))

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after a move operation")
	})

	t.Run("Copying a file greater than 1 MB should generate randomized challenges", func(t *testing.T) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err, "error creating file")

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"destpath":   "/dest/",
		}, true)
		require.Nil(t, err, "error copying file", strings.Join(output, "\n"))

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after a move operation")
	})

	t.Run("Updating a file greater than 1 MB should generate randomized challenges", func(t *testing.T) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err, "error creating file")

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

		localfile := generateRandomTestFileName(t)
		err = createFileWithSize(localfile, 2*MB)
		require.Nil(t, err)

		output, err = updateFileWithWallet(t, escapedTestName(t), configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  localfile,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after an update operation")
	})

	t.Run("Renaming a file greater than 1 MB should generate randomized challenges", func(t *testing.T) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err, "error creating file")

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"destname":   "newFile",
		}, true)
		require.Nil(t, err, "error renaming file", strings.Join(output, "\n"))

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after a rename operation")
	})
}

func getAllSharderBaseURLs(sharders map[string]*climodel.Sharder) []string {
	sharderURLs := make([]string, 0)
	for _, sharder := range sharders {
		sharderURLs = append(sharderURLs, getNodeBaseURL(sharder.Host, sharder.Port))
	}
	return sharderURLs
}

func apiGetOpenChallenges(t require.TestingT, sharderBaseURLs []string, blobberId string, offset int, limit int) *apimodel.BlobberChallenge {
	for _, sharderBaseURL := range sharderBaseURLs {
		res, err := http.Get(fmt.Sprintf(sharderBaseURL + "/v1/screst/" + storageSmartContractAddress +
			"/openchallenges" + "?blobber=" + blobberId + "&offset=" + strconv.Itoa(offset) + "&limit=" + strconv.Itoa(limit)))
		if err != nil || res.StatusCode < 200 || res.StatusCode >= 300 {
			continue
		}

		require.Nil(t, err, "error getting challenges", res)
		require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get open challenges for blobber id: %s", blobberId)
		require.NotNil(t, res.Body, "Open challenges API response must not be nil")

		resBody, err := io.ReadAll(res.Body)
		defer res.Body.Close()

		require.Nil(t, err, "Error reading response body")
		var openChallengesInBlobber apimodel.BlobberChallenge
		err = json.Unmarshal(resBody, &openChallengesInBlobber)
		require.Nil(t, err, "error unmarshalling response body")

		return &openChallengesInBlobber
	}
	t.Errorf("all sharders gave an error at endpoint /openchallenges")

	return nil
}

func openChallengesForAllBlobbers(t *testing.T, sharderBaseURLs, blobbers []string) (openChallenges []string) {
	for _, blobberId := range blobbers {
		offset := 0
		limit := 20
		for {
			openChallengesInBlobber := apiGetOpenChallenges(t, sharderBaseURLs, blobberId, offset, limit)
			if openChallengesInBlobber == nil || len(openChallengesInBlobber.Challenges) == 0 {
				break
			}
			for _, challenge := range openChallengesInBlobber.Challenges {
				openChallenges = append(openChallenges, challenge.ID)
			}
			offset += limit
		}
	}

	return openChallenges
}

func areNewChallengesOpened(t *testing.T, sharderBaseURLs, blobbers []string, openChallengesBefore []string) bool {
	t.Log("Checking for new challenges to open...")
	for i := 0; i < 150; i++ {
		openChallengesAfter := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)
		for _, id := range openChallengesAfter {
			if contains(id, openChallengesBefore) {
				continue // this challenge already existed before we made a commit_connection
			} else {
				return true // this challenge was generated after we
			}
		}
		cliutils.Wait(t, time.Second*1)
	}
	return false
}

func contains(id string, challenges []string) bool {
    for _, challenge := range challenges {
        if challenge == id {
            return true
        }
    }
    return false
}