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
func Test___FlakyBrokenScenarios(t *testing.T) {
	balance := 0.8 // 800.000 mZCN
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Parallel()

	// The test is failling due to sync function inability to detect the file changes in local folder
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
		err = createFileWithSize(path.Join(localFolderRoot, "root.txt"), 32*KB)
		require.Nil(t, err, "Cannot create a local file")

		output, err := syncFolder(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"encryptpath": false,
			"localpath":   localFolderRoot,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		var file_initial climodel.AllocationFile
		for _, item := range files {
			if item.Name == "root.txt" {
				file_initial = item
			}
		}
		require.NotNil(t, file_initial, "sync error, file 'root.txt' must be uploaded to allocation", files)

		// Update the local file in root
		err = createFileWithSize(path.Join(localFolderRoot, "root.txt"), 128*KB)
		require.Nil(t, err, "Cannot update the local file")

		output, err = getDifferences(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  localFolderRoot,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		var differences []climodel.FileDiff
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&differences)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, differences, 1, "we updated a file, we except 1 change but we got %v", len(differences), differences)

		output, err = syncFolder(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  localFolderRoot,
		}, true)
		require.Nil(t, err, "Error in syncing the folder: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Error in listing the allocation files: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files2 []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files2)
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

	// The test is failling due to sync function inability to detect the file changes in local folder
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

	// based on zbox documents, exlude path switch expected to exclude a REMOTE path in allocation from being updated by sync.
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

	// The test is failing due to a possible bug.
	// When owner downloads the file the cost is deduced from the read pool,
	// But it seems the collaborators can download the file for free
	t.Run("Add Collaborator _ file owner must pay for collaborators' reads", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Read pool balance is not being updated at all.")
		}
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

	t.Run("Send with description", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Send ZCN with description is temporarily broken due to json object enforcement")
		}
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = sendZCN(t, configPath, target.ClientID, "1", "rich description", true)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "Send tokens success", output[0])
		// cannot verify transaction payload at this moment due to transaction hash not being printed.
	})

	t.Run("Tokens should move from write pool balance to challenge pool acc. to expected upload cost", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Blobber write pool balance is not being updated correctly")
		}
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

	t.Run("Each blobber's read pool balance should reduce by download cost", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Blobber read pool balance is not being updated correctly")
		}
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   0.6,
			"size":   10485760,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		path, err := filepath.Abs("tmp")
		require.Nil(t, err)

		filename := cliutils.RandomAlphaNumericString(10) + "_test.txt"
		fullPath := fmt.Sprintf("%s/%s", path, filename)
		err = createFileWithSize(fullPath, 1024*5)
		require.Nil(t, err, "error while generating file: ", err)

		// upload a dummy 5 MB file
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fullPath,
			"remotepath": "/",
		})

		// Lock read pool tokens
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "5m",
		})
		output, err = readPoolLock(t, configPath, params, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Read pool before download
		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		initialReadPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Id)
		require.InEpsilon(t, 0.4, intToZCN(initialReadPool[0].Balance), epsilon, "read pool balance did not match expected")
		require.IsType(t, int64(1), initialReadPool[0].ExpireAt)
		require.Equal(t, allocationID, initialReadPool[0].AllocationId)
		require.Less(t, 0, len(initialReadPool[0].Blobber))
		require.Equal(t, true, initialReadPool[0].Locked)

		for i := 0; i < len(initialReadPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), initialReadPool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] balance is [%v]", i, intToZCN(initialReadPool[0].Blobber[i].Balance))
		}

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filename,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))

		expectedDownloadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN = unitToZCN(expectedDownloadCostInZCN, unit)

		// Download the file
		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filename,
			"localpath":  "../../internal/dummy_file/five_MB_test_file_dowloaded",
		}), true)
		require.Nil(t, err, "Downloading the file failed", strings.Join(output, "\n"))

		defer os.Remove("../../internal/dummy_file/five_MB_test_file_dowloaded")

		require.Len(t, output, 2)
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filename, output[1])

		// Read pool after download
		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		finalReadPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Id)
		require.Less(t, intToZCN(finalReadPool[0].Balance), 0.4)
		require.IsType(t, int64(1), finalReadPool[0].ExpireAt)
		require.Equal(t, allocationID, finalReadPool[0].AllocationId)
		require.Equal(t, len(initialReadPool[0].Blobber), len(finalReadPool[0].Blobber))
		require.True(t, finalReadPool[0].Locked)

		for i := 0; i < len(finalReadPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), finalReadPool[0].Blobber[i].Balance)

			// amount deducted
			diff := intToZCN(initialReadPool[0].Blobber[i].Balance) - intToZCN(finalReadPool[0].Blobber[i].Balance)
			t.Logf("blobber [%v] read pool was deducted by [%v]", i, diff)
			require.InEpsilon(t, expectedDownloadCostInZCN, diff, epsilon, "blobber [%v] read pool was deducted by [%v] rather than the expected [%v]", i, diff, expectedDownloadCostInZCN)
		}
	})

	t.Run("update file with thumbnail", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Downloading thumbnail is not working")
		}
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		thumbnailFile := updateFileWithThumbnailURL(t, "https://en.wikipedia.org/static/images/project-logos/enwiki-2x.png", allocationID, "/"+filepath.Base(localFilePath), localFilePath, int64(filesize))

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"thumbnail":  true,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		defer func() {
			// Delete the downloaded thumbnail file
			err = os.Remove(thumbnailFile)
			require.Nil(t, err)
		}()
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update thumbnail of uploaded file", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Downloading thumbnail is not working")
		}
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		thumbnail := "upload_thumbnail_test.png"
		output, err := cliutils.RunCommandWithoutRetry(fmt.Sprintf("wget %s -O %s", "https://en.wikipedia.org/static/images/project-logos/enwiki-2x.png", thumbnail))
		require.Nil(t, err, "Failed to download thumbnail png file: ", strings.Join(output, "\n"))

		localFilePath := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{"thumbnailpath": thumbnail})

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"thumbnail":  true,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		err = os.Remove(thumbnail)
		require.Nil(t, err)

		// Update with new thumbnail
		thumbnail = updateFileWithThumbnailURL(t, "https://icons-for-free.com/iconfiles/png/512/eps+file+format+png+file+icon-1320167140989998942.png", allocationID, "/"+filepath.Base(localFilePath), localFilePath, int64(filesize))

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"thumbnail":  true,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		err = os.Remove(thumbnail)
		require.Nil(t, err)

		createAllocationTestTeardown(t, allocationID)
	})
}
