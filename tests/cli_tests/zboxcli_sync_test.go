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

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSyncWithBlobbers(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Sync path with 1 file to empty allocation should work")

	t.Parallel()

	t.Run("Sync path with 1 file to empty allocation should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

		// The folder structure tree
		// Integer values will be considered as files with that size
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

	t.Run("Sync path with 1 file encrypted to empty allocation should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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

	t.Run("Sync path with 1 file to empty allocation and download the file should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

		filename := "file1.txt"

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			filename: 64 * KB,
		}

		// This will create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		originalFileChecksum := generateChecksum(t, path.Join(localpath, filename))
		require.NotNil(t, originalFileChecksum)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 1, "1 file must be uploaded", files)
		file := files[0]

		require.NotNil(t, file, "sync error, file 'file1.txt' must be uploaded to allocation", files)

		downloadPath := path.Join(localpath, "download")
		err = os.MkdirAll(downloadPath, os.ModePerm)
		require.Nil(t, err, "Error in creating local folders")

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file.Path,
			"localpath":  downloadPath,
		}), true)

		// FIXME file cannot be downloaded so we get an error here
		// check the issue on github https://github.com/0chain/zboxcli/issues/142
		// FIXME after issue is solved
		fixed := false
		if !fixed {
			t.Log("FIXME", strings.Join(output, "\n"))
		} else {
			require.Nil(t, err, "Error in downloading the file", strings.Join(output, "\n"))
			require.Len(t, output, 2)

			expected := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filename)
			require.Equal(t, expected, output[1])

			downloadedFileChecksum := generateChecksum(t, path.Join(downloadPath, filename))

			require.Equal(t, originalFileChecksum, downloadedFileChecksum, "Downloaded file checksum is different than the uploaded file checksum")
		}
	})

	t.Run("Sync path with 1 file encrypted to empty allocation and download the file should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

		filename := "file1.txt"

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			filename: 64 * KB,
		}

		// This will create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		originalFileChecksum := generateChecksum(t, path.Join(localpath, filename))

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": true,
			"localpath":   localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 1, "1 file must be uploaded", files)
		file := files[0]

		require.NotNil(t, file, "sync error, file 'file1.txt' must be uploaded to allocation", files)

		downloadPath := path.Join(localpath, "download")
		err = os.MkdirAll(downloadPath, os.ModePerm)
		require.Nil(t, err, "Error in creating local folders")

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file.Path,
			"localpath":  downloadPath,
		}), true)

		// FIXME file cannot be downloaded so we get an error here
		// check the issue on github https://github.com/0chain/zboxcli/issues/142
		// FIXME after issue is solved
		fixed := false
		if !fixed {
			t.Log("FIXME", strings.Join(output, "\n"))
		} else {
			require.Nil(t, err, "Error in downloading the file", strings.Join(output, "\n"))
			require.Len(t, output, 2)

			expected := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filename)
			require.Equal(t, expected, output[1])

			downloadedFileChecksum := generateChecksum(t, path.Join(downloadPath, filename))

			require.Equal(t, originalFileChecksum, downloadedFileChecksum, "Downloaded file checksum is different than the uploaded file checksum")
		}
	})

	t.Run("Sync path with multiple files encrypted to empty allocation should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with multiple files to empty allocation should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with multiple files in nested directories to empty allocation should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path to NON-empty allocation (No filename Clashes) should work", func(t *test.SystemTest) {
		originalFileName := "no clash filename.txt"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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

	t.Run("Sync path to NON-empty allocation (Replace Existing File) should work", func(t *test.SystemTest) {
		originalFileName := "must Be Updated File.txt"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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
		require.Equal(t, 128*KB, foundItem.ActualSize, "The original file doesn't exist anymore", files)
	})

	t.Run("Sync path with chunk number specified should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
			"allocation":  allocationID,
			"chunknumber": 2,
			"localpath":   localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.Run("Sync path with cache flag should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
		require.GreaterOrEqual(t, len(output), 2, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-2], strings.Join(output, "\n"))
		require.Equal(t, "Local cache saved.", output[len(output)-1], strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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

	t.Run("Sync path with uploadonly flag should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
			"uploadonly": true,
			"localpath":  localpath,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// This will traverse the tree and asserts the existent of the files
		assertFileExistenceRecursively(t, mockFolderStructure, files)
	})

	t.RunWithTimeout("Attempt to Sync to allocation not owned must fail", 2*time.Minute, func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

		notOwnerWalletName := escapedTestName(t) + "_NOT_OWNER_WALLET"
		output, err := createWalletForName(t, configPath, notOwnerWalletName)
		require.Nil(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"abc.txt": 128 * KB,
		}

		// Create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err = syncFolderWithWallet(t, notOwnerWalletName, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  localpath,
		}, false) // Do not retry when expecting failure

		require.NotNil(t, err)
		require.Len(t, output, 2)
		require.Contains(t, strings.Join(output, "\n"), "error from server list response:", strings.Join(output, "\n"))

		// no file must be uploaded to allocation
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "[]")
	}) //todo: too slow

	t.Run("Attempt to Sync to non-existing allocation must fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		allocationID := "invalid-allocation-id"

		// The folder structure tree
		// Integer values will be consider as files with that size
		// Map values will be considered as folders
		mockFolderStructure := map[string]interface{}{
			"abc.txt": 128 * KB,
		}

		// Create files and folders based on defined structure recursively
		localpath, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, localpath)
		defer os.RemoveAll(localpath)

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  localpath,
		}, false)
		require.NotNil(t, err, "Unexpected success in syncing the folder: ", strings.Join(output, "\n"))

		require.GreaterOrEqual(t, len(output), 1, "unexpected number of lines in output", strings.Join(output, "\n"))

		require.Equal(t, "Error fetching the allocation allocation_fetch_error: "+
			"Error fetching the allocation.internal_error: can't get allocation: error retrieving allocation: "+
			"invalid-allocation-id, error: record not found", output[0], strings.Join(output, "\n"))
	})

	t.Run("Sync path to non-empty allocation - locally updated files (in root) must be updated in allocation", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

		localFolderRoot := filepath.Join(os.TempDir(), "to-sync", cliutils.RandomAlphaNumericString(10))
		err := os.MkdirAll(localFolderRoot, os.ModePerm)
		require.Nil(t, err, "Error in creating the folders", localFolderRoot)
		defer os.RemoveAll(localFolderRoot)

		// Create a local file in root
		err = createFileWithSize(filepath.Join(localFolderRoot, "root.txt"), 32*KB)
		require.Nil(t, err, "Cannot create a local file")

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   localFolderRoot,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		var file_initial climodel.AllocationFile
		for _, item := range files {
			if item.Name == "root.txt" {
				file_initial = item
			}
		}
		require.NotNil(t, file_initial, "sync error, file 'root.txt' must be uploaded to allocation", files)

		// Update the local file in root
		err = createFileWithSize(filepath.Join(localFolderRoot, "root.txt"), 128*KB)
		require.Nil(t, err, "Cannot update the local file")

		output, err = getDifferences(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  localFolderRoot,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var differences []climodel.FileDiff
		err = json.Unmarshal([]byte(output[0]), &differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, differences, 1, "we updated a file, we except 1 change but we got %v", len(differences), differences)

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  localFolderRoot,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files2 []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files2)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		var file climodel.AllocationFile
		for _, item := range files2 {
			if item.Name == "root.txt" {
				file = item
			}
		}
		require.NotNil(t, file, "sync error, file 'root.txt' must've been uploaded to the allocation", files2)

		require.Greater(t, file.Size, file_initial.Size, "file expected to be updated to bigger size")
	})

	t.Run("Sync path to non-empty allocation - locally updated files (in sub folder) must be updated in allocation but is not see zboxcli/issues/250", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
			"allocation": allocationID,
			"localpath":  rootLocalFolder,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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
		err = createFileWithSize(filepath.Join(rootLocalFolder, "folder1", "file-in-folder1.txt"), 128*KB)
		require.Nil(t, err, "Cannot update the local file")
		err = createFileWithSize(filepath.Join(rootLocalFolder, "folder2", "file-in-folder2.txt"), 128*KB)
		require.Nil(t, err, "Cannot update the local file")

		output, err = getDifferences(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  rootLocalFolder,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var differences []climodel.FileDiff
		err = json.Unmarshal([]byte(output[0]), &differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, differences, 2, "Since we updated 2 files we expect 2 differences but we got %v", len(differences), differences)

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   rootLocalFolder,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files2 []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files2)
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

	t.Run("Sync path to non-empty allocation - exclude a path should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		createAllocationTestTeardown(t, allocationID)

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
			"decdisdbejdkcdqo3udewd.txt": 32 * KB,
		}

		// Create files and folders based on defined structure recursively
		rootLocalFolder, err := createMockFolders(t, "", mockFolderStructure)
		require.Nil(t, err, "Error in creating mock folders: ", err, rootLocalFolder)
		defer os.RemoveAll(rootLocalFolder)

		output, err := getDifferences(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"localpath":   rootLocalFolder,
			"excludepath": "/" + excludedFolderName,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var differences []climodel.FileDiff
		err = json.Unmarshal([]byte(output[0]), &differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, differences, 2, "Since we added a file and we updated 2 files (1 excluded) we expect 2 differences but we got %v", len(differences))

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  rootLocalFolder,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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
		err = createFileWithSize(filepath.Join(rootLocalFolder, excludedFolderName, excludedFileName), 128*KB)
		require.Nil(t, err, "Cannot change the file size")
		err = createFileWithSize(filepath.Join(rootLocalFolder, includedFolderName, includedFileName), 128*KB)
		require.Nil(t, err, "Cannot change the file size")
		err = createFileWithSize(filepath.Join(rootLocalFolder, "decdisdbejdkcdqo3udewd.txt"), 128*KB)
		require.Nil(t, err, "Cannot change the file size")

		output, err = getDifferences(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"localpath":   rootLocalFolder,
			"excludepath": "/" + excludedFolderName,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, differences, 2, "Since we added a file and we updated 2 files (1 excluded) we expect 2 differences but we got %v", len(differences))

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"localpath":   rootLocalFolder,
			"excludepath": "/" + excludedFolderName,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Sync Complete", output[len(output)-1])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files2 []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files2)
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
}

// This will traverse the tree and asserts the existent of the files
func assertFileExistenceRecursively(t *test.SystemTest, structure map[string]interface{}, files []climodel.AllocationFile) {
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

func syncFolder(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return syncFolderWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func syncFolderWithWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
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

func getDifferences(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return getDifferencesWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func getDifferencesWithWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
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

// nolint
// This will create files and folders based on defined structure recursively inside the root folder
//
//   - rootFolder: Leave empty or send "/" to create on os temp folder
//
//   - structure: Map of the desired folder structure to be created; Int values will represent a file with that size, Map values will be considered as folders
//
//   - returns local root folder
//
//   - sample structure:
//
//     map[string]interface{} {
//     "FolderA": map[string]interface{}{
//     "file1.txt": 64*KB + 1,
//     "file2.txt": 64*KB + 1,
//     },
//     "FolderB": map[string]interface{}{},
//     }
func createMockFolders(t *test.SystemTest, rootFolder string, structure map[string]interface{}) (string, error) {
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
