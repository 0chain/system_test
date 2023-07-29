package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	t.RunWithTimeout("Number of challenges between 2 blocks should be equal to the number of blocks (given that we have active allocations", 4*time.Minute, func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 9,
		})

		startBlock := getLatestFinalizedBlock(t)

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

		time.Sleep(2 * time.Minute)

		endBlock := getLatestFinalizedBlock(t)

		challengesCountQuery := fmt.Sprintf("round_created_at >= %d AND round_created_at < %d", startBlock.Round, endBlock.Round)
		challenges, err := countChallengesByBlocks(t, challengesCountQuery, sharderBaseURLs)
		require.Nil(t, err, "error counting challenges", strings.Join(output, "\n"))

		require.Equal(t, endBlock.Round-startBlock.Round, challenges["total"], "number of challenges should be equal to the number of blocks")
		require.Equal(t, endBlock.Round-startBlock.Round, challenges["passed"], "number of challenges should be equal to the number of blocks")
	})
}

func getAllSharderBaseURLs(sharders map[string]*climodel.Sharder) []string {
	sharderURLs := make([]string, 0)
	for _, sharder := range sharders {
		sharderURLs = append(sharderURLs, getNodeBaseURL(sharder.Host, sharder.Port))
	}
	return sharderURLs
}

func apiGetOpenChallenges(t require.TestingT, sharderBaseURLs []string, blobberId string, offset, limit int, lastChallengeRound int64) *climodel.BlobberChallenge {
	for _, sharderBaseURL := range sharderBaseURLs {
		res, err := http.Get(fmt.Sprintf(sharderBaseURL + "/v1/screst/" + storageSmartContractAddress +
			"/openchallenges" + "?blobber=" + blobberId + "&offset=" + strconv.Itoa(offset) + "&limit=" + strconv.Itoa(limit) + "&from=" + strconv.FormatInt(lastChallengeRound, 10)))
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
		limit := 50
		lastChallengeRound := int64(0)

		for {
			openChallengesInBlobber := apiGetOpenChallenges(t, sharderBaseURLs, blobberId, offset, limit, lastChallengeRound)
			if openChallengesInBlobber == nil || len(openChallengesInBlobber.Challenges) == 0 {
				break
			}
			for _, challenge := range openChallengesInBlobber.Challenges {
				openChallenges[challenge.ID] = challenge

				if challenge.RoundCreatedAt > lastChallengeRound {
					lastChallengeRound = challenge.RoundCreatedAt
				}
			}
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

func countChallengesByBlocks(t *test.SystemTest, query string, sharderBaseURLs []string) (map[string]int64, error) {
	for _, sharderBaseURL := range sharderBaseURLs {
		res, err := http.Get(fmt.Sprintf(sharderBaseURL + "/v1/screst/" + storageSmartContractAddress +
			"/count-challenges" + "?query=" + query))
		if err != nil || res.StatusCode < 200 || res.StatusCode >= 300 {
			continue
		}

		require.Nil(t, err, "error getting challenges count", res)
		require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get challenges count between blocks")
		require.NotNil(t, res.Body, "get challenges count between blocks API response must not be nil")

		resBody, err := io.ReadAll(res.Body)
		func() { defer res.Body.Close() }()

		require.Nil(t, err, "Error reading response body")

		var challengesCount map[string]int64
		err = json.Unmarshal(resBody, &challengesCount)
		require.Nil(t, err, "error unmarshalling response body")

		return challengesCount, nil
	}
	t.Errorf("all sharders gave an error at endpoint /count-challenges")

	return nil, nil
}
