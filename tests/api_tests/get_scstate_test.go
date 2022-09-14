package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetScState(t *testing.T) {
	t.Parallel()

	t.Run("Get SCState of faucet SC, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucet(t, registeredWallet, keyPair)

		sharderSCStateResponse, restyResponse, err := client.v1SharderGetSCState(t, client.FaucetSmartContractAddress, registeredWallet.ClientID, nil)
		require.Nil(t, err)
		require.Equal(t, client.HttpOkStatus, restyResponse.Status())
		require.Equal(t, tokenomics.ZcnToInt(sharderSCStateResponse.Used), int64(1), "SCState does not seem to be valid")
	})

	t.Run("Get SCState of faucet SC, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, _ := registerWallet(t)

		_, restyResponse, err := client.v1SharderGetSCState(t, client.FaucetSmartContractAddress, registeredWallet.ClientID, nil)
		require.Nil(t, err)
		require.Equal(t, client.HttpNotFoundStatus, restyResponse.Status())
	})
}
