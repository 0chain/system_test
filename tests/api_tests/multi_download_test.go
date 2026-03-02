package api_tests

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestMultiDownload(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Multi download should work")

	t.RunSequentiallyWithTimeout("Multi download should work", 10*time.Minute, func(t *test.SystemTest) {
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

		err := os.MkdirAll("temp_download", os.ModePerm)
		require.NoError(t, err, "error creating temp dir")
		defer func() {
			_ = os.RemoveAll("temp_download")
		}()

		for _, op := range ops {
			// RemotePath starts with "/"; DownloadFile prepends "/" internally, so strip it
			remotePath := strings.TrimPrefix(op.FileMeta.RemotePath, "/")
			sdkClient.DownloadFile(t, allocationID, remotePath, "temp_download/")
		}

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
