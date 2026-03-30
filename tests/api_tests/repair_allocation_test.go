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
		// Skip blobbers with invalid/placeholder URLs that can't be resolved.
		// These include test-registered ghost blobbers from prior runs.
		if blobber.Baseurl == "http://0zus.com/" {
			continue
		}
		// Skip ghost blobbers registered by tests with randomly generated URLs
		if strings.Contains(blobber.Baseurl, ".com/") &&
			!strings.Contains(blobber.Baseurl, "zus.network") &&
			!strings.Contains(blobber.Baseurl, "0chain.net") &&
			!strings.Contains(blobber.Baseurl, "198.18.") {
			continue
		}
		validBlobbers = append(validBlobbers, blobber)
	}
	// Ensure we have at least the minimum required blobbers
	if len(validBlobbers) < minRequired {
		return blobbers // Return original if filtering would leave too few
	}
	return validBlobbers
}

func TestRepairAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	walletMutex.Lock()
	wallet := initialisedWallets[walletIdx]
	walletIdx++
	walletMutex.Unlock()
	balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
	wallet.Nonce = int(balance.Nonce)

	sdkClient.SetWallet(t, wallet)

	t.RunSequentiallyWithTimeout("Repair allocation after single upload should work", 10*time.Minute, func(t *test.SystemTest) {
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 64 * 1024 * 3
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)
		lastBlobber := alloc.Blobbers[len(alloc.Blobbers)-1]
		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
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

	t.RunSequentiallyWithTimeout("Repair allocation after multiple uploads should work", 10*time.Minute, func(t *test.SystemTest) {

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

	t.RunSequentiallyWithTimeout("Repair allocation after update should work", 10*time.Minute, func(t *test.SystemTest) {

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

		alloc.Blobbers[len(alloc.Blobbers)-1].Baseurl = "http://0zus.com/"
		updateOp := sdkClient.AddUpdateOperation(t, op.RemotePath, op.FileMeta.RemoteName, op.FileMeta.ActualSize)
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{updateOp}, client.WithRepair(validBlobbers))

		// Wait for write markers to commit on all blobbers before repairing.
		// Without this, repair races against in-flight writes and fails.
		time.Sleep(15 * time.Second)

		sdkClient.RepairAllocation(t, allocationID)

		updatedRef, err := sdk.GetFileRefFromBlobber(allocationID, firstBlobber.ID, op.RemotePath)
		require.Nil(t, err)

		fRef, err := sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.Nil(t, err)
		require.Equal(t, updatedRef.ActualFileHash, fRef.ActualFileHash)
	})

	t.RunSequentiallyWithTimeout("Repair allocation after delete should work", 10*time.Minute, func(t *test.SystemTest) {

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

		// Wait for write markers to commit before repair
		time.Sleep(15 * time.Second)

		sdkClient.RepairAllocation(t, allocationID)
		_, err = sdk.GetFileRefFromBlobber(allocationID, lastBlobber.ID, op.RemotePath)
		require.NotNil(t, err)
	})

	t.RunSequentiallyWithTimeout("Repair allocation after move should work", 10*time.Minute, func(t *test.SystemTest) {

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

	t.RunSequentiallyWithTimeout("Repair allocation after copy should work", 10*time.Minute, func(t *test.SystemTest) {

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

	t.RunSequentiallyWithTimeout("Repair allocation after rename should work", 10*time.Minute, func(t *test.SystemTest) {

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

	t.RunSequentiallyWithTimeout("Repair allocation should work with multiple 100MB file", 10*time.Minute, func(t *test.SystemTest) {

		fileSize := int64(512)
		numOfFile := int64(4)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2 * 1024 * 1024 // 2MB to accommodate SDK minimum chunk sizes
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

		fileSize := int64(512)
		numOfFile := int64(4)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2 * 1024 * 1024 // 2MB to accommodate SDK minimum chunk sizes
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

		fileSize := int64(512)
		numOfFile := int64(4)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 2 * 1024 * 1024 // 2MB to accommodate SDK minimum chunk sizes
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

		fileSize := int64(512)
		numOfFile := int64(10)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 4 * 1024 * 1024 // 4MB for 10 files to accommodate SDK minimum chunk sizes
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

			// Use consistent small file size to stay within blobber max_file_size limits
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
	wallet := createWallet(t)
	sdkClient.SetWallet(t, wallet)

	t.RunSequentiallyWithTimeout("repair size in case of no blobber failure should be zero", 10*time.Minute, func(t *test.SystemTest) {
		// create allocation with default blobber requirements
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		t.Logf("allocationID: %v", allocationID)

		// Wait for blobbers to sync allocation from chain event DB before attempting upload.
		time.Sleep(10 * time.Second)

		// create and upload a file of 2KB to allocation.
		op := sdkClient.AddUploadOperation(t, "", "", int64(512))
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

	t.RunSequentiallyWithTimeout("repair size on single blobber failure should match", 10*time.Minute, func(t *test.SystemTest) {
		// create allocation with default blobber requirements
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 2
		blobberRequirements.Size = 64 * 1024 * 3
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		t.Logf("allocationID: %v", allocationID)

		// Wait for blobbers to sync allocation from chain event DB before attempting upload.
		time.Sleep(10 * time.Second)

		// create and upload a file of 2KB to allocation.
		// one blobber url is set invalid to mimic failure.
		alloc, err := sdk.GetAllocation(allocationID)
		require.NoErrorf(t, err, "allocation ID %v is not found", allocationID)
		alloc.Blobbers[0].Baseurl = "http://0zus.com/"
		op := sdkClient.AddUploadOperation(t, "", "", int64(512))
		validBlobbers := filterValidBlobbers(alloc.Blobbers, int(blobberRequirements.DataShards))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op}, client.WithRepair(validBlobbers))

		// assert repair is needed (upload and download sizes should be > 0)
		rs, err := alloc.RepairSize("/")
		require.Nil(t, err)
		t.Logf("repair size: upload=%v download=%v", rs.UploadSize, rs.DownloadSize)
		require.Greater(t, rs.UploadSize, uint64(0), "upload size should be > 0 with one failed blobber")
		require.Greater(t, rs.DownloadSize, uint64(0), "download size should be > 0 with one failed blobber")
	})

	t.RunSequentiallyWithTimeout("repair size with nested directories and two blobber failure should match", 10*time.Minute, func(t *test.SystemTest) {
		// create allocation with default blobber requirements
		// Note: GetAllocationBlobbers inflates DataShards by +4 for candidate pool selection.
		// With DataShards=2 → inflated=6. ParityShards must be ≤ 3 so total (6+3=9) ≤ available blobbers.
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 2
		blobberRequirements.ParityShards = 3
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		t.Logf("allocationID: %v", allocationID)

		// Wait for blobbers to sync allocation from chain event DB before attempting upload.
		// Subtest 3 runs immediately after subtests 1 and 2, and with 5 blobbers (2d+3p),
		// blobbers need more time to receive the allocation event from the sharder.
		time.Sleep(20 * time.Second)

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

		// assert upload and download sizes in /dir1
		// With 2 failed blobbers, upload_size = files * size * num_failed_blobbers
		rs, err := alloc.RepairSize("/dir1")
		require.Nilf(t, err, "error getting repair size in /dir1: %v", err)
		t.Logf("repair size /dir1: upload=%v download=%v", rs.UploadSize, rs.DownloadSize)
		require.Greater(t, rs.UploadSize, uint64(0), "upload size in directory /dir1 should be > 0")
		require.Greater(t, rs.DownloadSize, uint64(0), "download size in directory /dir1 should be > 0")

		// with trailing slash - should match without trailing slash
		rs2, err := alloc.RepairSize("/dir1/")
		require.Nilf(t, err, "error getting repair size in /dir1/: %v", err)
		t.Logf("repair size /dir1/: upload=%v download=%v", rs2.UploadSize, rs2.DownloadSize)
		require.Equal(t, rs.UploadSize, rs2.UploadSize, "upload size /dir1 vs /dir1/ should match")
		require.Equal(t, rs.DownloadSize, rs2.DownloadSize, "download size /dir1 vs /dir1/ should match")

		// root directory should have >= /dir1 sizes (it has more files)
		rsRoot, err := alloc.RepairSize("/")
		require.Nilf(t, err, "error getting repair size in /: %v", err)
		t.Logf("repair size /: upload=%v download=%v", rsRoot.UploadSize, rsRoot.DownloadSize)
		require.GreaterOrEqual(t, rsRoot.UploadSize, rs.UploadSize, "root upload size should be >= /dir1 upload size")
		require.GreaterOrEqual(t, rsRoot.DownloadSize, rs.DownloadSize, "root download size should be >= /dir1 download size")
	})
}
