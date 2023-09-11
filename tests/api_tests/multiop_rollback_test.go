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

func TestMultiOperationRollback(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Multi different operations rollback should work")
	t.RunSequentiallyWithTimeout("Multi upload operations rollback should work", 90*time.Second, func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 5)

		for i := 0; i < 1; i++ {
			op := sdkClient.AddUploadOperation(t, allocationID)
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		newOps := make([]sdk.OperationRequest, 0, 5)
		time.Sleep(1 * time.Second)
		for i := 0; i < 2; i++ {
			op := sdkClient.AddUploadOperation(t, allocationID)
			newOps = append(newOps, op)
		}
		sdkClient.MultiOperation(t, allocationID, newOps)

		time.Sleep(1 * time.Second)
		sdkClient.Rollback(t, allocationID)

		moreOps := make([]sdk.OperationRequest, 0, 1)
		for i := 0; i < 1; i++ {
			op := sdkClient.AddUploadOperation(t, allocationID)
			moreOps = append(moreOps, op)
		}
		sdkClient.MultiOperation(t, allocationID, moreOps)
		moreOps = nil
		for i := 0; i < 1; i++ {
			op := sdkClient.AddUploadOperation(t, allocationID)
			moreOps = append(moreOps, op)
		}
		sdkClient.MultiOperation(t, allocationID, moreOps)
		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 3, len(listResult.Children), "files count mismatch expected %v actual %v", 3, len(listResult.Children))
	})

	t.RunSequentially("Multi delete operations rollback should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, allocationID)
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		newOps := make([]sdk.OperationRequest, 0, 10)
		time.Sleep(2 * time.Second)
		for i := 0; i < 5; i++ {
			op := sdkClient.AddDeleteOperation(t, allocationID, ops[i].FileMeta.RemotePath)
			newOps = append(newOps, op)
		}
		sdkClient.MultiOperation(t, allocationID, newOps)

		time.Sleep(5 * time.Second)
		sdkClient.Rollback(t, allocationID)
		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 10, len(listResult.Children), "files count mismatch expected 5 got %v", len(listResult.Children))
	})

	t.RunSequentially("Multi rename operations rollback should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, allocationID)
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)
		time.Sleep(2 * time.Second)
		newOps := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddRenameOperation(t, allocationID, ops[i].FileMeta.RemotePath, randName())
			newOps = append(newOps, op)
		}

		sdkClient.MultiOperation(t, allocationID, newOps)

		time.Sleep(5 * time.Second)
		sdkClient.Rollback(t, allocationID)
		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 10, len(listResult.Children), "files count mismatch expected %v actual %v", 10, len(listResult.Children))
	})
	t.RunSequentially("Multi different operations rollback should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, allocationID)
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)
		time.Sleep(2 * time.Second)
		newOps := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			switch i % 3 {
			case 0:
				op := sdkClient.AddDeleteOperation(t, allocationID, ops[i].FileMeta.RemotePath)
				newOps = append(newOps, op)
			case 1:
				op := sdkClient.AddUpdateOperation(t, allocationID, ops[i].FileMeta.RemotePath, ops[i].FileMeta.RemoteName)
				newOps = append(newOps, op)
			case 2:
				op := sdkClient.AddRenameOperation(t, allocationID, ops[i].FileMeta.RemotePath, randName())
				newOps = append(newOps, op)
			}
		}

		sdkClient.MultiOperation(t, allocationID, newOps)

		time.Sleep(5 * time.Second)
		sdkClient.Rollback(t, allocationID)
		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 10, len(listResult.Children), "files count mismatch expected %v actual %v", 10, len(listResult.Children))
	})
}
