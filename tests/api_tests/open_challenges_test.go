package api_tests

import (
	"bytes"
	"encoding/json"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestOpenChallenges(t *testing.T) {
	t.Parallel()

	t.Run("Open Challenges API response should be successful decode given a valid request", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)

		blobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 2147483648, 2, 2, 3600000000000, time.Minute*2)
		require.NotNil(t, blobbers)
		require.Greater(t, len(*blobbers), 3)
		require.NotNil(t, blobberRequirements)

		blobberId := (*blobbers)[0]
		response, err := zeroChain.GetOpenChallenges(t, STORAGE_SMART_CONTRACT_ADDRESS, blobberId)
		require.Nil(t, err)
		bytesReader := bytes.NewBuffer(response.Body())
		d := json.NewDecoder(bytesReader)
		d.UseNumber()

		var blobberChallenges model.BCChallengeResponse
		blobberChallenges.Challenges = make([]*model.ChallengeEntity, 0)
		err = d.Decode(&blobberChallenges)
		require.Nil(t, err)
	})
}
