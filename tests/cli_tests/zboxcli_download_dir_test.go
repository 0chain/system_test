package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

const StatusCompletedCBDD = "Status completed callback"

func TestDownloadDir(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Download Directory Should Work")
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	// Success Scenarios
	t.Run("Download Directory from Root Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/dir1"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		originalFileChecksum := generateChecksum(t, filename)

		// Delete the uploaded file, since we will be downloading it now
		err := os.Remove(filename)
		require.Nil(t, err)

		output, err := downloadDirectory(t, configPath, createParams(map[string]interface{}{
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

	// 	t.RunWithTimeout("Download Directory Concurrently Should Work for two Different Directory", 6*time.Minute, func(t *test.SystemTest) {
	// 		allocSize := int64(4096)
	// 		filesize := int64(1024)
	// 		remoteFilePaths := [2]string{"/dir1/", "/dir2/"}

	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   allocSize,
	// 			"tokens": 9,
	// 		})

	// 		fileNameOfFirstDirectory := generateFileAndUpload(t, allocationID, remoteFilePaths[0], filesize)
	// 		fileNameOfSecondDirectory := generateFileAndUpload(t, allocationID, remoteFilePaths[1], filesize)
	// 		originalFirstFileChecksum := generateChecksum(t, fileNameOfFirstDirectory)
	// 		originalSecondFileChecksum := generateChecksum(t, fileNameOfSecondDirectory)

	// 		// deleting uploaded file from /dir1 since we will be downloading it now
	// 		err := os.Remove(fileNameOfFirstDirectory)
	// 		require.Nil(t, err)

	// 		// deleting uploaded file from /dir2 since we will be downloading it now
	// 		err = os.Remove(fileNameOfSecondDirectory)
	// 		require.Nil(t, err)

	// 		var outputList [2][]string
	// 		var errorList [2]error
	// 		var wg sync.WaitGroup

	// 		fileNames := [2]string{fileNameOfFirstDirectory, fileNameOfSecondDirectory}
	// 		for index, fileName := range fileNames {
	// 			wg.Add(1)
	// 			go func(currentFileName string, currentIndex int) {
	// 				defer wg.Done()
	// 				op, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 					"allocation": allocationID,
	// 					"remotepath": remoteFilePaths[currentIndex] + filepath.Base(currentFileName),
	// 					"localpath":  "tmp/",
	// 				}), true)
	// 				errorList[currentIndex] = err
	// 				outputList[currentIndex] = op
	// 			}(fileName, index)
	// 		}

	// 		wg.Wait()

	// 		require.Nil(t, errorList[0], strings.Join(outputList[0], "\n"))
	// 		require.Len(t, outputList[0], 2)

	// 		require.Contains(t, outputList[0][1], StatusCompletedCB)
	// 		require.Contains(t, outputList[0][1], filepath.Base(fileNameOfFirstDirectory))
	// 		downloadedFileFromFirstDirectoryChecksum := generateChecksum(t, "tmp/"+filepath.Base(fileNameOfFirstDirectory))

	// 		require.Equal(t, originalFirstFileChecksum, downloadedFileFromFirstDirectoryChecksum)
	// 		require.Nil(t, errorList[1], strings.Join(outputList[1], "\n"))
	// 		require.Len(t, outputList[1], 2)

	// 		require.Contains(t, outputList[1][1], StatusCompletedCB)
	// 		require.Contains(t, outputList[1][1], filepath.Base(fileNameOfSecondDirectory))
	// 		downloadedFileFromSecondDirectoryChecksum := generateChecksum(t, "tmp/"+filepath.Base(fileNameOfSecondDirectory))
	// 		require.Equal(t, originalSecondFileChecksum, downloadedFileFromSecondDirectoryChecksum)
	// 	})

	// 	t.Run("Download Directory from a Directory Should Work", func(t *test.SystemTest) {
	// 		allocSize := int64(2048)
	// 		filesize := int64(256)
	// 		remotepath := "/dir/"

	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   allocSize,
	// 			"tokens": 9,
	// 		})

	// 		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 		originalFileChecksum := generateChecksum(t, filename)

	// 		// Delete the uploaded file, since we will be downloading it now
	// 		err := os.Remove(filename)
	// 		require.Nil(t, err)

	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"remotepath": remotepath + filepath.Base(filename),
	// 			"localpath":  "tmp/",
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[1], StatusCompletedCB)
	// 		require.Contains(t, output[1], filepath.Base(filename))

	// 		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

	// 		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	// 	})

	// 	t.Run("Download Directory from Nested Directory Should Work", func(t *test.SystemTest) {
	// 		allocSize := int64(2048)
	// 		filesize := int64(256)
	// 		remotepath := "/nested/dir/"

	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   allocSize,
	// 			"tokens": 9,
	// 		})

	// 		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 		originalFileChecksum := generateChecksum(t, filename)

	// 		// Delete the uploaded file, since we will be downloading it now
	// 		err := os.Remove(filename)
	// 		require.Nil(t, err)

	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"remotepath": remotepath + filepath.Base(filename),
	// 			"localpath":  "tmp/",
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[1], StatusCompletedCB)
	// 		require.Contains(t, output[1], filepath.Base(filename))

	// 		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

	// 		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	// 	})

	// 	t.RunWithTimeout("Download Entire Directory Should Work but does not see blobber/issues/588", 3*time.Minute, func(t *test.SystemTest) { // todo: slow
	// 		allocSize := int64(2048)
	// 		filesize := int64(256)
	// 		remotepath := "/nested/dir/"

	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   allocSize,
	// 			"tokens": 9,
	// 		})

	// 		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

	// 		// Delete the uploaded file, since we will be downloading it now
	// 		err := os.Remove(filename)
	// 		require.Nil(t, err)

	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"remotepath": remotepath,
	// 			"localpath":  "tmp/dir",
	// 		}), false)
	// 		require.Error(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 1)
	// 		require.Contains(t, output[0], "consensus_not_met")
	// 	})

	// 	t.RunWithTimeout("Download Directory From Shared Folder Should Work but does not see blobber/issues/588", 3*time.Minute, func(t *test.SystemTest) {
	// 		var authTicket, filename string

	// 		filesize := int64(10)
	// 		remotepath := "/"

	// 		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
	// 		t.Run("Share Entire Folder from Another Wallet", func(t *test.SystemTest) {
	// 			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 				"size":   10 * 1024,
	// 				"tokens": 9,
	// 			})
	// 			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)

	// 			require.NotEqual(t, "", filename)

	// 			// Delete the uploaded file from tmp folder if it exist,
	// 			// since we will be downloading it now
	// 			err := os.RemoveAll("tmp/" + filepath.Base(filename))
	// 			require.Nil(t, err)

	// 			shareParam := createParams(map[string]interface{}{
	// 				"allocation": allocationID,
	// 				"remotepath": remotepath,
	// 			})

	// 			output, err := shareFolderInAllocation(t, configPath, shareParam)
	// 			require.Nil(t, err, strings.Join(output, "\n"))
	// 			require.Len(t, output, 1)

	// 			authTicket, err = extractAuthToken(output[0])
	// 			require.Nil(t, err, "extract auth token failed")
	// 			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
	// 		})

	// 		// Just create a wallet so that we can work further
	// 		createWallet(t)

	// 		// Download file using auth-ticket: should work
	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"localpath":  "tmp/dir",
	// 			"remotepath": "/" + filename,
	// 		}), false)
	// 		require.NotNil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 1)
	// 		aggregatedOutput := strings.Join(output, " ")
	// 		require.Contains(t, aggregatedOutput, "consensus_not_met")
	// 	})

	// 	t.RunWithTimeout("Download Shared File Should Work", 5*time.Minute, func(t *test.SystemTest) { // todo: too slow
	// 		var authTicket, filename, originalFileChecksum string

	// 		filesize := int64(10)
	// 		remotepath := "/"

	// 		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
	// 		t.Run("Share Directory from Another Wallet", func(t *test.SystemTest) {
	// 			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 				"size":   10 * 1024,
	// 				"tokens": 9,
	// 			})
	// 			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 			originalFileChecksum = generateChecksum(t, filename)

	// 			require.NotEqual(t, "", filename)

	// 			// Delete the uploaded file from tmp folder if it exist,
	// 			// since we will be downloading it now
	// 			err := os.RemoveAll("tmp/" + filepath.Base(filename))
	// 			require.Nil(t, err)

	// 			shareParam := createParams(map[string]interface{}{
	// 				"allocation": allocationID,
	// 				"remotepath": remotepath + filepath.Base(filename),
	// 			})

	// 			output, err := shareFolderInAllocation(t, configPath, shareParam)
	// 			require.Nil(t, err, strings.Join(output, "\n"))
	// 			require.Len(t, output, 1)

	// 			authTicket, err = extractAuthToken(output[0])
	// 			require.Nil(t, err, "extract auth token failed")
	// 			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
	// 		})

	// 		// Just create a wallet so that we can work further
	// 		err := createWalletAndLockReadTokens(t, configPath)
	// 		require.Nil(t, err)

	// 		// Download file using auth-ticket: should work
	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"localpath":  "tmp/",
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[1], StatusCompletedCB)
	// 		require.Contains(t, output[1], filepath.Base(filename))

	// 		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

	// 		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	// 	})

	// 	t.RunWithTimeout("Download Encrypted Directory Should Work", 5*time.Minute, func(t *test.SystemTest) {
	// 		allocSize := int64(10 * MB)
	// 		filesize := int64(10)
	// 		remotepath := "/"

	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   allocSize,
	// 			"tokens": 9,
	// 		})

	// 		filename := generateRandomTestFileName(t)
	// 		err := createFileWithSize(filename, filesize)
	// 		require.Nil(t, err)
	// 		originalFileChecksum := generateChecksum(t, filename)

	// 		// Upload parameters
	// 		uploadWithParam(t, configPath, map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"localpath":  filename,
	// 			"remotepath": remotepath + filepath.Base(filename),
	// 			"encrypt":    "",
	// 		})

	// 		// Delete the uploaded file, since we will be downloading it now
	// 		err = os.Remove(filename)
	// 		require.Nil(t, err)

	// 		// Downloading encrypted file should work
	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"remotepath": remotepath + filepath.Base(filename),
	// 			"localpath":  os.TempDir(),
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)
	// 		require.Contains(t, output[len(output)-1], StatusCompletedCB)
	// 		require.Contains(t, output[len(output)-1], filepath.Base(filename))
	// 		downloadedFileChecksum := generateChecksum(t, strings.TrimSuffix(os.TempDir(), "/")+"/"+filepath.Base(filename))
	// 		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	// 	})

	// 	t.RunWithTimeout("Download Shared Encrypted Directory Should Work", 5*time.Minute, func(t *test.SystemTest) { //todo: slow
	// 		var authTicket, filename string

	// 		filesize := int64(10)
	// 		remotepath := "/"
	// 		var allocationID string

	// 		// create viewer wallet
	// 		viewerWalletName := escapedTestName(t) + "_viewer"
	// 		createWalletForNameAndLockReadTokens(t, configPath, viewerWalletName)

	// 		viewerWallet, err := getWalletForName(t, configPath, viewerWalletName)
	// 		require.Nil(t, err)
	// 		require.NotNil(t, viewerWallet)

	// 		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
	// 		t.Run("Share File from Another Wallet", func(t *test.SystemTest) {
	// 			allocationID = setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 				"size":   10 * 1024,
	// 				"tokens": 9,
	// 			})
	// 			filename = generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{
	// 				"encrypt": "",
	// 			})
	// 			require.NotEqual(t, "", filename)

	// 			// Delete the uploaded file from tmp folder if it exist,
	// 			// since we will be downloading it now
	// 			err := os.RemoveAll("tmp/" + filepath.Base(filename))
	// 			require.Nil(t, err)

	// 			shareParam := createParams(map[string]interface{}{
	// 				"allocation":          allocationID,
	// 				"remotepath":          remotepath + filepath.Base(filename),
	// 				"encryptionpublickey": viewerWallet.EncryptionPublicKey,
	// 			})

	// 			output, err := shareFolderInAllocation(t, configPath, shareParam)
	// 			require.Nil(t, err, strings.Join(output, "\n"))
	// 			require.Len(t, output, 1)

	// 			authTicket, err = extractAuthToken(output[0])
	// 			require.Nil(t, err, "extract auth token failed")
	// 			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
	// 		})

	// 		file := "tmp/" + filepath.Base(filename)

	// 		// Download file using auth-ticket: should work
	// 		output, err := downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"localpath":  file,
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[len(output)-1], StatusCompletedCB)
	// 		require.Contains(t, output[len(output)-1], filepath.Base(filename))

	// 		os.Remove(file) //nolint

	// 		// Download file using auth-ticket and lookuphash: should work
	// 		output, err = downloadFileForWallet(t, viewerWalletName, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"lookuphash": GetReferenceLookup(allocationID, remotepath+filepath.Base(filename)),
	// 			"localpath":  file,
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[len(output)-1], StatusCompletedCB)
	// 		require.Contains(t, output[len(output)-1], filepath.Base(filename))
	// 	})

	// 	t.RunWithTimeout("Download Directory From Shared Folder by Remotepath Should Work", 5*time.Minute, func(t *test.SystemTest) {
	// 		var authTicket, filename, originalFileChecksum string

	// 		filesize := int64(10)
	// 		remotepath := "/dir/"

	// 		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
	// 		t.Run("Share Directory from Another Wallet", func(t *test.SystemTest) {
	// 			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 				"size":   10 * 1024,
	// 				"tokens": 9,
	// 			})
	// 			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 			originalFileChecksum = generateChecksum(t, filename)
	// 			require.NotEqual(t, "", filename)

	// 			// Delete the uploaded file from tmp folder if it exist,
	// 			// since we will be downloading it now
	// 			err := os.RemoveAll("tmp/" + filepath.Base(filename))
	// 			require.Nil(t, err)

	// 			shareParam := createParams(map[string]interface{}{
	// 				"allocation": allocationID,
	// 				"remotepath": remotepath,
	// 			})

	// 			output, err := shareFolderInAllocation(t, configPath, shareParam)
	// 			require.Nil(t, err, strings.Join(output, "\n"))
	// 			require.Len(t, output, 1)

	// 			authTicket, err = extractAuthToken(output[0])
	// 			require.Nil(t, err, "extract auth token failed")
	// 			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
	// 		})

	// 		// Just create a wallet so that we can work further
	// 		err := createWalletAndLockReadTokens(t, configPath)
	// 		require.Nil(t, err)

	// 		// Download file using auth-ticket: should work
	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"localpath":  "tmp/",
	// 			"remotepath": remotepath + filepath.Base(filename),
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[1], StatusCompletedCB)
	// 		require.Contains(t, output[1], filepath.Base(filename))

	// 		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

	// 		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	// 	})

	// 	t.RunWithTimeout("Download Directory From Shared Folder by Lookup Hash Should Work", 5*time.Minute, func(t *test.SystemTest) {
	// 		var authTicket, lookuphash, filename, originalFileChecksum string

	// 		filesize := int64(10)
	// 		remotepath := "/dir/"

	// 		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
	// 		t.Run("Share Directory from Another Wallet", func(t *test.SystemTest) {
	// 			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 				"size":   10 * 1024,
	// 				"tokens": 9,
	// 			})
	// 			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 			originalFileChecksum = generateChecksum(t, filename)
	// 			require.NotEqual(t, "", filename)

	// 			// Delete the uploaded file from tmp folder if it exist,
	// 			// since we will be downloading it now
	// 			err := os.RemoveAll("tmp/" + filepath.Base(filename))
	// 			require.Nil(t, err)

	// 			shareParam := createParams(map[string]interface{}{
	// 				"allocation": allocationID,
	// 				"remotepath": remotepath,
	// 			})

	// 			output, err := shareFolderInAllocation(t, configPath, shareParam)
	// 			require.Nil(t, err, strings.Join(output, "\n"))
	// 			require.Len(t, output, 1)

	// 			authTicket, err = extractAuthToken(output[0])
	// 			require.Nil(t, err, "extract auth token failed")
	// 			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)

	// 			h := sha3.Sum256([]byte(fmt.Sprintf("%s:%s%s", allocationID, remotepath, filepath.Base(filename))))
	// 			lookuphash = fmt.Sprintf("%x", h)
	// 			require.NotEqual(t, "", lookuphash, "Lookup Hash: ", lookuphash)
	// 		})

	// 		// Just create a wallet so that we can work further
	// 		err := createWalletAndLockReadTokens(t, configPath)
	// 		require.Nil(t, err)

	// 		// Download file using auth-ticket: should work
	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"localpath":  "tmp/",
	// 			"lookuphash": lookuphash,
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[1], StatusCompletedCB)
	// 		require.Contains(t, output[1], filepath.Base(filename))

	// 		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

	// 		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	// 	})

	// 	t.RunWithTimeout("Download Shared Directory without Paying Should Not Work", 5*time.Minute, func(t *test.SystemTest) {
	// 		t.Skip()
	// 		var authTicket, filename string

	// 		filesize := int64(10)
	// 		remotepath := "/"

	// 		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
	// 		t.Run("Share Directory from Another Wallet", func(t *test.SystemTest) {
	// 			allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 				"size":   10 * 1024,
	// 				"tokens": 9,
	// 			})
	// 			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 			require.NotEqual(t, "", filename)

	// 			// Delete the uploaded file from tmp folder if it exist,
	// 			// since we will be downloading it now
	// 			err := os.RemoveAll("tmp/" + filepath.Base(filename))
	// 			require.Nil(t, err)

	// 			shareParam := createParams(map[string]interface{}{
	// 				"allocation": allocationID,
	// 				"remotepath": remotepath + filepath.Base(filename),
	// 			})

	// 			output, err := shareFolderInAllocation(t, configPath, shareParam)
	// 			require.Nil(t, err, strings.Join(output, "\n"))
	// 			require.Len(t, output, 1)

	// 			authTicket, err = extractAuthToken(output[0])
	// 			require.Nil(t, err, "extract auth token failed")
	// 			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
	// 		})

	// 		// Just create a wallet so that we can work further
	// 		createWallet(t)

	// 		// Download file using auth-ticket: shouldn't work
	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"localpath":  "tmp/",
	// 		}), false)
	// 		require.NotNil(t, err)
	// 		require.Greater(t, len(output), 0)
	// 		aggregatedOutput := strings.Join(output, " ")
	// 		require.Contains(t, aggregatedOutput, "pre-redeeming read marker")
	// 	})

	// 	t.RunWithTimeout("Download Shared Directory by Paying Should Work", 5*time.Minute, func(t *test.SystemTest) {
	// 		var allocationID, authTicket, filename string

	// 		filesize := int64(10)
	// 		remotepath := "/"

	// 		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
	// 		t.Run("Share Directory from Another Wallet", func(t *test.SystemTest) {
	// 			allocationID = setupAllocation(t, configPath, map[string]interface{}{
	// 				"size": 10 * 1024,
	// 				"lock": 9,
	// 			})
	// 			filename = generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 			require.NotEqual(t, "", filename)

	// 			// Delete the uploaded file from tmp folder if it exist,
	// 			// since we will be downloading it now
	// 			err := os.RemoveAll("tmp/" + filepath.Base(filename))
	// 			require.Nil(t, err)

	// 			shareParam := createParams(map[string]interface{}{
	// 				"allocation": allocationID,
	// 				"remotepath": remotepath + filepath.Base(filename),
	// 			})

	// 			output, err := shareFolderInAllocation(t, configPath, shareParam)
	// 			require.Nil(t, err, strings.Join(output, "\n"))
	// 			require.Len(t, output, 1)
	// 			authTicket, err = extractAuthToken(output[0])
	// 			require.Nil(t, err, "extract auth token failed")
	// 			require.NotEqual(t, "", authTicket, "Ticket: ", authTicket)
	// 		})

	// 		err = createWalletAndLockReadTokens(t, configPath)
	// 		require.Nil(t, err)
	// 		// Download file using auth-ticket: should work
	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"authticket": authTicket,
	// 			"localpath":  "tmp/",
	// 		}), false)

	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)
	// 		aggregatedOutput := strings.Join(output, " ")
	// 		require.Contains(t, aggregatedOutput, filepath.Base(filename))
	// 	})

	// 	t.Run("Download to Non-Existent Path Should Work", func(t *test.SystemTest) {
	// 		allocSize := int64(2048)
	// 		filesize := int64(256)
	// 		remotepath := "/"

	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   allocSize,
	// 			"tokens": 9,
	// 		})

	// 		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 		originalFileChecksum := generateChecksum(t, filename)

	// 		// Delete the uploaded file, since we will be downloading it now
	// 		err := os.Remove(filename)
	// 		require.Nil(t, err)

	// 		newLocalPath := "tmp/tmp2/" + filepath.Base(filename)
	// 		defer func() {
	// 			os.Remove(newLocalPath) //nolint: errcheck
	// 		}()

	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"remotepath": remotepath + filepath.Base(filename),
	// 			"localpath":  newLocalPath,
	// 		}), true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[1], StatusCompletedCB)
	// 		require.Contains(t, output[1], filepath.Base(filename))

	// 		downloadedFileChecksum := generateChecksum(t, newLocalPath)

	// 		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	// 	})

	// 	t.Run("Download without any Parameter Should Fail", func(t *test.SystemTest) {
	// 		createWallet(t)

	// 		output, err := downloadFile(t, configPath, "", false)
	// 		require.NotNil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 1)

	// 		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
	// 	})

	// 	t.Run("Download from Allocation without other Parameter Should Fail", func(t *test.SystemTest) {
	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   10000,
	// 			"tokens": 9,
	// 		})

	// 		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"allocation": allocationID,
	// 		}), false)

	// 		require.NotNil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 1)
	// 		require.Equal(t, "Error: remotepath / authticket flag is missing", output[0])
	// 	})

	// 	t.RunWithTimeout("Download Moved Directory Should Work", 5*time.Minute, func(t *test.SystemTest) {
	// 		allocSize := int64(2048)
	// 		filesize := int64(256)
	// 		remotepath := "/"

	// 		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
	// 			"size":   allocSize,
	// 			"tokens": 9,
	// 		})

	// 		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
	// 		originalFileChecksum := generateChecksum(t, filename)

	// 		// Delete the uploaded file, since we will be downloading it now
	// 		err := os.Remove(filename)
	// 		require.Nil(t, err)

	// 		remotepath += filepath.Base(filename)
	// 		destpath := "/child/"
	// 		output, err := moveFile(t, configPath, map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"remotepath": remotepath,
	// 			"destpath":   destpath,
	// 		}, true)
	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 1)
	// 		require.Equal(t, fmt.Sprintf(remotepath+" moved"), output[0])

	// 		defer func() {
	// 			os.Remove("tmp/" + filepath.Base(filename)) //nolint: errcheck
	// 		}()

	// 		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
	// 			"allocation": allocationID,
	// 			"remotepath": destpath + filepath.Base(filename),
	// 			"localpath":  "tmp/",
	// 		}), true)

	// 		require.Nil(t, err, strings.Join(output, "\n"))
	// 		require.Len(t, output, 2)

	// 		require.Contains(t, output[1], StatusCompletedCB)
	// 		require.Contains(t, output[1], filepath.Base(filename))

	// 		downloadedFileChecksum := generateChecksum(t, "tmp/"+filepath.Base(filename))

	//		require.Equal(t, originalFileChecksum, downloadedFileChecksum)
	//	})
}

func downloadDirectory(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return downloadDirectoryFromWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func downloadDirectoryFromWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 15*time.Second) // TODO replace with pollers
	t.Logf("Downloading file...")
	cmd := fmt.Sprintf(
		"./zbox downloaddir %s --silent --wallet %s --configDir ./config --config %s",
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
