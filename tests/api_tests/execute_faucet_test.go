package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
)

func TestExecuteFaucet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *test.SystemTest) {
		wallet := apiClient.CreateWallet(t)

		apiClient.ExecuteFaucetWithAssertions(t, wallet, client.TxSuccessfulStatus)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		require.Equal(t, *tokenomics.IntToZCN(1), walletBalance.Balance)
	})
}
