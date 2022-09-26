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

		wallet := apiClient.RegisterWalletWrapper(t, client.HttpOkStatus)
		apiClient.ExecuteFaucetWrapper(t, wallet, client.HttpOkStatus, client.TxSuccessfulStatus)

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

		wallet := apiClient.RegisterWalletWrapper(t, client.HttpOkStatus)

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
