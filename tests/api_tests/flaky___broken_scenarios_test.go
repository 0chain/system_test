//nolint:gocritic
//nolint:gocyclo
package api_tests

import (
	"encoding/hex"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

/*
Tests in here are skipped until the feature has been fixed
*/
func Test___BrokenScenariosRegisterWallet(t *testing.T) {
	t.Parallel()

	t.Run("Register wallet API call should be successful, ignoring invalid creation date", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonic(t)
		expectedPublicKey, _ := crypto.GenerateKeys(t, mnemonic)
		publicKeyBytes, _ := hex.DecodeString(expectedPublicKey)
		expectedClientId := crypto.Sha3256(publicKeyBytes)
		invalidCreationDate := -1

		walletRequest := model.Wallet{Id: expectedClientId, PublicKey: expectedPublicKey, CreationDate: &invalidCreationDate}

		registeredWallet, httpResponse, err := v1ClientPut(t, walletRequest)

		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, httpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", httpResponse)
		require.Equal(t, "200 OK", httpResponse.Status())
		require.Equal(t, registeredWallet.Id, expectedClientId)
		require.Equal(t, registeredWallet.PublicKey, expectedPublicKey)
		require.Greater(t, *registeredWallet.CreationDate, 0, "Creation date is an invalid value!")
		require.NotNil(t, registeredWallet.Version)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - client id invalid", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonic(t)
		publicKeyHex, _ := crypto.GenerateKeys(t, mnemonic)
		walletRequest := model.Wallet{Id: "invalid", PublicKey: publicKeyHex}

		walletResponse, httpResponse, err := v1ClientPut(t, walletRequest)

		require.Nil(t, walletResponse, "Expected returned wallet to be nil but was [%s] with http response [%s]", walletResponse, httpResponse)
		require.NotNil(t, err, "Expected error when registering wallet but was nil.")
		require.Equal(t, "400 Bad Request", httpResponse.Status())
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - public key invalid", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonic(t)
		publicKeyHex, _ := crypto.GenerateKeys(t, mnemonic)
		publicKeyBytes, _ := hex.DecodeString(publicKeyHex)
		clientId := crypto.Sha3256(publicKeyBytes)
		walletRequest := model.Wallet{Id: clientId, PublicKey: "invalid"}

		walletResponse, httpResponse, err := v1ClientPut(t, walletRequest)

		require.Nil(t, walletResponse, "Expected returned wallet to be nil but was [%s] with http response [%s]", walletResponse, httpResponse)
		require.NotNil(t, err, "Expected error when registering wallet but was nil.")
		require.Equal(t, "400 Bad Request", httpResponse.Status())
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - empty json body", func(t *testing.T) {
		t.Parallel()

		walletRequest := model.Wallet{}
		walletResponse, httpResponse, err := v1ClientPut(t, walletRequest)

		require.Nil(t, walletResponse, "Expected returned wallet to be nil but was [%s] with http response [%s]", walletResponse, httpResponse)
		require.NotNil(t, err, "Expected error when registering wallet but was nil.")
		require.Equal(t, "400 Bad Request", httpResponse.Status())
	})
}
