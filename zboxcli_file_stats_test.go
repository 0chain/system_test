package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileStats(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	// Success Scenarios

	t.Run("get file stats in root directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		filesize := int64(5)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, v := range stats {
			require.Equal(t, fname, v.Name)
			require.Equal(t, remoteFilePath, v.Path)
		}
	})

	t.Run("get file stats in sub directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/dir/"
		filesize := int64(5)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, v := range stats {
			require.Equal(t, fname, v.Name)
			require.Equal(t, remoteFilePath, v.Path)
		}
	})

	t.Run("get file stats in nested sub directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/nested/dir/"
		filesize := int64(5)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, v := range stats {
			require.Equal(t, fname, v.Name)
			require.Equal(t, remoteFilePath, v.Path)
		}
	})

	t.Run("get file stats on an empty allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/"

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, v := range stats {
			require.Equal(t, "", v.Name)
			require.Equal(t, "", v.Path)
		}
	})

	t.Run("get file stats for a file that does not exists", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		absentFileName := "randomFileName.txt"
		filesize := int64(5)
		generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": path.Join(remotepath, absentFileName),
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, v := range stats {
			require.Equal(t, "", v.Name)
			require.Equal(t, "", v.Path)
		}
	})
}

func getFileStats(t *testing.T, cliConfigFilename, param string) ([]string, error) {
	t.Logf("Getting file stats...")
	cmd := fmt.Sprintf(
		"./zbox stats %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(cmd)
}
