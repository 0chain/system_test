package cli_tests

import (
	"fmt"
	"math"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

func Test___FlakyScenariosCommonUserFunctions(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	// FIXME: WRITEPOOL TOKEN ACCOUNTING
	t.RunWithTimeout("File Update with a different size - Blobbers should be paid for the extra file size", (1*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// Logic: Upload a 0.5 MB file and get the upload cost. Update the 0.5 MB file with a 1 MB file
		// and see that blobber's write pool balances are deduced again for the cost of uploading extra
		// 0.5 MBs.

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock":   "0.5",
			"size":   10 * MB,
			"data":   2,
			"parity": 2,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		fileSize := int64(0.5 * MB)

		// Get expected upload cost for 0.5 MB
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)
		output, _ = getUploadCostInUnit(t, configPath, allocationID, localpath)
		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))
		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := (expectedUploadCostInZCN / (2 + 2))

		// Wait for write pool balance to be deduced for initial 0.5 MB
		cliutils.Wait(t, time.Minute)

		initialAllocation := getAllocation(t, allocationID)

		require.Equal(t, 0.5-actualExpectedUploadCostInZCN, intToZCN(initialAllocation.WritePool))

		remotepath := "/" + filepath.Base(localpath)
		updateFileWithRandomlyGeneratedData(t, allocationID, remotepath, int64(1*MB))

		// Wait before fetching final write pool
		cliutils.Wait(t, time.Minute)

		finalAllocation := getAllocation(t, allocationID)
		require.Equal(t, (0.5 - 2*actualExpectedUploadCostInZCN), intToZCN(finalAllocation.WritePool))

		// Blobber pool balance should reduce by expected cost of 0.5 MB
		totalChangeInWritePool := intToZCN(initialAllocation.WritePool - finalAllocation.WritePool)
		require.Equal(t, actualExpectedUploadCostInZCN, totalChangeInWritePool)
		createAllocationTestTeardown(t, allocationID)
	})
}

func Test___FlakyTransferAllocation(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	t.SetRunAllTestsAsSmokeTest()

	t.RunWithTimeout("transfer allocation accounting test", 6*time.Minute, func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(1024000),
		})

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, 204800)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = createWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, newOwner, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		initialAllocation := getAllocation(t, allocationID)

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))

		transferred := pollForAllocationTransferToEffect(t, newOwner, allocationID)
		require.True(t, transferred, "allocation was not transferred to new owner within time allotted")

		// balance of old owner should be unchanged
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.4, balance)

		// balance of new owner should be unchanged
		balance, err = getBalanceZCN(t, configPath, newOwner)
		require.NoError(t, err)
		require.Equal(t, 5.9, balance)

		// write lock pool of old owner should remain locked
		cliutils.Wait(t, 2*time.Minute)

		// Get expected upload cost
		output, _ = getUploadCostInUnit(t, configPath, allocationID, file)

		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost is given in "per 720 hours", we need 1 hour
		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := expectedUploadCostInZCN / ((2 + 2) * 720)

		finalAllocation := getAllocation(t, allocationID)
		actualCost := initialAllocation.WritePool - finalAllocation.WritePool

		// If a challenge has passed for upload, writepool balance should reduce, else, remain same
		require.True(t, actualCost == 0 || intToZCN(actualCost) == actualExpectedUploadCostInZCN)
	})

	// todo: some allocation transfer operations are very slow
	t.RunWithTimeout("transfer allocation and upload file", 6*time.Minute, func(t *test.SystemTest) { // todo: very slow
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(20480),
		})

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = createWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, newOwner, configPath, 1)
		require.Nil(t, err, "faucet execution failed for non-owner wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))

		transferred := pollForAllocationTransferToEffect(t, newOwner, allocationID)
		require.True(t, transferred, "allocation was not transferred to new owner within time allotted")

		output, err = writePoolLockWithWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.5,
		}), false)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "write pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "write pool lock - Unexpected output", strings.Join(output, "\n"))

		output, err = uploadFileForWallet(t, newOwner, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/new" + remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))
	}) //todo:slow
}

func Test___FlakyFileDelete(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetRunAllTestsAsSmokeTest()

	t.RunWithTimeout("Delete file concurrently in existing directory, should work", 6*time.Minute, func(t *test.SystemTest) { // TODO: slow
		const allocSize int64 = 2048
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		var fileNames [2]string

		const remotePathPrefix = "/"

		var outputList [2][]string
		var errorList [2]error
		var wg sync.WaitGroup

		for i, fileName := range fileNames {
			wg.Add(1)
			go func(currentFileName string, currentIndex int) {
				defer wg.Done()

				fileName := filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
				fileNames[currentIndex] = fileName

				remoteFilePath := filepath.Join(remotePathPrefix, fileName)

				op, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
					"allocation": allocationID,
					"remotepath": remoteFilePath,
				}), true)

				errorList[currentIndex] = err
				outputList[currentIndex] = op
			}(fileName, i)
		}

		wg.Wait()

		const expectedPattern = "%s deleted"

		for i := 0; i < 2; i++ {
			require.Nil(t, errorList[i], strings.Join(outputList[i], "\n"))
			require.Len(t, outputList, 2, strings.Join(outputList[i], "\n"))

			require.Equal(t, fmt.Sprintf(expectedPattern, fileNames[i]), filepath.Base(outputList[i][0]), "Output is not appropriate")
		}

		for i := 0; i < 2; i++ {
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": path.Join(remotePathPrefix, fileNames[i]),
				"json":       "",
			}), true)

			require.NotNil(t, err, strings.Join(output, "\n"))
			require.Contains(t, strings.Join(output, "\n"), "Invalid path record not found")
		}
	})
}

func Test___FlakyFileRename(testSetup *testing.T) { // nolint:gocyclo
	t := test.NewSystemTest(testSetup)
	t.SetRunAllTestsAsSmokeTest()

	t.Parallel()

	t.RunWithTimeout("Rename and delete file concurrently, should work", 6*time.Minute, func(t *test.SystemTest) { // todo: unacceptably slow
		const allocSize int64 = 2048
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		var renameFileNames [2]string
		var destFileNames [2]string

		var deleteFileNames [2]string

		const remotePathPrefix = "/"

		var renameOutputList, deleteOutputList [2][]string
		var renameErrorList, deleteErrorList [2]error
		var wg sync.WaitGroup

		renameFileName := filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
		renameFileNames[0] = renameFileName

		destFileName := filepath.Base(generateRandomTestFileName(t))
		destFileNames[0] = destFileName

		renameFileName = filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
		renameFileNames[1] = renameFileName

		destFileName = filepath.Base(generateRandomTestFileName(t))
		destFileNames[1] = destFileName

		deleteFileName := filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
		deleteFileNames[0] = deleteFileName

		deleteFileName = filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
		deleteFileNames[1] = deleteFileName

		for i := 0; i < 2; i++ {
			wg.Add(2)

			go func(currentIndex int) {
				defer wg.Done()

				op, err := renameFile(t, configPath, map[string]interface{}{
					"allocation": allocationID,
					"remotepath": filepath.Join(remotePathPrefix, renameFileNames[currentIndex]),
					"destname":   destFileNames[currentIndex],
				}, true)

				renameErrorList[currentIndex] = err
				renameOutputList[currentIndex] = op
			}(i)

			go func(currentIndex int) {
				defer wg.Done()

				op, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
					"allocation": allocationID,
					"remotepath": filepath.Join(remotePathPrefix, deleteFileNames[currentIndex]),
				}), true)

				deleteErrorList[currentIndex] = err
				deleteOutputList[currentIndex] = op
			}(i)
		}

		wg.Wait()

		const renameExpectedPattern = "%s renamed"

		for i := 0; i < 2; i++ {
			require.Nil(t, renameErrorList[i], strings.Join(renameOutputList[i], "\n"))
			require.Len(t, renameOutputList[i], 1, strings.Join(renameOutputList[i], "\n"))

			require.Equal(t, fmt.Sprintf(renameExpectedPattern, renameFileNames[i]), filepath.Base(renameOutputList[i][0]), "Rename output is not appropriate")
		}

		const deleteExpectedPattern = "%s deleted"

		for i := 0; i < 2; i++ {
			require.Nil(t, deleteErrorList[i], strings.Join(deleteOutputList[i], "\n"))
			require.Len(t, deleteOutputList[i], 1, strings.Join(deleteOutputList[i], "\n"))

			require.Equal(t, fmt.Sprintf(deleteExpectedPattern, deleteFileNames[i]), filepath.Base(deleteOutputList[i][0]), "Delete output is not appropriate")
		}

		for i := 0; i < 2; i++ {
			output, err := listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": path.Join(remotePathPrefix, deleteFileNames[i]),
				"json":       "",
			}), true)

			require.Error(t, err)
			require.Len(t, output, 1)
		}
	})
}

func Test___FlakyFileCopy(testSetup *testing.T) { // nolint:gocyclo
	t := test.NewSystemTest(testSetup)
	t.SetRunAllTestsAsSmokeTest()

	t.RunWithTimeout("File copy - Users should not be charged for moving a file ", 60*time.Second, func(t *test.SystemTest) { // see https://github.com/0chain/zboxcli/issues/334
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 4 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		fileSize := int64(math.Floor(1 * MB))

		// Upload 1 MB file
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)

		// Get initial write pool
		cliutils.Wait(t, 10*time.Second)

		initialAllocation := getAllocation(t, allocationID)

		// Move file
		remotepath := "/" + filepath.Base(localpath)

		// copy file
		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"destpath":   "/newdir/",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotepath+" copied"), output[0])

		// Get expected upload cost
		output, _ = getUploadCostInUnit(t, configPath, allocationID, localpath)

		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost is given in "per 720 hours", we need 1 hour
		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := expectedUploadCostInZCN / ((2 + 2) * 720)

		finalAllocation := getAllocation(t, allocationID)

		actualCost := initialAllocation.WritePool - finalAllocation.WritePool
		require.True(t, actualCost == 0 || intToZCN(actualCost) == actualExpectedUploadCostInZCN)

		createAllocationTestTeardown(t, allocationID)
	})
}
