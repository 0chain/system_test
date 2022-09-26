package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExecuteFaucet(t *testing.T) {
	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		apiClient.ExecuteFaucetWithAssertions(t, wallet, client.TxSuccessfulStatus)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		require.Equal(t, *tokenomics.IntToZCN(1), walletBalance.Balance)
	})
}
