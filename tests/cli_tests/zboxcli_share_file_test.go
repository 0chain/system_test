package cli_tests

//nolint:errcheck // defer statements for cleanup don't need error handling

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
