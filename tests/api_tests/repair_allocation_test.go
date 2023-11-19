package api_tests

import (
	"path/filepath"
	"testing"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestRepairAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)
	apiClient.CreateReadPool(t, sdkWallet, 0.5, client.TxSuccessfulStatus)
	t.RunSequentially("Repair allocation after single upload should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2056
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[0]
		alloc.Blobbers[0].Baseurl = "http://0zus.com/"
		op := sdkClient.AddUploadOperation(t, "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.Nil(t, err)
	})

	t.RunSequentially("Repair allocation after multiple uploads should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]
		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"

		ops := make([]sdk.OperationRequest, 0, 4)
		for i := 0; i < 4; i++ {
			op := sdkClient.AddUploadOperation(t, "")
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		for _, op := range ops {
			_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
			require.Nil(t, err)
		}
	})

	t.RunSequentially("Repair allocation after update should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 6096
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		firstBlobber := alloc.Blobbers[0]
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[0].Baseurl = "http://0zus.com/"
		updateOp := sdkClient.AddUpdateOperation(t, allocationID, op.RemotePath, op.FileMeta.RemoteName)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{updateOp}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)

		updatedRef, err := sdk.GetFileRefFromBlobber(allocationID, firstBlobber.ID, op.RemotePath)
		require.Nil(t, err)

		fRef, err := sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.Nil(t, err)
		require.Equal(t, updatedRef.ActualFileHash, fRef.ActualFileHash)
	})

	t.RunSequentially("Repair allocation after delete should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 2056
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
		deleteOp := sdkClient.AddDeleteOperation(t, allocationID, op.RemotePath)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{deleteOp}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.NotNil(t, err)
	})

	t.RunSequentially("Repair allocation after move should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 6096
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
		newPath := "/new/" + filepath.Join("", filepath.Base(op.FileMeta.Path))
		moveOp := sdkClient.AddMoveOperation(t, allocationID, op.RemotePath, newPath)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{moveOp}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, newPath)
		require.Nil(t, err)
	})

	t.RunSequentially("Repair allocation after copy should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 8096
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
		newPath := "/new/" + filepath.Join("", filepath.Base(op.FileMeta.Path))
		copyOP := sdkClient.AddCopyOperation(t, allocationID, op.RemotePath, newPath)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{copyOP}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, newPath)
		require.Nil(t, err)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.Nil(t, err)
	})

	t.RunSequentially("Repair allocation after rename should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 6096
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
		newName := randName()
		renameOp := sdkClient.AddRenameOperation(t, allocationID, op.RemotePath, newName)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{renameOp}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, "/"+newName)
		require.Nil(t, err)
	})
}
