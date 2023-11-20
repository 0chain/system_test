package api_tests

import (
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAllocationUpdateLockAmount(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Run("Extend Allocation Size", func(t *test.SystemTest) {
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 10, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		blobberRequirements.Size = 1 * GB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.131, client.TxSuccessfulStatus)
		t.Log("Allocation ID: ", allocationID)

		uar := &model.UpdateAllocationRequest{
			ID:   allocationID,
			Size: 1 * GB,
		}

		minLockRequired, err := sdk.GetUpdateAllocationMinLock(allocationID, 1*GB, false, "", "")
		require.NoError(t, err)

		t.Logf("Min lock required: %v", minLockRequired)

		apiClient.UpdateAllocation(t, sdkWallet, allocationID, uar, float64(minLockRequired), client.TxSuccessfulStatus)
		alloc := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		require.Equal(t, 1*GB, alloc.Size, "Allocation size is not updated")
	})
}
