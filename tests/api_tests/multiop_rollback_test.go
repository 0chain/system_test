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

	t.RunSequentially("Multi upload operations rollback should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 5)

		for i := 0; i < 1; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		newOps := make([]sdk.OperationRequest, 0, 5)
		time.Sleep(2 * time.Second)
		for i := 0; i < 2; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
			newOps = append(newOps, op)
		}
		sdkClient.MultiOperation(t, allocationID, newOps)

		time.Sleep(5 * time.Second)
		sdkClient.Rollback(t, allocationID)

		moreOps := make([]sdk.OperationRequest, 0, 5)
		for i := 0; i < 3; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
			moreOps = append(moreOps, op)
		}
		sdkClient.MultiOperation(t, allocationID, moreOps)

		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 4, len(listResult.Children), "files count mismatch expected %v actual %v", 4, len(listResult.Children))
	})

	t.RunSequentially("Multi delete operations rollback should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
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
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
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
	t.Run("Multi different operations rollback should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)
		time.Sleep(2 * time.Second)
		newOps := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			switch i % 2 {
			case 0:
				op := sdkClient.AddDeleteOperation(t, allocationID, ops[i].FileMeta.RemotePath)
				newOps = append(newOps, op)
			case 1:
				op := sdkClient.AddUpdateOperation(t, ops[i].FileMeta.RemotePath, ops[i].FileMeta.RemoteName, ops[i].FileMeta.ActualSize)
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
