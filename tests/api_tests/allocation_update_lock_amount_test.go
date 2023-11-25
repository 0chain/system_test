package api_tests

import (
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAllocationUpdateLockAmount(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Extend Allocation Size", 1*time.Minute, func(t *test.SystemTest) {
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 10, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
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

		apiClient.UpdateAllocation(t, sdkWallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(2*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Extend Allocation Size where used size is greater than 0", 5*time.Minute, func(t *test.SystemTest) {
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 10, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uploadOp := sdkClient.AddUploadOperation(t, "", 10*MB)
		chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(2 * time.Minute)

		uar := &model.UpdateAllocationRequest{
			ID:   allocationID,
			Size: 1 * GB,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 1*GB, false, "", "")
		require.NoError(t, err)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		require.Greater(t, 0.2, minLockRequiredInZcn, "Min lock required is not correct")

		t.Logf("Min lock required: %v", minLockRequired)

		apiClient.UpdateAllocation(t, sdkWallet, allocationID, uar, 0.2, client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(2*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Extend Allocation Duration", 1*time.Minute, func(t *test.SystemTest) {
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 10, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.16, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uar := &model.UpdateAllocationRequest{
			ID:     allocationID,
			Extend: true,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 0, true, "", "")
		require.NoError(t, err)

		t.Logf("Min lock required: %v", minLockRequired)

		minLockRequiredInZcn := float64(minLockRequired) / 1e10

		apiClient.UpdateAllocation(t, sdkWallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Add blobber to allocation", 1*time.Minute, func(t *test.SystemTest) {
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 10, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.16, client.TxSuccessfulStatus)
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

		apiClient.UpdateAllocation(t, sdkWallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})

	t.RunSequentiallyWithTimeout("Replace blobber", 1*time.Minute, func(t *test.SystemTest) {
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 10, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.16, client.TxSuccessfulStatus)
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

		apiClient.UpdateAllocation(t, sdkWallet, allocationID, uar, minLockRequiredInZcn, client.TxSuccessfulStatus)
		alloc = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, int64(1*GB), alloc.Size, "Allocation size is not updated")
	})
}

//https://dev-4.devnet-0chain.net/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation-update-min-lock?data=%7B%22add_blobber_id%22%3A%22%22%2C%22extend%22%3Afalse%2C%22id%22%3A%2274613ab72f1cfb671994eec8ec9bde9181b273a62908afc955a3efc73ce5d4dd%22%2C%22owner_id%22%3A%226d02a02cb9cbddd76f7e3981eae473b86dc488e558cbaffbe0549b31926605b3%22%2C%22owner_public_key%22%3A%22%22%2C%22remove_blobber_id%22%3A%22%22%2C%22size%22%3A0%7D
