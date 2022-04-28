package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonic(t)
		registeredWallet, rawHttpResponse, err := registerWalletForMnemonic(t, mnemonic)
		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, rawHttpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", rawHttpResponse)
	})
}
