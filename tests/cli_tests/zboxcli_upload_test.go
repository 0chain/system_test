package cli_tests

import (
	"encoding/json"
	"fmt"
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

	})

	// upload with thumbnail
	t.Run("Upload File with Thumbnail Should Work", func(t *testing.T) {

	})

	// upload encrypted
	t.Run("Upload File with Encryption Should Work", func(t *testing.T) {

	})

	// try to upload to non-existent allocation
	t.Run("Upload File to Non-Existent Allocation Should Fail", func(t *testing.T) {

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
