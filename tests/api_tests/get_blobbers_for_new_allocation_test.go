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

		wallet := apiClient.RegisterWalletWrapper(t)

		scRestGetAllocationBlobbersResponse, resp, err := apiClient.V1SCRestGetAllocationBlobbers(
			&model.SCRestGetAllocationBlobbersRequest{
				ClientID:  wallet.ClientID,
				ClientKey: wallet.ClientKey,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocationBlobbersResponse)

		require.NotNil(t, scRestGetAllocationBlobbersResponse.Blobbers)
		require.Greater(t, len(*scRestGetAllocationBlobbersResponse.Blobbers), 3)
		require.NotNil(t, scRestGetAllocationBlobbersResponse.BlobberRequirements)
	})

	// FIXME lack of field validation leads to error see https://github.com/0chain/0chain/issues/1319
	t.Run("BROKEN Alloc blobbers API call should fail gracefully given valid request but does not see 0chain/issues/1319", func(t *testing.T) {
		t.Parallel()
		t.Skip("FIXME: lack of field validation leads to error see https://github.com/0chain/0chain/issues/1319")

		scRestGetAllocationBlobbersResponse, resp, err := apiClient.V1SCRestGetAllocationBlobbers(
			nil,
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocationBlobbersResponse)
		require.NotNil(t, scRestGetAllocationBlobbersResponse.Blobbers)
	})
}
