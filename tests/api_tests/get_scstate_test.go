package api_tests

import (
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

		sharderSCStateResponse, restyResponse, err := v1SharderGetSCState(t, FAUCET_SMART_CONTRACT_ADDRESS, registeredWallet.Id, nil)
		require.Nil(t, err)
		require.Equal(t, util.HttpOkStatus, restyResponse.Status())
		require.Equal(t, util.ZcnToInt(sharderSCStateResponse.Used), int64(1), "SCState does not seem to be valid")
	})

	t.Run("Get SCState of faucet SC, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, _ := registerWallet(t)

		_, restyResponse, err := v1SharderGetSCState(t, FAUCET_SMART_CONTRACT_ADDRESS, registeredWallet.Id, nil)
		require.Nil(t, err)
		require.Equal(t, util.HttpNotFoundStatus, restyResponse.Status())
	})
}
