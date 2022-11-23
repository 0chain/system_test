package api_tests

import (
	"encoding/hex"
	"github.com/0chain/system_test/internal/api/util/test"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestRegisterWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Register wallet API call should be successful given a valid request", func(t *test.SystemTest) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)

		registeredWallet, rawHttpResponse, err := apiClient.RegisterWalletForMnemonicWithoutAssertion(t, mnemonic, client.HttpOkStatus)

		publicKeyBytes, _ := hex.DecodeString(registeredWallet.Keys.PublicKey.SerializeToHexStr())
		expectedClientId := crypto.Sha3256(publicKeyBytes)
		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, rawHttpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", rawHttpResponse)
		require.Equal(t, "200 OK", rawHttpResponse.Status())
		require.Equal(t, registeredWallet.Id, expectedClientId)
		require.Equal(t, registeredWallet.PublicKey, registeredWallet.Keys.PublicKey.SerializeToHexStr())
		require.Greater(t, *registeredWallet.CreationDate, 0, "Creation date is an invalid value!")
		require.NotNil(t, registeredWallet.Version)
	})
}
