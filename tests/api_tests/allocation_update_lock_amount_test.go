package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestAllocationUpdateLockAmount(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Extend Allocation Size with used size > 0", 5*time.Minute, func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uploadOp := sdkClient.AddUploadOperation(t, "", "", 10*MB)
		chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(2 * time.Minute)

		uar := &model.UpdateAllocationRequest{
			ID:   allocationID,
			Size: 1 * GB,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 1*GB, false, "", "")
		require.NoError(t, err)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		require.Greater(t, minLockRequiredInZcn, 0.2, "Min lock required is not correct")

		t.Logf("Min lock required: %v", minLockRequired)

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(2*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Extend Allocation Duration with used size > 0", 5*time.Minute, func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uploadOp := sdkClient.AddUploadOperation(t, "", "", 10*MB)
		chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(2 * time.Minute)

		uar := &model.UpdateAllocationRequest{
			ID:     allocationID,
			Extend: true,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, true, "", "")
		require.NoError(t, err)

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Add blobber to allocation with used size > 0", 5*time.Minute, func(t *test.SystemTest) {

		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uploadOp := sdkClient.AddUploadOperation(t, "", "", 10*MB)
		chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(2 * time.Minute)

		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, alloc.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		uar := &model.UpdateAllocationRequest{
			ID:           allocationID,
			AddBlobberId: newBlobberID,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, false, newBlobberID, "")
		require.NoError(t, err)

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunWithTimeout("Extend Allocation Duration", 1*time.Minute, func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uar := &model.UpdateAllocationRequest{
			ID:     allocationID,
			Extend: true,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, true, "", "")
		require.NoError(t, err)

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		require.Equal(t, float64(0), minLockRequiredInZcn, "Min lock required is not correct")

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunWithTimeout("Add blobber to allocation", 1*time.Minute, func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, alloc.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		uar := &model.UpdateAllocationRequest{
			ID:           allocationID,
			AddBlobberId: newBlobberID,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, false, newBlobberID, "")
		require.NoError(t, err)

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		require.Greater(t, minLockRequiredInZcn, 0.1, "Min lock required should be more than 0.1")
		require.Less(t, minLockRequiredInZcn, 0.105, "Min lock required should be less than 0.105")

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunWithTimeout("Replace blobber", 1*time.Minute, func(t *test.SystemTest) {

		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.3, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, alloc.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		removeBlobberID := alloc.Blobbers[0].ID

		uar := &model.UpdateAllocationRequest{
			ID:              allocationID,
			AddBlobberId:    newBlobberID,
			RemoveBlobberId: removeBlobberID,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, false, newBlobberID, removeBlobberID)
		require.NoError(t, err)

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunWithTimeout("Extend Allocation Size", 1*time.Minute, func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uar := &model.UpdateAllocationRequest{
			ID:   allocationID,
			Size: 1 * GB,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 1*GB, false, "", "")
		require.NoError(t, err)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		require.Equal(t, 0.21, minLockRequiredInZcn, "Min lock required is not correct")

		t.Logf("Min lock required: %v", minLockRequired)

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(2*GB), alloc.Size, "Allocation size is not updated")
	})
}
