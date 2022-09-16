package api_tests

import (
	"encoding/json"
	"github.com/0chain/gosdk/zboxcore/blockchain"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAddBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Add new blobber to allocation, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 2, 1, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore+1)
	})

	t.Run("Add new blobber without provided blobber ID to allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		allocationUpdate := getAllocationUpdate(allocation.ID, "", "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, endpoint.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Add new blobber with incorrect ID to allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		allocationUpdate := getAllocationUpdate(allocation.ID, strconv.Itoa(rand.Intn(10)), "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, endpoint.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Add blobber which already exists in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, oldBlobberID, "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, endpoint.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
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

func updateAllocation(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair, allocationUpdate *model.AllocationUpdate) (*model.TransactionResponse, *model.Confirmation) {
	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "update_allocation_request", InputArgs: allocationUpdate})
	require.Nil(t, err)

	updateAllocationRequest := model.Transaction{
		PublicKey:        wallet.ClientKey,
		TxnOutputHash:    "",
		TransactionValue: 1000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  string(txnDataString),
		ToClientId:       endpoint.StorageSmartContractAddress,
		CreationDate:     time.Now().Unix(),
		ClientId:         wallet.ClientID,
		Version:          "1.0",
		TransactionNonce: wallet.Nonce + 1,
	}

	addBlobberTransaction := executeTransaction(t, &updateAllocationRequest, keyPair)
	confirmation, _ := confirmTransaction(t, wallet, addBlobberTransaction.Entity, 5*time.Minute)

	return addBlobberTransaction, confirmation
}

//Returns "StorageNode" ID which is not used for created allocation yet, if such one exists
func getNotUsedStorageNodeID(availableStorageNodeIDs *[]string, usedStorageNodes []*blockchain.StorageNode) string {
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
func getFirstUsedStorageNodeID(availableStorageNodeIDs *[]string, usedStorageNodes []*blockchain.StorageNode) string {
	for _, availableStorageNodeID := range *availableStorageNodeIDs {
		for _, usedStorageNode := range usedStorageNodes {
			if usedStorageNode.ID == availableStorageNodeID {
				return availableStorageNodeID
			}
		}
	}
	return ""
}

func getBlobberURL(blobberID string, blobbers []*blockchain.StorageNode) string {
	for _, blobber := range blobbers {
		if blobber.ID == blobberID {
			return blobber.Baseurl
		}
	}
	return ""
}

func isBlobberExist(blobberID string, blobbers []*blockchain.StorageNode) bool {
	return getBlobberURL(blobberID, blobbers) != ""
}

//Creates new file and fills it with random data
func createFileWithSize(name string, size int64) (*os.File, error) {
	buffer := make([]byte, size)
	_, err := rand.Read(buffer)
	if err != nil {
		return nil, err
	} //nolint:gosec,revive

	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	_, err = file.Write(buffer)
	if err != nil {
		return nil, err
	}
	return file, nil
}
