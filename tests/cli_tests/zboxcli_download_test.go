package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestDownload(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	// Success Scenarios

	t.Run("Download File from Root Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download File from a Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download File from Nested Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/nested/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	//TODO: Directory download seems broken
	t.Run("Download Entire Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/nested/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/dir",
		}), false)
		require.Error(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})

	//TODO: Directory share seems broken
	t.Run("Download File From Shared Folder Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Entire Folder from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)

			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/dir",
			"remotepath": "/" + filename,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})

	t.Run("Download Entire Shared Folder Should Fail", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Entire Folder from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)

			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/dir",
			"remotepath": "/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error in file operation: please get files from folder, and download them one by one", output[0])
	})

	t.Run("Download Shared File Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			originalFileChecksum = generateChecksum(t, filename)

			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath + filepath.Base(filename),
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download Encrypted File Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(10 * MB)
		filesize := int64(10)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)
		originalFileChecksum := generateChecksum(t, filename)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath + filepath.Base(filename),
			"encrypt":    "",
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		// Downloading encrypted file should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  os.TempDir(),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[len(output)-1])
		downloadedFileChecksum := generateChecksum(t, strings.TrimSuffix(os.TempDir(), "/")+"/"+filepath.Base(filename))
		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download Shared Encrypted File Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"
		var allocationID string

		// register viewer wallet
		viewerWalletName := escapedTestName(t) + "_viewer"
		_, err = registerWalletForName(t, configPath, viewerWalletName)
		require.Nil(t, err)

		viewerWallet, err := getWalletForName(t, configPath, viewerWalletName)
		require.Nil(t, err)
		require.NotNil(t, viewerWallet)

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID = setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{
				"encrypt": "",
			})
			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation":          allocationID,
				"remotepath":          remotepath + filepath.Base(filename),
				"encryptionpublickey": viewerWallet.EncryptionPublicKey,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)

		file := "tmp/" + filepath.Base(filename)

		// Download file using auth-ticket: should work
		output, err := downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  file,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Equal(t, expected, output[len(output)-1])

		os.Remove(file) //nolint

		// Download file using auth-ticket and lookuphash: should work
		output, err = downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"lookuphash": GetReferenceLookup(allocationID, remotepath+filepath.Base(filename)),
			"localpath":  file,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Equal(t, expected, output[len(output)-1])
	})

	t.Run("Download From Shared Folder by Remotepath Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/dir/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			originalFileChecksum = generateChecksum(t, filename)
			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
			"remotepath": remotepath + filepath.Base(filename),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download From Shared Folder by Lookup Hash Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, lookuphash, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/dir/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			originalFileChecksum = generateChecksum(t, filename)
			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath,
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)

			h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
			lookuphash = fmt.Sprintf("%x", h)
			require.NotEqual(t, "", lookuphash, "Lookup Hash: ", lookuphash)
		})

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
			"lookuphash": lookuphash,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download Shared File without Paying Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			originalFileChecksum = generateChecksum(t, filename)
			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath + filepath.Base(filename),
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	//FIXME: POSSIBLE BUG: Can't download by self-paying for shared file (rx-pay)
	t.Run("Download Shared File by Paying Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 1,
			})
			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
			require.NotEqual(t, "", filename)

			// Delete the uploaded file from tmp folder if it exist,
			// since we will be downloading it now
			err := os.RemoveAll("tmp/" + filepath.Base(filename))
			require.Nil(t, err)

			shareParam := createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": remotepath + filepath.Base(filename),
			})

			output, err := shareFolderInAllocation(t, configPath, shareParam)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)

			authTicket, err = extractAuthToken(output[0])
			require.Nil(t, err, "extract auth token failed")
			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
		})

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * 1024,
			"tokens": 1,
		})

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"authticket": authTicket,
			"localpath":  "tmp/",
			"rx_pay":     "",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1])
	})

	t.Run("Download File Thumbnail Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		thumbnail := escapedTestName(t) + "thumbnail.png"
		//nolint
		thumbnailSize := generateThumbnail(t, thumbnail)

		defer func() {
			// Delete the downloaded thumbnail file
			err = os.Remove(thumbnail)
			require.Nil(t, err)
		}()

		filename := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{
			"thumbnailpath": thumbnail,
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		localPath := filepath.Join(os.TempDir(), filepath.Base(filename))

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  localPath,
			"thumbnail":  nil,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		stat, err := os.Stat(localPath)
		require.Nil(t, err)
		require.Equal(t, thumbnailSize, int(stat.Size()))
	})

	t.Run("Download to Non-Existent Path Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/tmp2/" + filepath.Base(filename),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/tmp2/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download File With Only startblock Should Work", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(filename),
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)
		var data climodel.FileStats
		for _, data = range stats {
			break
		}

		startBlock := int64(5) // blocks 5 to 10 should be downloaded
		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		info, err := os.Stat("tmp/" + filepath.Base(filename))
		require.Nil(t, err, "error getting file stats")
		// downloaded file size should equal to ratio of block downloaded by original file size
		require.Equal(t, float64(info.Size()), (float64((data.NumOfBlocks-(startBlock-1)))/float64(data.NumOfBlocks))*float64((filesize)))
	})

	t.Run("Download File With Only endblock Should Work", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(filename),
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)
		var data climodel.FileStats
		for _, data = range stats {
			break
		}

		endBlock := int64(5) // blocks 1 to 5 should be downloaded
		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"endblock":   endBlock,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		// FIXME: Type = application/octet-stream.
		expected := fmt.Sprintf(
			"Status completed callback. Type = . Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		info, err := os.Stat("tmp/" + filepath.Base(filename))
		require.Nil(t, err, "error getting file stats")
		// downloaded file size should equal to ratio of block downloaded by original file size
		// FIXME: giving only endblock should download from block 1 to that endblock.
		// Instead, empty file is downloaded.
		require.NotEqual(t, float64(info.Size()), (float64(endBlock)/float64(data.NumOfBlocks))*float64((filesize)))
	})

	t.Run("Download File With startblock And endblock Should Work", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536, we upload 20 blocks and download 1 block
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(filename),
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)
		var data climodel.FileStats
		for _, data = range stats {
			break
		}

		startBlock := 1
		endBlock := 1
		// Minimum Startblock value should be 1 (since gosdk subtracts 1 from start block, so 0 would lead to startblock being -1).
		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
			"endblock":   endBlock,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		info, err := os.Stat("tmp/" + filepath.Base(filename))
		require.Nil(t, err, "error getting file stats")
		// downloaded file size should equal to ratio of block downloaded by original file size
		require.Equal(t, float64(info.Size()), (float64((endBlock-(startBlock-1)))/float64(data.NumOfBlocks))*float64((filesize)))
	})

	t.Run("Download File With startblock 0 and non-zero endblock should fail", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		startBlock := 0
		endBlock := 5
		// Minimum Startblock value should be 1 (since gosdk subtracts 1 from start block, so 0 would lead to startblock being -1).
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
			"endblock":   endBlock,
		}), true)
		// FIXME: error should not be nil, as this is unexpected behavior.
		// An empty File is downloaded instead of first 5 blocks.
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.NotEqual(t, output[0], "invalid startblock. Please input greater than or equal to 1")
	})

	t.Run("Download File With endblock greater than number of blocks should fail", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		startBlock := 1
		endBlock := 40
		// Minimum Startblock value should be 1 (since gosdk subtracts 1 from start block, so 0 would lead to startblock being -1).
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
			"endblock":   endBlock,
		}), true)
		// FIXME: error should not be nil, as this is unexpected behavior.
		// 40 blocks cannot be downloaded as that many blocks don't exist.
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.NotEqual(t, output[0], "invalid startblock. Please input greater than or equal to 1")
	})

	t.Run("Download with endblock less than startblock should fail", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		startBlock := 6
		endBlock := 4
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
			"endblock":   endBlock,
		}), true)
		// FIXME: error should not be nil, as this is unexpected behavior.
		// An empty file is downloaded.
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.NotEqual(t, output[0], "invalid startblock. Please input greater than or equal to 1")
	})

	t.Run("Download with negative startblock should fail", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		startBlock := -6
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
		}), true)
		// FIXME: error should not be nil, as this is unexpected behavior.
		// An unexpected amount of blocks are downloaded.
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.NotEqual(t, output[0], "invalid startblock. Please input greater than or equal to 1")
	})

	t.Run("Download with negative endblock should fail", func(t *testing.T) {
		t.Parallel()

		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		endBlock := -6
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"endblock":   endBlock,
		}), true)
		// FIXME: error should not be nil, as this is unexpected behavior.
		// An empty file is downloaded.
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.NotEqual(t, output[0], "invalid startblock. Please input greater than or equal to 1")
	})

	t.Run("Download File With commit Flag Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"commit":     "",
		}), true)
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
		err = json.Unmarshal([]byte(match[1]), &commitResp)
		require.Nil(t, err)
		require.NotEmpty(t, commitResp)

		require.Equal(t, "application/octet-stream", commitResp.MetaData.MimeType)
		require.Equal(t, filesize, commitResp.MetaData.Size)
		require.Equal(t, filepath.Base(filename), commitResp.MetaData.Name)
		require.Equal(t, remotepath+filepath.Base(filename), commitResp.MetaData.Path)
		require.Equal(t, "", commitResp.MetaData.EncryptedKey)
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download File With blockspermarker Flag Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation":      allocationID,
			"remotepath":      remotepath + filepath.Base(filename),
			"localpath":       "tmp/",
			"blockspermarker": 1,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	// Failure Scenarios

	t.Run("Download File from Non-Existent Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": "12334qe",
			"remotepath": "/",
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error fetching the allocation allocation_fetch_error: Error fetching the allocation.consensus_failed: consensus failed on sharders", output[0])
	})

	t.Run("Download File from Other's Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		var otherAllocationID, otherFilename string

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		t.Run("Get Other Allocation ID", func(t *testing.T) {
			otherAllocationID = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
			otherFilename = generateFileAndUpload(t, otherAllocationID, remotepath, filesize)
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(otherFilename)
		require.Nil(t, err)

		// Download using otherAllocationID: should not work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": otherAllocationID,
			"remotepath": remotepath + filepath.Base(otherFilename),
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0)

		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[len(output)-1])
	})

	t.Run("Download Non-Existent File Should Fail", func(t *testing.T) {
		t.Parallel()

		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10000,
			"tokens": 1,
		})

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + "hello.txt",
			"localpath":  "tmp/",
		}), false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})

	t.Run("Download without any Parameter Should Fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = downloadFile(t, configPath, "", false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
	})

	t.Run("Download from Allocation without other Parameter Should Fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10000,
			"tokens": 1,
		})

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}), false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
	})

	t.Run("Download File Without read-lock Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "not enough tokens")
	})

	t.Run("Download File using Expired Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
			"expire": "1h",
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "-1h",
		})
		output, err := updateAllocation(t, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})

	t.Run("Download File to Existing File Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  os.TempDir(),
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := fmt.Sprintf(
			"Download failed. Local file already exists '%s'",
			strings.TrimSuffix(os.TempDir(), "/")+"/"+filepath.Base(filename),
		)
		require.Equal(t, expected, output[0])
	})
}

func setupAllocationAndReadLock(t *testing.T, cliConfigFilename string, extraParam map[string]interface{}) string {
	tokens := float64(1)
	if tok, ok := extraParam["tokens"]; ok {
		token, err := strconv.ParseFloat(fmt.Sprintf("%v", tok), 64)
		require.Nil(t, err)
		tokens = token
	}

	allocationID := setupAllocation(t, cliConfigFilename, extraParam)

	// Lock half the tokens for read pool
	output, err := readPoolLock(t, cliConfigFilename, createParams(map[string]interface{}{
		"allocation": allocationID,
		"tokens":     tokens / 2,
		"duration":   "10m",
	}), true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

	return allocationID
}

func downloadFile(t *testing.T, cliConfigFilename, param string, retry bool) ([]string, error) {
	return downloadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func downloadFileForWallet(t *testing.T, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 15*time.Second) // TODO replace with pollers
	t.Logf("Downloading file...")
	cmd := fmt.Sprintf(
		"./zbox download %s --silent --wallet %s --configDir ./config --config %s",
		param,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func generateChecksum(t *testing.T, filePath string) string {
	t.Logf("Generating checksum for file [%v]...", filePath)

	output, err := cliutils.RunCommandWithoutRetry("shasum -a 256 " + filePath)
	require.Nil(t, err, "Checksum generation for file %v failed", filePath, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)

	matcher := regexp.MustCompile("(.*) " + filePath + "$")
	require.Regexp(t, matcher, output[0], "Checksum execution output did not match expected", strings.Join(output, "\n"))

	return matcher.FindAllStringSubmatch(output[0], 1)[0][1]
}
