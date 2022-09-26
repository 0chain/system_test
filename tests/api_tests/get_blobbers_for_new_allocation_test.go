package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetBlobbersForNewAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Alloc blobbers API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet, nil, client.HttpOkStatus)

		require.NotNil(t, allocationBlobbers.Blobbers)
		require.Greater(t, len(*allocationBlobbers.Blobbers), 3)
		require.NotNil(t, allocationBlobbers.BlobberRequirements)
	})

	// FIXME lack of field validation leads to error see https://github.com/0chain/0chain/issues/1319
	t.Run("BROKEN Alloc blobbers API call should fail gracefully given valid request but does not see 0chain/issues/1319", func(t *testing.T) {
		t.Parallel()
		t.Skip("FIXME: lack of field validation leads to error see https://github.com/0chain/0chain/issues/1319")

		wallet := apiClient.RegisterWalletWrapper(t, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet, &model.BlobberRequirements{}, client.HttpOkStatus)

		require.NotNil(t, allocationBlobbers.Blobbers)
	})
}
