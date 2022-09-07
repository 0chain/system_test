package api_tests

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestAddBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Add new blobber to allocation, should work", func(t *testing.T) {
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

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, "")
		updateAllocation(t, registeredWallet, keyPair, allocationUpdate)

		allocation = getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Replace blobber in allocation, should work", func(t *testing.T) {
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

		oldBlobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, oldBlobberID)
		updateAllocation(t, registeredWallet, keyPair, allocationUpdate)

		allocation = getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
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

		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, "")
		updateAllocation(t, registeredWallet, keyPair, allocationUpdate)

		fmt.Println(allocation.BlobberDetails)

		allocation = getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)

		const fileName = "test"
		const filePath = "/test"
		const actualSize int64 = 1024

		newFile, err := createFileWithSize(fileName, actualSize)
		require.Nil(t, err, "new file is not created")

		hash, err := crypto.HashOfFile(newFile)
		require.Nil(t, err, "hash for new file is not created")

		newBlobberURL := getBlobberURL(newBlobberID, allocation.Blobbers)
		require.NotZero(t, newBlobberURL, "can't get URL of a new blobber")

		fmt.Println(getBalance(t, registeredWallet.Id))

		fmt.Println(allocation.BlobberDetails)
		blobberUploadFileRequest := model.BlobberUploadFileRequest{
			KeyPair: sys.KeyPair{
				PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
				PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
			},
			URL:          newBlobberURL,
			ClientID:     registeredWallet.Id,
			ClientKey:    registeredWallet.PublicKey,
			AllocationID: allocation.ID,
			File:         newFile,
			Meta: model.BlobberUploadFileMeta{
				ConnectionID: crypto.NewConnectionID(),
				FileName:     fileName,
				FilePath:     filePath,
				ActualHash:   hash,
				ActualSize:   actualSize,
			},
		}
		blobberUploadFileResponse, restyResponse, err := v1BlobberFileUpload(t, blobberUploadFileRequest)
		require.Nil(t, err)
		require.NotNil(t, blobberUploadFileResponse)
		require.NotNil(t, restyResponse)

		fmt.Println(getBalance(t, registeredWallet.Id))

		fmt.Println(string(restyResponse.Body()))
	})
}

//Returns new "AllocationUpdate" model for testing
func getAllocationUpdate(allocationID string, newBlobberID, oldBlobberID string) *model.AllocationUpdate {
	return &model.AllocationUpdate{
		ID:              allocationID,
		AddBlobberId:    newBlobberID,
		RemoveBlobberId: oldBlobberID,
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

//Returns "StorageNode" ID which is not used for created allocation yet, if such one exists
func getFirstUsedStorageNodeID(availableStorageNodeIDs *[]string, usedStorageNodes []*model.StorageNode) string {
	for _, availableStorageNodeID := range *availableStorageNodeIDs {
		for _, usedStorageNode := range usedStorageNodes {
			if usedStorageNode.ID == availableStorageNodeID {
				return availableStorageNodeID
			}
		}
	}
	return ""
}

func getBlobberURL(blobberID string, blobbers []*model.StorageNode) string {
	for _, blobber := range blobbers {
		if blobber.ID == blobberID {
			return blobber.BaseURL
		}
	}
	return ""
}

//Creates new file and fills it with random data
func createFileWithSize(name string, size int64) (*os.File, error) {
	buffer := make([]byte, size)
	_, err := rand.Read(buffer)
	if err != nil {
		return nil, err
	} //nolint:gosec,revive

	err = ioutil.WriteFile(name, buffer, 0777)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return file, nil
}
