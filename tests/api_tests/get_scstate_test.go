package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetSCState(t *testing.T) {
	t.Parallel()

	t.Run("Get SCState of faucet SC, should work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		transactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData()},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionPutResponse)

		transactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: transactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionGetConfirmationResponse)

		scStateGetResponse, resp, err := apiClient.V1SharderGetSCState(
			model.SCStateGetRequest{
				SCAddress: client.FaucetSmartContractAddress,
				Key:       wallet.ClientID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scStateGetResponse)
	})

	t.Run("Get SCState of faucet SC, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		scStateGetResponse, resp, err := apiClient.V1SharderGetSCState(
			model.SCStateGetRequest{
				SCAddress: client.FaucetSmartContractAddress,
				Key:       wallet.ClientID,
			},
			client.HttpNotFoundStatus)

		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scStateGetResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})
}
