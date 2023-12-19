package api_tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestTxnMaxFee(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Run("Txn put should fail with more than max_txn_fee", func(t *test.SystemTest) {
		wallet := apiClient.CreateWallet(t)
		_, resp, err := apiClient.ExecuteFaucetWithTokensWithFee(t, wallet, 2, 50)
		require.Error(t, err)
		require.Equal(t, err.Error(), "execution consensus is not reached")
		require.Nil(t, resp)
	})
}
