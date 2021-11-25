package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSyncWithBlobbers(t *testing.T) {
	t.Parallel()

	t.Run("sync a folder to empty allocation should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		mockFolderStructure := map[string]interface{}{
			"FolderA": map[string]interface{}{
				"file1.txt": 64*KB + 1,
				"file2.txt": 64*KB + 1,
			},
			"FolderB": map[string]interface{}{},
		}
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		assertFileExistanceRecursively(t, mockFolderStructure, files)
	})
}

func assertFileExistanceRecursively(t *testing.T, structure map[string]interface{}, files []climodel.AllocationFile) {
	for name, value := range structure {
		switch v := value.(type) {
		case int:
			found := false
			for _, item := range files {
				if item.Name == name {
					found = true
					break
				}
			}
			require.True(t, found, "File %s is not found in allocation files", name)
		case map[string]interface{}:
			assertFileExistanceRecursively(t, v, files)
		}
	}
}

func syncFolder(t *testing.T, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return syncFolderWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func syncFolderWithWallet(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Syncing folder...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox sync %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*40)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func createMockFolders(t *testing.T, rootFolder string, structure map[string]interface{}) (string, error) {
	if rootFolder == "" || rootFolder == "/" {
		rootFolder = filepath.Join(os.TempDir(), "to-synce", cliutils.RandomAlphaNumericString(10))
	}
	err := os.MkdirAll(rootFolder, os.ModePerm)
	if err != nil {
		return rootFolder, err
	}

	for name, value := range structure {
		switch v := value.(type) {
		case int:
			localpath := path.Join(rootFolder, name)
			err := createFileWithSize(localpath, int64(v))
			if err != nil {
				return rootFolder, err
			}
		case map[string]interface{}:
			_, err := createMockFolders(t, path.Join(rootFolder, name), v)
			if err != nil {
				return rootFolder, err
			}
		}
	}
	return rootFolder, nil
}
