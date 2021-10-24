package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestShareFile(t *testing.T) {
	t.Parallel()
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		configPath = "./zbox_config.yaml"
		cliutils.Logger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	t.Run("Share file with another wallet", func(t *testing.T) {
		t.Parallel()
		var output []string
		var err error

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		allocationReceiver, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}

		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Nil(t, err)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, err)
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := os.TempDir() + escapedTestName(t) + ".txt"
		download_params := map[string]interface{}{
			"allocation": allocationReceiver,
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": filename,
		}
		output, err = downloadFileWithParams(t, receiverWallet, configPath, download_params)

		require.Nil(t, err, "Error in file operation", strings.Join(output, "\n"))
		defer os.Remove(dummyFilePath)

		require.Len(t, output, 2)
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filename, output[1])
	})

	t.Run("Revoke shared file", func(t *testing.T) {
		t.Parallel()

		var output []string
		var err error

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		allocationReceiver, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationReceiver,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}

		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Nil(t, err)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, err)
		require.NotEqual(t, "", authTicket)

		// revoke file
		share_params = map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
			"--revoke":            "",
		}
		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Nil(t, err)

		// Download the file
		dummyFilePath := os.TempDir() + escapedTestName(t) + ".txt"
		download_params := map[string]interface{}{
			"allocation": allocationReceiver,
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": filename,
		}
		output, err = downloadFileWithParams(t, receiverWallet, configPath, download_params)

		require.NotNil(t, err, "Error in file operation", strings.Join(output, "\n"))
		defer os.Remove(dummyFilePath)
	})

	t.Run("Share file with expiration", func(t *testing.T) {
		t.Parallel()

		var output []string
		var err error

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		allocationReceiver, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":           allocationID,
			"clientid":             clientId,
			"encryptionpublickey":  encKey,
			"remotepath":           filename,
			"--expiration-seconds": 1,
		}

		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Nil(t, err)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, err)
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := os.TempDir() + escapedTestName(t) + ".txt"
		download_params := map[string]interface{}{
			"allocation": allocationReceiver,
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": filename,
		}
		output, err = downloadFileWithParams(t, receiverWallet, configPath, download_params)

		require.NotNil(t, err, "Error in file operation", strings.Join(output, "\n"))
		defer os.Remove(dummyFilePath)
	})

	t.Run("Download shared file from wrong clientId", func(t *testing.T) {
		t.Parallel()

		var output []string
		var err error

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		uploadFileForShare(t, allocationID, walletOwner, filename, filename)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		allocationReceiver, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey

		walletOwnerModel, err := getWalletForName(t, configPath, walletOwner)
		require.Nil(t, err)

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            walletOwnerModel.ClientID,
			"encryptionpublickey": encKey,
			"remotepath":          filename,
		}

		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Nil(t, err)
		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, err)
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := os.TempDir() + escapedTestName(t) + ".txt"
		download_params := map[string]interface{}{
			"allocation": allocationReceiver,
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": filename,
		}
		output, err = downloadFileWithParams(t, receiverWallet, configPath, download_params)
		require.NotNil(t, err, "Error:", strings.Join(output, "\n"))

		defer os.Remove(dummyFilePath)
	})

	t.Run("Share folder with another wallet", func(t *testing.T) {

		var output []string
		var err error

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		filename := generateRandomTestFileName(t)
		remoteOwnerPath := "/subfolder1/subfolder2/test_file.txt"
		uploadFileForShare(t, allocationID, walletOwner, filename, remoteOwnerPath)

		// receiver wallet operations
		receiverWallet := escapedTestName(t) + "_second"
		allocationReceiver, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/subfolder1",
		}

		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Nil(t, err)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, err)
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := os.TempDir() + escapedTestName(t) + ".txt"
		download_params := map[string]interface{}{
			"allocation": allocationReceiver,
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		}
		output, err = downloadFileWithParams(t, receiverWallet, configPath, download_params)

		require.Nil(t, err, "Error in file operation", strings.Join(output, "\n"))
		defer os.Remove(dummyFilePath)

		require.Len(t, output, 2)
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filename, output[1])
	})

	t.Run("Share folder with another wallet wrong download", func(t *testing.T) {

		var output []string
		var err error

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
		allocationReceiver, walletReceiver := registerAndCreateAllocation(t, configPath, receiverWallet)

		encKey := walletReceiver.EncryptionPublicKey
		clientId := walletReceiver.ClientID

		share_params := map[string]interface{}{
			"allocation":          allocationID,
			"clientid":            clientId,
			"encryptionpublickey": encKey,
			"remotepath":          "/subfolder2",
		}

		output, err = shareFile(t, walletOwner, configPath, share_params)
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Nil(t, err)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, err)
		require.NotEqual(t, "", authTicket)

		// Download the file
		dummyFilePath := os.TempDir() + escapedTestName(t) + ".txt"
		download_params := map[string]interface{}{
			"allocation": allocationReceiver,
			"localpath":  dummyFilePath,
			"authticket": authTicket,
			"remotepath": remoteOwnerPath,
		}
		output, err = downloadFileWithParams(t, receiverWallet, configPath, download_params)

		require.NotNil(t, err, "Error:", strings.Join(output, "\n"))
		defer os.Remove(dummyFilePath)
	})
}

func uploadFileForShare(t *testing.T, allocationID, wallet, localpath, filename string) {
	var output []string

	// upload file
	fileSize := int64(256)
	err := createFileWithSize(filename, fileSize)
	require.Nil(t, err)

	output, err = uploadFileForWallet(t, wallet, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": filename,
		"localpath":  filename,
		"encrypt":    "",
	})
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)

	expected := fmt.Sprintf(
		"Status completed callback. Type = application/octet-stream. Name = %s",
		filepath.Base(filename),
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

	return cliutils.RunCommand(cmd)
}

func registerWalletAndFaucet(t *testing.T, configPath, wallet string) error {
	faucetTokens := 3.0
	// First create a wallet and run faucet command
	output, err := registerWalletForName(configPath, wallet)
	require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

	output, err = executeFaucetWithTokensForWallet(t, wallet, configPath, faucetTokens)
	require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

	return nil
}

func registerAndCreateAllocation(t *testing.T, configPath, wallet string) (string, *climodel.Wallet) {
	registerWalletAndFaucet(t, configPath, wallet)

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

	// Lock 1 token in Write pool amongst all blobbers
	params := createParams(map[string]interface{}{
		"allocation": allocationID,
		"duration":   "30m",
		"tokens":     "1",
	})
	output, err = writePoolLock(t, wallet, configPath, params)
	require.Nil(t, err, "Failed to lock write tokens", strings.Join(output, "\n"))
	require.Equal(t, "locked", output[0])

	output, err = writePoolInfo(t, configPath)
	require.Nil(t, err, "Error fetching write pool", strings.Join(output, "\n"))

	writePools := []climodel.WritePoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &writePools)
	require.Nil(t, err, "Error unmarshalling write pool", strings.Join(output, "\n"))

	// locking tokens for read pool
	output, err = readPoolLockWithWallet(t, wallet, configPath, allocationID, 0.4)
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

func writePoolLock(t *testing.T, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Locking write tokens...")
	return cliutils.RunCommand(fmt.Sprintf("./zbox wp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
}
