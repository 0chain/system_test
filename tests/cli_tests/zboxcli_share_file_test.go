package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	//TODO: all share operations take ~40s except for PRE which takes ~2mins 30s!
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Share to public a folder with no encrypted file using auth ticket with zero expiration")

	t.Parallel()
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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		output, err = createWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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

		output, err = createWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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
		faucetTokens := 9.0

		output, err := createWalletForName(t, configPath, walletOwner)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		for i := 0; i < 3; i++ {
			output, err = executeFaucetWithTokensForWallet(t, walletOwner, configPath, faucetTokens)
			require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
		}

		allocParam := createParams(map[string]interface{}{
			"lock":   24,
			"size":   1024000,
			"expire": "30m",
			"parity": 1,
			"data":   1,
		})

		output, err = createNewAllocationForWallet(t, walletOwner, configPath, allocParam)

		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		// locking tokens for read pool
		readPoolParams := createParams(map[string]interface{}{
			"tokens": 3,
		})
		output, err = readPoolLockWithWallet(t, walletOwner, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		output, err = createWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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

		output, err = createWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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
		require.Len(t, output, 3, "download file - Unexpected output", strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "Error cipher: message authentication failed")
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

		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

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

		output, err = createWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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

		output, err := createWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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
		output, err = shareFile(t, configPath, shareParams)
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

		output, err := createWalletForName(t, configPath, sharerWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_receiver"

		output, err = createWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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
		output, err = shareFileWithWallet(t, sharerWallet, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "file_meta_error: Error getting object meta data from blobbers", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share file with missing allocation should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"remotepath": "/blahblah.txt",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: allocation flag is missing", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share file with missing remotepath should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"allocation": "dummy",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: remotepath flag is missing", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Share encrypted file using auth ticket - download accounting test - proxy re-encryption ", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/" + filepath.Base(file)
		fileSize := int64(1024 * 1024 * 10) // must upload bigger file to ensure has noticeable cost
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

		// locking 1 read tokens to readPool via wallet
		createWalletForNameAndLockReadTokens(t, configPath, receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          remoteOwnerPath,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// fetching readPool info for receiver
		output, err = readPoolInfoWithWallet(t, receiverWallet, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		initialReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, initialReadPool)

		// instead of 0.4*1e10 we need to use 0.1*1e11 as we are staking single token via receiver wallet
		require.Equal(t, 0.1*1e11, float64(initialReadPool.Balance))

		// download cost functions works fine with no issues.
		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteOwnerPath,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expectedDownloadCost, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInSas := unitToZCN(expectedDownloadCost, unit) * 1e10
		t.Logf("Download cost: %v sas", expectedDownloadCostInSas)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})

		// downloading file for receiver walet
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// waiting 60 seconds for blobber to redeem tokens
		cliutils.Wait(t, 60*time.Second)

		output, err = readPoolInfoWithWallet(t, receiverWallet, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		finalReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, finalReadPool)

		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		expectedRPBalance := initialReadPool.Balance - int64(expectedDownloadCostInSas)

		expectedRPBalance = int64(expectedRPBalance*95) / 100 // reducing it to 5% to deal with the rounding off issue

		// getDownloadCost returns download cost when all the associated blobbers of an allocation are required
		// In current enhancement/verify-download PR, it gets data from minimum blobbers possible.
		// So the download cost will be in between initial balance and expected balance.
		require.Equal(t, true,
			finalReadPool.Balance < initialReadPool.Balance &&
				finalReadPool.Balance >= int64(expectedRPBalance))
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
	faucetTokens := 3.0
	// First create a wallet and run faucet command
	// Output:
	// 		[0]:"No wallet in path  C:\Users\kisha\.zcn\unique found. Creating wallet..."
	// 		[1]:"ZCN wallet created!!"
	// 		[2]:"Execute faucet smart contract success with txn : ${hash}"
	output, err := createWalletForName(t, configPath, wallet)
	require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))
	require.Len(t, output, 3, strings.Join(output, "\n"))

	output, err = executeFaucetWithTokensForWallet(t, wallet, configPath, faucetTokens)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	allocParam := createParams(map[string]interface{}{
		"lock":   2,
		"size":   1024 * 1024 * 1024,
		"expire": "10m",
		"parity": 1,
		"data":   1,
	})

	output, err = createNewAllocationForWallet(t, wallet, configPath, allocParam)

	require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

	require.Len(t, output, 1)
	matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
	require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

	allocationID := strings.Fields(output[0])[2]

	// locking tokens for read pool
	readPoolParams := createParams(map[string]interface{}{
		"tokens": 0.4,
	})
	output, err = readPoolLockWithWallet(t, wallet, configPath, readPoolParams, true)
	require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

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
