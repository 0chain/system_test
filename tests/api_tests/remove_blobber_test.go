package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRemoveBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Remove blobber in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucet(t, registeredWallet, keyPair)
		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		transactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, "", oldBlobberID)
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, endpoint.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
}
