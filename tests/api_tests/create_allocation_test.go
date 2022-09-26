package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"testing"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t, client.HttpOkStatus)
		apiClient.ExecuteFaucetWrapper(t, wallet, client.HttpOkStatus, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers, client.HttpOkStatus, client.TxSuccessfulStatus)
		apiClient.GetAllocationWrapper(t, allocationID, client.HttpOkStatus)
	})
}
