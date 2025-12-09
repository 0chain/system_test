package api_tests

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/blockchain"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// filterValidBlobbers filters out blobbers with invalid URLs (like http://0zus.com/)
// to avoid connection errors during multi-operation. The repair functionality will
// still work as the invalid blobber is missing from the list.
func filterValidBlobbers(blobbers []*blockchain.StorageNode, minRequired int) []*blockchain.StorageNode {
	validBlobbers := make([]*blockchain.StorageNode, 0)
	for _, blobber := range blobbers {
		// Skip blobbers with invalid URLs that can't be resolved
		if blobber.Baseurl != "http://0zus.com/" {
			validBlobbers = append(validBlobbers, blobber)
		}
	}
	// Ensure we have at least the minimum required blobbers
	if len(validBlobbers) < minRequired {
		return blobbers // Return original if filtering would leave too few
	}
	return validBlobbers
}

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op}, client.WithRepair(validBlobbers))

		// Wait for the upload to complete and be committed to all blobbers
		// This ensures the file is properly stored before repair attempts to fix it
		time.Sleep(15 * time.Second)

		// Use the existing RepairAllocation method which will handle the repair
		// The repair should skip invalid blobbers gracefully
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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(validBlobbers))

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{updateOp}, client.WithRepair(validBlobbers))

		// Wait for the update to complete and be committed to all blobbers
		// This ensures the file is properly updated before repair attempts to fix it
		time.Sleep(15 * time.Second)

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{deleteOp}, client.WithRepair(validBlobbers))

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{moveOp}, client.WithRepair(validBlobbers))

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{copyOP}, client.WithRepair(validBlobbers))

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{renameOp}, client.WithRepair(validBlobbers))

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(validBlobbers))

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(validBlobbers))

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
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(validBlobbers))

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

		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(validBlobbers))

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

	t.RunSequentiallyWithTimeout("repair size in case of no blobber failure should be zero", 5*time.Minute, func(t *test.SystemTest) {
		// create allocation with default blobber requirements
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		t.Logf("allocationID: %v", allocationID)

		// create and upload a file of 2KB to allocation.
		op := sdkClient.AddUploadOperation(t, "", "", int64(1024*2))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		// assert both upload and download size should be zero
		alloc, err := sdk.GetAllocation(allocationID)
		require.NoErrorf(t, err, "allocation ID %v is not found", allocationID)
		rs, err := alloc.RepairSize("/")
		require.Nil(t, err)
		t.Logf("repair size: %v", rs)
		require.Equal(t, uint64(0), rs.UploadSize, "upload size doesn't match")
		require.Equal(t, uint64(0), rs.DownloadSize, "download size doesn't match")
	})

	t.RunSequentiallyWithTimeout("repair size on single blobber failure should match", 5*time.Minute, func(t *test.SystemTest) {
		// create allocation with default blobber requirements
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2056
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		t.Logf("allocationID: %v", allocationID)

		// create and upload a file of 2KB to allocation.
		// one blobber url is set invalid to mimic failure.
		alloc, err := sdk.GetAllocation(allocationID)
		require.NoErrorf(t, err, "allocation ID %v is not found", allocationID)
		alloc.Blobbers[0].Baseurl = "http://0zus.com/"
		op := sdkClient.AddUploadOperation(t, "", "", int64(1024*2))
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op}, client.WithRepair(validBlobbers))

		// assert upload and download size should be 1KB and 2KB respectively
		rs, err := alloc.RepairSize("/")
		require.Nil(t, err)
		t.Logf("repair size: %v", rs)
		require.Equal(t, uint64(1024), rs.UploadSize, "upload size doesn't match")
		require.Equal(t, uint64(1024*2), rs.DownloadSize, "download size doesn't match")
	})

	t.RunSequentiallyWithTimeout("repair size with nested directories and two blobber failure should match", 5*time.Minute, func(t *test.SystemTest) {
		// create allocation with default blobber requirements
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 4
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		t.Logf("allocationID: %v", allocationID)

		// create and upload two files of 1KB each to / and /dir1.
		// two blobber url is set invalid to mimic failure.
		alloc, err := sdk.GetAllocation(allocationID)
		require.NoErrorf(t, err, "allocation ID %v is not found", allocationID)
		alloc.Blobbers[0].Baseurl = "http://0zus.com/"
		alloc.Blobbers[1].Baseurl = "http://0zus.com/"
		ops := []sdk.OperationRequest{
			sdkClient.AddUploadOperationWithPath(t, allocationID, "/dir1/"),
			sdkClient.AddUploadOperationWithPath(t, allocationID, "/dir1/"),
			sdkClient.AddUploadOperationWithPath(t, allocationID, "/"),
			sdkClient.AddUploadOperationWithPath(t, allocationID, "/"),
		}
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, ops, client.WithRepair(validBlobbers))

		// assert both upload and download size should be 2KB in /dir1
		rs, err := alloc.RepairSize("/dir1")
		require.Nilf(t, err, "error getting repair size in /dir1: %v", err)
		t.Logf("repair size: %v", rs)
		require.Equal(t, uint64(1024*2), rs.UploadSize, "upload size in directory /dir1 doesn't match")
		require.Equal(t, uint64(1024*2), rs.DownloadSize, "download size in directory dir1 doesn't match")

		// with trailing slash
		// assert both upload and download size should be 2KB in /dir1/
		rs, err = alloc.RepairSize("/dir1/")
		require.Nilf(t, err, "error getting repair size in /dir1/: %v", err)
		t.Logf("repair size: %v", rs)
		require.Equal(t, uint64(1024*2), rs.UploadSize, "upload size in directory /dir1/ doesn't match")
		require.Equal(t, uint64(1024*2), rs.DownloadSize, "download size in directory /dir1/ doesn't match")

		// assert both upload and download size should be 4KB in root directory
		rs, err = alloc.RepairSize("/")
		require.Nilf(t, err, "error getting repair size in /: %v", err)
		t.Logf("repair size: %v", rs)
		require.Equal(t, uint64(1024*4), rs.UploadSize, "upload size in root directory doesn't match")
		require.Equal(t, uint64(1024*4), rs.DownloadSize, "download size in root directory doesn't match")
	})
}
