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

	t.Run("Sync path to non-empty allocation - locally updated files must be updated in allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"folder1": map[string]interface{}{
				"file-in-folder1.txt": 32 * KB,
			},
			"folder2": map[string]interface{}{
				"file-in-folder2.txt": 16 * KB,
			},
		}

		// Create files and folders based on the defined structure recursively
		rootLocalFolder, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, rootLocalFolder)
		defer os.RemoveAll(rootLocalFolder)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   rootLocalFolder,
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

		var file1_initial climodel.AllocationFile
		var file2_initial climodel.AllocationFile
		for _, item := range files {
			if item.Name == "file-in-folder1.txt" {
				file1_initial = item
			} else if item.Name == "file-in-folder2.txt" {
				file2_initial = item
			}
		}

		// Update the local files in sub folders
		err = createFileWithSize(path.Join(rootLocalFolder, "folder1", "file-in-folder1.txt"), 128*KB)
		require.Nil(t, err, "Cannot update the local file")
		err = createFileWithSize(path.Join(rootLocalFolder, "folder2", "file-in-folder2.txt"), 128*KB)
		require.Nil(t, err, "Cannot update the local file")
		err = createFileWithSize(path.Join(rootLocalFolder, "root.txt"), 128*KB)
		require.Nil(t, err, "Cannot update the local file")

		output, err = getDifferences(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  rootLocalFolder,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		var differences []climodel.FileDiff
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, differences, 3, "Since we updated 2 files we expect 2 differences but we got %v", len(differences), differences)

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   rootLocalFolder,
			// "excludepath": excludedFolderName,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files2 []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files2)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		var file1 climodel.AllocationFile
		var file2 climodel.AllocationFile
		for _, item := range files2 {
			if item.Name == "file-in-folder1.txt" {
				file1 = item
			} else if item.Name == "file-in-folder2.txt" {
				file2 = item
			}
		}
		require.NotNil(t, file1, "sync error, file 'file-in-folder1.txt' must be uploaded to allocation", files2)
		require.NotNil(t, file2, "sync error, file 'file-in-folder2.txt' must be uploaded to allocation", files2)

		require.Greater(t, file1.Size, file1_initial.Size, "file1 expected to be updated to bigger size")
		require.Greater(t, file2.Size, file2_initial.Size, "file2 expected to be updated to bigger size")

	})

	t.Run("Sync path to empty allocation - exclude a path should work", func(t *testing.T) {
		// t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		// We want to exclude the folder containing this file from being synced
		excludedFileName := "file1.txt"
		excludedFolderName := "excludedFolder"
		includedFileName := "file2.txt"
		includedFolderName := "includedFolder"

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			includedFolderName: map[string]interface{}{
				includedFileName: 8 * KB,
			},
			excludedFolderName: map[string]interface{}{
				excludedFileName: 16 * KB,
			},
			"abc.txt": 32 * KB,
		}

		// Create files and folders based on defined structure recursively
		rootLocalFolder, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, rootLocalFolder)
		defer os.RemoveAll(rootLocalFolder)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   rootLocalFolder,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		var includedFile_initial climodel.AllocationFile
		var excludedFile_initial climodel.AllocationFile

		for _, item := range files {
			if item.Name == includedFileName {
				includedFile_initial = item
			} else if item.Name == excludedFileName {
				excludedFile_initial = item
			}
		}
		require.NotNil(t, includedFile_initial, "sync error, file '%s' must be uploaded to allocation", includedFileName, files)
		require.NotNil(t, excludedFile_initial, "sync error, file '%s' must be uploaded to allocation", excludedFile_initial, files)

		// Update the local files
		err = createFileWithSize(path.Join(rootLocalFolder, excludedFolderName, excludedFileName), 128*KB)
		require.Nil(t, err, "Cannot change the file size")
		err = createFileWithSize(path.Join(rootLocalFolder, includedFolderName, includedFileName), 128*KB)
		require.Nil(t, err, "Cannot change the file size")
		err = createFileWithSize(path.Join(rootLocalFolder, "abc.txt"), 128*KB)
		require.Nil(t, err, "Cannot change the file size")

		output, err = getDifferences(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  rootLocalFolder,
			// "excludepath": excludedFolderName,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		var differences []climodel.FileDiff
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, differences, 3, "Since we added a file and we updated 2 files we expect 3 differences but we got %v", len(differences))

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   rootLocalFolder,
			// "excludepath": excludedFolderName,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files2 []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files2)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		var includedFile_final climodel.AllocationFile
		var excludedFile_final climodel.AllocationFile
		for _, item := range files2 {
			if item.Name == includedFileName {
				includedFile_final = item
			} else if item.Name == excludedFileName {
				excludedFile_final = item
			}
		}
		require.NotNil(t, includedFile_final, "sync error, file '%s' must be uploaded to allocation", includedFileName, files2)
		require.NotNil(t, excludedFile_final, "sync error, file '%s' must be uploaded to allocation", excludedFileName, files2)

		require.Greater(t, includedFile_final.Size, includedFile_initial.Size, "included file expected to be updated to bigger size")
		require.Equal(t, excludedFile_initial.Size, excludedFile_final.Size, "excluded file expected to NOT be updated")

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
