package cli_tests

import (
	"encoding/json"
	"fmt"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

var reCommitResponse = regexp.MustCompile(`^Commit Metadata successful, Response : (.*)$`)

func TestUpload(t *testing.T) {
	t.Parallel()

	// Success Scenarios

	t.Run("Upload File to Root Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(256)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload File to a Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/" + filepath.Base(filename),
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	//FIXME: POSSIBLE BUG: Uploading file to a remote directory without
	// filename causes the file to be renamed to directory's name and upload to root
	t.Run("Upload File to a Directory without Filename Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/",
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprint(
			"Status completed callback. Type = application/octet-stream. Name = dir",
		)
		require.Equal(t, expected, output[1])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/",
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var listResults []climodel.ListFileResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
		require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

		require.Len(t, listResults, 1)
		result := listResults[0]

		require.Equal(t, "dir", result.Name)
		require.Equal(t, "/dir", result.Path)
		require.Equal(t, fileSize, result.ActualSize)
		require.Equal(t, "f", result.Type)
		require.Equal(t, "", result.EncryptionKey)
	})

	t.Run("Upload File to Nested Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/nested/dir/" + filepath.Base(filename),
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload File with Thumbnail Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(10 * 1024 * 1024)
		fileSize := int64(256)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		thumbnail, err := filepath.Abs("../../internal/dummy_file/0.png")
		require.Nil(t, err, thumbnail)

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":    allocationID,
			"remotepath":    "/",
			"localpath":     filename,
			"thumbnailpath": thumbnail,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload Image File Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(10 * 1024 * 1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename, err := filepath.Abs("../../internal/dummy_file/0.png")
		require.Nil(t, err, filename)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = image/png. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload Video File Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(100 * 1024 * 1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9.9,
		})

		filename, err := filepath.Abs("../../internal/dummy_file/0Chain.mp4")
		require.Nil(t, err, filename)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = video/mp4. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload File with Encryption Should Work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, 10)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": "/",
			"encrypt":    "",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload File with Commit Should Work", func(t *testing.T) {
		t.Parallel()

		filesize := int64(1024)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 2048,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/" + filepath.Base(filename),
			"localpath":  filename,
			"commit":     "",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		match := reCommitResponse.FindStringSubmatch(output[2])
		require.Len(t, match, 2)

		var commitResp climodel.CommitResponse
		err = json.NewDecoder(strings.NewReader(match[1])).Decode(&commitResp)
		require.Nil(t, err)

		require.Equal(t, "application/octet-stream", commitResp.MetaData.MimeType)
		require.Equal(t, filesize, commitResp.MetaData.Size)
		require.Equal(t, filepath.Base(filename), commitResp.MetaData.Name)
		require.Equal(t, "/dir/"+filepath.Base(filename), commitResp.MetaData.Path)
		require.Equal(t, "", commitResp.MetaData.EncryptedKey)
	})

	t.Run("Upload Encrypted File with Commit Should Work", func(t *testing.T) {
		t.Parallel()

		filesize := int64(10)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 100000,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/" + filepath.Base(filename),
			"localpath":  filename,
			"commit":     "",
			"encrypt":    "",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		match := reCommitResponse.FindStringSubmatch(output[2])
		require.Len(t, match, 2)

		var commitResp climodel.CommitResponse
		err = json.NewDecoder(strings.NewReader(match[1])).Decode(&commitResp)
		require.Nil(t, err)

		require.Equal(t, "application/octet-stream", commitResp.MetaData.MimeType)
		require.Equal(t, filesize, commitResp.MetaData.Size)
		require.Equal(t, filepath.Base(filename), commitResp.MetaData.Name)
		require.Equal(t, "/dir/"+filepath.Base(filename), commitResp.MetaData.Path)
		require.NotEqual(t, "", commitResp.MetaData.EncryptedKey)
	})

	// Failure Scenarios

	t.Run("Upload File to Existing File Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		// Upload the file again to same directory
		output, err = uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		require.Equal(t, "Error in file operation: Upload failed: Consensus_rate:NaN, expected:10.000000", output[1])
		require.Equal(t, "Upload failed. Upload failed: Consensus_rate:NaN, expected:10.000000", output[2])
	})

	t.Run("Upload File to Non-Existent Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		fileSize := int64(256)

		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": "ab12mn34as90",
			"remotepath": "/",
			"localpath":  filename,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := fmt.Sprint(
			"Error fetching the allocation. allocation_fetch_error: Error fetching the allocation.consensus_failed: consensus failed on sharders",
		)
		require.Equal(t, expected, output[0])
	})

	t.Run("Upload File to Other's Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		var otherAllocationID string

		allocSize := int64(2048)
		fileSize := int64(256)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		t.Run("Get Other Allocation ID", func(t *testing.T) {
			otherAllocationID = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
		})

		// Upload using allocationID: should work
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		// Upload using otherAllocationID: should not work
		output, err = uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": otherAllocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		require.Equal(t, "Error in file operation: Upload failed: Consensus_rate:NaN, expected:10.000000", output[1])
		require.Equal(t, "Upload failed. Upload failed: Consensus_rate:NaN, expected:10.000000", output[2])
	})

	t.Run("Upload Non-Existent File Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		filename := "non-existent-file.txt"

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  "non-existent-file.txt",
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := fmt.Sprintf(
			"Upload failed. open %s: no such file or directory",
			filename,
		)
		require.Equal(t, expected, output[0])
	})

	t.Run("Upload Blank File Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		fileSize := int64(0)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Upload failed. EOF", output[0])
	})

	t.Run("Upload without any Parameter Should Fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = uploadFileWithoutRetry(t, configPath, nil)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("Upload to Allocation without remotepath and authticket Should Fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 2048,
		})

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
		})

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: remotepath flag is missing", output[0])
	})
}

func uploadWithParam(t *testing.T, cliConfigFilename string, param map[string]interface{}) {
	filename, ok := param["localpath"].(string)
	require.True(t, ok)

	output, err := uploadFile(t, cliConfigFilename, param)
	require.Nil(t, err, "Upload file failed due to error ", err, strings.Join(output, "\n"))

	require.Len(t, output, 2)

	expected := fmt.Sprintf(
		"Status completed callback. Type = application/octet-stream. Name = %s",
		filepath.Base(filename),
	)
	require.Equal(t, expected, output[1])
}

func uploadFile(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithRetry(cmd, 3, time.Second*20)
}

func uploadFileWithoutRetry(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommand(cmd)
}

func generateFileAndUpload(t *testing.T, allocationID, remotepath string, size int64) string {
	filename := generateRandomTestFileName(t)

	err := createFileWithSize(filename, size)
	require.Nil(t, err)

	// Upload parameters
	uploadWithParam(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"localpath":  filename,
		"remotepath": remotepath + filepath.Base(filename),
	})

	return filename
}
