package api_tests

import (
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

func TestAddBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Add new blobber to allocation, should work")

	t.Parallel()

	t.RunWithTimeout("Add new blobber to allocation, should work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		// Use fewer shards (1+1=2) so there are spare blobbers available to add
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		require.NotNil(t, allocationBlobbers.Blobbers, "Failed to get allocation blobbers")
		require.Greater(t, len(*allocationBlobbers.Blobbers), 0, "No blobbers available for allocation")
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		require.NotNil(t, allocation, "Failed to get allocation")
		require.Greater(t, len(allocation.Blobbers), 0, "Allocation has no blobbers")
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add. Available blobbers: %d, Used blobbers: %d", len(*allocationBlobbers.Blobbers), len(allocation.Blobbers))

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, "", allocationID, client.TxSuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore+1
		})
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore+1)
	})

	t.RunWithTimeout("Add new blobber without provided blobber ID to allocation, shouldn't work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		require.NotNil(t, allocationBlobbers.Blobbers, "Failed to get allocation blobbers")
		require.Greater(t, len(*allocationBlobbers.Blobbers), 0, "No blobbers available for allocation")
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		require.NotNil(t, allocation, "Failed to get allocation")
		require.Greater(t, len(allocation.Blobbers), 0, "Allocation has no blobbers")
		numberOfBlobbersBefore := len(allocation.Blobbers)

		matched := apiClient.TryUpdateAllocationBlobbers(t, wallet, "", "", allocationID, client.TxUnsuccessfulStatus)
		if !matched {
			t.Log("UpdateAllocationBlobbers failed at client/transaction level (expected for empty blobber ID)")
		}

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.RunWithTimeout("Add new blobber with incorrect ID to allocation, shouldn't work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		require.NotNil(t, allocationBlobbers.Blobbers, "Failed to get allocation blobbers")
		require.Greater(t, len(*allocationBlobbers.Blobbers), 0, "No blobbers available for allocation")
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		require.NotNil(t, allocation, "Failed to get allocation")
		require.Greater(t, len(allocation.Blobbers), 0, "Allocation has no blobbers")
		numberOfBlobbersBefore := len(allocation.Blobbers)

		result, err := rand.Int(rand.Reader, big.NewInt(10))
		require.Nil(t, err)

		apiClient.UpdateAllocationBlobbers(t, wallet, result.String(), "", allocationID, client.TxUnsuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.RunWithTimeout("Add blobber which already exists in allocation, shouldn't work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		require.NotNil(t, allocationBlobbers.Blobbers, "Failed to get allocation blobbers")
		require.Greater(t, len(*allocationBlobbers.Blobbers), 0, "No blobbers available for allocation")
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		require.NotNil(t, allocation, "Failed to get allocation")
		require.Greater(t, len(allocation.Blobbers), 0, "Allocation has no blobbers")
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value - allocation has no matching blobbers. Available: %d, Used: %d", len(*allocationBlobbers.Blobbers), len(allocation.Blobbers))

		apiClient.UpdateAllocationBlobbers(t, wallet, oldBlobberID, "", allocationID, client.TxUnsuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})
}

// Returns "StorageNode" ID which is not used for created allocation yet, if such one exists
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

// Returns a non-enterprise blobber ID not used in the allocation
func getNotUsedNonEnterpriseBlobberID(t *test.SystemTest, availableStorageNodeIDs *[]string, usedStorageNodes []*model.StorageNode) string {
	// Get all blobbers to check enterprise status
	allBlobbers, _, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
	if err != nil {
		t.Logf("Warning: could not get all blobbers to filter enterprise: %v", err)
		return getNotUsedStorageNodeID(availableStorageNodeIDs, usedStorageNodes)
	}

	skipIDs := make(map[string]bool)
	for _, b := range allBlobbers {
		if b.IsEnterprise || b.IsRestricted {
			skipIDs[b.ID] = true
		}
	}

	for _, availableStorageNodeID := range *availableStorageNodeIDs {
		if skipIDs[availableStorageNodeID] {
			continue // skip enterprise and restricted blobbers
		}
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

// Returns "StorageNode" ID which is not used for created allocation yet, if such one exists
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
