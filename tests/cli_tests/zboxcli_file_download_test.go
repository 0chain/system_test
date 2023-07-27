package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"golang.org/x/crypto/sha3"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const StatusCompletedCB = "Status completed callback"

func TestDownload(testSetup *testing.T) {
	//todo: too mnay test cases are slow in here
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Download File from Root Directory Should Work")
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	// Success Scenarios
	t.Run("Download File from Root Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.RunWithTimeout("Download File Concurrently Should Work from two Different Directory", 6*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(4096)
		filesize := int64(1024)
		remoteFilePaths := [2]string{"/dir1/", "/dir2/"}

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		fileNameOfFirstDirectory := generateFileAndUpload(t, allocationID, remoteFilePaths[0], filesize)
		fileNameOfSecondDirectory := generateFileAndUpload(t, allocationID, remoteFilePaths[1], filesize)
		originalFirstFileChecksum := generateChecksum(t, fileNameOfFirstDirectory)
		originalSecondFileChecksum := generateChecksum(t, fileNameOfSecondDirectory)

		// deleting uploaded file from /dir1 since we will be downloading it now
		err := os.Remove(fileNameOfFirstDirectory)
		require.Nil(t, err)

		// deleting uploaded file from /dir2 since we will be downloading it now
		err = os.Remove(fileNameOfSecondDirectory)
		require.Nil(t, err)

		var outputList [2][]string
		var errorList [2]error
		var wg sync.WaitGroup

		fileNames := [2]string{fileNameOfFirstDirectory, fileNameOfSecondDirectory}
		for index, fileName := range fileNames {
			wg.Add(1)
			go func(currentFileName string, currentIndex int) {
				defer wg.Done()
				op, err := downloadFile(t, configPath, createParams(map[string]interface{}{
					"allocation": allocationID,
					"remotepath": remoteFilePaths[currentIndex] + filepath.Base(currentFileName),
					"localpath":  "tmp/",
				}), true)
				errorList[currentIndex] = err
				outputList[currentIndex] = op
			}(fileName, index)
		}

		wg.Wait()

		require.Nil(t, errorList[0], strings.Join(outputList[0], "\n"))
		require.Len(t, outputList[0], 2)

		require.Contains(t, outputList[0][1], StatusCompletedCB)
		require.Contains(t, outputList[0][1], filepath.Base(fileNameOfFirstDirectory))
		downloadedFileFromFirstDirectoryChecksum := generateChecksum(t, "tmp/"+filepath.Base(fileNameOfFirstDirectory))

		require.Equal(t, originalFirstFileChecksum, downloadedFileFromFirstDirectoryChecksum)
		require.Nil(t, errorList[1], strings.Join(outputList[1], "\n"))
		require.Len(t, outputList[1], 2)

		require.Contains(t, outputList[1][1], StatusCompletedCB)
		require.Contains(t, outputList[1][1], filepath.Base(fileNameOfSecondDirectory))
		downloadedFileFromSecondDirectoryChecksum := generateChecksum(t, "tmp/"+filepath.Base(fileNameOfSecondDirectory))
		require.Equal(t, originalSecondFileChecksum, downloadedFileFromSecondDirectoryChecksum)
	})

	t.Run("Download File from a Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.Run("Download File from Nested Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/nested/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	//TODO: Directory download seems broken see https://github.com/0chain/blobber/issues/588
	t.RunWithTimeout("Download Entire Directory Should Work but does not see blobber/issues/588", 3*time.Minute, func(t *test.SystemTest) { // todo: slow
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/nested/dir/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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
		require.Contains(t, output[0], "consensus_not_met")
	})

	//TODO: Directory share seems broken see https://github.com/0chain/blobber/issues/588
	t.RunWithTimeout("Download File From Shared Folder Should Work but does not see blobber/issues/588", 3*time.Minute, func(t *test.SystemTest) {
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share Entire Folder from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
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

		// Just create a wallet so that we can work further
		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/dir",
			"remotepath": "/" + filename,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "consensus_not_met")
	})

	t.RunWithTimeout("Download Shared File Should Work", 5*time.Minute, func(t *test.SystemTest) { // todo: too slow
		var authTicket, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
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

		// Just create a wallet so that we can work further
		err := createWalletAndLockReadTokens(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.RunWithTimeout("Download Encrypted File Should Work", 5*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(10 * MB)
		filesize := int64(10)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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
		require.Contains(t, output[len(output)-1], StatusCompletedCB)
		require.Contains(t, output[len(output)-1], filepath.Base(filename))
		downloadedFileChecksum := generateChecksum(t, strings.TrimSuffix(os.TempDir(), "/")+"/"+filepath.Base(filename))
		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.RunWithTimeout("Download Shared Encrypted File Should Work", 5*time.Minute, func(t *test.SystemTest) { //todo: slow
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"
		var allocationID string

		// create viewer wallet
		viewerWalletName := escapedTestName(t) + "_viewer"
		createWalletForNameAndLockReadTokens(t, configPath, viewerWalletName)

		viewerWallet, err := getWalletForName(t, configPath, viewerWalletName)
		require.Nil(t, err)
		require.NotNil(t, viewerWallet)

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID = setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
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

		file := "tmp/" + filepath.Base(filename)

		// Download file using auth-ticket: should work
		output, err := downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  file,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[len(output)-1], StatusCompletedCB)
		require.Contains(t, output[len(output)-1], filepath.Base(filename))

		os.Remove(file) //nolint

		// Download file using auth-ticket and lookuphash: should work
		output, err = downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"lookuphash": GetReferenceLookup(allocationID, remotepath+filepath.Base(filename)),
			"localpath":  file,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[len(output)-1], StatusCompletedCB)
		require.Contains(t, output[len(output)-1], filepath.Base(filename))
	})

	t.RunWithTimeout("Download From Shared Folder by Remotepath Should Work", 5*time.Minute, func(t *test.SystemTest) {
		var authTicket, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/dir/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
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

		// Just create a wallet so that we can work further
		err := createWalletAndLockReadTokens(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
			"remotepath": remotepath + filepath.Base(filename),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.RunWithTimeout("Download From Shared Folder by Lookup Hash Should Work", 5*time.Minute, func(t *test.SystemTest) {
		var authTicket, lookuphash, filename, originalFileChecksum string

		filesize := int64(10)
		remotepath := "/dir/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
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

		// Just create a wallet so that we can work further
		err := createWalletAndLockReadTokens(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
			"lookuphash": lookuphash,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	t.RunWithTimeout("Download Shared File without Paying Should Not Work", 5*time.Minute, func(t *test.SystemTest) {
		var authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
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

		// Just create a wallet so that we can work further
		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		// Download file using auth-ticket: shouldn't work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err)
		require.Greater(t, len(output), 0)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "pre-redeeming read marker")
	})

	t.RunWithTimeout("Download Shared File by Paying Should Work", 5*time.Minute, func(t *test.SystemTest) {
		var allocationID, authTicket, filename string

		filesize := int64(10)
		remotepath := "/"

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
			allocationID = setupAllocation(t, configPath, map[string]interface{}{
				"size":   10 * 1024,
				"tokens": 9,
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

		err = createWalletAndLockReadTokens(t, configPath)
		require.Nil(t, err)
		// Download file using auth-ticket: should work
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"authticket": authTicket,
			"localpath":  "tmp/",
		}), false)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, filepath.Base(filename))
	})

	t.Run("Download File Thumbnail Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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
		os.Remove(thumbnail) // nolint: errcheck
		err = os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  thumbnail,
			"thumbnail":  nil,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		stat, err := os.Stat(thumbnail)
		require.Nil(t, err)
		require.Equal(t, thumbnailSize, int(stat.Size()))
	})

	t.Run("Download Encrypted File Thumbnail Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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
			"encrypt":       "",
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		os.Remove(thumbnail) // nolint: errcheck
		require.Nil(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  thumbnail,
			"thumbnail":  nil,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		stat, err := os.Stat(thumbnail)
		require.Nil(t, err)
		require.Equal(t, thumbnailSize, int(stat.Size()))
	})

	t.Run("Download to Non-Existent Path Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		newLocalPath := "tmp/tmp2/" + filepath.Base(filename)
		defer func() {
			os.Remove(newLocalPath) //nolint: errcheck
		}()

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  newLocalPath,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, newLocalPath)

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})
	t.Run("Download File With Only startblock Should Work", func(t *test.SystemTest) {
		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		info, err := os.Stat("tmp/" + filepath.Base(filename))
		require.Nil(t, err, "error getting file stats")
		// downloaded file size should equal to ratio of block downloaded by original file size
		require.Equal(t, float64(info.Size()), (float64(data.NumOfBlocks-(startBlock-1))/float64(data.NumOfBlocks))*float64(filesize))
	})

	t.Run("Download File With Only endblock Should Work", func(t *test.SystemTest) {
		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		endBlock := int64(5)
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"endblock":   endBlock,
		}), false)

		require.NoError(t, err)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, StatusCompletedCB)
		require.Contains(t, aggregatedOutput, filepath.Base(filename))
	})

	t.Run("Download File With startblock And endblock Should Work", func(t *test.SystemTest) {
		// 1 block is of size 65536, we upload 20 blocks and download 1 block
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		info, err := os.Stat("tmp/" + filepath.Base(filename))
		require.Nil(t, err, "error getting file stats")
		// downloaded file size should equal to ratio of block downloaded by original file size
		require.Equal(t, float64(info.Size()), (float64(endBlock-(startBlock-1))/float64(data.NumOfBlocks))*float64(filesize))
	})

	t.RunWithTimeout("Download File With startblock 0 and non-zero endblock should fail", 5*time.Minute, func(t *test.SystemTest) { //todo: too slow
		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		startBlock := 0
		endBlock := 5
		// Minimum Startblock value should be 1 (since gosdk subtracts 1 from start block, so 1 would lead to startblock being 0).
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"startblock": startBlock,
			"endblock":   endBlock,
		}), true)

		require.Error(t, err)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "Error: start block should not be less than 1")
	})

	t.Run("Download File With endblock greater than number of blocks should work", func(t *test.SystemTest) {
		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(10240)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "Status completed callback.")
	})

	t.Run("Download with endblock less than startblock should fail", func(t *test.SystemTest) {
		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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
			"localpath":  "/tmp",
			"startblock": startBlock,
			"endblock":   endBlock,
		}), false)

		require.NotNil(t, err)
		require.Len(t, output, 1)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "start block should be less than end block")
	})

	t.RunWithTimeout("Download with negative startblock should fail", 5*time.Minute, func(t *test.SystemTest) { //todo: too slow
		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.NotNil(t, err, strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "start block should not be less than 1")
	})

	t.Run("Download with negative endblock should fail", func(t *test.SystemTest) {
		// 1 block is of size 65536
		allocSize := int64(655360 * 4)
		filesize := int64(655360 * 2)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		endBlock := -6
		startBlock := 1
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
			"endblock":   endBlock,
			"startblock": startBlock,
		}), false)

		require.NotNil(t, err)
		require.Len(t, output, 1)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "start block or end block or both cannot be negative.")
	})

	t.Run("Download File With blockspermarker Flag Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
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

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})

	// Failure Scenarios

	t.Run("Download File from Non-Existent Allocation Should Fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": "12334qe",
			"remotepath": "/",
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error fetching the allocation allocation_fetch_error: "+
			"Error fetching the allocation.internal_error: can't get allocation: error retrieving allocation: 12334qe, error: record not found", output[0])
	})

	t.Run("Download File from Other's Allocation Should Fail", func(t *test.SystemTest) {
		var otherAllocationID, otherFilename string

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		t.Run("Get Other Allocation ID", func(t *test.SystemTest) {
			otherAllocationID = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
			otherFilename = generateFileAndUpload(t, otherAllocationID, remotepath, filesize)
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(otherFilename)
		require.NoError(t, err)

		// Download using otherAllocationID: should not work
		_, err = createWallet(t, configPath)
		require.NoError(t, err)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": otherAllocationID,
			"remotepath": remotepath + filepath.Base(otherFilename),
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0)

		require.Contains(t, output[len(output)-1], "consensus_not_met")
	})

	t.Run("Download Non-Existent File Should Fail", func(t *test.SystemTest) {
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10000,
			"tokens": 9,
		})

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + "hello.txt",
			"localpath":  "tmp/",
		}), false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "consensus_not_met")
	})

	t.Run("Download without any Parameter Should Fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = downloadFile(t, configPath, "", false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
	})

	t.Run("Download from Allocation without other Parameter Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10000,
			"tokens": 9,
		})

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}), false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
	})

	t.Run("Download File Without read-lock Should Fail", func(t *test.SystemTest) {
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
		require.Greater(t, len(output), 0)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "pre-redeeming read marker")
	})

	t.Run("Download File to Existing File Should Fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := "Error in file operation: Error while calculating shard params. Error: exceeded_max_offset_value: file is already downloaded"
		require.Equal(t, expected, output[0])
	})

	t.RunWithTimeout("Download Moved File Should Work", 5*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		remotepath += filepath.Base(filename)
		destpath := "/child/"
		output, err := moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotepath+" moved"), output[0])

		defer func() {
			os.Remove("tmp/" + filepath.Base(filename)) //nolint: errcheck
		}()

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": destpath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))

		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	})
}

func setupAllocationAndReadLock(t *test.SystemTest, cliConfigFilename string, extraParam map[string]interface{}) string {
	tokens := float64(1)
	if tok, ok := extraParam["tokens"]; ok {
		token, err := strconv.ParseFloat(fmt.Sprintf("%v", tok), 64)
		require.Nil(t, err)
		tokens = token
	}

	allocationID := setupAllocation(t, cliConfigFilename, extraParam)

	// Lock half the tokens for read pool
	readPoolParams := createParams(map[string]interface{}{
		"tokens": tokens / 3,
	})
	output, err := readPoolLock(t, cliConfigFilename, readPoolParams, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

	return allocationID
}

func downloadFile(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return downloadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func downloadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
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

func generateChecksum(t *test.SystemTest, filePath string) string {
	t.Logf("Generating checksum for file [%v]...", filePath)

	output, err := cliutils.RunCommandWithoutRetry("shasum -a 256 " + filePath)
	require.Nil(t, err, "Checksum generation for file %v failed", filePath, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)

	matcher := regexp.MustCompile("(.*) " + filePath + "$")
	require.Regexp(t, matcher, output[0], "Checksum execution output did not match expected", strings.Join(output, "\n"))

	return matcher.FindAllStringSubmatch(output[0], 1)[0][1]
}
