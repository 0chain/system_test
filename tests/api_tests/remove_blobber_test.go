package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRemoveBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Remove blobber in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbers(t, wallet, "", oldBlobberID, allocationID, client.TxSuccessfulStatus)

		allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
}
