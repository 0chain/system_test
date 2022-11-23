package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestGetBlobbersForNewAllocation(testSetup *testing.T) {
	t := &test.SystemTest{T: testSetup}

	t.Parallel()

	t.Run("Alloc blobbers API call should be successful given a valid request", func(t *test.SystemTest) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)

		require.NotNil(t, allocationBlobbers.Blobbers)
		require.Greater(t, len(*allocationBlobbers.Blobbers), 3)
		require.NotNil(t, allocationBlobbers.BlobberRequirements)
	})

	// FIXME lack of field validation leads to error see https://github.com/0chain/0chain/issues/1319
	t.Run("BROKEN Alloc blobbers API call should fail gracefully given valid request but does not see 0chain/issues/1319", func(t *test.SystemTest) {
		t.Parallel()
		t.Skip("FIXME: lack of field validation leads to error see https://github.com/0chain/0chain/issues/1319")

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &model.BlobberRequirements{}, client.HttpOkStatus)

		require.NotNil(t, allocationBlobbers.Blobbers)
	})
}
