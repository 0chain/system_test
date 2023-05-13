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

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBlobberChallenge(testSetup *testing.T) {
	// todo: all of these tests poll for up to 2mins30s - is this reasonable?
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Uploading a file greater than 1 MB should generate randomized challenges")

	var blobberList []climodel.BlobberInfo
	var sharderBaseURLs []string

	t.TestSetup("Get list of sharders and blobbers", func() {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

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
		sharderBaseURLs = getAllSharderBaseURLs(sharders)
		require.Greater(t, len(sharderBaseURLs), 0, "No sharder URLs found.")

		blobberList = []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")
	})

	t.RunSequentiallyWithTimeout("Uploading a file greater than 1 MB should generate randomized challenges", 3*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
		})

		var blobbers []string
		for _, blobber := range blobberList {
			blobbers = append(blobbers, blobber.Id)
		}

		openChallengesBefore := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)

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

		passed := areNewChallengesOpened(t, sharderBaseURLs, blobbers, openChallengesBefore)
		require.True(t, passed, "expected new challenges to be created after an upload operation")
	})

	t.RunSequentiallyWithTimeout("Downloading a file greater than 1 MB should generate randomized challenges", 3*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

	t.RunSequentiallyWithTimeout("Moving a file greater than 1 MB should generate randomized challenges", 3*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

	t.RunSequentiallyWithTimeout("Deleting a file greater than 1 MB should generate randomized challenges", 3*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

	t.RunSequentiallyWithTimeout("Copying a file greater than 1 MB should generate randomized challenges", 3*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

	t.RunSequentiallyWithTimeout("Updating a file greater than 1 MB should generate randomized challenges", 3*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

	t.RunSequentiallyWithTimeout("Renaming a file greater than 1 MB should generate randomized challenges", 3*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
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

func apiGetOpenChallenges(t require.TestingT, sharderBaseURLs []string, blobberId string, offset, limit int) *climodel.BlobberChallenge {
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
		func() { defer res.Body.Close() }()

		require.Nil(t, err, "Error reading response body")
		var openChallengesInBlobber climodel.BlobberChallenge
		err = json.Unmarshal(resBody, &openChallengesInBlobber)
		require.Nil(t, err, "error unmarshalling response body")

		return &openChallengesInBlobber
	}
	t.Errorf("all sharders gave an error at endpoint /openchallenges")

	return nil
}

func openChallengesForAllBlobbers(t *test.SystemTest, sharderBaseURLs, blobbers []string) (openChallenges map[string]climodel.Challenges) {
	openChallenges = make(map[string]climodel.Challenges)
	for _, blobberId := range blobbers {
		offset := 0
		limit := 20
		for {
			openChallengesInBlobber := apiGetOpenChallenges(t, sharderBaseURLs, blobberId, offset, limit)
			if openChallengesInBlobber == nil || len(openChallengesInBlobber.Challenges) == 0 {
				break
			}
			for _, challenge := range openChallengesInBlobber.Challenges {
				openChallenges[challenge.ID] = challenge
			}
			offset += limit
		}
	}

	return openChallenges
}

func areNewChallengesOpened(t *test.SystemTest, sharderBaseURLs, blobbers []string, openChallengesBefore map[string]climodel.Challenges) bool {
	t.Log("Checking for new challenges to open...")
	for i := 0; i < 150; i++ {
		openChallengesAfter := openChallengesForAllBlobbers(t, sharderBaseURLs, blobbers)
		for _, challenge := range openChallengesAfter {
			if _, ok := openChallengesBefore[challenge.ID]; !ok {
				return true
			}
		}
		cliutils.Wait(t, time.Second)
	}
	return false
}
