package cli_tests

import (
	"encoding/json"
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

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// Fixed
func TestFileRename(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("rename file should work")

	t.Parallel()

	t.Run("rename folder should work", func(t *test.SystemTest) {
		allocSize := int64(2048)

		remotePath := "/child"

		destPath := "/child_modified/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := createDir(t, configPath, allocationID, remotePath, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, remotePath+" directory created", output[0])

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destPath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf(remotePath+" renamed"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 1)

		// check if expected file has been renamed
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destPath {
				foundAtDest = true
				require.Equal(t, destPath, f.Name, strings.Join(output, "\n"))
				require.Equal(t, "d", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("rename file should work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destName := "new_" + filename
		destPath := "/child/" + destName

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" renamed"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been renamed
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destPath {
				foundAtDest = true
				require.Equal(t, destName, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Rename file concurrently to existing directory, should work", 6*time.Minute, func(t *test.SystemTest) { // todo: slow
		const allocSize int64 = 2048
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		var fileNames [2]string
		var destFileNames []string

		const remotePathPrefix = "/"

		var outputList [2][]string
		var errorList [2]error
		var wg sync.WaitGroup

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(currentIndex int) {
				defer wg.Done()

				fileName := filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
				fileNames[currentIndex] = fileName

				destFileName := filepath.Base(generateRandomTestFileName(t))
				destFileNames = append(destFileNames, destFileName)

				op, err := renameFile(t, configPath, map[string]interface{}{
					"allocation": allocationID,
					"remotepath": filepath.Join(remotePathPrefix, fileName),
					"destname":   destFileName,
				}, true)

				errorList[currentIndex] = err
				outputList[currentIndex] = op
			}(i)
		}

		wg.Wait()

		const renameExpectedPattern = "%s renamed"

		for i := 0; i < 2; i++ {
			require.Nil(t, errorList[i], strings.Join(outputList[i], "\n"))
			require.Len(t, outputList[i], 1, strings.Join(outputList[i], "\n"))

			require.Equal(t, fmt.Sprintf(renameExpectedPattern, fileNames[i]), filepath.Base(outputList[i][0]), "Rename output is not appropriate")
		}

		output, err := listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Len of output is not enough")

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2, "Amount of files is not enough")

		for _, file := range files {
			_, ok := cliutils.Contains(destFileNames, file.Name)

			require.True(t, ok, strings.Join(output, "\n"))
			require.GreaterOrEqual(t, file.Size, int(fileSize), strings.Join(output, "\n"))
			require.Equal(t, "f", file.Type, strings.Join(output, "\n"))
			require.NotEmpty(t, file.Hash, "File hash is empty")
		}
	})

	t.Run("rename file to same filename (no change) shouldn't work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destName := filename

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if file is still there
		found := false
		for _, f := range files {
			if f.Path == remotePath { // nolint:gocritic // this is better than inverted if cond
				found = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, found, "file not found: ", strings.Join(output, "\n"))
	})

	t.Run("rename file to with 90-char (below 100-char filename limit)", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		b := make([]rune, 90-4) // subtract chars for extension
		for i := range b {
			b[i] = 'a'
		}
		destName := string(b) + ".txt"
		destPath := "/child/" + destName

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		// rename file
		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" renamed"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has not been renamed
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destPath {
				foundAtDest = true
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("rename file to with 110-char (above 100-char filename limit) should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		b := make([]rune, 110-4) // subtract chars for extension
		for i := range b {
			b[i] = 'a'
		}
		destName := string(b) + ".txt"
		destPath := "/child/" + destName

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "filename is longer than 100 characters")

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has not been renamed
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destPath {
				foundAtDest = true
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.False(t, foundAtDest, "file is found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("rename file to containing special characters", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destName := "!@#$%^&*()<>{}[]:;'?,." + filename
		destPath := "/child/" + destName

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		// rename file
		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" renamed"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been renamed
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destPath {
				foundAtDest = true
				require.Equal(t, destName, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("File Rename - Users should not be charged for renaming a file", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "5",
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

		// Wait for write marker to be redeemed
		time.Sleep(16 * time.Second)
		// Get initial write pool
		initialAllocation := getAllocation(t, allocationID)

		// Rename file
		remotepath := filepath.Base(localpath)
		renameAllocationFile(t, allocationID, remotepath, remotepath+"_renamed")

		// Get expected upload cost
		output, _ = getUploadCostInUnit(t, configPath, allocationID, localpath)

		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost is given in "per 720 hours", we need 1 hour
		actualExpectedUploadCostInZCN := expectedUploadCostInZCN / 720

		finalAllocation := getAllocation(t, allocationID)

		actualCost := initialAllocation.WritePool - finalAllocation.WritePool
		require.True(t, actualCost == 0 || intToZCN(actualCost) == actualExpectedUploadCostInZCN)
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("rename root path should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)

		remotePath := "/"
		destName := "new_"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Rename failed. invalid_operation: cannot rename root path", output[0])
	}) //todo: too slow

	t.Run("rename non-existing file should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		// try to rename a file
		output, err := renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/child/nonexisting.txt",
			"destname":   "/child/newnonexisting.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Rename failed")
	})

	t.Run("rename file from someone else's allocation should fail", func(t *test.SystemTest) {
		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err := createWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destName := "new_" + filename
		destPath := "/child/" + destName

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = renameFileWithWallet(t, configPath, nonAllocOwnerWallet, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Rename failed")

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file was not renamed
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destPath {
				foundAtDest = true
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.False(t, foundAtDest, "file is found at destination: ", strings.Join(output, "\n"))
	}) //todo: too slow

	t.Run("rename file with no allocation param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs on rename
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = renameFile(t, configPath, map[string]interface{}{
			"remotepath": "/abc.txt",
			"destname":   "def.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("rename file with no remotepath param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs on rename
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"destname":   "def.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath flag is missing", output[0])
	})

	t.Run("rename file with no destname param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs on rename
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"remotepath": "/abc.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: destname flag is missing", output[0])
	})

	t.Run("File Rename - Users should not be charged for renaming a file", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "5",
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
		time.Sleep(16 * time.Second)
		initialAllocation := getAllocation(t, allocationID)

		// Rename file
		remotepath := filepath.Base(localpath)
		renameAllocationFile(t, allocationID, remotepath, remotepath+"_renamed")

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

	t.Run("rename file with allocation rename file option forbidden should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		destName := "new_" + filename
		destPath := "/" + destName

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":          allocSize,
			"forbid_rename": nil,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destname":   destName,
		}, true)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Rename failed. this options for this file is not permitted for this allocation")

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.NotContains(t, strings.Join(output, "\n"), destPath)
	})

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

			require.Error(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
		}
	})
}

func renameFileWithWallet(t *test.SystemTest, cliConfigFilename, wallet string, param map[string]interface{}) ([]string, error) {
	t.Logf("Renaming file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox rename %s --silent --wallet %s --configDir ./config --config %s",
		p,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommand(t, cmd, 3, time.Second*20)
}
