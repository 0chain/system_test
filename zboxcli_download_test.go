package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
			"tokens": 9.9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Download File from a Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9.9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Download File from Nested Directory Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/nested/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9.9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	//FIXME: POSSIBLE BUG: Can't download encrypted file as owner
	t.Run("Download Encrypted File Should Work", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(10 * 1024 * 1024)
		filesize := int64(10)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9.9,
		})

		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

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

		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1])
	})

	t.Run("Download Shared File Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9.9,
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

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Download From Shared Folder by Remotepath Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/dir/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9.9,
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
		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
			"remotepath": remotepath + filepath.Base(filename),
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Download From Shared Folder by Lookup Hash Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, lookuphash, filename string

		filesize := int64(10)
		remotepath := "/dir/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9.9,
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

			h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
			lookuphash = fmt.Sprintf("%x", h)
			require.NotEqual(t, "", lookuphash, "Lookup Hash: ", lookuphash)
		})

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
			"lookuphash": lookuphash,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Download Shared File without Paying Should Work", func(t *testing.T) {
		t.Parallel()

		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *testing.T) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9.9,
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

		// Just register a wallet so that we can work further
		_, err := registerWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
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
				"tokens": 9.9,
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
			"tokens": 9.9,
		})

		// Download file using auth-ticket: should work
		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"authticket": authTicket,
			"localpath":  "tmp/",
			"rx_pay":     "",
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1])
	})

	// Failure Scenarios

	t.Run("Download File to Existing File Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9.9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := fmt.Sprintf(
			"Download failed. Local file already exists '%s'",
			"tmp/"+filepath.Base(filename),
		)
		require.Equal(t, expected, output[0])
	})

	t.Run("Download File from Non-Existent Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": "12334qe",
			"remotepath": "/",
			"localpath":  "tmp/",
		}))
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

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		t.Run("Get Other Allocation ID", func(t *testing.T) {
			otherAllocationID = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
			otherFilename = generateFileAndUpload(t, otherAllocationID, remotepath, filesize)
		})

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		// Download using allocationID: should work
		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(otherFilename)
		require.Nil(t, err)

		// Download using otherAllocationID: should not work
		output, err = downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": otherAllocationID,
			"remotepath": remotepath + filepath.Base(otherFilename),
			"localpath":  "tmp/",
		}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1])
	})

	t.Run("Download Non-Existent File Should Fail", func(t *testing.T) {
		t.Parallel()

		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10000,
			"tokens": 9.9,
		})

		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + "hello.txt",
			"localpath":  "tmp/",
		}))

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})

	t.Run("Download without any Parameter Should Fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = downloadWithParam(t, configPath, "")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
	})

	t.Run("Download from Allocation without other Parameter Should Fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10000,
			"tokens": 9.9,
		})

		output, err := downloadWithParam(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
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
	output, err := readPoolLock(t, cliConfigFilename, allocationID, tokens/2)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

	return allocationID
}

func downloadWithParam(t *testing.T, cliConfigFilename, param string) ([]string, error) {
	t.Logf("Downloading file...")
	time.Sleep(15 * time.Second) // TODO replace with pollers
	cmd := fmt.Sprintf(
		"./zbox download %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommandWithRetry(t, cmd, 3, time.Second*20)
}
