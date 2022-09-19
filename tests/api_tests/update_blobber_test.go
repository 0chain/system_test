package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpdateBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Update blobber in allocation without correct delegated client, shouldn't work", func(t *testing.T) {
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

		blobberID := getFirstUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, blobberID)

		scRestGetBlobberResponse, resp, err := apiClient.V1SCRestGetBlobber(
			model.SCRestGetBlobberRequest{
				BlobberID: blobberID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetBlobberResponse)
		require.NotEqual(t, wallet.ClientID, scRestGetBlobberResponse.StakePoolSettings.DelegateWallet)

		updateBlobberTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateBlobberTransactionData(scRestGetBlobberResponse),
				Value:           tokenomics.IntToZCN(0.1),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateBlobberTransactionPutResponse)

		updateBlobberTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: updateBlobberTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxUnsuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateBlobberTransactionGetConfirmationResponse)
	})
}
