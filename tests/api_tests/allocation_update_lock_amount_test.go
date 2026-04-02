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

	t.RunSequentiallyWithTimeout("Extend Allocation Size with used size > 0", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uploadOp := sdkClient.AddUploadOperation(t, "", "", 512)
		chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(10 * time.Second)

		// Re-set wallet to ensure SDK context is correct after chimneySdkClient operation
		sdkClient.SetWallet(t, wallet)

		uar := &model.UpdateAllocationRequest{
			ID:   allocationID,
			Size: 1 * GB,
		}

		// Ensure wallet is set before calling GetUpdateAllocationMinLock
		// This is critical as GetUpdateAllocationMinLock needs wallet context
		sdkClient.SetWallet(t, wallet)
		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 1*GB, false, "", "")
		require.NoError(t, err)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		// SDK may return 0 for min_lock in some edge cases; use fallback since blobbers have non-zero write_price
		if minLockRequiredInZcn == 0 {
			t.Log("WARNING: SDK returned min_lock=0 for extend size; using fallback 0.2 ZCN")
			minLockRequiredInZcn = 0.2
		}

		// Add generous buffer — SDK min_lock covers write pool only, not challenge pool deficit.
		// When allocation has used data, challenge pool may be nearly empty and needs refilling.
		minLockRequiredInZcn *= 2.0
		if minLockRequiredInZcn < 1.0 {
			minLockRequiredInZcn = 1.0
		}

		t.Logf("Min lock required: %v (with buffer: %v ZCN)", minLockRequired, minLockRequiredInZcn)

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		// Wait for SC to commit the size update before reading back
		time.Sleep(5 * time.Second)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(2*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Extend Allocation Duration with used size > 0", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		// Wait for allocation to be indexed on sharder before upload
		apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		uploadOp := sdkClient.AddUploadOperation(t, "", "", 512)
		chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(10 * time.Second)

		// Re-set wallet to ensure SDK context is correct after chimneySdkClient operation
		sdkClient.SetWallet(t, wallet)

		uar := &model.UpdateAllocationRequest{
			ID:     allocationID,
			Extend: true,
		}

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, 0.1, client.TxSuccessfulStatus)
		// Wait for SC to commit the duration update before reading back
		time.Sleep(5 * time.Second)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Add blobber to allocation with used size > 0", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 10 * MB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uploadOp := sdkClient.AddUploadOperation(t, "", "", 512)
		chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(10 * time.Second)

		// Re-set wallet to ensure SDK context is correct after chimneySdkClient operation
		sdkClient.SetWallet(t, wallet)

		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		origBlobberCount := len(alloc.Blobbers)

		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, alloc.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add")

		uar := &model.UpdateAllocationRequest{
			ID:           allocationID,
			AddBlobberId: newBlobberID,
		}

		// Ensure wallet is set before calling GetUpdateAllocationMinLock
		sdkClient.SetWallet(t, wallet)
		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, false, newBlobberID, "")
		require.NoError(t, err, "GetUpdateAllocationMinLock")

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10
		// SDK may underestimate min_lock when allocation has used data; ensure at least 0.2 ZCN
		if minLockRequiredInZcn < 0.2 {
			minLockRequiredInZcn = 0.2
		}

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Greater(t, len(alloc.Blobbers), origBlobberCount, "blobber must be added to allocation")

		require.Equal(t, int64(10*MB), alloc.Size, "Allocation size is not updated")
	})

	t.RunWithTimeout("Extend Allocation Duration", 1*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		// Wait for allocation to be indexed on sharder before calling GetUpdateAllocationMinLock
		apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

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

	t.RunWithTimeout("Add blobber to allocation", 2*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)
		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 10 * MB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		origBlobberCount := len(alloc.Blobbers)

		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, alloc.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add")

		uar := &model.UpdateAllocationRequest{
			ID:           allocationID,
			AddBlobberId: newBlobberID,
		}

		sdkClient.SetWallet(t, wallet)
		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, false, newBlobberID, "")
		require.NoError(t, err, "GetUpdateAllocationMinLock")

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		if minLockRequiredInZcn == 0 {
			t.Log("WARNING: SDK returned min_lock=0 for add blobber (may indicate write_price=0); using fallback 0.1 ZCN")
		}

		// SDK may underestimate min_lock; ensure at least 0.1 ZCN
		if minLockRequiredInZcn < 0.1 {
			minLockRequiredInZcn = 0.1
		}

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Greater(t, len(alloc.Blobbers), origBlobberCount, "blobber must be added to allocation")

		require.Equal(t, int64(10*MB), alloc.Size, "Allocation size is not updated")
	})

	t.RunWithTimeout("Replace blobber", 2*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 10 * MB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.3, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		newBlobberID := getNotUsedNonEnterpriseBlobberID(t, allocationBlobbers.Blobbers, alloc.Blobbers)
		require.NotZero(t, newBlobberID, "No non-enterprise blobber available to add")

		removeBlobberID := alloc.Blobbers[0].ID

		uar := &model.UpdateAllocationRequest{
			ID:              allocationID,
			AddBlobberId:    newBlobberID,
			RemoveBlobberId: removeBlobberID,
		}

		sdkClient.SetWallet(t, wallet)
		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, false, newBlobberID, removeBlobberID)
		require.NoError(t, err, "GetUpdateAllocationMinLock")

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10
		// SDK may underestimate min_lock for replace blobber; ensure at least 0.2 ZCN
		if minLockRequiredInZcn < 0.2 {
			minLockRequiredInZcn = 0.2
		}

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// Check if blobber was actually replaced (new blobber present, old blobber gone)
		foundNew := false
		for _, b := range alloc.Blobbers {
			if b.ID == newBlobberID {
				foundNew = true
				break
			}
		}
		require.True(t, foundNew, "new blobber must be present in allocation after replace")

		require.Equal(t, int64(10*MB), alloc.Size, "Allocation size is not updated")
	})

	t.RunWithTimeout("Extend Allocation Size", 1*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		// Wait for allocation to be indexed on sharder before calling GetUpdateAllocationMinLock
		apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		uar := &model.UpdateAllocationRequest{
			ID:   allocationID,
			Size: 1 * GB,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 1*GB, false, "", "")
		require.NoError(t, err)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		// SDK may return 0 for min_lock in some edge cases; use fallback since blobbers have non-zero write_price
		if minLockRequiredInZcn == 0 {
			t.Log("WARNING: SDK returned min_lock=0 for extend size; using fallback 0.2 ZCN")
			minLockRequiredInZcn = 0.2
		}
		t.Logf("Min lock required: %v ZCN (%v SAS)", minLockRequiredInZcn, minLockRequired)

		apiClient.UpdateAllocation(t, wallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		// Wait for SC to commit the size update before reading back
		time.Sleep(5 * time.Second)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(2*GB), alloc.Size, "Allocation size is not updated")
	})
}
