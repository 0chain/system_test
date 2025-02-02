package api_tests

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestRepairAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	wallet := initialisedWallets[walletIdx]
	walletIdx++
	balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
	wallet.Nonce = int(balance.Nonce)

	sdkClient.SetWallet(t, wallet)

	t.RunSequentially("Repair allocation after single upload should work", func(t *test.SystemTest) {
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 64 * 1024 * 3
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[0]
		alloc.Blobbers[0].Baseurl = "http://0zus.com/"
		op := sdkClient.AddUploadOperation(t, "", "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.Nil(t, err)
	})

	t.RunSequentially("Repair allocation after multiple uploads should work", func(t *test.SystemTest) {
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]
		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"

		ops := make([]sdk.OperationRequest, 0, 4)
		for i := 0; i < 4; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
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
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 64 * 1024 * 10 * 2
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		firstBlobber := alloc.Blobbers[0]
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "", "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[0].Baseurl = "http://0zus.com/"
		updateOp := sdkClient.AddUpdateOperation(t, op.RemotePath, op.FileMeta.RemoteName, op.FileMeta.ActualSize)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{updateOp}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)

		updatedRef, err := sdk.GetFileRefFromBlobber(allocationID, firstBlobber.ID, op.RemotePath)
		require.Nil(t, err)

		fRef, err := sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.Nil(t, err)
		require.Equal(t, updatedRef.ActualFileHash, fRef.ActualFileHash)
	})

	t.RunSequentially("Repair allocation after delete should work", func(t *test.SystemTest) {
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 64 * 1024 * 3
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "", "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
		deleteOp := sdkClient.AddDeleteOperation(t, allocationID, op.RemotePath)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{deleteOp}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.NotNil(t, err)
	})

	t.RunSequentially("Repair allocation after move should work", func(t *test.SystemTest) {
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 64 * 1024 * 10 * 2
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "", "")
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
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 64 * 1024 * 20 * 2
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "", "")
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
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 64 * 1024 * 10 * 2
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]

		op := sdkClient.AddUploadOperation(t, "", "")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
		newName := randName()
		renameOp := sdkClient.AddRenameOperation(t, allocationID, op.RemotePath, newName)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{renameOp}, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, "/"+newName)
		require.Nil(t, err)
	})

	t.RunSequentially("Repair allocation should work with multiple 100MB file", func(t *test.SystemTest) {
		fileSize := int64(1024 * 1024 * 10) // 100MB
		numOfFile := int64(4)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2 * numOfFile * int64(fileSize)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]
		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"

		ops := make([]sdk.OperationRequest, 0, 4)
		for i := 0; i < int(numOfFile); i++ {
			path := fmt.Sprintf("dummy_%d", i)
			op := sdkClient.AddUploadOperation(t, path, "", fileSize)
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		for _, op := range ops {
			_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
			require.Nil(t, err)
		}
	})

	t.RunSequentiallyWithTimeout("Repair allocation should work with multiple 500MB file", 10*time.Minute, func(t *test.SystemTest) {
		fileSize := int64(1024 * 1024 * 500) // 500MB
		numOfFile := int64(4)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2 * numOfFile * fileSize
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]
		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"

		ops := make([]sdk.OperationRequest, 0, 4)
		for i := 0; i < int(numOfFile); i++ {
			path := fmt.Sprintf("dummy_%d", i)
			op := sdkClient.AddUploadOperation(t, path, "", fileSize)
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		for _, op := range ops {
			_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
			require.Nil(t, err)
		}
	})

	t.RunSequentiallyWithTimeout("Repair allocation should work with multiple combination of file type & size", 10*time.Minute, func(t *test.SystemTest) {
		fileSize := int64(1024 * 1024 * 500) // 500MB
		numOfFile := int64(4)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2 * numOfFile * fileSize
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]
		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"

		ops := make([]sdk.OperationRequest, 0, 4)
		for i := 0; i < int(numOfFile); i++ {
			path := fmt.Sprintf("dummy_%d", i)
			format := ""
			if i%2 == 0 {
				format = "pdf"
				fileSize = int64(1024 * 1024 * 300)
			}
			op := sdkClient.AddUploadOperation(t, path, format, fileSize)
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		for _, op := range ops {
			_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
			require.Nil(t, err)
		}
	})

	t.RunSequentiallyWithTimeout("Repair allocation should work with multiple combination of file type & size & nested folders", 10*time.Minute, func(t *test.SystemTest) {
		fileSize := int64(1024 * 1024 * 1000) // 1 GB
		numOfFile := int64(10)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2 * numOfFile * fileSize
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]
		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"

		ops := make([]sdk.OperationRequest, 0, 4)

		for i := 0; i < int(numOfFile); i++ {
			numOfNestedFolders, err := rand.Int(rand.Reader, big.NewInt(20))
			require.Nil(t, err)
			folderStructure := fmt.Sprintf("test_%d/", i)
			path := strings.Repeat(folderStructure, int(numOfNestedFolders.Int64()))

			switch {
			case i%5 == 0:
				fileSize = int64(1024 * 1024 * 500)
			case i%3 == 0:
				fileSize = int64(1024 * 1024 * 300)
			case i%2 == 0:
				fileSize = int64(1024 * 1024 * 200)
			}
			// for default case size would be 1 GB
			op := sdkClient.AddUploadOperation(t, path, "", fileSize)
			ops = append(ops, op)
		}

		t.Log(ops)

		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(alloc.Blobbers))

		sdkClient.RepairAllocation(t, allocationID)
		for _, op := range ops {
			_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
			require.Nil(t, err)
		}
	})
}

func TestRepairSize(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Skip("Skipping as repair size is not implemented for V2")
	wallet := createWallet(t)
	sdkClient.SetWallet(t, wallet)
}
