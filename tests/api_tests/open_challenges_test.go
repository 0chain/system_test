package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOpenChallenges(t *testing.T) {
	t.Parallel()

	t.Run("Open Challenges API response should be successful decode given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		scRestOpenChallengeResponse, resp, err := apiClient.V1SCRestOpenChallenge(
			model.SCRestOpenChallengeRequest{
				BlobberID: blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestOpenChallengeResponse)
	})
}
