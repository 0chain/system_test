package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/stretchr/testify/require"
)

func TestHashnodeRoot(t *testing.T) {
	t.Parallel()

	t.Run("Update blobber in allocation without correct delegated client, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)

		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		blobberRequirements.Blobbers = availableBlobbers
		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status)
		require.NotNil(t, createAllocationTransactionResponse)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		blobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobberRequest := &model.BlobberGetHashnodeRequest{
			AllocationID: allocation.ID,
			URL:          getBlobberURL(blobberID, allocation.Blobbers),
		}

		getBlobberResponse, restyResponse, err := v1BlobberGetHashNodeRoot(t, *blobberRequest)
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.NotNil(t, getBlobberResponse)
		t.Log(getBlobberResponse)
	})
}
