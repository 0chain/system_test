package cli_tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
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

		remotepath := "/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		// Upload parameters
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath,
			"localpath":  filename,
			"commit":     "",
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		cliutils.Wait(t, 30*time.Second)

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

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		err = sdk.GenerateChallenges(0)
		require.Nil(t, err, "error calling Generate Challenges")

		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"json":       "",
		}), true)
		require.Nil(t, err, "error getting file stats")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err, "error unmarshalling file stats json")

		var challenge int
		for _, stat := range stats {
			challenge += int(stat.NumOfChallenges)
		}

		require.Greater(t, challenge, 0)
	})
}
