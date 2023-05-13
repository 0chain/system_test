package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/client"
)

// this test is working fine in local
func TestCreateAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create allocation API call should be successful given a valid request")

	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *test.SystemTest) {
		wallet := apiClient.CreateWallet(t)
		for i := 0; i < 2; i++ {
			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)
		}

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	})
}
