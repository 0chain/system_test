package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestReadMarker(t *testing.T) {
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	t.Run("Read-markers retrieved after download", func(t *testing.T) {
		const defaultBlobberCount = 4
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotePath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotePath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilePath := remotePath + filepath.Base(filename)

		sharderUrl := getSharderUrl(t)
		beforeCount := CountReadMarkers(t, allocationId, sharderUrl)
		require.Zerof(t, beforeCount.ReadMarkersCount, "non zero read-marker count before download")
		fmt.Println("beforeCount", beforeCount)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilePath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		time.Sleep(time.Second * 10)

		readMarkers := GetReadMarkers(t, allocationId, "", sharderUrl)
		fmt.Println("readmarkers", len(readMarkers))
		//require.Len(t, readMarkers, defaultBlobberCount)
		afterCount := CountReadMarkers(t, allocationId, sharderUrl)
		fmt.Println("readmarker count after", afterCount)
		//require.EqualValuesf(t, afterCount.ReadMarkersCount, len(readMarkers), "should equal length of read-markers", len(readMarkers))
	})
}

func CountReadMarkers(t *testing.T, allocationId string, sharderBaseUrl string) *climodel.ReadMarkersCount {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + cliutils.StorageScAddress + "/count_readmarkers")
	params := map[string]string{
		"allocation_id": allocationId,
	}
	return cliutils.ApiGet[climodel.ReadMarkersCount](t, url, params)
}

func GetReadMarkers(t *testing.T, allocationId, authTicket string, sharderBaseUrl string) []climodel.ReadMarker {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + cliutils.StorageScAddress + "/readmarkers")
	params := make(map[string]string)
	if len(allocationId) > 0 {
		params["allocation_id"] = allocationId
	}
	if len(authTicket) > 0 {
		params["auth_ticket"] = authTicket
	}
	return cliutils.ApiGetList[climodel.ReadMarker](t, url, params, 0, 100)
}
