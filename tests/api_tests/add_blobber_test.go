package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
	"testing"
)

func TestAddBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Add new blobber to allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		apiClient.ExecuteFaucetWrapper(t, wallet)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              allocationID,
					AddBlobberId:    newBlobberID,
					RemoveBlobberId: "",
				}),
				Value: tokenomics.IntToZCN(0.1),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionPutResponse)

		updateAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: allocationID,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

		allocation = apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore+1)
	})

	t.Run("Add new blobber without provided blobber ID to allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		apiClient.ExecuteFaucetWrapper(t, wallet)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              allocationID,
					AddBlobberId:    "",
					RemoveBlobberId: "",
				}),
				Value: tokenomics.IntToZCN(0.1),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionPutResponse)

		updateAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: allocationID,
			},
			client.HttpOkStatus,
			client.TxUnsuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

		allocation = apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Add new blobber with incorrect ID to allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		apiClient.ExecuteFaucetWrapper(t, wallet)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              allocationID,
					AddBlobberId:    strconv.Itoa(rand.Intn(10)),
					RemoveBlobberId: "",
				}),
				Value: tokenomics.IntToZCN(0.1),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionPutResponse)

		updateAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: allocationID,
			},
			client.HttpOkStatus,
			client.TxUnsuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

		allocation = apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Add blobber which already exists in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		apiClient.ExecuteFaucetWrapper(t, wallet)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              allocationID,
					AddBlobberId:    oldBlobberID,
					RemoveBlobberId: "",
				}),
				Value: tokenomics.IntToZCN(0.1),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionPutResponse)

		updateAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: allocationID,
			},
			client.HttpOkStatus,
			client.TxUnsuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

		allocation = apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
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

func isBlobberExist(blobberID string, blobbers []*model.StorageNode) bool {
	return getBlobberURL(blobberID, blobbers) != ""
}
