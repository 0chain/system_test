package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestOpenChallenges(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Open Challenges API response should be successful decode given a valid request", func(t *test.SystemTest) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		scRestOpenChallengeResponse, resp, err := apiClient.V1SCRestOpenChallenge(
			t,
			model.SCRestOpenChallengeRequest{
				BlobberID: blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestOpenChallengeResponse)
	})
}
