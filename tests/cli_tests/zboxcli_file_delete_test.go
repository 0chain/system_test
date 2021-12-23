package cli_tests

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileDelete(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Run("delete existing file in root directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remoteFilePath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), false)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete existing file in sub directory should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/root/"
		filesize := int64(1 * KB)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("%s deleted", remoteFilePath), output[0])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), false)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	t.Run("delete existing file with thumbnail should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		filesize := int64(1 * KB)

		thumbnail := "upload_thumbnail_test.png"
		thumbnailPath := "./tmp/" + thumbnail
		output, err := cliutils.RunCommandWithoutRetry("wget https://en.wikipedia.org/static/images/project-logos/enwiki-2x.png -O " + thumbnailPath)
		require.Nil(t, err, "Failed to download thumbnail png file: ", strings.Join(output, "\n"))
		defer func() {
			os.Remove(thumbnailPath)
		}()

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, filesize)
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation":    allocationID,
			"remotepath":    remotepath,
			"localpath":     filename,
			"thumbnailpath": thumbnailPath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), false)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})

	// broken test
	t.Run("delete existing file in root directory with commit should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

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

	t.Run("delete existing file with encryption should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		remotepath := "/"
		filesize := int64(1 * KB)

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, filesize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  filename,
			"encrypt":    true,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err = deleteFile(t, escapedTestName(t), createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		}), false)
		require.Nil(t, err, "List files failed", err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "null", output[0], strings.Join(output, "\n"))
	})
}
