package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"testing"

	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/client"
)

func TestCreateAllocation(testSetup *testing.T) {
	t := test.SystemTest{T: testSetup}

	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	})
}
