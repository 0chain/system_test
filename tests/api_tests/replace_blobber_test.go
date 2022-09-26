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

func TestReplaceBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Replace blobber in allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		apiClient.ExecuteFaucetWrapper(t, wallet)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              allocationID,
					AddBlobberId:    newBlobberID,
					RemoveBlobberId: oldBlobberID,
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

		allocation = apiClient.GetAllocationWrapper(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.True(t, isBlobberExist(newBlobberID, allocation.Blobbers))

	})

	t.Run("Replace blobber with the same one in allocation, shouldn't work", func(t *testing.T) {
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
					RemoveBlobberId: oldBlobberID,
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

	t.Run("Replace blobber with incorrect blobber ID of an old blobber, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		apiClient.ExecuteFaucetWrapper(t, wallet)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "Old blobber ID contains zero value")

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              allocationID,
					AddBlobberId:    newBlobberID,
					RemoveBlobberId: strconv.Itoa(rand.Intn(10)),
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

	t.Run("Check token accounting of a blobber replacing in allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		apiClient.ExecuteFaucetWrapper(t, wallet)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		walletBalance := apiClient.GetWalletBalanceWrapper(t, wallet)
		balanceBeforeAllocationUpdate := walletBalance.Balance

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              allocationID,
					AddBlobberId:    newBlobberID,
					RemoveBlobberId: oldBlobberID,
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

		walletBalance = apiClient.GetWalletBalanceWrapper(t, wallet)
		balanceAfterAllocationUpdate := walletBalance.Balance

		allocation = apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.Greater(t, balanceBeforeAllocationUpdate, balanceAfterAllocationUpdate)
	})
}
