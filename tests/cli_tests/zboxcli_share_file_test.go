package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestShareFile(t *testing.T) {
	t.Parallel()

	// share non-existent file
	// revoke all auth ticket for remote path
	// publicly share encrypted file should result to download failure

	// pre tests for unencrypted file

	// param validation
	// missing alloc
	// missing remotepath

	// missing clientid but pubkey provided
	// missing pubkey but clientid provided

	// add output on error message
	// require equal on share output

	t.Run("Publicly share unencrypted file using auth ticket", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename, false)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": filename,
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.Nil(t, err, "Error in file operation", strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	// FIXME is auth ticket with no target wallet and expiration meant to be eternal and cannot be revoked?
	t.Run("Revoke auth ticket on publicly-shared unencrypted file should fail", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename, false)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": filename,
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// revoke file
		shareParams = map[string]interface{}{
			"allocation": allocationID,
			"remotepath": filename,
			"revoke":     "",
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "share not reached", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.Nil(t, err, "Error in file operation", strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Publicly share unencrypted file using auth ticket with expiration", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename, false)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         filename,
			"expiration-seconds": 10,
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		time.Sleep(10 * time.Second)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Publicly share folder with no encrypted file using auth ticket", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(filename)
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPath, false)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/subfolder1",
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share encrypted file using auth ticket - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename, true)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Revoke auth ticket of encrypted file - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename, true)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
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
			"remotepath":          filename,
			"revoke":              "",
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Share revoked for client "+clientId, strings.Join(output, "\n"),
			"share file - Unexpected output", strings.Join(output, "\n"))

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Expired auth ticket of encrypted file should fail - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename, true)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
			"expiration-seconds":  10,
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		time.Sleep(10 * time.Second)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.NotNil(t, err, "Error in file operation", strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"share file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Auth ticket but for wrong clientId should fail - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename, true)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		encKey := walletReceiver.EncryptionPublicKey

		walletOwnerModel, err := getWalletForName(t, configPath, walletOwner)
		require.Nil(t, err)

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletOwnerModel.ClientID,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Share folder with encrypted file using auth ticket - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/" + filepath.Base(filename)
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPath, true)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

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
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Downloading from folder not shared should fail - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/test_file.txt"
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPath, true)

		remoteOwnerPathSubfolder := "/subfolder2/subfolder3/test_file.txt"
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPathSubfolder, true)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"

		output, err := registerWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

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
		output, err = shareFile(t, walletOwner, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})
}

func uploadFileForShare(t *testing.T, allocationID, wallet, localpath, remotepath string, encrypt bool) {
	var output []string

	// upload file
	fileSize := int64(256)
	err := createFileWithSize(localpath, fileSize)
	require.Nil(t, err)

	uploadParams := map[string]interface{}{
		"allocation": allocationID,
		"localpath":  localpath,
		"remotepath": remotepath,
	}

	if encrypt {
		uploadParams["encrypt"] = ""
	}

	output, err = uploadFileForWallet(t, wallet, configPath, uploadParams, false)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)

	expected := fmt.Sprintf(
		"Status completed callback. Type = application/octet-stream. Name = %s",
		filepath.Base(remotepath),
	)
	require.Equal(t, expected, output[1])
}

func shareFile(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
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

func registerWalletAndFaucet(t *testing.T, configPath, wallet string) error {
	faucetTokens := 3.0
	// First create a wallet and run faucet command
	output, err := registerWalletForName(t, configPath, wallet)
	require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

	output, err = executeFaucetWithTokensForWallet(t, wallet, configPath, faucetTokens)
	require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

	return nil
}

func registerAndCreateAllocation(t *testing.T, configPath, wallet string) (string, *climodel.Wallet) {
	err := registerWalletAndFaucet(t, configPath, wallet)
	require.Nil(t, err)

	allocParam := createParams(map[string]interface{}{
		"lock":   0.5,
		"size":   10485760,
		"expire": "1h",
		"parity": 1,
		"data":   1,
	})
	output, err := createNewAllocationForWallet(t, wallet+"_wallet.json", configPath, allocParam)
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
	output, err = readPoolLockWithWallet(t, wallet, configPath, readPoolParams)
	require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

	walletModel, err := getWalletForName(t, configPath, wallet)
	require.Nil(t, err)

	return allocationID, walletModel
}
