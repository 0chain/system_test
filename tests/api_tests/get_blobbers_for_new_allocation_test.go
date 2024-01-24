package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestGetBlobbersForNewAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Alloc blobbers API call should be successful given a valid request")

	t.Parallel()

	t.Run("Alloc blobbers API call should be successful given a valid request", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)

		require.NotNil(t, allocationBlobbers.Blobbers)
		require.Greater(t, len(*allocationBlobbers.Blobbers), 3)
		require.NotNil(t, allocationBlobbers.BlobberRequirements)
	})

	t.Run("BROKEN Alloc blobbers API call should fail gracefully given valid request but does not see 0chain/issues/1319", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &model.BlobberRequirements{}, client.HttpBadRequestStatus)

		require.NotNil(t, allocationBlobbers.Blobbers)
	})
}
