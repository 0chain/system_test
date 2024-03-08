package api_tests

import (
	"os"
	"sync"
	"testing"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestMultiDownload(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()
	t.SetSmokeTests("Multi download should work")

	t.Run("Multi download should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		apiClient.CreateReadPool(t, wallet, float64(1.5), client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, "", "")
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err, "error getting allocation")
		err = os.MkdirAll("temp_download", os.ModePerm)
		require.NoError(t, err, "error creating temp dir")
		defer func() {
			err = os.RemoveAll("temp_download")
			require.NoError(t, err, "error removing temp dir")
		}()
		wg := &sync.WaitGroup{}
		for i := 0; i < 9; i++ {
			sdkClient.DownloadFileWithParam(t, alloc, ops[i].FileMeta.RemotePath, "temp_download/", wg, false)
		}
		sdkClient.DownloadFileWithParam(t, alloc, ops[9].FileMeta.RemotePath, "temp_download/", wg, true)
		wg.Wait()
		files, err := os.ReadDir("temp_download")
		require.NoError(t, err, "error reading temp dir")
		require.Equal(t, 10, len(files), "files count mismatch expected %v actual %v", 10, len(files))
		for _, file := range files {
			sz, err := file.Info()
			require.NoError(t, err, "error getting file info")
			require.Equal(t, int64(1024), sz.Size(), "file size mismatch expected %v actual %v", 1024, sz.Size())
		}
	})
}
