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

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestShareFile(t *testing.T) {
	t.Parallel()

	t.Run("Share unencrypted file to public using auth ticket", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Shared encrypted file to public using auth ticket should fail to download", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3, "download file - Unexpected output", strings.Join(output, "\n"))
		aggregatedOutput := strings.ToLower(strings.Join(output, " "))
		require.Contains(t, aggregatedOutput, "invalid ed25519 curve point")
	})

	t.Run("Revoke auth ticket on publicly-shared unencrypted file should fail to download", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Len(t, output, 3)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "share revoked")
	})

	t.Run("Expired auth ticket of a publicly-shared file should fail to download", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share to public a folder with no encrypted file using auth ticket", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share encrypted file using auth ticket - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share encrypted huge file using auth ticket - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(102400) // this is big enough to cause problem with download
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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)

		// download with authticket and lookuphash should work
		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"lookuphash": GetReferenceLookup(allocationID, file),
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Equal(t, expected, output[len(output)-1])

		os.Remove(file) //nolint
		// download with authticket should work
		downloadParams = createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Equal(t, expected, output[len(output)-1])
		os.Remove(file) //nolint
	})

	t.Run("Revoke auth ticket of encrypted file - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Len(t, output, 3)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "share revoked")
	})

	t.Run("Expired auth ticket of an encrypted file should fail to download - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Auth ticket for wrong clientId should fail to download - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Auth ticket for wrong encryption public key should fail to download - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share folder with encrypted file using auth ticket - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Folder not shared should fail to download - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share non-existent file should fail", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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

	t.Run("Share someone else's allocation file should fail", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// sharer wallet operations
		sharerWallet := escapedTestName(t) + "_sharer"

		output, err := registerWalletForName(t, configPath, sharerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_receiver"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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

	t.Run("Share file with missing allocation should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"remotepath": "/blahblah.txt",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: allocation flag is missing", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share file with missing remotepath should fail", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"allocation": "dummy",
		}
		output, err = shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: remotepath flag is missing", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	// FIXME download cost is not affecting read pool if downloading through auth ticket
	t.Run("Share encrypted file using auth ticket - download accounting test - proxy re-encryption ", func(t *testing.T) {
		t.Skip("Skipped for merging nonce")
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(10240) // must upload bigger file to ensure has noticeable cost
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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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

		// Read pool before download
		output, err = readPoolInfo(t, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		initialReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, initialReadPool)

		require.Equal(t, 0.4, initialReadPool.OwnerBalance)
		t.Logf("Read pool balance: %v", initialReadPool.OwnerBalance)

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
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
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))

		// Read pool after download
		output, err = readPoolInfo(t, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		finalReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, finalReadPool)

		expectedRPBalance := 0.4*1e10 - expectedDownloadCostInSas
		// todo: finalReadPool.OwnerBalance might be in ZCN format
		require.InEpsilon(t, expectedRPBalance, float64(finalReadPool.OwnerBalance), epsilon)
	})

	// FIXME download cost is not affecting read pool if downloading through auth ticket
	t.Run("Share unencrypted file using auth ticket - download accounting test", func(t *testing.T) {
		t.Skip("Skipped for nonce merge")
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(10240) // must upload bigger file to ensure has noticeable cost
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
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

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

		// Read pool before download
		output, err = readPoolInfo(t, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		initialReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, initialReadPool)

		require.Equal(t, 0.4, initialReadPool.OwnerBalance)

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
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
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))

		// Wait for blobber to redeem read-tokens
		// Blobber runs worker in the interval of usually 10 seconds.
		time.Sleep(time.Second * 20)
		// Read pool after download
		output, err = readPoolInfo(t, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		finalReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, finalReadPool)

		expectedRPBalance := 0.4*1e10 - expectedDownloadCostInSas
		// todo: finalReadPool.OwnerBalance might be in ZCN format
		require.InEpsilon(t, expectedRPBalance, float64(finalReadPool.OwnerBalance), epsilon)
	})

	t.Run("Share encrypted file using auth ticket - download accounting test where 3rd party pays - proxy re-encryption ", func(t *testing.T) {
		t.Skip("Skipped for nonce merge")
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(10240) // must upload bigger file to ensure has noticeable cost
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation":              allocationID,
			"localpath":               file,
			"remotepath":              file,
			"encrypt":                 "",
			"attr-who-pays-for-reads": "3rd_party",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))
		require.Len(t, output, 4, strings.Join(output, "\n"))
		require.Equal(t, "Read pool created successfully", output[2], strings.Join(output, "\n"))
		require.Equal(t, "Wallet registered", output[3], strings.Join(output, "\n"))

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

		output, err = executeFaucetWithTokensForWallet(t, receiverWallet, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		readPoolParams := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "1h",
		})
		output, err = readPoolLockWithWallet(t, receiverWallet, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Read pool before download
		output, err = readPoolInfoWithwallet(t, receiverWallet, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		initialReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, initialReadPool)

		require.Equal(t, 0.4, initialReadPool.OwnerBalance)

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expectedDownloadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN = unitToZCN(expectedDownloadCostInZCN, unit) * 1e10
		t.Logf("Download cost: %v", expectedDownloadCostInZCN)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
			"lookuphash": GetReferenceLookup(allocationID, file),
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))

		// Wait for blobber to redeem read-tokens
		// Blobber runs worker in the interval of usually 10 seconds.
		time.Sleep(time.Second * 20)
		// Read pool after download
		output, err = readPoolInfoWithwallet(t, receiverWallet, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		finalReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, finalReadPool)

		// todo: finalReadPool.OwnerBalance might be in ZCN format
		require.EqualValues(t, 0.4*1e10, float64(finalReadPool.OwnerBalance))
	})

	t.Run("Share unencrypted file using auth ticket - download accounting test where 3rd party pays ", func(t *testing.T) {
		t.Skip("Skipped for nonce merge")
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		fileSize := int64(10240) // must upload bigger file to ensure has noticeable cost
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation":              allocationID,
			"localpath":               filename,
			"remotepath":              filename,
			"attr-who-pays-for-reads": "3rd_party",
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(filename)), output[1])

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err = registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))
		require.Len(t, output, 4, strings.Join(output, "\n"))
		require.Equal(t, "Read pool created successfully", output[2], strings.Join(output, "\n"))
		require.Equal(t, "Wallet registered", output[3], strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": filename,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		output, err = executeFaucetWithTokensForWallet(t, receiverWallet, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		readPoolParams := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "1h",
		})
		output, err = readPoolLockWithWallet(t, receiverWallet, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Read pool before download
		output, err = readPoolInfoWithwallet(t, receiverWallet, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		initialReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, initialReadPool)

		require.InEpsilon(t, 0.4*1e10, initialReadPool.OwnerBalance, epsilon, "read pool balance did not match expected")

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": filename,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expectedDownloadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN = unitToZCN(expectedDownloadCostInZCN, unit) * 1e10
		t.Logf("Download cost: %v", expectedDownloadCostInZCN)

		// Download the file (delete local copy first)
		os.Remove(filename)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  filename,
			"authticket": authTicket,
			"lookuphash": GetReferenceLookup(allocationID, filename),
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))

		// Read pool after download
		output, err = readPoolInfoWithwallet(t, receiverWallet, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		finalReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, finalReadPool)

		// todo: finalReadPool.OwnerBalance might be in ZCN format
		require.EqualValues(t, 0.4*1e10, float64(finalReadPool.OwnerBalance))
	})
}

func shareFile(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	return shareFileWithWallet(t, escapedTestName(t), cliConfigFilename, param)
}

func shareFileWithWallet(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
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

func registerAndCreateAllocation(t *testing.T, configPath, wallet string) (string, *climodel.Wallet) {
	faucetTokens := 3.0
	// First create a wallet and run faucet command
	// Output:
	// 		[0]:"ZCN wallet created"
	// 		[1]:"Creating related read pool for storage smart-contract..."
	// 		[2]:"Read pool created successfully"
	// 		[3]:"Wallet registered"
	output, err := registerWalletForName(t, configPath, wallet)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))
	require.Len(t, output, 4, strings.Join(output, "\n"))
	require.Equal(t, "Read pool created successfully", output[2], strings.Join(output, "\n"))
	require.Equal(t, "Wallet registered", output[3], strings.Join(output, "\n"))

	output, err = executeFaucetWithTokensForWallet(t, wallet, configPath, faucetTokens)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	allocParam := createParams(map[string]interface{}{
		"lock":   0.5,
		"size":   10485760,
		"expire": "2h",
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
		"allocation": allocationID,
		"tokens":     0.4,
		"duration":   "1h",
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
