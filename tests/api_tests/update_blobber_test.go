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

		wallet := apiClient.RegisterWalletWrapper(t, client.HttpOkStatus)
		apiClient.ExecuteFaucetWrapper(t, wallet, client.HttpOkStatus, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers, client.HttpOkStatus, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocationWrapper(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
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
