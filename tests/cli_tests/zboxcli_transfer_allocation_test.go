package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestTransferAllocation(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	// todo: some allocation transfer operations are very slow
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("transfer allocation by owner should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err := registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0])
	})

	t.Run("transfer allocation by non-owner", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		nonOwner := escapedTestName(t) + "_NON_OWNER"

		output, err := registerWalletForName(t, configPath, nonOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		nonOwnerWallet, err := getWalletForName(t, configPath, nonOwner)
		require.Nil(t, err, "Error occurred when retrieving non-owner wallet")

		output, err = transferAllocationOwnershipWithWallet(t, nonOwner, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": nonOwnerWallet.ClientPublicKey,
			"new_owner":     nonOwnerWallet.ClientID,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		reg := regexp.MustCompile("Error transferring allocation:allocation_updating_failed: only owner can update the allocation")
		require.Regexp(t, reg, output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	//todo: unacceptably slow

	t.RunWithTimeout("transfer allocation and download non-encrypted file", 6*time.Minute, func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, newOwner, configPath, 1)
		require.Nil(t, err, "faucet execution failed for non-owner wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))

		transferred := pollForAllocationTransferToEffect(t, newOwner, allocationID)
		require.True(t, transferred, "allocation was not transferred to new owner within time allotted")

		output, err = readPoolLockWithWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"tokens": 0.5,
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "read pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "read pool lock - Unexpected output", strings.Join(output, "\n"))

		downloadFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/"
		os.Remove(downloadFilePath + "/" + filename)

		output, err = downloadFileForWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"localpath":  downloadFilePath,
			"remotepath": remotePath,
		}), true)
		require.Nil(t, err, "Error in file operation", strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output length", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(filename))
	}) //todo:slow

	t.RunWithTimeout("transfer allocation and download with auth ticket should fail", 6*time.Minute, func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))

		// share publicly
		output, err = shareFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
		})
		require.Nil(t, err, "Error:", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEqual(t, "", authTicket)

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, newOwner, configPath, 1)
		require.Nil(t, err, "faucet execution failed for non-owner wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))

		transferred := pollForAllocationTransferToEffect(t, newOwner, allocationID)
		require.True(t, transferred, "allocation was not transferred to new owner within time allotted")

		readPoolParams := createParams(map[string]interface{}{
			"tokens": 0.5,
		})
		output, err = readPoolLock(t, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "read pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "read pool lock - Unexpected output", strings.Join(output, "\n"))

		output, err = readPoolLockWithWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"tokens": 0.5,
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "read pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "read pool lock - Unexpected output", strings.Join(output, "\n"))

		downloadFilePath := strings.TrimSuffix(os.TempDir(), "/") + "/"
		os.Remove(downloadFilePath + "/" + filename)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"localpath":  downloadFilePath,
			"authticket": authTicket,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3, "download file - Unexpected output", strings.Join(output, "\n"))
		aggregatedOutput := strings.ToLower(strings.Join(output, " "))
		require.Contains(t, aggregatedOutput, "failed")
	}) //todo:slow

	t.RunWithTimeout("transfer allocation and update allocation", 6*time.Minute, func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(4096),
		})

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, newOwner, configPath, 1)
		require.Nil(t, err, "faucet execution failed for non-owner wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))

		transferred := pollForAllocationTransferToEffect(t, newOwner, allocationID)
		require.True(t, transferred, "allocation was not transferred to new owner within time allotted")

		output, err = writePoolLockWithWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.5,
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "write pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "write pool lock - Unexpected output", strings.Join(output, "\n"))

		output, err = updateAllocationWithWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "24h",
		}), true)
		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1, "update allocation - Unexpected output", strings.Join(output, "\n"))
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
	}) //todo:slow

	t.Run("transfer allocation with no allocation param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"new_owner_key": ownerWallet.ClientPublicKey,
			"new_owner":     ownerWallet.ClientID,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: allocation flag is missing", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation with no new_owner_key param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation": "dummy",
			"new_owner":  ownerWallet.ClientID,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: new_owner_key flag is missing", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation with no new_owner param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    "dummy",
			"new_owner_key": ownerWallet.ClientPublicKey,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error: new_owner flag is missing", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation with invalid allocation param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    "badallocationid",
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error transferring allocation:couldnt_find_allocation: Couldn't find the allocation required for update", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	// FIXME is this expected to fail?
	t.Run("transfer allocation with invalid new_owner param", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err := registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     "badclientid",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to badclientid", allocationID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	// FIXME is this expected to fail?
	t.Run("transfer allocation with invalid new_owner_key param should fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err := registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": "badclientpubkey",
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})
}

func transferAllocationOwnership(t *test.SystemTest, param map[string]interface{}, retry bool) ([]string, error) {
	return transferAllocationOwnershipWithWallet(t, escapedTestName(t), param, retry)
}

func transferAllocationOwnershipWithWallet(t *test.SystemTest, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Transferring allocation ownership...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox transferallocation %s --silent --wallet %s --configDir ./config --config %s",
		p,
		walletName+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func pollForAllocationTransferToEffect(t *test.SystemTest, newOwner, allocationID string) bool {
	t.Logf("Polling for 5 minutes until allocation ownership changed...")
	timeout := time.After(time.Minute * 5)

	// this requires the allocation has file uploaded to work properly.
	for {
		// using `list all` to verify transfer as this check blobber content as opposed to `get allocation` which is based on sharder
		output, err := listAllWithWallet(t, newOwner, configPath, allocationID, true)

		// if not empty, the transfer of allocation contents has occurred on blobbers.
		// there is only one content expected so once it is no longer empty, transfer is deemed complete.
		if err == nil && len(output) == 1 && output[0] != "[]" {
			return true
		}

		// on timeout, exit with failed transfer allocation.
		// otherwise, wait and try again
		select {
		case <-timeout:
			return false
		default:
			cliutils.Wait(t, time.Second*10)
		}
	}
}
