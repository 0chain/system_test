package cli_tests

import (
	"encoding/json"
	"fmt"
	cli_model "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"testing"
)

func pretty(data interface{}) {
	bts, _ := json.MarshalIndent(data, "", "\t")
	fmt.Println(string(bts))
}

func TestUpload(t *testing.T) {

	// Scenarios //

	//upload file to root
	t.Run("Upload File to Root Directory Should Work", func(t *testing.T) {

		allocSize := int64(2048)
		fileSize := int64(256)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	// upload file to directory
	t.Run("Upload File to a Directory Should Work", func(t *testing.T) {

		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/" + filepath.Base(filename),
			"localpath":  filename,
		}))
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

		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/",
			"localpath":  filename,
		}))
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

		var listResults []cli_model.ListFileResult
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

	// upload file to nested directory
	t.Run("Upload File to Nested Directory Should Work", func(t *testing.T) {

		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/nested/dir/" + filepath.Base(filename),
			"localpath":  filename,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	// try to upload to existing file
	t.Run("Upload File to Existing File Should Fail", func(t *testing.T) {
		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		// Upload the file again to same directory
		output, err = uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected = fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

	})

	// upload with thumbnail
	t.Run("Upload File with Thumbnail Should Work", func(t *testing.T) {

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

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation":    allocationID,
			"remotepath":    "/",
			"localpath":     filename,
			"thumbnailpath": thumbnail,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	// upload encrypted
	t.Run("Upload File with Encryption Should Work", func(t *testing.T) {

	})

	// try to upload to non-existent allocation
	t.Run("Upload File to Non-Existent Allocation Should Fail", func(t *testing.T) {

		fileSize := int64(256)

		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": "ab12mn34as90",
			"remotepath": "/",
			"localpath":  filename,
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := fmt.Sprint(
			"Error fetching the allocation. allocation_fetch_error: Error fetching the allocation.consensus_failed: consensus failed on sharders",
		)
		require.Equal(t, expected, output[0])
	})

	// try to upload to someone else's allocation
	t.Run("Upload File to Other's Allocation Should Fail", func(t *testing.T) {

	})

	// try to upload a file that does not exist on the local disk
	t.Run("Upload Non-Existent File Should Fail", func(t *testing.T) {

	})

	// try to upload blank file
	t.Run("Upload Blank File Should Work", func(t *testing.T) {

	})

	// token accounting (where do tokens go on lock, file upload etc.)

}
