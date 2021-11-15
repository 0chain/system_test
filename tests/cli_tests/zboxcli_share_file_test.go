package cli_tests

import (
	"encoding/json"
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

	t.Run("Share file using auth ticket with another wallet - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		_, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}

		output, err := shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))

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
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params)

		require.Nil(t, err, "Error in file operation", strings.Join(output, "\n"))

		require.Len(t, output, 2)
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1])
	})

	t.Run("Revoke auth ticket - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		_, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}

		output, err := shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// revoke file
		share_params = map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
			"revoke":              "",
		}
		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Equal(t, "Share revoked for client "+clientId, strings.Join(output, "\n"))

		// Download the file
		dummyFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/" + filepath.Base(filename)
		os.Remove(dummyFilePath)

		download_params := createParams(map[string]interface{}{
			"localpath":  dummyFilePath,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params)

		require.NotNil(t, err, "Error in file operation", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1])
	})

	t.Run("Share file using auth ticket with expiration - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		_, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
			"expiration-seconds":  10,
		}

		output, err := shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))

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
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params)

		require.NotNil(t, err, "Error in file operation", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})

	t.Run("Download shared file using auth ticket from wrong clientId - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		_, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey

		walletOwnerModel, err := getWalletForName(t, configPath, walletOwner)
		require.Nil(t, err)

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletOwnerModel.ClientID,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}

		output, err := shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))

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
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params)
		require.NotNil(t, err, "Error:", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})

	t.Run("Share folder using auth ticket with another wallet - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/test_file.txt"
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPath)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		_, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/subfolder1",
		}

		output, err := shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))

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
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
	})

	t.Run("Share folder using auth ticket with another wallet, wrong download path - proxy re-encryption", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/test_file.txt"
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPath)

		remoteOwnerPathSubfolder := "/subfolder2/subfolder3/test_file.txt"
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPathSubfolder)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		_, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/subfolder2",
		}

		output, err := shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))

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
		output, err = downloadFileForWallet(t, receiverWallet, configPath, download_params)

		require.NotNil(t, err, "Error:", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0])
	})
}

func uploadFileForShare(t *testing.T, allocationID, wallet, localpath, remotepath string) {
	var output []string

	// upload file
	fileSize := int64(256)
	err := createFileWithSize(localpath, fileSize)
	require.Nil(t, err)

	output, err = uploadFileForWallet(t, wallet, configPath, map[string]interface{}{
		"allocation": allocationID,
		"localpath":  localpath,
		"remotepath": remotepath,
		"encrypt":    "",
	})
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)

	expected := fmt.Sprintf(
		"Status completed callback. Type = application/octet-stream. Name = %s",
		filepath.Base(remotepath),
	)
	require.Equal(t, expected, output[1])
}

func shareFile(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Share file...")
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

	output, err = readPoolInfo(t, configPath, allocationID)
	require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

	readPool := []climodel.ReadPoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &readPool)
	require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

	walletModel, err := getWalletForName(t, configPath, wallet)
	require.Nil(t, err)

	return allocationID, walletModel
}
