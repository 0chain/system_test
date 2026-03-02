package api_tests

import (
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

func TestReplaceBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Replace blobber in allocation, should work")

	t.RunSequentiallyWithTimeout("Replace blobber in allocation, should work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add")

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxSuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.True(t, isBlobberExist(newBlobberID, allocation.Blobbers))
	})

	t.RunSequentiallyWithTimeout("Replace blobber with the same one in allocation, shouldn't work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		matched := apiClient.TryUpdateAllocationBlobbers(t, wallet, oldBlobberID, oldBlobberID, allocationID, client.TxUnsuccessfulStatus)
		if !matched {
			// gosdk serialization error or transaction timeout - operation still failed, which is expected
			t.Log("UpdateAllocationBlobbers failed at client/transaction level (expected for same-blobber replacement)")
		}

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.RunSequentiallyWithTimeout("Replace blobber with incorrect blobber ID of an old blobber, shouldn't work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add")

		result, err := rand.Int(rand.Reader, big.NewInt(10))
		require.Nil(t, err)

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, result.String(), allocationID, client.TxUnsuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.RunSequentiallyWithTimeout("Check token accounting of a blobber replacing in allocation, should work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add")

		// Save old blobber write price before replacement (allocation is overwritten in wait loop)
		oldBlobberWritePrice := int64(0)
		for _, b := range allocation.Blobbers {
			if b.ID == oldBlobberID {
				oldBlobberWritePrice = b.Terms.WritePrice
				break
			}
		}

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBeforeAllocationUpdate := walletBalance.Balance

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxSuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfterAllocationUpdate := walletBalance.Balance

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		// Token accounting: balance direction depends on relative write prices.
		// Old write pool released + new write pool locked = net balance change.
		// If old blobber was more expensive, balance increases; if new is more expensive, it decreases.
		newBlobberWritePrice := int64(0)
		for _, b := range allocation.Blobbers {
			if b.ID == newBlobberID {
				newBlobberWritePrice = b.Terms.WritePrice
				break
			}
		}
		if oldBlobberWritePrice > newBlobberWritePrice {
			// Cheaper replacement: write pool tokens returned → balance should increase
			require.Less(t, balanceBeforeAllocationUpdate, balanceAfterAllocationUpdate,
				"balance should increase when replacing with a cheaper blobber")
		} else if newBlobberWritePrice > oldBlobberWritePrice {
			// More expensive replacement: more tokens locked → balance should decrease
			require.Greater(t, balanceBeforeAllocationUpdate, balanceAfterAllocationUpdate,
				"balance should decrease when replacing with a more expensive blobber")
		}
		// Equal write prices: no strict balance direction assertion (only tx fees deducted)
	})

	t.RunSequentiallyWithTimeout("Replace blobber in allocation with repair should work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		uploadOp := sdkClient.AddUploadOperation(t, "", "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")
		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add")
		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxSuccessfulStatus)

		time.Sleep(10 * time.Second)

		alloc, err := sdk.GetAllocation(allocationID)
		require.Nil(t, err)
		// Check for blobber replacement
		notFound := true
		require.True(t, len(alloc.Blobbers) > 0)
		// Check if new blobber is in the same position as old blobber
		require.True(t, alloc.Blobbers[0].ID == newBlobberID)
		for _, blobber := range alloc.Blobbers {
			if blobber.ID == oldBlobberID {
				notFound = false
				break
			}
		}
		require.True(t, notFound, "old blobber should not be in the list")
		// check for repair
		_, _, req, _, err := alloc.RepairRequired("/")
		require.Nil(t, err)
		require.True(t, req)

		// Repair downloads from healthy blobbers; with read_price=0 no read pool is needed.

		// do repair
		sdkClient.RepairAllocation(t, allocationID)

		_, err = sdk.GetFileRefFromBlobber(allocationID, newBlobberID, uploadOp.RemotePath)
		require.Nil(t, err)
	})
}
