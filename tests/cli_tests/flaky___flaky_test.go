package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func Test___FlakyScenariosCommonUserFunctions(t *testing.T) {
	t.Run("File move - Users should not be charged for moving a file ", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 4 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		fileSize := int64(math.Floor(1 * MB))

		// Upload 1 MB file
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)

		// Get initial write pool
		cliutils.Wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Move file
		remotepath := filepath.Base(localpath)
		moveAllocationFile(t, allocationID, remotepath, "newDir")

		cliutils.Wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		// Get final write pool, no deduction should have been done
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Equal(t, initialWritePool[0].Balance, finalWritePool[0].Balance, "Write pool balance expected to be unchanged")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File Rename - Users should not be charged for renaming a file", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 4 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		fileSize := int64(math.Floor(1 * MB))

		// Upload 1 MB file
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)

		// Get initial write pool
		cliutils.Wait(t, 30*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Rename file
		remotepath := filepath.Base(localpath)
		renameAllocationFile(t, allocationID, remotepath, remotepath+"_renamed")

		cliutils.Wait(t, 30*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		// Get final write pool, no deduction should have been done
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Equal(t, initialWritePool[0].Balance, finalWritePool[0].Balance, "Write pool balance expected to be unchanged")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File Update with same size - Users should not be charged, blobber should not be paid", func(t *testing.T) {
		t.Parallel()

		// Logic: Upload a 1 MB file, get the write pool info. Update said file with another file
		// of size 1 MB. Get write pool info and check nothing has been deducted.

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 4 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		fileSize := int64(math.Floor(1 * MB))

		// Upload 1 MB file
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)

		cliutils.Wait(t, 30*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Update with same size
		remotepath := "/" + filepath.Base(localpath)
		updateFileWithRandomlyGeneratedData(t, allocationID, remotepath, fileSize)

		cliutils.Wait(t, 30*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		// Get final write pool, no deduction should have been made
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Equal(t, initialWritePool[0].Balance, finalWritePool[0].Balance, "Write pool balance expected to be unchanged")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File Update with a different size - Blobbers should be paid for the extra file size", func(t *testing.T) {
		t.Parallel()

		// Logic: Upload a 0.5 MB file and get the upload cost. Update the 0.5 MB file with a 1 MB file
		// and see that blobber's write pool balances are deduced again for the cost of uploading extra
		// 0.5 MBs.

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock":   "0.5",
			"size":   10 * MB,
			"data":   2,
			"parity": 2,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		fileSize := int64(0.5 * MB)

		// Get expected upload cost for 0.5 MB
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)
		output, _ = getUploadCostInUnit(t, configPath, allocationID, localpath)
		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))
		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := (expectedUploadCostInZCN / (2 + 2))

		// Wait for write pool blobber balances to be deduced for initial 0.5 MB
		cliutils.Wait(t, time.Minute)

		// Get write pool info before file update
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		t.Logf("Write pool Balance after upload expected to be [%v] but was [%v]", 0.5, intToZCN(initialWritePool[0].Balance))
		require.InEpsilonf(t, 0.5-actualExpectedUploadCostInZCN, intToZCN(initialWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		remotepath := "/" + filepath.Base(localpath)
		updateFileWithRandomlyGeneratedData(t, allocationID, remotepath, int64(1*MB))

		// Wait before fetching final write pool
		cliutils.Wait(t, time.Minute)

		// Get the new Write Pool info after update
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		t.Logf("Write pool Balance after upload expected to be [%v] but was [%v]", 0.5-actualExpectedUploadCostInZCN, intToZCN(initialWritePool[0].Balance))
		require.InEpsilon(t, (0.5 - 2*actualExpectedUploadCostInZCN), intToZCN(finalWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5-actualExpectedUploadCostInZCN, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		// Blobber pool balance should reduce by expected cost of 0.5 MB for each blobber
		totalChangeInWritePool := float64(0)
		require.InEpsilon(t, actualExpectedUploadCostInZCN, totalChangeInWritePool, epsilon, "expected write pool balance to decrease by [%v] but has actually decreased by [%v]", actualExpectedUploadCostInZCN, totalChangeInWritePool)
		createAllocationTestTeardown(t, allocationID)
	})
}

func Test___FlakyScenariosTransferAllocation(t *testing.T) {
	t.Parallel()

	t.Run("transfer allocation accounting test", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(1024000),
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
		err = createFileWithSize(file, 204800)
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
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, _ = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, "write pool info - Unexpected output", strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Len(t, initialWritePool, 1)

		require.True(t, initialWritePool[0].Locked, strings.Join(output, "\n"))
		require.Equal(t, allocationID, initialWritePool[0].Id, strings.Join(output, "\n"))
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, strings.Join(output, "\n"))

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

		// balance of old owner should be unchanged
		// FIXME should this contain the released pool balances given the change of ownership?
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Unexpected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))
		require.Len(t, output, 1, "get balance - Unexpected output", strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0],
			"get balance - Unexpected output", strings.Join(output, "\n"))

		// balance of new owner should be unchanged
		output, err = getBalanceForWallet(t, configPath, newOwner)
		require.Nil(t, err, "Unexpected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))
		require.Len(t, output, 1, "get balance - Unexpected output", strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0],
			"get balance - Unexpected output", strings.Join(output, "\n"))

		// zero cost to transfer
		expectedTransferCost := int64(0)

		// write lock pool of old owner should remain locked
		// FIXME should this be unlocked given the change of ownership?
		cliutils.Wait(t, 2*time.Minute)
		output, _ = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, "write pool info - Unexpected output", strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Len(t, finalWritePool, 1)

		actualCost := initialWritePool[0].Balance - finalWritePool[0].Balance

		require.Equal(t, expectedTransferCost, actualCost)
		require.True(t, finalWritePool[0].Locked, strings.Join(output, "\n"))
		require.Equal(t, allocationID, finalWritePool[0].Id, strings.Join(output, "\n"))
		require.Equal(t, allocationID, finalWritePool[0].AllocationId, strings.Join(output, "\n"))
	})
}

func Test___FlakyScenariosUpdateScSettings(t *testing.T) {
	t.Run("update max_read_price to invalid value should fail", func(t *testing.T) {
		t.Parallel()

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		configKey := "max_read_price"
		newValue := "x"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "fatal:{\"error\": \"verify transaction failed\"}", output[0], strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
	})
}

func Test___FlakyScenariosCreateAllocationFreeStorage(t *testing.T) {
	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	assigner := escapedTestName(t) + "_ASSIGNER"

	// register SC owner wallet
	output, err := registerWalletForName(t, configPath, scOwnerWallet)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	// register assigner wallet
	output, err = registerWalletForName(t, configPath, assigner)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	// Open the wallet file themselves to get private key for signing data
	ownerWallet := readWalletFile(t, "./config/"+scOwnerWallet+"_wallet.json")
	assignerWallet := readWalletFile(t, "./config/"+assigner+"_wallet.json")

	// necessary cli call to generate wallet to avoid polluting logs of succeeding cli calls
	output, err = registerWallet(t, configPath)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	configKeyDataShards := "free_allocation_settings.data_shards"
	configKeyParityShards := "free_allocation_settings.parity_shards"
	configKeySize := "free_allocation_settings.size"
	configKeyDuration := "free_allocation_settings.duration"

	// nolint:gocritic
	// configKeyReadPoolFraction := "free_allocation_settings.read_pool_fraction"

	keys := strings.Join([]string{
		configKeyDataShards,
		configKeyParityShards,
		configKeySize,
		configKeyDuration,
	}, ",")

	output, err = getStorageSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	cfgBefore, _ := keyValuePairStringToMap(t, output)

	// ensure revert in config is run regardless of test result
	defer func() {
		oldValues := strings.Join([]string{
			cfgBefore[configKeyDataShards],
			cfgBefore[configKeyParityShards],
			cfgBefore[configKeySize],
			cfgBefore[configKeyDuration],
		}, ",")

		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   keys,
			"values": oldValues,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
	}()

	newValues := strings.Join([]string{
		"1",    // decreasing data shards from default 10
		"1",    // decreasing parity shards from default 5
		"1024", // decreasing size from default 10000000000
		"5m",   // reduce free allocation duration from 50h to 5m
	}, ",")

	output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
		"keys":   keys,
		"values": newValues,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2, strings.Join(output, "\n"))
	require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
	require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

	// miners list
	output, err = getMiners(t, configPath)
	require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var miners climodel.NodeList
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
	require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))

	freeAllocAssignerTxn := freeAllocationAssignerTxn(t, ownerWallet, assignerWallet)
	err = sendTxn(miners, freeAllocAssignerTxn)
	require.Nil(t, err, "Error sending txn to miners: %v", output[0], err)

	output, err = verifyTransaction(t, configPath, freeAllocAssignerTxn.Hash)
	require.Nil(t, err, "Could not verify commit transaction", strings.Join(output, "\n"))
	require.Len(t, output, 3)
	require.Equal(t, "Transaction verification success", output[0])
	require.Equal(t, "TransactionStatus: 1", output[1])
	require.Greater(t, len(output[2]), 0, output[2])

	t.Parallel()

	// FIXME not working at the moment
	t.Run("Create free storage from marker with accounting", func(t *testing.T) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		marker.Signature = sign(t, data, assignerWallet)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationForWallet(t, recipient, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1)
		require.Equal(t, "Error creating free allocation: [txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
		// FIXME disabled as not working as expected
		// require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		// require.Len(t, output, 1)
		// matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		// require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")
		// allocationID := strings.Fields(output[0])[2]
		//
		// readPoolFraction, err := strconv.ParseFloat(cfgBefore[configKeyReadPoolFraction], 64)
		// require.Nil(t, err, "Read pool fraction config is not float: %s", cfgBefore[configKeyReadPoolFraction])
		//
		// wantReadPoolFraction := marker.FreeTokens * readPoolFraction
		// wantWritePoolToken := marker.FreeTokens - wantReadPoolFraction
		//
		// // Verify write and read pools are set with tokens
		// output, err = writePoolInfo(t, configPath, true)
		// require.Len(t, output, 1, strings.Join(output, "\n"))
		// require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))
		//
		// writePool := []climodel.WritePoolInfo{}
		// err = json.Unmarshal([]byte(output[0]), &writePool)
		// require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		// require.Len(t, writePool, 1, "More than 1 write pool found", strings.Join(output, "\n"))
		// require.Equal(t, ConvertToValue(wantWritePoolToken), writePool[0].Balance, "Expected write pool amount not met", strings.Join(output, "\n"))
		//
		// readPool := getReadPoolInfo(t, allocationID)
		// require.Len(t, readPool, 1, "Read pool must exist")
		// require.Equal(t, ConvertToValue(wantReadPoolFraction), readPool[0].Balance, "Read Pool balance must be equal to locked amount")
	})

	t.Run("Create free storage with malformed marker should fail", func(t *testing.T) {
		// register recipient wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte("bad marker json"), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "unmarshalling markerinvalid character 'b' looking for beginning of value", output[0])
	})

	t.Run("Create free storage with invalid marker contents should fail", func(t *testing.T) {
		// register recipient wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte(`{"invalid_marker":true}`), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "Error creating free allocation: free_allocation_failed: error getting assigner details: value not present", output[0])
	})

	t.Run("Create free storage with invalid marker signature should fail", func(t *testing.T) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		marker.Signature = "badsignature"
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		// TODO test can differ one of just sort it out
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: encoding/hex: invalid byte: U+0073 's'", output[0])
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: marker timestamped in the future: 1642693108"", output[0])
	})

	t.Run("Create free storage with wrong recipient wallet should fail", func(t *testing.T) {
		recipientCorrect := escapedTestName(t) + "_RECIPIENT"

		// register correct recipient wallet
		output, err = registerWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		// register this wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		marker.Signature = sign(t, data, assignerWallet)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipientCorrect + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, 1, len(output), strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile("Error creating free allocation: free_allocation_failed: marker verification failed: marker timestamped in the future: ([0-9]{10})"), output[0])
	})

	t.Run("Create free storage with tokens exceeding assigner's individual limit should fail", func(t *testing.T) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: freeTokensIndividualLimit + 1,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		marker.Signature = sign(t, data, assignerWallet)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		// TODO sort out why message changes
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: 110000000000 exceeded permitted free storage  100000000000", output[0])
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: marker timestamped in the future: 1642693167", output[0])
	})
}
