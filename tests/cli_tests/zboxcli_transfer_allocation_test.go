package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestTransferAllocation(t *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t.Parallel()

	t.Run("transfer allocation by curator should work", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		curatorWalletName := escapedTestName(t) + "CURATOR"
		output, err := registerWalletForName(t, configPath, curatorWalletName)
		require.Nil(t, err, "Error occurred when retrieving owner wallet", strings.Join(output, "\n"))

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting owner wallet")

		curatorWallet, err := getWalletForName(t, configPath, curatorWalletName)
		require.Nil(t, err, "error getting curator wallet")

		output, err = addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    curatorWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, curatorWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnershipWithWallet(t, curatorWalletName, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation by owner should work", func(t *testing.T) {
		t.Parallel()

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

	t.Run("transfer allocation by non-owner and non-curator should fail", func(t *testing.T) {
		t.Parallel()

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
		reg := regexp.MustCompile("Error adding curator:curator_transfer_allocation_failed: only curators or the owner can transfer allocations; [a-z0-9]{64} is neither")
		require.Regexp(t, reg, output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation to self", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": ownerWallet.ClientPublicKey,
			"new_owner":     ownerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, ownerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer an expired allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		// expire the allocation
		output, err = updateAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "-1h",
		}), true)
		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1, "update allocation - Unexpected output", strings.Join(output, "\n"))
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		alloc := getAllocation(t, allocationID)
		require.False(t, alloc.Finalized)

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
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
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer a canceled allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "cancel allocation - Unexpected output", strings.Join(output, "\n"))
		require.Regexp(t, "Allocation canceled with txId : [0-9a-f]+", output[0],
			"cancel allocation - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
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
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer a finalized allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		// expire the allocation first
		output, err = updateAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "-1h",
		}), true)
		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1, "update allocation - Unexpected output", strings.Join(output, "\n"))
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// FIXME this does not work at the moment
		output, err = finalizeAllocation(t, configPath, allocationID, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "finalize allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: allocation is not expired yet, or waiting a challenge completion", output[0],
			"finalize allocation - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		// FIXME should fail with finalized allocation
		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation and download non-encrypted file", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err = uploadFile(t, configPath, map[string]interface{}{
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
			"allocation": allocationID,
			"tokens":     0.5,
			"duration":   "1h",
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
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(filename), output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
	})

	// FIXME was expecting it to work given the change of allocation ownership
	t.Run("transfer allocation and download encrypted file should fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(204800),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, 25600) // cannot be a small file when uploading with encrypt
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
			"encrypt":    "",
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
			"allocation": allocationID,
			"tokens":     0.5,
			"duration":   "1h",
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
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1], "download file - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation and download with auth ticket should fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err = uploadFile(t, configPath, map[string]interface{}{
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

		output, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.5,
			"duration":   "1h",
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "read pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "read pool lock - Unexpected output", strings.Join(output, "\n"))

		output, err = readPoolLockWithWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.5,
			"duration":   "1h",
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
		require.Contains(t, aggregatedOutput, "owner id mismatch")

		/* Authticket is redundant for owner and collaborator
		output, err = downloadFileForWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"localpath":  downloadFilePath,
			"authticket": authTicket,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: File content didn't match with uploaded file", output[1],
			"download file - Unexpected output", strings.Join(output, "\n"))
		*/
	})

	t.Run("transfer allocation and update allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(4096),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err = uploadFile(t, configPath, map[string]interface{}{
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
			"duration":   "1h",
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
	})

	t.Run("transfer allocation with no allocation param should fail", func(t *testing.T) {
		t.Parallel()

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
		// FIXME error should be missing allocation param
		require.Equal(t, "Error: curator flag is missing", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation with no new_owner_key param should fail", func(t *testing.T) {
		t.Parallel()

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
		// FIXME error should be missing new_owner_key param
		require.Equal(t, "Error: curator flag is missing", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation with no new_owner param should fail", func(t *testing.T) {
		t.Parallel()

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
		// FIXME error should be missing new_owner param
		require.Equal(t, "Error: curator flag is missing", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation with invalid allocation param should fail", func(t *testing.T) {
		t.Parallel()

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
		require.Equal(t, "Error adding curator:curator_transfer_allocation_failed: value not present", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	// FIXME is this expected to fail?
	t.Run("transfer allocation with invalid new_owner param", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
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
	t.Run("transfer allocation with invalid new_owner_key param should fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
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

	t.Run("transfer allocation and upload file", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(20480),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, 256)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err = uploadFile(t, configPath, map[string]interface{}{
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
			"duration":   "24h",
		}), false)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "write pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "write pool lock - Unexpected output", strings.Join(output, "\n"))

		output, err = uploadFileForWallet(t, newOwner, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/new" + remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))
	})
}

func transferAllocationOwnership(t *testing.T, param map[string]interface{}, retry bool) ([]string, error) {
	return transferAllocationOwnershipWithWallet(t, escapedTestName(t), param, retry)
}

func transferAllocationOwnershipWithWallet(t *testing.T, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
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

func pollForAllocationTransferToEffect(t *testing.T, newOwner, allocationID string) bool {
	t.Logf("Polling for 5 minutes until allocation ownership changed...")
	timeout := time.After(time.Minute * 5)

	// this requires the allocation has file uploaded to work properly.
	for {
		// using `list all` to verify transfer as this check blobber content as opposed to `get allocation` which is based on sharder
		output, err := listAllWithWallet(t, newOwner, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// if not empty, the transfer of allocation contents has occurred on blobbers.
		// there is only one content expected so once it is no longer empty, transfer is deemed complete.
		if output[0] != "[]" {
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
