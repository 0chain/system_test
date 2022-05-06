//nolint:gocritic
package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

/*
Tests in here are skipped until the feature has been fixed
*/

//nolint:gocyclo
func Test___FlakyBrokenScenarios(t *testing.T) {
	balance := 0.8 // 800.000 mZCN
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Parallel()

	// FIXME The test is failling due to sync function inability to detect the file changes in local folder
	// https://0chain.slack.com/archives/G014PQ61WNT/p1638477374103000
	t.Run("Sync path to non-empty allocation - locally updated files (in root) must be updated in allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

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

	// FIXME The test is failling due to sync function inability to detect the file changes in local folder
	// https://0chain.slack.com/archives/G014PQ61WNT/p1638477374103000
	t.Run("Sync path to non-empty allocation - locally updated files (in sub folder) must be updated in allocation", func(t *testing.T) {
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

	// FIXME based on zbox documents, exclude path switch expected to exclude a REMOTE path in allocation from being updated by sync.
	// So this is failing due to the whole update in sync is failing.
	t.Run("Sync path to non-empty allocation - exclude a path should work", func(t *testing.T) {
		t.Parallel()

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

	// The test is failing due to a possible bug.
	// When owner downloads the file the cost is deduced from the read pool,
	// But it seems the collaborators can download the file for free
	t.Run("Add Collaborator _ file owner must pay for collaborators' reads", func(t *testing.T) {
		t.Parallel()

		collaboratorWalletName := escapedTestName(t) + "_collaborator"

		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 4 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 1*MB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))

		expectedDownloadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN = unitToZCN(expectedDownloadCostInZCN, unit)
		expectedDownloadCost := ConvertToValue(expectedDownloadCostInZCN)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Equal(t, 1, len(meta.Collaborators), "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "1h",
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0])

		readPool := getReadPoolInfo(t, allocationID)
		require.Len(t, readPool, 1, "Read pool must exist")
		require.Equal(t, ConvertToValue(0.4), readPool[0].Balance, "Read Pool balance must be equal to locked amount")

		output, err = downloadFileForWallet(t, collaboratorWalletName, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, "Error in downloading the file as collaborator", strings.Join(output, "\n"))
		defer os.Remove("tmp" + remotepath)
		require.Equal(t, 2, len(output), "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(localpath))
		require.Equal(t, expectedOutput, output[1], "Unexpected output", strings.Join(output, "\n"))

		// Wait for read markers to be redeemed
		cliutils.Wait(t, 5*time.Second)

		readPool = getReadPoolInfo(t, allocationID)
		require.Len(t, readPool, 1, "Read pool must exist")
		// expected download cost times to the number of blobbers
		expectedPoolBalance := ConvertToValue(0.4) - int64(len(readPool[0].Blobber))*expectedDownloadCost
		require.InEpsilon(t, expectedPoolBalance, readPool[0].Balance, 0.000001, "Read Pool balance must be equal to (initial balace-download cost)")
		t.Logf("Expected Read Pool Balance: %v\nActual Read Pool Balance: %v", expectedPoolBalance, readPool[0].Balance)
	})

	t.Run("Tokens should move from write pool balance to challenge pool acc. to expected upload cost", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

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

		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		require.InEpsilon(t, 0.8, intToZCN(initialWritePool[0].Balance), epsilon)
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId)
		require.Less(t, 0, len(initialWritePool[0].Blobber))
		require.Equal(t, true, initialWritePool[0].Locked)

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
		actualExpectedUploadCostInZCN := (expectedUploadCostInZCN / ((2 + 2) * 720))

		// upload a dummy 5 MB file
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": "/",
		})

		// Get the new Write-Pool info after upload
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		require.InEpsilon(t, 0.8, intToZCN(finalWritePool[0].Balance), epsilon)
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, finalWritePool[0].AllocationId)
		require.Less(t, 0, len(finalWritePool[0].Blobber))
		require.Equal(t, true, finalWritePool[0].Locked)

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

		// Blobber pool balance should reduce by (write price*filesize) for each blobber
		totalChangeInWritePool := float64(0)
		for i := 0; i < len(finalWritePool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), finalWritePool[0].Blobber[i].Balance)

			// deduce tokens
			diff := intToZCN(initialWritePool[0].Blobber[i].Balance) - intToZCN(finalWritePool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] write pool has decreased by [%v] tokens after upload", i, diff)
			totalChangeInWritePool += diff
		}

		require.InEpsilon(t, actualExpectedUploadCostInZCN, totalChangeInWritePool, epsilon, "expected write pool balance to decrease by [%v] but has actually decreased by [%v]", actualExpectedUploadCostInZCN, totalChangeInWritePool)
		require.InEpsilon(t, totalChangeInWritePool, intToZCN(challengePool.Balance), epsilon, "expected challenge pool balance to match deducted amount from write pool [%v] but balance was actually [%v]", totalChangeInWritePool, intToZCN(challengePool.Balance))
	})

	t.Run("delete existing file in root directory with commit should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		remotepath := "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"commit":     true,
		}), true)

		// FIXME: error in deleting file with commit
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
	})

	t.Run("should allow update of owner", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		ownerKey := "owner_id"
		newOwner := "22e412a350036944f9762a3d6b5687ee4f64d20d2cf6faf2571a490defd10f17"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwner,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "interest pool smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getInterestPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newOwner, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwner)

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": oldOwner,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "interest pool smart contract settings updated", output[0], strings.Join(output, "\n"))
	})

	// FIXME: interest rate is invalid config key
	t.Run("should allow update of interest_rate", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "interest_rate"
		newValue := "0.1"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgBefore, _ := keyValuePairStringToMap(t, output)

		// ensure revert in config is run regardless of test result
		defer func() {
			oldValue := cfgBefore[configKey]
			output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": oldValue,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			// FIXME should say miner instead of storagesc
			require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
			require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
		}()

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		// FIXME should say miner instead of storagesc
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newValue, cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	// FIXME: interest rate is not a valid configkey
	t.Run("update interest_rate to invalid value should fail", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "interest_rate"
		newValue := "x"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unsupported key interest_rate", output[0], strings.Join(output, "\n"))
	})

	// FIXME: Commented out because these cases hang the broken test suite till timeout

	// FIXME: add param validation
	// t.Run("Upload from local webcam feed with a negative chunksize should fail", func(t *testing.T) {
	// 	output, err := registerWallet(t, configPath)
	// 	require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

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
	// t.Run("Upload from youtube feed with a negative chunksize should fail", func(t *testing.T) {
	// 	t.Parallel()

	// 	output, err := registerWallet(t, configPath)
	// 	require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

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
	// t.Run("Uploading youtube feed with negative delay should fail", func(t *testing.T) {
	// 	output, err := registerWallet(t, configPath)
	// 	require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

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
	// t.Run("Uploading local webcam feed with negative delay should fail", func(t *testing.T) {
	// 	output, err := registerWallet(t, configPath)
	// 	require.Nil(t, err, "failed to register wallet", strings.Join(output, "\n"))

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

// func runUploadFeed(t *testing.T, cliConfigFilename, params string) error {
// 	t.Logf("Starting upload of live stream to zbox...")
// 	commandString := fmt.Sprintf("./zbox upload %s --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, params)
// 	_, err := cliutils.RunCommandWithoutRetry(commandString)
// 	return err
// }
