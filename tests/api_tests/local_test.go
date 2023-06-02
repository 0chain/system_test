package api_tests

import (
	"sync"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
)

func TestLocalOperation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	// t.RunSequentially("Upload 10 files and update each of them 10 times", func(t *test.SystemTest) {
	// 	apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

	// 	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	// 	allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
	// 	allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

	// 	ops := make([]sdk.OperationRequest, 0, 10)
	// 	var str string
	// 	for i := 0; i < 10; i++ {
	// 		if i == 0 {
	// 			str = str + fmt.Sprintf("/%v/", i)
	// 		} else {
	// 			str = str + fmt.Sprintf("%v/", i)
	// 		}
	// 		fmt.Println(str)
	// 		op := sdkClient.AddUploadOperationWithPath(t, allocationID, str)
	// 		ops = append(ops, op)
	// 	}
	// 	sdkClient.MultiOperation(t, allocationID, ops)
	// 	newOps := make([]sdk.OperationRequest, 0)
	// 	for i := 0; i < 10; i++ {
	// 		for j := 0; j < 10; j++ {
	// 			op := sdkClient.AddUpdateOperation(t, allocationID, ops[j].FileMeta.RemotePath, ops[j].FileMeta.RemoteName)
	// 			newOps = append(newOps, op)
	// 		}
	// 		sdkClient.MultiOperation(t, allocationID, newOps)
	// 		newOps = nil
	// 	}
	// })

	// t.RunSequentially("Upload 10 files and move each of them 10 times", func(t *test.SystemTest) {
	// 	apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

	// 	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	// 	allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
	// 	allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

	// 	ops := make([]sdk.OperationRequest, 0, 10)
	// 	var str string
	// 	for i := 0; i < 10; i++ {
	// 		if i == 0 {
	// 			str = str + fmt.Sprintf("/%v/", i)
	// 		} else {
	// 			str = str + fmt.Sprintf("%v/", i)
	// 		}
	// 		fmt.Println(str)
	// 		op := sdkClient.AddUploadOperationWithPath(t, allocationID, str)
	// 		ops = append(ops, op)
	// 	}
	// 	sdkClient.MultiOperation(t, allocationID, ops)

	// 	newOps := make([]sdk.OperationRequest, 0)
	// 	x := 0
	// 	for i := 0; i < 10; i++ {
	// 		var newPath string
	// 		x += 9
	// 		for j := 9; j >= 0; j-- {
	// 			if j == 9 {
	// 				newPath = newPath + fmt.Sprintf("/%v/", x)
	// 			} else {
	// 				newPath = newPath + fmt.Sprintf("%v/", x)
	// 			}
	// 			op := sdkClient.AddMoveOperation(t, allocationID, ops[j].FileMeta.RemotePath, newPath)
	// 			newOps = append(newOps, op)
	// 			sdkClient.MultiOperation(t, allocationID, newOps)
	// 			ops[j].FileMeta.RemotePath = filepath.Join(newPath, ops[j].FileMeta.RemoteName)
	// 			newOps = nil
	// 			x++
	// 		}
	// 	}
	// })

	// t.RunSequentially("Upload 5 files and copy each of them 10 times", func(t *test.SystemTest) {
	// 	apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

	// 	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	// 	allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
	// 	allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

	// 	ops := make([]sdk.OperationRequest, 0, 10)
	// 	var str string
	// 	for i := 0; i < 5; i++ {
	// 		if i == 0 {
	// 			str = str + fmt.Sprintf("/%v/", i)
	// 		} else {
	// 			str = str + fmt.Sprintf("%v/", i)
	// 		}
	// 		fmt.Println(str)
	// 		op := sdkClient.AddUploadOperationWithPath(t, allocationID, str)
	// 		ops = append(ops, op)
	// 	}
	// 	sdkClient.MultiOperation(t, allocationID, ops)

	// 	newOps := make([]sdk.OperationRequest, 0)
	// 	x := 0
	// 	for i := 0; i < 10; i++ {
	// 		var newPath string
	// 		x += 9
	// 		for j := 4; j >= 0; j-- {
	// 			if j == 4 {
	// 				newPath = newPath + fmt.Sprintf("/%v/", x)
	// 			} else {
	// 				newPath = newPath + fmt.Sprintf("%v/", x)
	// 			}
	// 			op := sdkClient.AddCopyOperation(t, allocationID, ops[j].FileMeta.RemotePath, newPath)
	// 			newOps = append(newOps, op)
	// 			sdkClient.CopyObject(t, allocationID, ops[j].FileMeta.RemotePath, newPath)
	// 			ops[j].FileMeta.RemotePath = filepath.Join(newPath, ops[j].FileMeta.RemoteName)
	// 			newOps = nil
	// 			x++
	// 		}
	// 	}
	// })
	t.RunSequentially("upload and copy concurrently", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)
		wg := sync.WaitGroup{}
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				op := sdkClient.AddUploadOperation(t, allocationID)
				sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})
				time.Sleep(2 * time.Second)
				copyOp := sdkClient.AddCopyOperation(t, allocationID, op.FileMeta.RemotePath, "/copy/")
				sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{copyOp})
			}()
		}
		wg.Wait()
	})
}
