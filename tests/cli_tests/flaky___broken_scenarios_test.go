//nolint:gocritic
package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

/*
Tests in here are skipped until the feature has been fixed
*/

//nolint:gocyclo

func Test___FlakyBrokenScenarios(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Skip()
	balance := 0.8 // 800.000 mZCN
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Parallel()

	// FIXME The test is failing due to sync function inability to detect the file changes in local folder see https://github.com/0chain/zboxcli/issues/250
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

	// FIXME The test is failling due to sync function inability to detect the file changes in local folder see <tbd>
	t.Run("BROKEN Sync path to non-empty allocation - locally updated files (in sub folder) must be updated in allocation but is not see zboxcli/issues/250", func(t *test.SystemTest) {
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

	// FIXME based on zbox documents, exclude path switch expected to exclude a REMOTE path in allocation from being updated by sync. see <tbd>
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
			"abc.txt": 32 * KB,
		}

		// Create files and folders based on defined structure recursively
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
		err = createFileWithSize(filepath.Join(rootLocalFolder, "abc.txt"), 128*KB)
		require.Nil(t, err, "Cannot change the file size")

		output, err = getDifferences(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"localpath":   rootLocalFolder,
			"excludepath": excludedFolderName,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var differences []climodel.FileDiff
		err = json.Unmarshal([]byte(output[0]), &differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, differences, 2, "Since we added a file and we updated 2 files (1 excluded) we expect 2 differences but we got %v", len(differences))

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"localpath":   rootLocalFolder,
			"excludepath": excludedFolderName,
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

	// FIXME: WRITEPOOL TOKEN ACCOUNTING
	t.Run("Tokens should move from write pool balance to challenge pool acc. to expected upload cost", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   balance,
			"size":   10485760,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		// Write pool balance should increment to 1
		initialAllocation := getAllocation(t, allocationID)
		require.Equal(t, 0.8, intToZCN(initialAllocation.WritePool))

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, 1024*5)
		require.Nil(t, err, "error while generating file: ", err)

		// Get expected upload cost
		output, _ = getUploadCostInUnit(t, configPath, allocationID, filename)

		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost is given in "per 720 hours", we need 1 hour
		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := expectedUploadCostInZCN / ((2 + 2) * 720)
		// upload a dummy 5 MB file
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": "/",
		})

		finalAllocation := getAllocation(t, allocationID)
		require.Equal(t, 0.8, intToZCN(finalAllocation.WritePool))

		// Get Challenge-Pool info after upload
		output, err = challengePoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Could not fetch challenge pool", strings.Join(output, "\n"))

		challengePool := climodel.ChallengePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &challengePool)
		require.Nil(t, err, "Error unmarshalling challenge pool info", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("([a-f0-9]{64}):challengepool:%s", allocationID)), challengePool.Id)
		require.IsType(t, int64(1), challengePool.StartTime)
		require.IsType(t, int64(1), challengePool.Expiration)
		require.IsType(t, int64(1), challengePool.Balance)
		require.False(t, challengePool.Finalized)

		// FIXME: Blobber details are empty
		// Blobber pool balance should reduce by (write price*filesize) for each blobber
		totalChangeInWritePool := intToZCN(initialAllocation.WritePool - finalAllocation.WritePool)

		require.Equal(t, actualExpectedUploadCostInZCN, totalChangeInWritePool, "expected write pool balance to decrease by [%v] but has actually decreased by [%v]", actualExpectedUploadCostInZCN, totalChangeInWritePool)
		require.Equal(t, totalChangeInWritePool, intToZCN(challengePool.Balance), "expected challenge pool balance to match deducted amount from write pool [%v] but balance was actually [%v]", totalChangeInWritePool, intToZCN(challengePool.Balance))
	})

	// FIXME: Commented out because these cases hang the broken test suite till timeout

	// FIXME: add param validation
	// t.Run("Upload from local webcam feed with a negative chunksize should fail", func(t *test.SystemTest) {
	// 	output, err := createWallet(t, configPath)
	// 	require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

	// 	output, err = executeFaucetWithTokens(t, configPath, 2.0)
	// 	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	// 	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
	// 		"lock": 1,
	// 	}))
	// 	require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
	// 	require.Len(t, output, 1)
	// 	require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
	// 	allocationID := strings.Fields(output[0])[2]

	// 	remotepath := "/live/stream.m3u8"
	// 	localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
	// 	localpath := filepath.Join(localfolder, "up.m3u8")
	// 	err = os.MkdirAll(localpath, os.ModePerm)
	// 	require.Nil(t, err, "Error in creating the folders", localpath)
	// 	defer os.RemoveAll(localfolder)

	// 	chunksize := -655360
	// 	// FIXME: negative chunksize works without error, after implementing fix change startUploadFeed to
	// 	// runUploadFeed below
	// 	err = startUploadFeed(t, configPath, createParams(map[string]interface{}{
	// 		"allocation": allocationID,
	// 		"localpath":  localpath,
	// 		"remotepath": remotepath,
	// 		"live":       "",
	// 		"chunksize":  chunksize,
	// 	}))
	// 	require.Nil(t, err, "expected error when using negative chunksize")
	// 	KillFFMPEG()
	// })

	// FIXME: add param validation
	// t.Run("Upload from youtube feed with a negative chunksize should fail", func(t *test.SystemTest) {
	//

	// 	output, err := createWallet(t, configPath)
	// 	require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

	// 	output, err = executeFaucetWithTokens(t, configPath, 2.0)
	// 	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	// 	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
	// 		"lock": 1,
	// 	}))
	// 	require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
	// 	require.Len(t, output, 1)
	// 	require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
	// 	allocationID := strings.Fields(output[0])[2]

	// 	remotepath := "/live/stream.m3u8"
	// 	localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
	// 	localpath := filepath.Join(localfolder, "up.m3u8")
	// 	err = os.MkdirAll(localpath, os.ModePerm)
	// 	require.Nil(t, err, "Error in creating the folders", localpath)
	// 	defer os.RemoveAll(localfolder)

	// 	chunksize := -655360
	// 	// FIXME: negative chunksize works without error, after implementing fix change startUploadFeed to
	// 	// runUploadFeed below
	// 	err = startUploadFeed(t, configPath, createParams(map[string]interface{}{
	// 		"allocation": allocationID,
	// 		"localpath":  localpath,
	// 		"remotepath": remotepath,
	// 		"feed":       `https://www.youtube.com/watch?v=5qap5aO4i9A`,
	// 		"sync":       "",
	// 		"chunksize":  chunksize,
	// 	}))
	// 	require.Nil(t, err, "expected error when using negative chunksize")
	// 	KillFFMPEG()
	// })

	// FIXME: add param validation
	// t.Run("Uploading youtube feed with negative delay should fail", func(t *test.SystemTest) {
	// 	output, err := createWallet(t, configPath)
	// 	require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

	// 	output, err = executeFaucetWithTokens(t, configPath, 2.0)
	// 	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	// 	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
	// 		"lock": 1,
	// 	}))
	// 	require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
	// 	require.Len(t, output, 1)
	// 	require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
	// 	allocationID := strings.Fields(output[0])[2]

	// 	remotepath := "/live/stream.m3u8"
	// 	localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
	// 	localpath := filepath.Join(localfolder, "up.m3u8")
	// 	err = os.MkdirAll(localpath, os.ModePerm)
	// 	require.Nil(t, err, "Error in creating the folders", localpath)
	// 	defer os.RemoveAll(localfolder)

	// 	err = runUploadFeed(t, configPath, createParams(map[string]interface{}{
	// 		"allocation": allocationID,
	// 		"localpath":  localpath,
	// 		"remotepath": remotepath,
	// 		"feed":       `https://www.youtube.com/watch?v=5qap5aO4i9A`,
	// 		"sync":       "",
	// 		"delay":      -10,
	// 	}))
	// 	require.NotNil(t, err, "negative delay should fail")
	// 	KillFFMPEG()
	// })

	// FIXME: add param validation
	// t.Run("Uploading local webcam feed with negative delay should fail", func(t *test.SystemTest) {
	// 	output, err := createWallet(t, configPath)
	// 	require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

	// 	output, err = executeFaucetWithTokens(t, configPath, 2.0)
	// 	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	// 	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
	// 		"lock": 1,
	// 	}))
	// 	require.Nil(t, err, "error creating allocation", strings.Join(output, "\n"))
	// 	require.Len(t, output, 1)
	// 	require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
	// 	allocationID := strings.Fields(output[0])[2]

	// 	remotepath := "/live/stream.m3u8"
	// 	localfolder := filepath.Join(os.TempDir(), escapedTestName(t))
	// 	localpath := filepath.Join(localfolder, "up.m3u8")
	// 	err = os.MkdirAll(localpath, os.ModePerm)
	// 	require.Nil(t, err, "Error in creating the folders", localpath)
	// 	defer os.RemoveAll(localfolder)

	// 	err = runUploadFeed(t, configPath, createParams(map[string]interface{}{
	// 		"allocation": allocationID,
	// 		"localpath":  localpath,
	// 		"remotepath": remotepath,
	// 		"live":       "",
	// 		"delay":      -10,
	// 	}))
	// 	require.NotNil(t, err, "negative delay should fail")
	// 	KillFFMPEG()
	// })
}

// func runUploadFeed(t *test.SystemTest, cliConfigFilename, params string) error {
// 	t.Logf("Starting upload of live stream to zbox...")
// 	commandString := fmt.Sprintf("./zbox upload %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, params)
// 	_, err := cliutils.RunCommandWithoutRetry(commandString)
// 	return err
// }
