package api_tests

import (
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestReplaceBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Replace blobber in allocation, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 1, 1, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, oldBlobberID)
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.True(t, isBlobberExist(newBlobberID, allocation.Blobbers))
	})

	t.Run("Replace blobber with the same one in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 1, 1, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, oldBlobberID, oldBlobberID)
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Replace blobber with incorrect blobber ID of an old blobber, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 1, 1, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "Old blobber ID contains zero value")

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, strconv.Itoa(rand.Intn(10)))
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api.TxUnsuccessfulStatus, confirmation.Status)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Check token accounting of a blobber replacing in allocation, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 1, 1, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		balanceBeforeAllocationUpdate := getBalance(t, registeredWallet.ClientID)
		require.NotNil(t, balanceBeforeAllocationUpdate)

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, oldBlobberID)
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		balanceAfterAllocationUpdate := getBalance(t, registeredWallet.ClientID)
		require.NotNil(t, balanceAfterAllocationUpdate)

		allocation = getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
}
