package api_tests

import (
	"testing"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestStreamUpload(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Stream upload should work")

	t.RunSequentially("Stream upload should work", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)
		sdkClient.SetWallet(t, wallet)
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.Size = 150 * MB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		op := sdkClient.AddUploadOperation(t, "", "", 100*MB)
		op.FileMeta.ActualSize = 0
		op.StreamUpload = true
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})
		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 1, len(listResult.Children), "files count mismatch expected %v actual %v", 10, len(listResult.Children))
	})
}
