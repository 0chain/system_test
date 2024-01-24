package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/client"
)

func TestCreateAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create allocation API call should be successful given a valid request")

	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		wallet.Nonce = int(balance.Nonce)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	})
}
