package api_tests

import (
	"github.com/0chain/gosdk/zboxcore/blockchain"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAddBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Add new blobber to allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		transactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData()},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionPutResponse)

		transactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: transactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionGetConfirmationResponse)

		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 1, 1, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore+1)
	})

	t.Run("Add new blobber without provided blobber ID to allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		transactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData()},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionPutResponse)

		transactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: transactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionGetConfirmationResponse)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		allocationUpdate := getAllocationUpdate(allocation.ID, "", "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api_client.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Add new blobber with incorrect ID to allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		faucetTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData()},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, faucetTransactionPutResponse)

		faucetTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: faucetTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, faucetTransactionGetConfirmationResponse)

		scRestGetAllocationBlobbersResponse, resp, err := apiClient.V1SCRestGetAllocationBlobbers(
			&model.SCRestGetAllocationBlobbersRequest{
				ClientID:  wallet.ClientID,
				ClientKey: wallet.ClientKey,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocationBlobbersResponse)

		createAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.StorageSmartContractAddress,
				TransactionData: model.NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createAllocationTransactionPutResponse)

		createAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createAllocationTransactionGetConfirmationResponse)

		scRestGetAllocation, resp, err := apiClient.V1SCRestGetAllocation(
			model.SCRestGetAllocationRequest{
				AllocationID: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocation)

		numberOfBlobbersBefore := len(scRestGetAllocation.Blobbers)

		//allocationUpdate := getAllocationUpdate(allocation.ID, strconv.Itoa(rand.Intn(10)), "")
		//updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		//require.NotNil(t, updateAllocationTransactionResponse)
		//require.Equal(t, api_client.TxUnsuccessfulStatus, confirmation.Status)

		scRestGetAllocation, resp, err = apiClient.V1SCRestGetAllocation(
			model.SCRestGetAllocationRequest{
				AllocationID: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocation)

		numberOfBlobbersAfter := len(scRestGetAllocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Add blobber which already exists in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		transactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData()},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionPutResponse)

		transactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: transactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionGetConfirmationResponse)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, oldBlobberID, "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api_client.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
}

////Returns new "AllocationUpdate" model for testing
//func getAllocationUpdate(allocationID string, newBlobberID, oldBlobberID string) *model.AllocationUpdate {
//	return &model.AllocationUpdate{
//		ID:              allocationID,
//		AddBlobberId:    newBlobberID,
//		RemoveBlobberId: oldBlobberID,
//	}
//}

//func updateAllocation(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair, allocationUpdate *model.AllocationUpdate) (*model.TransactionResponse, *model.Confirmation) {
//	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "update_allocation_request", InputArgs: allocationUpdate})
//	require.Nil(t, err)
//
//	updateAllocationRequest := model.Transaction{
//		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
//		TxnOutputHash:    "",
//		TransactionValue: 1000000000,
//		TransactionType:  1000,
//		TransactionFee:   0,
//		TransactionData:  string(txnDataString),
//		ToClientId:       api_client.StorageSmartContractAddress,
//		CreationDate:     time.Now().Unix(),
//		ClientId:         wallet.ClientID,
//		Version:          "1.0",
//		TransactionNonce: wallet.Nonce + 1,
//	}
//
//	addBlobberTransaction := executeTransaction(t, &updateAllocationRequest, keyPair)
//	confirmation, _ := confirmTransaction(t, wallet, addBlobberTransaction.Entity, 1*time.Minute)
//
//	return addBlobberTransaction, confirmation
//}

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
