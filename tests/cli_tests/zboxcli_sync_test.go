package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSyncWithBlobbers(t *testing.T) {
	t.Parallel()

	t.Run("Sync path with cache flag should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"Folder A": map[string]interface{}{
				"file 1.txt": 128 * KB,
				"file 2.txt": 64 * KB,
			},
			"Folder B": map[string]interface{}{
				"file 3.txt": 64 * KB,
				"Folder C": map[string]interface{}{
					"file 4.txt": 64 * KB,
				},
			},
			"abc.txt": 128 * KB, // Create a file with same name but different size
		}

		rootFolder := filepath.Join(os.TempDir(), cliutils.RandomAlphaNumericString(10))
		localCachePath := filepath.Join(rootFolder, "localcache.json")

		// Create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, filepath.Join(rootFolder, "files"), mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localcache": localCachePath,
			"localpath":  localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files in th allocation
		assertFileExistenceRecursively(t, mockFolderStructure, files)

		localCacheFile, err := os.Open(localCachePath)
		require.Nil(t, err, "Unable to read the local cache file due to the error:", err)
		defer localCacheFile.Close()

		var localCacheFileList map[string]interface{}
		err = json.NewDecoder(localCacheFile).Decode(&localCacheFileList)
		require.Nil(t, err, "Error deserializing local cache file: `%v`: %v", localCachePath, err)

		require.Len(t, localCacheFileList, 8, "all files and folders must be appeared in the local cache list")
	})

	t.Run("Sync path with commit should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"Folder A": map[string]interface{}{
				"file 1.txt": 128 * KB,
				"file 2.txt": 64 * KB,
			},
			"Folder B": map[string]interface{}{
				"file 3.txt": 64 * KB,
				"Folder C": map[string]interface{}{
					"file 4.txt": 64 * KB,
				},
			},
			"abc.txt": 128 * KB,
		}

		// Create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"commit":     true,
			"localpath":  localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with chunk size specified should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"Folder A": map[string]interface{}{
				"file 1.txt": 128 * KB,
				"file 2.txt": 64 * KB,
			},
			"Folder B": map[string]interface{}{
				"file 3.txt": 64 * KB,
				"Folder C": map[string]interface{}{
					"file 4.txt": 64 * KB,
				},
			},
			"abc.txt": 128 * KB,
		}

		// Create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"chunksize":  128 * KB,
			"localpath":  localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path to NON-empty allocation (Replace Existing File) should work", func(t *testing.T) {
		t.Parallel()

		originalFileName := "must Be Updated File.txt"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// Create a file locally
		fileLocalFolder := filepath.Join(os.TempDir(), cliutils.RandomAlphaNumericString(10))
		err := os.MkdirAll(fileLocalFolder, os.ModePerm)
		require.Nil(t, err, "cannot create local path folders")
		fileLocalPath := filepath.Join(fileLocalFolder, originalFileName)
		err = createFileWithSize(fileLocalPath, 64*KB)
		require.Nil(t, err, "cannot create local file")

		// Upload the file to allocation root before sync
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": `"/` + filepath.Base(fileLocalPath) + `"`,
			"localpath":  `"` + fileLocalPath + `"`,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, 2, len(output))
		require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"Folder A": map[string]interface{}{
				"file 1.txt": 128 * KB,
				"file 2.txt": 64 * KB,
			},
			"Folder B": map[string]interface{}{
				"file 3.txt": 64 * KB,
				"Folder C": map[string]interface{}{
					"file 4.txt": 64 * KB,
				},
			},
			originalFileName: 128 * KB, // Create a file with same name but different size
		}

		// Create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err = syncFolder(t, configPath, map[string]interface{}{
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

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)

		var foundItem climodel.AllocationFile
		// Assert that the original file before sync, still exists
		for _, item := range files {
			if item.Name == originalFileName {
				foundItem = item
				break
			}
		}
		require.NotNil(t, foundItem, "The original file doesn't exist anymore", files)
		require.Equal(t, 128*KB, foundItem.Size, "The original file doesn't exist anymore", files)
	})

	t.Run("Sync path to NON-empty allocation (No filename Clashes) should work", func(t *testing.T) {
		t.Parallel()

		originalFileName := "no clash filename.txt"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// Create a file locally
		fileLocalFolder := filepath.Join(os.TempDir(), cliutils.RandomAlphaNumericString(10))
		err := os.MkdirAll(fileLocalFolder, os.ModePerm)
		require.Nil(t, err, "cannot create local path folders")
		fileLocalPath := filepath.Join(fileLocalFolder, originalFileName)
		err = createFileWithSize(fileLocalPath, 32*KB)
		require.Nil(t, err, "cannot create local file")

		// Upload the file to allocation root before sync
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": `"/` + filepath.Base(fileLocalPath) + `"`,
			"localpath":  `"` + fileLocalPath + `"`,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, 2, len(output))
		require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"Folder A": map[string]interface{}{
				"file 1.txt": 128 * KB,
				"file 2.txt": 64 * KB,
			},
			"Folder B": map[string]interface{}{
				"file 3.txt": 64 * KB,
				"Folder C": map[string]interface{}{
					"file 4.txt": 64 * KB,
				},
			},
			"file 5.txt": 128 * KB,
		}

		// Create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err = syncFolder(t, configPath, map[string]interface{}{
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

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)

		var foundItem climodel.AllocationFile
		// Assert that the original file before sync, still exists
		for _, item := range files {
			if item.Name == originalFileName {
				foundItem = item
				break
			}
		}
		require.NotNil(t, foundItem, "The original file doesn't exist anymore", files)
	})

	t.Run("Sync path with multiple files in nested directories to empty allocation should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"Folder A": map[string]interface{}{
				"file1.txt": 128 * KB,
				"file2.txt": 64 * KB,
			},
			"Folder B": map[string]interface{}{
				"file 3.txt": 64 * KB,
				"Folder C": map[string]interface{}{
					"file 4.txt": 64 * KB,
				},
			},
		}

		// This will create files and folders based on defined structure recursively
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

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with multiple files to empty allocation should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"file 1.txt": 64 * KB,
			"file 2.txt": 20 * KB,
			"file 3.txt": 1 * KB,
			"file 4.txt": 140 * KB,
		}

		// This will create files and folders based on defined structure recursively
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

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with multiple files encrypted to empty allocation should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"file 1.txt": 64 * KB,
			"file 2.txt": 20 * KB,
			"file 3.txt": 1 * KB,
			"file 4.txt": 140 * KB,
		}

		// This will create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": true,
			"localpath":   localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with 1 file encrypted to empty allocation should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"file1.txt": 64 * KB,
		}

		// This will create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": true,
			"localpath":   localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with 1 file to empty allocation should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"file1.txt": 64*KB + 1,
		}

		// This will create files and folders based on defined structure recursively
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

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})
}

// This will traverse the tree and asserts the existent of the files
func assertFileExistenceRecursively(t *testing.T, structure map[string]interface{}, files []climodel.AllocationFile) {
	for name, value := range structure {
		switch v := value.(type) {
		case int:
			var foundItem climodel.AllocationFile
			for _, item := range files {
				if item.Name == name {
					foundItem = item
					break
				}
			}
			require.NotNil(t, foundItem, "File %s is not found in allocation files", name)
		case map[string]interface{}:
			assertFileExistenceRecursively(t, v, files)
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

func getDifferences(t *testing.T, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return getDifferencesWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func getDifferencesWithWallet(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Get Differences...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox get-diff %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
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

// This will create files and folders based on defined structure recursively inside the root folder
//
//	- rootFolder: Leave empty or send "/" to create on os temp folder
//	- structure: Map of the desired folder structure to be created; Int values will represent a file with that size, Map values will be considered as folders
//	- returns local root folder
//	- sample structure:
// map[string]interface{}{
// 	"FolderA": map[string]interface{}{
// 		"file1.txt": 64*KB + 1,
// 		"file2.txt": 64*KB + 1,
// 	},
// 	"FolderB": map[string]interface{}{},
// }
func createMockFolders(t *testing.T, rootFolder string, structure map[string]interface{}) (string, error) {
	if rootFolder == "" || rootFolder == "/" {
		rootFolder = filepath.Join(os.TempDir(), "to-sync", cliutils.RandomAlphaNumericString(10))
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
