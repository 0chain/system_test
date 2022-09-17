package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetScState(t *testing.T) {
	t.Parallel()

	t.Run("Get SCState of faucet SC, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucet(t, registeredWallet, keyPair)

		sharderSCStateRequest := model.SharderSCStateRequest{
			SCAddress: FAUCET_SMART_CONTRACT_ADDRESS,
			Key:       registeredWallet.Id,
		}

		sharderSCStateResponse, restyResponse, err := v1SharderGetSCState(t, sharderSCStateRequest, util.ConsensusByHttpStatus(util.HttpOkStatus))
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.NotNil(t, sharderSCStateResponse)
		require.Equal(t, util.ZcnToInt(sharderSCStateResponse.Used), int64(1), "SCState does not seem to be valid")
	})

	t.Run("Get SCState of faucet SC, which has no tokens, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, _ := registerWallet(t)

		sharderSCStateRequest := model.SharderSCStateRequest{
			SCAddress: FAUCET_SMART_CONTRACT_ADDRESS,
			Key:       registeredWallet.Id,
		}

		sharderSCStateResponse, restyResponse, err := v1SharderGetSCState(t, sharderSCStateRequest, util.ConsensusByHttpStatus(util.HttpNotFoundStatus))
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.Nil(t, sharderSCStateResponse)
	})
}
