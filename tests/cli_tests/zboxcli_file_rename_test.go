package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
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

func TestFileRename(t *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t.Parallel()

	t.Run("rename file", func(t *testing.T) {
		t.Parallel()

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
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("File Rename - Users should not be charged for renaming a file", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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

	t.Run("rename file to same filename (no change)", func(t *testing.T) {
		t.Parallel()

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

		// check if file is still there
		found := false
		for _, f := range files {
			if f.Path == remotePath { // nolint:gocritic // this is better than inverted if cond
				found = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, found, "file not found: ", strings.Join(output, "\n"))
	})

	t.Run("rename file to with 90-char (below 100-char filename limit)", func(t *testing.T) {
		t.Parallel()

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
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
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

	t.Run("rename file to with 110-char (above 100-char filename limit) should fail", func(t *testing.T) {
		t.Parallel()

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
		//  FIXME: Should be `Renamed failed see https://github.com/0chain/zboxcli/issues/240`
		require.Equal(t, "Delete failed: Commit consensus failed", output[0])

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
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
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

	t.Run("rename file to containing special characters", func(t *testing.T) {
		t.Parallel()

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
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("rename file with commit param", func(t *testing.T) {
		t.Parallel()

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
			"commit":     "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)
		require.Equal(t, fmt.Sprintf(remotePath+" renamed"), output[0])

		match := reCommitResponse.FindStringSubmatch(output[2])
		require.Len(t, match, 2)

		var commitResp climodel.CommitResponse
		err = json.Unmarshal([]byte(match[1]), &commitResp)
		require.Nil(t, err)
		require.NotEmpty(t, commitResp)

		require.Equal(t, "application/octet-stream", commitResp.MetaData.MimeType)
		require.Equal(t, fileSize, commitResp.MetaData.Size)
		require.Equal(t, filepath.Base(filename), commitResp.MetaData.Name)
		require.Equal(t, remotePath, commitResp.MetaData.Path)
		require.Equal(t, "", commitResp.MetaData.EncryptedKey)

		// verify commit txn
		output, err = verifyTransaction(t, configPath, commitResp.TxnID)
		require.Nil(t, err, "Could not verify commit transaction", strings.Join(output, "\n"))
		require.Len(t, output, 3)
		require.Equal(t, "Transaction verification success", output[0])
		require.Equal(t, "TransactionStatus: 1", output[1])
		require.Greater(t, len(output[2]), 0, output[2])
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
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("rename non-existing file should fail", func(t *testing.T) {
		t.Parallel()

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
		require.Equal(t, "Rename failed: Rename request failed. Operation failed.", output[0])
	})

	t.Run("rename file from someone else's allocation should fail", func(t *testing.T) {
		t.Parallel()

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err := registerWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Rename failed: Rename request failed. Operation failed.", output[0])

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
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
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

	t.Run("rename file with no allocation param should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs on rename
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = renameFile(t, configPath, map[string]interface{}{
			"remotepath": "/abc.txt",
			"destname":   "def.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("rename file with no remotepath param should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs on rename
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"destname":   "def.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath flag is missing", output[0])
	})

	t.Run("rename file with no destname param should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs on rename
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"remotepath": "/abc.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: destname flag is missing", output[0])
	})
}

func renameFileWithWallet(t *testing.T, cliConfigFilename, wallet string, param map[string]interface{}) ([]string, error) {
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
