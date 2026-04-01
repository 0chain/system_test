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

func TestSharePublicPrivate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Public share of unencrypted file should allow any wallet to download")

	t.Parallel()

	// 1. Public share of unencrypted file — share publicly (no specific recipient), download by any wallet
	t.RunWithTimeout("Public share of unencrypted file should allow any wallet to download", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// Share publicly (no clientid, no encryptionpublickey)
		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Create a completely separate receiver wallet and download
		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

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

		// Verify a second different wallet can also download
		receiverWallet2 := escapedTestName(t) + "_receiver2"
		createWalletForName(receiverWallet2)

		os.Remove(file)

		output, err = downloadFileForWallet(t, receiverWallet2, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})

	// 2. Private share of unencrypted file — share to specific recipient, only that recipient can download
	t.RunWithTimeout("Private share of unencrypted file to specific recipient should only allow that recipient to download", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// Create receiver wallet and get its client ID
		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		// Share to specific recipient (private share)
		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          remotePath,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Intended recipient should be able to download
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

		// A different wallet should NOT be able to download with this auth ticket
		otherWallet := escapedTestName(t) + "_other"
		createWalletForName(otherWallet)

		otherLocalFile := generateRandomTestFileName(t)

		downloadParams2 := createParams(map[string]interface{}{
			"localpath":  otherLocalFile,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, otherWallet, configPath, downloadParams2, false)
		require.NotNil(t, err, "Expected error when unauthorized wallet downloads", strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "consensus_not_met")
	})

	// 3. Public share of folder — share entire folder publicly, download files from it
	t.RunWithTimeout("Public share of folder should allow downloading files within the folder", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// Create folder and upload files
		remoteDir := "/sharedFolder/"
		output, err := createDir(t, configPath, allocationID, remoteDir, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, remoteDir+" directory created", output[0])

		file1 := generateRandomTestFileName(t)
		fileSize := int64(256)
		err = createFileWithSize(file1, fileSize)
		require.Nil(t, err)

		remoteFile1 := remoteDir + filepath.Base(file1)
		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file1,
			"remotepath": remoteFile1,
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		file2 := generateRandomTestFileName(t)
		err = createFileWithSize(file2, fileSize)
		require.Nil(t, err)

		remoteFile2 := remoteDir + filepath.Base(file2)
		uploadParams = map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file2,
			"remotepath": remoteFile2,
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Share folder publicly
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

		// Receiver downloads file1 from shared folder
		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

		os.Remove(file1)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file1,
			"authticket": authTicket,
			"remotepath": remoteFile1,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file1))

		// Receiver downloads file2 from shared folder
		os.Remove(file2)

		downloadParams = createParams(map[string]interface{}{
			"localpath":  file2,
			"authticket": authTicket,
			"remotepath": remoteFile2,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file2))
	})

	// 4. Group share — share file to multiple recipients, verify each can download
	t.RunWithTimeout("Group share to multiple recipients should allow each recipient to download", 6*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// Create 3 receiver wallets and share to each
		type recipientInfo struct {
			walletName string
			authTicket string
		}

		recipients := make([]recipientInfo, 3)
		for i := 0; i < 3; i++ {
			walletName := fmt.Sprintf("%s_recipient_%d", escapedTestName(t), i)
			createWalletForName(walletName)

			walletModel, err := getWalletForName(t, configPath, walletName)
			require.Nil(t, err)

			shareParams := map[string]interface{}{
				"allocation":          allocationID,
				"remotepath":          remotePath,
				"clientid":            walletModel.ClientID,
				"encryptionpublickey": walletModel.EncryptionPublicKey,
			}
			output, err = shareFile(t, configPath, shareParams)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

			authTicket, err := extractAuthToken(output[0])
			require.Nil(t, err, "Error extracting auth token")
			require.NotEqual(t, "", authTicket)

			recipients[i] = recipientInfo{
				walletName: walletName,
				authTicket: authTicket,
			}
		}

		// Verify each recipient can download
		for i, r := range recipients {
			localFile := generateRandomTestFileName(t)

			downloadParams := createParams(map[string]interface{}{
				"localpath":  localFile,
				"authticket": r.authTicket,
			})
			output, err = downloadFileForWallet(t, r.walletName, configPath, downloadParams, false)
			require.Nil(t, err, fmt.Sprintf("Recipient %d failed to download: %s", i, strings.Join(output, "\n")))
			require.Len(t, output, 2, fmt.Sprintf("Recipient %d download - Unexpected output: %s", i, strings.Join(output, "\n")))
			require.Contains(t, output[1], StatusCompletedCB)
			require.Contains(t, output[1], filepath.Base(file))
		}
	})

	// 5. Revoke public share — share publicly, verify download works, revoke, verify download fails
	t.RunWithTimeout("Revoke public share should prevent further downloads", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// Share publicly
		shareParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Verify download works before revocation
		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

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

		// Revoke the public share
		revokeParams := map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"revoke":     "",
		}
		output, err = shareFile(t, configPath, revokeParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "revoke share - Unexpected output", strings.Join(output, "\n"))

		// Verify download fails after revocation
		os.Remove(file)

		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, true)
		require.NotNil(t, err, "Expected error after revocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "consensus_not_met")
	})

	// 6. Remove recipient from group share — share to 3 recipients, remove one, verify removed can't download but others can
	t.RunWithTimeout("Remove one recipient from group share should only block that recipient", 6*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// Create 3 recipients and share to each
		type recipientInfo struct {
			walletName string
			clientID   string
			encKey     string
			authTicket string
		}

		recipients := make([]recipientInfo, 3)
		for i := 0; i < 3; i++ {
			walletName := fmt.Sprintf("%s_rcpt_%d", escapedTestName(t), i)
			createWalletForName(walletName)

			walletModel, err := getWalletForName(t, configPath, walletName)
			require.Nil(t, err)

			shareParams := map[string]interface{}{
				"allocation":          allocationID,
				"remotepath":          remotePath,
				"clientid":            walletModel.ClientID,
				"encryptionpublickey": walletModel.EncryptionPublicKey,
			}
			output, err = shareFile(t, configPath, shareParams)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

			authTicket, err := extractAuthToken(output[0])
			require.Nil(t, err, "Error extracting auth token")
			require.NotEqual(t, "", authTicket)

			recipients[i] = recipientInfo{
				walletName: walletName,
				clientID:   walletModel.ClientID,
				encKey:     walletModel.EncryptionPublicKey,
				authTicket: authTicket,
			}
		}

		// Revoke share for recipient 1 (middle one)
		revokeParams := map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          remotePath,
			"clientid":            recipients[1].clientID,
			"encryptionpublickey": recipients[1].encKey,
			"revoke":              "",
		}
		output, err = shareFile(t, configPath, revokeParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "revoke share - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Share revoked for client "+recipients[1].clientID, output[0])

		// Recipient 0 should still be able to download
		localFile0 := generateRandomTestFileName(t)
		downloadParams0 := createParams(map[string]interface{}{
			"localpath":  localFile0,
			"authticket": recipients[0].authTicket,
		})
		output, err = downloadFileForWallet(t, recipients[0].walletName, configPath, downloadParams0, false)
		require.Nil(t, err, fmt.Sprintf("Recipient 0 should still download: %s", strings.Join(output, "\n")))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)

		// Recipient 1 (revoked) should fail to download
		localFile1 := generateRandomTestFileName(t)
		downloadParams1 := createParams(map[string]interface{}{
			"localpath":  localFile1,
			"authticket": recipients[1].authTicket,
		})
		output, err = downloadFileForWallet(t, recipients[1].walletName, configPath, downloadParams1, false)
		require.NotNil(t, err, "Revoked recipient should fail to download", strings.Join(output, "\n"))
		aggregatedOutput := strings.Join(output, " ")
		require.Contains(t, aggregatedOutput, "consensus_not_met")

		// Recipient 2 should still be able to download
		localFile2 := generateRandomTestFileName(t)
		downloadParams2 := createParams(map[string]interface{}{
			"localpath":  localFile2,
			"authticket": recipients[2].authTicket,
		})
		output, err = downloadFileForWallet(t, recipients[2].walletName, configPath, downloadParams2, false)
		require.Nil(t, err, fmt.Sprintf("Recipient 2 should still download: %s", strings.Join(output, "\n")))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
	})

	// 7a. Edge case: share non-existent file should fail
	t.RunWithTimeout("Share non-existent file should fail", 3*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          "/does_not_exist.txt",
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
		}
		output, err := shareFile(t, configPath, shareParams)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "Expected error output")
	})

	// 7b. Edge case: share with expired auth ticket should fail to download
	t.RunWithTimeout("Expired auth ticket should fail to download", 4*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Share with short expiration (10 seconds)
		shareParams := map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         remotePath,
			"expiration-seconds": 10,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Wait for auth ticket to expire
		cliutils.Wait(t, 10*time.Second)

		// Try to download - should fail
		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

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

	// 7c. Edge case: share file to self (owner downloads using own auth ticket)
	t.RunWithTimeout("Share file to self should allow owner to download via auth ticket", 4*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Get owner wallet info
		walletOwnerModel, err := getWalletForName(t, configPath, walletOwner)
		require.Nil(t, err)

		// Share to self
		shareParams := map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          remotePath,
			"clientid":            walletOwnerModel.ClientID,
			"encryptionpublickey": walletOwnerModel.EncryptionPublicKey,
		}
		output, err = shareFile(t, configPath, shareParams)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "share file - Unexpected output", strings.Join(output, "\n"))

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		// Owner downloads using auth ticket
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, walletOwner, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))
	})
}
