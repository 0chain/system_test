package api_tests

import (
	"bytes"
	"encoding/json"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOpenChallenges(t *testing.T) {
	t.Parallel()

	t.Run("Open Challenges API response should be successful decode given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

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

		blobberId := (*scRestGetAllocationBlobbersResponse.Blobbers)[0]

		resp, err = apiClient.V1SCRestOpenChallenges(
			model.SCRestOpenChallengesRequest{
				BlobberID: blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)

		bytesReader := bytes.NewBuffer(resp.Body())
		d := json.NewDecoder(bytesReader)
		d.UseNumber()

		var blobberChallenges model.BCChallengeResponse
		blobberChallenges.Challenges = make([]*model.ChallengeEntity, 0)
		err = d.Decode(&blobberChallenges)
		require.Nil(t, err)
	})
}
