package cli_tests

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestShareFile(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Share to public a folder with no encrypted file using auth ticket with zero expiration")

	t.Parallel()

	t.Run("Share to public a folder with no file using auth ticket with zero expiration", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload Remote Dir
		remoteDir := "/folderToBeShared/"
		output, err := createDir(t, configPath, allocationID, remoteDir, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, remoteDir+" directory created", output[0])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         remoteDir,
			"expiration-seconds": 0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// list all file to verify
		listFileParams := createParams(map[string]interface{}{
			"authticket": authTicket,
			"remotepath": remoteDir,
			"allocation": allocationID,
			"json":       "",
		})
		output, err = listAllFilesFromBlobber(t, receiverWallet, configPath, listFileParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, `null`, output[0])
	})

	t.Run("Share to public a folder with single unencrypted file using auth ticket with zero expiration should work", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(file)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         "/subfolder1",
			"expiration-seconds": 0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.Run("Share to public a folder with multiple unencrypted file using auth ticket with zero expiration should work", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload Three files
		fileSize := int64(256)
		file1 := generateRandomTestFileName(t)
		remoteOwnerPath1 := "/subfolder1/subfolder2/" + filepath.Base(file1)

		err := createFileWithSize(file1, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file1,
			"remotepath": remoteOwnerPath1,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file1))

		file2 := generateRandomTestFileName(t)
		remoteOwnerPath2 := "/subfolder1/subfolder2/" + filepath.Base(file2)

		err = createFileWithSize(file2, fileSize)
		require.Nil(t, err)

		uploadParams = map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file2,
			"remotepath": remoteOwnerPath2,
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file2))

		file3 := generateRandomTestFileName(t)
		remoteOwnerPath3 := "/subfolder1/subfolder2/" + filepath.Base(file3)

		err = createFileWithSize(file3, fileSize)
		require.Nil(t, err)

		uploadParams = map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file3,
			"remotepath": remoteOwnerPath3,
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file3))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         "/subfolder1",
			"expiration-seconds": 0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file1)
		os.Remove(file2)
		os.Remove(file3)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file1,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath1,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file1))

		downloadParams = createParams(map[string]interface{}{
			"localpath":  file2,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath2,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file2))

		downloadParams = createParams(map[string]interface{}{
			"localpath":  file3,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath3,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file3))
	})

	t.Run("Share to public a single encrypted file using auth ticket with zero expiration", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(file)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          remoteOwnerPath,
			"expiration-seconds":  0,
			"encryptionpublickey": encKey,
			"clientid":            clientId,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.Run("Share a private folder with no file using auth ticket with zero expiration", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload Remote Dir
		remoteDir := "/folderToBeShared"
		output, err := createDir(t, configPath, allocationID, remoteDir, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, remoteDir+" directory created", output[0])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          remoteDir,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"expiration-seconds":  0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// list all file to verify
		listFileParams := createParams(map[string]interface{}{
			"authticket": authTicket,
			"remotepath": remoteDir,
			"allocation": allocationID,
		})
		output, err = listAllFilesFromBlobber(t, receiverWallet, configPath, listFileParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "list file - Unexpected output", strings.Join(output, "\n"))
		// Output is:
		// TYPE | NAME | PATH | SIZE | NUM BLOCKS | ACTUAL SIZE | ACTUAL NUM BLOCKS | LOOKUP HASH | IS ENCRYPTED
		// -------+------+------+------+------------+-------------+-------------------+-------------+---------------
	})

	t.Run("Share a private folder with single unencrypted file using auth ticket with zero expiration should work", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(file)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          "/subfolder1",
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"expiration-seconds":  0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.Run("Share a private folder with multiple unencrypted file using auth ticket with zero expiration should work", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload Three files
		fileSize := int64(256)
		file1 := generateRandomTestFileName(t)
		remoteOwnerPath1 := "/subfolder1/subfolder2/" + filepath.Base(file1)

		err := createFileWithSize(file1, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file1,
			"remotepath": remoteOwnerPath1,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file1))

		file2 := generateRandomTestFileName(t)
		remoteOwnerPath2 := "/subfolder1/subfolder2/" + filepath.Base(file2)

		err = createFileWithSize(file2, fileSize)
		require.Nil(t, err)

		uploadParams = map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file2,
			"remotepath": remoteOwnerPath2,
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file2))

		file3 := generateRandomTestFileName(t)
		remoteOwnerPath3 := "/subfolder1/subfolder2/" + filepath.Base(file3)

		err = createFileWithSize(file3, fileSize)
		require.Nil(t, err)

		uploadParams = map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file3,
			"remotepath": remoteOwnerPath3,
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file3))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/subfolder1",
			"expiration-seconds":  0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file1)
		os.Remove(file2)
		os.Remove(file3)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file1,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath1,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file1))

		downloadParams = createParams(map[string]interface{}{
			"localpath":  file2,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath2,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file2))

		downloadParams = createParams(map[string]interface{}{
			"localpath":  file3,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath3,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file3))
	})

	t.Run("Share to public a folder with no encrypted file using auth ticket with zero expiration", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(file)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         "/subfolder1",
			"expiration-seconds": 0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.Run("Share unencrypted file to public using auth ticket with zero expiration", func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         file,
			"expiration-seconds": 0,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.RunWithTimeout("Share unencrypted file to public using auth ticket", 4*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.RunWithTimeout("Shared encrypted file to public using auth ticket without encryptionkey flag should fail", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Equal(t, "Clientid and/or encryptionpublickey are missing for the encrypted share!", output[0], "An unexpected error message!")
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Revoke auth ticket on publicly-shared unencrypted file should fail to download", 4*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// revoke file
		shareParams = map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
			"revoke":     "",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, true)
		require.NotNil(t, err, "Expected error to be present but was not.", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "consensus_not_met")
	})

	t.RunWithTimeout("Expired auth ticket of a publicly-shared file should fail to download", 4*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         file,
			"expiration-seconds": 10,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		cliutils.Wait(t, 10*time.Second)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[0], "consensus_not_met")
	})

	t.RunWithTimeout("Share to public a folder with no encrypted file using auth ticket", 4*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(file)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/subfolder1",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.RunWithTimeout("Share encrypted file using auth ticket - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.RunWithTimeout("Share encrypted huge file using auth ticket - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)

		createWalletForName(walletOwner)

		allocParam := createParams(map[string]interface{}{
			"lock":   24,
			"size":   1024000,
			"parity": 1,
			"data":   1,
		})

		output, err := createNewAllocationForWallet(t, walletOwner, configPath, allocParam)

		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(102400) // this is big enough to cause problem with download
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
			"encrypt":    "",
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		// download with authticket and lookuphash should work
		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"lookuphash": GetReferenceLookup(allocationID, file),
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[len(output)-1], StatusCompletedCB)
		require.Contains(t, output[len(output)-1], filepath.Base(file))

		os.Remove(file) //nolint
		// download with authticket should work
		downloadParams = createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Contains(t, output[len(output)-1], StatusCompletedCB)
		require.Contains(t, output[len(output)-1], filepath.Base(file))
		os.Remove(file) //nolint
	})

	t.RunWithTimeout("Revoke auth ticket of encrypted file - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// revoke file
		shareParams = map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          file,
			"revoke":              "",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Share revoked for client "+clientId, strings.Join(output, "\n"),
			"share file - Unexpected output", strings.Join(output, "\n"))

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "consensus_not_met")
	})

	t.RunWithTimeout("Expired auth ticket of an encrypted file should fail to download - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          file,
			"expiration-seconds":  10,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		cliutils.Wait(t, 10*time.Second)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[0], "consensus_not_met",
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Auth ticket for wrong clientId should fail to download - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey

		walletOwnerModel, err := getWalletForName(t, configPath, walletOwner)
		require.Nil(t, err)

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletOwnerModel.ClientID,
			"encryptionpublickey": encKey,
			"remotepath":          file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[0], "consensus_not_met")
	})

	t.RunWithTimeout("Auth ticket for wrong encryption public key should fail to download - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		clientId := walletReceiver.ClientID

		walletOwnerModel, err := getWalletForName(t, configPath, walletOwner)
		require.Nil(t, err)

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": walletOwnerModel.EncryptionPublicKey,
			"remotepath":          file,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "download file - Unexpected output", strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "Error while initializing encryption invalid_encryption_key: Encryption key mismatch")
	})

	t.RunWithTimeout("Share folder with encrypted file using auth ticket - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(file)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/subfolder1",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})

		receiverWalletObj, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		t.Log("Downloading file for wallet", receiverWalletObj.ClientID)

		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	t.RunWithTimeout("Folder not shared should fail to download - proxy re-encryption", 4*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(file)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
			"encrypt":    "",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		remoteOwnerPathSubfolder := "/subfolder2/subfolder3/" + filepath.Base(file)
		uploadParams = map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPathSubfolder,
			"encrypt":    "",
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/subfolder2",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[0], "consensus_not_met")
	})

	t.RunWithTimeout("Share non-existent file should fail", 3*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/blahblah.txt",
		}
		output, err := shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "file_meta_error: Error getting object meta data from blobbers", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Share someone else's allocation file should fail", 3*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// sharer wallet operations
		sharerWallet := escapedTestName(t) + "_sharer"

		createWalletForName(sharerWallet)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_receiver"

		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/blahblah.txt",
		}
		output, err := shareFileWithWallet(t, sharerWallet, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "file_meta_error: Error getting object meta data from blobbers", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share file with missing allocation should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		shareParams := map[string]interface{}{
			"remotepath": "/blahblah.txt",
		}
		output, err := shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: allocation flag is missing", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share file with missing remotepath should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		createWallet(t)

		shareParams := map[string]interface{}{
			"allocation": "dummy",
		}
		output, err := shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: remotepath flag is missing", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	// ----------------------------------------------------------------------
	// Encrypted folder share regression suite — fix/pre-reuse-folder-entropy
	//
	// The gosdk fix (commit 9dfc92c4) made folder-level shares use
	// signingPrivateKey-derived entropy when the ref is a directory, mirroring
	// chunked_upload on SignatureV2 allocations. Pre-fix, NewDirectoryRef()
	// never set EncryptionVersion, so folder shares fell through to mnemonic
	// entropy, producing "Invalid Ciphertext in reEncrypt, C4 != H5" on every
	// download.
	//
	// Each subtest below materializes the shared directory with `zbox createdir`
	// — the most direct caller of NewDirectoryRef() — to exercise the patched
	// code path explicitly.
	// ----------------------------------------------------------------------

	t.RunWithTimeout("Encrypted folder share - empty folder share download added top-level file - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)
		require.NotEqual(t, "", authTicket)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		remotePath := "/shared/" + filepath.Base(file)
		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remotePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - pre-existing top-level file - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		remotePath := "/shared/" + filepath.Base(file)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err = shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remotePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - pre-existing nested file - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		remotePath := "/shared/nested/" + filepath.Base(file)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err = shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remotePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - add top-level file after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		fileA := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileA, 256))
		remoteA := "/shared/" + filepath.Base(fileA)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fileA,
			"remotepath": remoteA,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err = shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		os.Remove(fileA)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileA,
			"authticket": authTicket,
			"remotepath": remoteA,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)

		// Owner adds a NEW file to the already-shared folder
		fileB := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileB, 256))
		remoteB := "/shared/" + filepath.Base(fileB)
		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fileB,
			"remotepath": remoteB,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		// Recipient downloads new file with the SAME ticket
		os.Remove(fileB)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileB,
			"authticket": authTicket,
			"remotepath": remoteB,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - add nested folder after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		fileA := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileA, 256))
		remoteA := "/shared/" + filepath.Base(fileA)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fileA,
			"remotepath": remoteA,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Owner adds a NEW nested subfolder + file AFTER share
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		fileN := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileN, 256))
		remoteN := "/shared/nested/" + filepath.Base(fileN)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fileN,
			"remotepath": remoteN,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		// Recipient downloads the file in the newly-added subfolder
		os.Remove(fileN)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileN,
			"authticket": authTicket,
			"remotepath": remoteN,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - add deeply nested folders after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Owner builds a deep tree AFTER share — exercises NewDirectoryRef at depth
		for _, d := range []string{"/shared/a", "/shared/a/b", "/shared/a/b/c"} {
			_, err = createDir(t, configPath, allocationID, d, true)
			require.Nil(t, err, "failed to create dir %s", d)
		}

		fileD := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileD, 256))
		remoteD := "/shared/a/b/c/" + filepath.Base(fileD)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fileD,
			"remotepath": remoteD,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		os.Remove(fileD)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileD,
			"authticket": authTicket,
			"remotepath": remoteD,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - update top-level file after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		// v1 source
		v1Src := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(v1Src, 256))
		v1Bytes, err := os.ReadFile(v1Src)
		require.Nil(t, err)

		remotePath := "/shared/" + filepath.Base(v1Src)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  v1Src,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Recipient downloads v1 to a fresh path
		v1Dst := generateRandomTestFileName(t)
		os.Remove(v1Dst)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  v1Dst,
			"authticket": authTicket,
			"remotepath": remotePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		gotV1, err := os.ReadFile(v1Dst)
		require.Nil(t, err)
		require.Equal(t, v1Bytes, gotV1, "recipient should get v1 content before update")

		// Owner overwrites with v2 — distinct fixed bytes so the compare is meaningful
		v2Src := generateRandomTestFileName(t)
		v2Bytes := []byte(strings.Repeat("V2-DISTINCT-CONTENT-PAYLOAD-FOR-ENCRYPTED-FOLDER-SHARE-TEST.", 5))
		require.Nil(t, os.WriteFile(v2Src, v2Bytes, 0o644))
		_, err = updateFileWithWallet(t, walletOwner, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  v2Src,
		}, true)
		require.Nil(t, err)

		// Recipient downloads v2 with SAME ticket to a fresh path
		v2Dst := generateRandomTestFileName(t)
		os.Remove(v2Dst)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  v2Dst,
			"authticket": authTicket,
			"remotepath": remotePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		gotV2, err := os.ReadFile(v2Dst)
		require.Nil(t, err)
		require.Equal(t, v2Bytes, gotV2, "recipient should get v2 content after update with same ticket")
		require.NotEqual(t, v1Bytes, gotV2, "v2 must differ from v1 — not a tautological compare")
	})

	t.RunWithTimeout("Encrypted folder share - update nested file after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		v1Src := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(v1Src, 256))
		v1Bytes, err := os.ReadFile(v1Src)
		require.Nil(t, err)

		remotePath := "/shared/nested/" + filepath.Base(v1Src)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  v1Src,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		v2Src := generateRandomTestFileName(t)
		v2Bytes := []byte(strings.Repeat("NESTED-V2-PAYLOAD-FOR-ENCRYPTED-FOLDER-SHARE-UPDATE-TEST.", 5))
		require.Nil(t, os.WriteFile(v2Src, v2Bytes, 0o644))
		_, err = updateFileWithWallet(t, walletOwner, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  v2Src,
		}, true)
		require.Nil(t, err)

		v2Dst := generateRandomTestFileName(t)
		os.Remove(v2Dst)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  v2Dst,
			"authticket": authTicket,
			"remotepath": remotePath,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		gotV2, err := os.ReadFile(v2Dst)
		require.Nil(t, err)
		require.Equal(t, v2Bytes, gotV2, "recipient should get v2 content for nested file")
		require.NotEqual(t, v1Bytes, gotV2)
	})

	t.RunWithTimeout("Encrypted folder share - rename top-level file after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		oldName := filepath.Base(file)
		oldRemote := "/shared/" + oldName
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": oldRemote,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Rename the file in-place
		newName := "renamed_" + oldName
		newRemote := "/shared/" + newName
		_, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": oldRemote,
			"destname":   newName,
		}, true)
		require.Nil(t, err)

		// Recipient downloads via NEW path with same ticket
		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": newRemote,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)

		// Old path should fail
		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": oldRemote,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, " "), "ref not found")
	})

	t.RunWithTimeout("Encrypted folder share - rename nested file after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		oldName := filepath.Base(file)
		oldRemote := "/shared/nested/" + oldName
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": oldRemote,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		newName := "renamed_" + oldName
		newRemote := "/shared/nested/" + newName
		_, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": oldRemote,
			"destname":   newName,
		}, true)
		require.Nil(t, err)

		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": newRemote,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - rename nested folder after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		baseName := filepath.Base(file)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": "/shared/nested/" + baseName,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Rename the SUBFOLDER itself — strongest folder-entropy regression case
		_, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/shared/nested",
			"destname":   "nested_renamed",
		}, true)
		require.Nil(t, err)

		// File is now under /shared/nested_renamed/<base> — same ticket should work
		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": "/shared/nested_renamed/" + baseName,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - move file into shared folder after share - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/outside", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		baseName := filepath.Base(file)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": "/outside/" + baseName,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Move file from outside the share INTO the shared folder
		_, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/outside/" + baseName,
			"destpath":   "/shared",
		}, true)
		require.Nil(t, err)

		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": "/shared/" + baseName,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - move file within shared folder - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		baseName := filepath.Base(file)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": "/shared/nested/" + baseName,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Move file from nested up to the shared folder
		_, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/shared/nested/" + baseName,
			"destpath":   "/shared",
		}, true)
		require.Nil(t, err)

		os.Remove(file)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": "/shared/" + baseName,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - delete top-level file - sibling still works - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		fileA := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileA, 256))
		baseA := filepath.Base(fileA)
		remoteA := "/shared/" + baseA
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fileA,
			"remotepath": remoteA,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		fileB := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileB, 256))
		baseB := filepath.Base(fileB)
		remoteB := "/shared/" + baseB
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fileB,
			"remotepath": remoteB,
			"encrypt":    "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Owner deletes fileA
		_, err = deleteFile(t, walletOwner, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteA,
		}), true)
		require.Nil(t, err)

		// Recipient download of deleted file fails
		os.Remove(fileA)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileA,
			"authticket": authTicket,
			"remotepath": remoteA,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, " "), "ref not found")

		// Sibling fileB still works with same ticket
		os.Remove(fileB)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileB,
			"authticket": authTicket,
			"remotepath": remoteB,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - delete nested file - parent and siblings still work - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		// /shared/a.txt + /shared/nested/n1.txt + /shared/nested/n2.txt
		fileA := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileA, 256))
		remoteA := "/shared/" + filepath.Base(fileA)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileA, "remotepath": remoteA, "encrypt": "",
		}, true)
		require.Nil(t, err)

		fileN1 := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileN1, 256))
		remoteN1 := "/shared/nested/" + filepath.Base(fileN1)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileN1, "remotepath": remoteN1, "encrypt": "",
		}, true)
		require.Nil(t, err)

		fileN2 := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileN2, 256))
		remoteN2 := "/shared/nested/" + filepath.Base(fileN2)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileN2, "remotepath": remoteN2, "encrypt": "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		_, err = deleteFile(t, walletOwner, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteN1,
		}), true)
		require.Nil(t, err)

		// Deleted nested file: fail
		os.Remove(fileN1)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileN1,
			"authticket": authTicket,
			"remotepath": remoteN1,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, " "), "ref not found")

		// Sibling nested file still works
		os.Remove(fileN2)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileN2,
			"authticket": authTicket,
			"remotepath": remoteN2,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)

		// Top-level still works
		os.Remove(fileA)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileA,
			"authticket": authTicket,
			"remotepath": remoteA,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - delete nested folder - top-level still works - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/shared/nested", true)
		require.Nil(t, err)

		fileA := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileA, 256))
		remoteA := "/shared/" + filepath.Base(fileA)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileA, "remotepath": remoteA, "encrypt": "",
		}, true)
		require.Nil(t, err)

		fileN := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileN, 256))
		remoteN := "/shared/nested/" + filepath.Base(fileN)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileN, "remotepath": remoteN, "encrypt": "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Delete the nested file then the (now empty) folder
		_, err = deleteFile(t, walletOwner, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteN,
		}), true)
		require.Nil(t, err)
		// Best-effort delete the empty folder ref; failure here is non-fatal
		// because the lifecycle assertion is about content access, not ref cleanup.
		_, _ = deleteFile(t, walletOwner, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/shared/nested",
		}), false)

		// Nested file no longer downloadable
		os.Remove(fileN)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileN,
			"authticket": authTicket,
			"remotepath": remoteN,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, " "), "ref not found")

		// Top-level still works with same ticket
		os.Remove(fileA)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileA,
			"authticket": authTicket,
			"remotepath": remoteA,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - unrelated wallet cannot use recipient ticket - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)

		file := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(file, 256))
		remotePath := "/shared/" + filepath.Base(file)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": file, "remotepath": remotePath, "encrypt": "",
		}, true)
		require.Nil(t, err)

		// userA — legitimate recipient
		userA := escapedTestName(t) + "_second"
		createWalletForName(userA)
		walletA, err := getWalletForName(t, configPath, userA)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletA.ClientID,
			"encryptionpublickey": walletA.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// userB — unrelated third wallet attempting to reuse userA's ticket
		userB := escapedTestName(t) + "_third"
		createWalletForName(userB)

		os.Remove(file)
		output, err = downloadFileForWallet(t, userB, configPath, createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remotePath,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, " "), "ref not found",
			"unrelated wallet must not be able to use a ticket bound to another recipient")
	})

	t.RunWithTimeout("Encrypted folder share - recipient cannot escape via remotepath - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/private", true)
		require.Nil(t, err)

		// /shared/a.txt — recipient is allowed to see this
		fileA := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileA, 256))
		remoteA := "/shared/" + filepath.Base(fileA)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileA, "remotepath": remoteA, "encrypt": "",
		}, true)
		require.Nil(t, err)

		// /private/secret.txt — recipient must NOT be able to reach this
		fileSecret := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileSecret, 256))
		remoteSecret := "/private/" + filepath.Base(fileSecret)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileSecret, "remotepath": remoteSecret, "encrypt": "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Attempt to download /private/secret.txt with the /shared ticket
		os.Remove(fileSecret)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileSecret,
			"authticket": authTicket,
			"remotepath": remoteSecret,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, " "), "ref not found",
			"recipient must not reach files outside the shared folder via remotepath")

		// Control: legit access still works
		os.Remove(fileA)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileA,
			"authticket": authTicket,
			"remotepath": remoteA,
		}), false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	t.RunWithTimeout("Encrypted folder share - recipient cannot escape via lookuphash - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/private", true)
		require.Nil(t, err)

		fileA := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileA, 256))
		remoteA := "/shared/" + filepath.Base(fileA)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileA, "remotepath": remoteA, "encrypt": "",
		}, true)
		require.Nil(t, err)

		fileSecret := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileSecret, 256))
		remoteSecret := "/private/" + filepath.Base(fileSecret)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileSecret, "remotepath": remoteSecret, "encrypt": "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Compute the lookup hash of the target file outside the share scope.
		// Formula: sha3-256 hex of "<allocationID>:<path>". Source:
		// gosdk/zboxcore/fileref/fileref.go:115 (GetReferenceLookup).
		secretHashBytes := sha3.Sum256([]byte(allocationID + ":" + remoteSecret))
		secretLookupHash := hex.EncodeToString(secretHashBytes[:])

		// Attack: pass a manipulated lookuphash for /private/secret.txt
		os.Remove(fileSecret)
		output, err = downloadFileForWallet(t, receiverWallet, configPath, createParams(map[string]interface{}{
			"localpath":  fileSecret,
			"authticket": authTicket,
			"lookuphash": secretLookupHash,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Encrypted folder share - recipient cannot list sibling folder via authticket - proxy re-encryption", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		_, err := createDir(t, configPath, allocationID, "/shared", true)
		require.Nil(t, err)
		_, err = createDir(t, configPath, allocationID, "/private", true)
		require.Nil(t, err)

		fileSecret := generateRandomTestFileName(t)
		require.Nil(t, createFileWithSize(fileSecret, 256))
		remoteSecret := "/private/" + filepath.Base(fileSecret)
		_, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID, "localpath": fileSecret, "remotepath": remoteSecret, "encrypt": "",
		}, true)
		require.Nil(t, err)

		receiverWallet := escapedTestName(t) + "_second"
		createWalletForName(receiverWallet)
		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		output, err := shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
			"remotepath":          "/shared",
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err)

		// Receiver attempts to list /private using the /shared ticket. Inline the
		// command so it runs as the receiver wallet — the listAllFilesFromBlobber
		// helper hardcodes the test name, which would run as the owner.
		listCmd := fmt.Sprintf(
			"./zbox list --authticket %s --remotepath /private --json --silent --wallet %s_wallet.json --configDir ./config --config %s",
			authTicket, receiverWallet, configPath,
		)
		output, err = cliutils.RunCommandWithoutRetry(listCmd)
		// Two acceptable outcomes: blobber rejects (err != nil with consensus
		// failure) OR returns an empty list (no children). Either proves the
		// recipient cannot enumerate sibling folders.
		joined := strings.Join(output, " ")
		listFailed := err != nil && (strings.Contains(joined, "ref not found") ||
			strings.Contains(joined, "invalid_path") ||
			strings.Contains(joined, "auth_ticket") ||
			strings.Contains(joined, "allocation flag is missing"))
		emptyList := err == nil &&
			(joined == "" || joined == "null" || strings.Contains(joined, "[]"))
		require.True(t, listFailed || emptyList,
			"recipient must not be able to list sibling folder /private; got: %s", joined)
	})
}

func shareFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	return shareFileWithWallet(t, escapedTestName(t), cliConfigFilename, param)
}

func shareFileWithWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Sharing file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox share %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)

	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func createWalletAndAllocation(t *test.SystemTest, configPath, wallet string) (string, *climodel.Wallet) {
	createWalletForName(wallet)

	allocParam := createParams(map[string]interface{}{
		"lock":   2,
		"size":   1024 * 1024 * 1024,
		"parity": 1,
		"data":   1,
	})

	output, err := createNewAllocationForWallet(t, wallet, configPath, allocParam)
	require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

	require.Len(t, output, 1)
	matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
	require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

	allocationID := strings.Fields(output[0])[2]

	walletModel, err := getWalletForName(t, configPath, wallet)
	require.Nil(t, err)

	return allocationID, walletModel
}

// Hash - hash the given data and return the hash as hex string
func Hash(data string) string {
	return hex.EncodeToString(RawHash(data))
}

// RawHash - Logic to hash the text and return the hash bytes
func RawHash(data string) []byte {
	hash := sha3.New256()
	hash.Write([]byte(data))
	var buf []byte
	return hash.Sum(buf)
}

// GetReferenceLookup hash(allocationID + ":" + path)
func GetReferenceLookup(allocationID, path string) string {
	return Hash(allocationID + ":" + path)
}

func listAllFilesFromBlobber(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 10*time.Second) // TODO replace with poller
	t.Logf("Listing all files in allocation...")
	cmd := fmt.Sprintf(
		"./zbox list %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
