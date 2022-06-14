package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileMove(t *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t.Parallel()

	t.Run("move file to existing directory", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destpath := "/"

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

		// move file
		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" moved"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been moved
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("File move - Users should not be charged for moving a file ", func(t *testing.T) {
		t.Skip("re-do till https://github.com/0chain/0chain/pull/1373 merges")
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
		cliutils.Wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Move file
		remotepath := filepath.Base(localpath)
		moveAllocationFile(t, allocationID, remotepath, "newDir")

		cliutils.Wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		// Get final write pool, no deduction should have been done
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Equal(t, initialWritePool[0].Balance, finalWritePool[0].Balance, "Write pool balance expected to be unchanged")

		for i := 0; i < len(finalWritePool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
			t.Logf("Initital blobber[%v] balance: [%v], final balance: [%v]", i, initialWritePool[0].Blobber[i].Balance, finalWritePool[0].Blobber[i].Balance)
			require.Equal(t, finalWritePool[0].Blobber[i].Balance, initialWritePool[0].Blobber[i].Balance, epsilon)
		}
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("move file to non-existing directory should work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		destpath := "/child/"

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

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" moved"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if expected file has been moved
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("move file to same directory (no change) should fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		destpath := "/"

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

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, false)
		require.Nil(t, err, strings.Join(output, "\n")) // FIXME zbox move should throw non-zero code see https://github.com/0chain/zboxcli/issues/251
		require.Len(t, output, 1)
		// FIXME: Error message is incorrect. Should be `Move failed https://github.com/0chain/zboxcli/issues/240`
		require.Equal(t, "Copy failed: Copy request failed. Operation failed.", output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 1)

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

	t.Run("move file to another directory with existing file with same name should fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		existingFileInDest := generateRandomTestFileName(t)
		err = createFileWithSize(existingFileInDest, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		remotePathAtDest := "/target/" + filename
		destpath := "/target/"

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

		// upload file to another directory with same name.
		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePathAtDest,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected = fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, false)
		require.Nil(t, err, strings.Join(output, "\n")) // FIXME zbox move should throw non-zero code see https://github.com/0chain/zboxcli/issues/251
		require.Len(t, output, 1)
		// FIXME: Error message is incorrect. Should be `Move failed`
		require.Equal(t, "Copy failed: Copy request failed. Operation failed.", output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 3)

		// check if both existing files are there
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
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("move file with commit param", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destpath := "/"

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

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
			"commit":     "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)
		require.Equal(t, fmt.Sprintf(remotePath+" moved"), output[0])

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

		// check if expected file has been moved
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Greater(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.False(t, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("move non-existing file should fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		// try to move a file
		output, err := moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/child/nonexisting.txt",
			"destpath":   "/",
		}, false)
		require.Nil(t, err, strings.Join(output, "\n")) // FIXME zbox move should throw non-zero code see https://github.com/0chain/zboxcli/issues/251
		require.Len(t, output, 1)
		// FIXME: Error message is incorrect. Should be `Move failed`
		require.Equal(t, "Copy failed: Copy request failed. Operation failed.", output[0])
	})

	t.Run("move file from someone else's allocation should fail", func(t *testing.T) {
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
		destpath := "/"

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

		output, err = moveFileWithWallet(t, nonAllocOwnerWallet, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n")) // FIXME zbox move should throw non-zero code see https://github.com/0chain/zboxcli/issues/251
		require.Len(t, output, 1)
		// FIXME: Error message is incorrect. Should be `Move failed`
		require.Equal(t, "Copy failed: Copy request failed. Operation failed.", output[0])

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
			if f.Path == destpath+filename {
				foundAtDest = true
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.False(t, foundAtDest, "file is found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("move file with no allocation param should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs on move
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = moveFile(t, configPath, map[string]interface{}{
			"remotepath": "/abc.txt",
			"destpath":   "/child/",
		}, false)
		require.Nil(t, err, strings.Join(output, "\n")) // FIXME zbox move should throw non-zero code see https://github.com/0chain/zboxcli/issues/251
		require.Len(t, output, 1)
		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("move file with no remotepath param should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs on move
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"destpath":   "/",
		}, false)
		require.Nil(t, err, strings.Join(output, "\n")) // FIXME zbox move should throw non-zero code see https://github.com/0chain/zboxcli/issues/251
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath flag is missing", output[0])
	})

	t.Run("move file with no destpath param should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs on move
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"remotepath": "/abc.txt",
		}, false)
		require.Nil(t, err, strings.Join(output, "\n")) // FIXME zbox move should throw non-zero code see https://github.com/0chain/zboxcli/issues/251
		require.Len(t, output, 1)
		require.Equal(t, "Error: destpath flag is missing", output[0])
	})
}

func moveFile(t *testing.T, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return moveFileWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func moveFileWithWallet(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Moving file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox move %s --silent --wallet %s --configDir ./config --config %s",
		p,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
