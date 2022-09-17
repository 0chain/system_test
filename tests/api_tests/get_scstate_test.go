package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetScState(t *testing.T) {
	t.Parallel()

	t.Run("Get SCState of faucet SC, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		sharderSCStateRequest := model.SharderSCStateRequest{
			SCAddress: endpoint.FaucetSmartContractAddress,
			Key:       registeredWallet.ClientID,
		}

		sharderSCStateResponse, restyResponse, err := v1SharderGetSCState(t, sharderSCStateRequest, endpoint.ConsensusByHttpStatus(endpoint.HttpOkStatus))
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.NotNil(t, sharderSCStateResponse)
		require.Equal(t, tokenomics.ZcnToInt(sharderSCStateResponse.Used), int64(1), "SCState does not seem to be valid")
	})

	t.Run("Get SCState of faucet SC, which has no tokens, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, _ := registerWallet(t)

		sharderSCStateRequest := model.SharderSCStateRequest{
			SCAddress: endpoint.FaucetSmartContractAddress,
			Key:       registeredWallet.ClientID,
		}

		sharderSCStateResponse, restyResponse, err := v1SharderGetSCState(t, sharderSCStateRequest, endpoint.ConsensusByHttpStatus(endpoint.HttpNotFoundStatus))
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.Nil(t, sharderSCStateResponse)
	})
}
