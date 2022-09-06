package api_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Adding a blobber as additional Parity shard, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		response := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, response)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		blobberRequirements.Blobbers = availableBlobbers
		transactionResponse, _ := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		allocation := getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID)
		updateAllocation(t, registeredWallet, keyPair, allocationUpdate)

		allocation = getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Token accounting of added blobber as additional Parity shard, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		response := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, response)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		blobberRequirements.Blobbers = availableBlobbers
		transactionResponse, _ := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		allocation := getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		fmt.Println(getBalance(t, registeredWallet.Id))

		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID)
		updateAllocation(t, registeredWallet, keyPair, allocationUpdate)

		allocation = getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		fmt.Println(getBalance(t, registeredWallet.Id))

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
}

//Returns new "AllocationUpdate" model for testing
func getAllocationUpdate(allocationID string, newBlobberID string) *model.AllocationUpdate {
	return &model.AllocationUpdate{
		ID:           allocationID,
		AddBlobberId: newBlobberID,
	}
}

func updateAllocation(t *testing.T, wallet *model.Wallet, keyPair model.KeyPair, allocationUpdate *model.AllocationUpdate) (*model.TransactionResponse, *model.Confirmation) {
	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "update_allocation_request", InputArgs: allocationUpdate})
	require.Nil(t, err)
	updateAllocationRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 1000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  string(txnDataString),
		ToClientId:       STORAGE_SMART_CONTRACT_ADDRESS,
		CreationDate:     time.Now().Unix(),
		ClientId:         wallet.Id,
		Version:          "1.0",
		TransactionNonce: wallet.Nonce + 1,
	}

	addBlobberTransaction := executeTransaction(t, &updateAllocationRequest, keyPair)
	confirmation, _ := confirmTransaction(t, wallet, addBlobberTransaction.Entity, 1*time.Minute)

	return addBlobberTransaction, confirmation
}

//Returns "StorageNode" ID which is not used for created allocation yet, if such one exists
func getNotUsedStorageNodeID(availableStorageNodeIDs *[]string, usedStorageNodes []*model.StorageNode) string {
	for _, availableStorageNodeID := range *availableStorageNodeIDs {
		var found bool
		for _, usedStorageNode := range usedStorageNodes {
			if usedStorageNode.ID == availableStorageNodeID {
				found = true
			}
		}
		if !found {
			return availableStorageNodeID
		}
	}
	return ""
}

func escapedTestName(t *testing.T) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func uploadFile(t *testing.T, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return uploadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func uploadFileForWallet(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Uploading file...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*40)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func uploadFileWithoutRetry(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Uploading file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithoutRetry(cmd)
}

func generateFileAndUpload(t *testing.T, allocationID, remotepath string, size int64) string {
	return generateFileAndUploadForWallet(t, escapedTestName(t), allocationID, remotepath, size)
}

func generateFileAndUploadForWallet(t *testing.T, wallet, allocationID, remotepath string, size int64) string {
	filename := generateRandomTestFileName(t)

	err := createFileWithSize(filename, size)
	require.Nil(t, err)

	// Upload parameters
	uploadWithParamForWallet(t, wallet, configPath, map[string]interface{}{
		"allocation": allocationID,
		"localpath":  filename,
		"remotepath": remotepath + filepath.Base(filename),
	})

	return filename
}
