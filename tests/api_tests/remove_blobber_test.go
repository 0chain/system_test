package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRemoveBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Remove blobber in allocation, shouldn't work", func(t *testing.T) {
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
					AddBlobberId:    "",
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

		allocation = apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
}
