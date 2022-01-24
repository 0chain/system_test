package cli_tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBlobberChallenge(t *testing.T) {
	t.Parallel()

	t.Run("Creating challenge against an allocation of used size > 1 MB should work", func(t *testing.T) {
		t.Parallel()

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		// Upload parameters
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
			"commit":     "",
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error getting file stats")
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats
		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err, "error unmarshalling file stats json")

		var blobberId string
		for _, stat := range stats {
			blobberId = stat.BlobberID
		}

		// Get sharder list.
		output, err = getSharders(t, configPath)
		require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1)
		require.Equal(t, "MagicBlock Sharders", output[0])

		var sharders map[string]climodel.Sharder
		err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
		require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

		// Use first sharder from map.
		sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

		// Get base URL for API calls.
		sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)
		res1, err := apiGetOpenChallenges(sharderBaseUrl, blobberId)
		require.Nil(t, err, "error getting challenges", res1)
		require.True(t, res1.StatusCode >= 200 && res1.StatusCode < 300, "Failed API request to get open challenges for blobber id: %s", blobberId)
		require.NotNil(t, res1.Body, "Open challenges API response must not be nil")

		res1Body, err := ioutil.ReadAll(res1.Body)
		require.Nil(t, err, "Error reading response body")
		var openChallengesBefore apimodel.BlobberChallenge
		err = json.Unmarshal(res1Body, &openChallengesBefore)
		require.Nil(t, err, "error unmarshalling response body")

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)

		res2, err := apiGetOpenChallenges(sharderBaseUrl, blobberId)
		require.Nil(t, err, "error getting challenges", res2)
		require.True(t, res2.StatusCode >= 200 && res2.StatusCode < 300, "Failed API request to get open challenges for blobber id: %s", blobberId)
		require.NotNil(t, res1.Body, "Open challenges API response must not be nil")

		res2Body, err := ioutil.ReadAll(res2.Body)
		require.Nil(t, err, "Error reading response body")
		var openChallengesAfter apimodel.BlobberChallenge
		err = json.Unmarshal(res2Body, &openChallengesAfter)
		require.Nil(t, err, "error unmarshalling response body")

		// New challenges must have been created after a download request
		require.Greater(t, len(openChallengesAfter.Challenges), len(openChallengesBefore.Challenges))
	})
}

func apiGetOpenChallenges(sharderBaseURL, blobberId string) (*http.Response, error) {
	return http.Get(fmt.Sprintf(sharderBaseURL + "/v1/screst/" + storageSmartContractAddress + "/openchallenges" + "?blobber=" + blobberId))
}
