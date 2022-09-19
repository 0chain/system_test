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

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
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
				Value:           tokenomics.IntToZCN(0.1),
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

		oldBlobberID := getFirstUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              scRestGetAllocation.ID,
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
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

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
		require.True(t, isBlobberExist(newBlobberID, scRestGetAllocation.Blobbers))
	})

	t.Run("Replace blobber with the same one in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
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
				Value:           tokenomics.IntToZCN(0.1),
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

		oldBlobberID := getFirstUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              scRestGetAllocation.ID,
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
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxUnsuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

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

	t.Run("Replace blobber with incorrect blobber ID of an old blobber, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
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
				Value:           tokenomics.IntToZCN(0.1),
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

		newBlobberID := getNotUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, newBlobberID, "Old blobber ID contains zero value")

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              scRestGetAllocation.ID,
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
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxUnsuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

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

	t.Run("Check token accounting of a blobber replacing in allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
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
				Value:           tokenomics.IntToZCN(0.1),
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

		oldBlobberID := getFirstUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		clientGetBalanceResponse, resp, err := apiClient.V1ClientGetBalance(
			model.ClientGetBalanceRequest{
				ClientID: wallet.ClientID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, clientGetBalanceResponse)

		balanceBeforeAllocationUpdate := clientGetBalanceResponse.Balance

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              scRestGetAllocation.ID,
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
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxUnsuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

		clientGetBalanceResponse, resp, err = apiClient.V1ClientGetBalance(
			model.ClientGetBalanceRequest{
				ClientID: wallet.ClientID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, clientGetBalanceResponse)

		balanceAfterAllocationUpdate := clientGetBalanceResponse.Balance

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
		require.Greater(t, balanceBeforeAllocationUpdate, balanceAfterAllocationUpdate)
	})
}
