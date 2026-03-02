package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestGetSCState(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get SCState of faucet SC, should work")

	t.Parallel()

	t.Run("Get SCState of faucet SC, should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		// Pour from faucet so the SC has state for this wallet to query.
		apiClient.FundWallet(t, wallet, 2.0, client.TxSuccessfulStatus)

		// SC state may have a 1-2 block delay after transaction confirmation; retry for up to 30s.
		var scStateGetResponse *model.SCStateGetResponse
		wait.PoolImmediately(t, 30*time.Second, func() bool {
			var err error
			scStateGetResponse, _, err = apiClient.V1SharderGetSCState(
				t,
				model.SCStateGetRequest{
					SCAddress: client.FaucetSmartContractAddress,
					Key:       wallet.Id,
				},
				client.HttpOkStatus)
			return err == nil && scStateGetResponse != nil
		})
		require.NotNil(t, scStateGetResponse)
	})

	t.Run("Get SCState with invalid SC address should fail", func(t *test.SystemTest) {
		wallet := createWallet(t)

		_, resp, err := apiClient.V1SharderGetSCState(
			t,
			model.SCStateGetRequest{
				SCAddress: "invalid_sc_address",
				Key:       wallet.Id,
			},
			client.HttpBadRequestStatus)

		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})
}
