package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/tokenomics"
)

func TestExecuteFaucet(t *testing.T) {
	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		apiClient.ExecuteFaucetWithAssertions(t, wallet, client.TxSuccessfulStatus)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		require.Equal(t, *tokenomics.IntToZCN(1), walletBalance.Balance)
	})
}
