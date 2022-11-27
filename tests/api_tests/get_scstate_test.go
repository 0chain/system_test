package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestGetSCState(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Get SCState of faucet SC, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		scStateGetResponse, resp, err := apiClient.V1SharderGetSCState(
			t,
			model.SCStateGetRequest{
				SCAddress: client.FaucetSmartContractAddress,
				Key:       wallet.Id,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scStateGetResponse)
	})

	t.Run("Get SCState of faucet SC, shouldn't work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		scStateGetResponse, resp, err := apiClient.V1SharderGetSCState(
			t,
			model.SCStateGetRequest{
				SCAddress: client.FaucetSmartContractAddress,
				Key:       wallet.Id,
			},
			client.HttpBadRequestStatus)

		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scStateGetResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})
}
