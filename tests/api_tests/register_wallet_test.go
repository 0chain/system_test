package api_tests

import (
	"encoding/hex"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRegisterWallet(t *testing.T) {
	t.Parallel()

	t.Run("Register wallet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonic(t)

		registeredWallet, keyPair, rawHttpResponse, err := registerWalletForMnemonicWithoutAssertion(t, mnemonic)

		publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
		expectedClientId := crypto.Sha3256(publicKeyBytes)
		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, rawHttpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", rawHttpResponse)
		require.Equal(t, "200 OK", rawHttpResponse.Status())
		require.Equal(t, registeredWallet.Id, expectedClientId)
		require.Equal(t, registeredWallet.PublicKey, keyPair.PublicKey.SerializeToHexStr())
		require.Greater(t, *registeredWallet.CreationDate, 0, "Creation date is an invalid value!")
		require.NotNil(t, registeredWallet.Version)
	})
}

func registerWallet(t *testing.T) (*model.Wallet, model.KeyPair) {
	mnemonic := crypto.GenerateMnemonic(t)

	return registerWalletForMnemonic(t, mnemonic)
}

func registerWalletForMnemonic(t *testing.T, mnemonic string) (*model.Wallet, model.KeyPair) {
	registeredWallet, keyPair, httpResponse, err := registerWalletForMnemonicWithoutAssertion(t, mnemonic)

	publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	clientId := crypto.Sha3256(publicKeyBytes)

	require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, httpResponse)
	require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())
	require.Equal(t, registeredWallet.Id, clientId)
	require.Equal(t, registeredWallet.PublicKey, keyPair.PublicKey.SerializeToHexStr())
	require.Greater(t, *registeredWallet.CreationDate, 0, "Creation date is an invalid value!")
	require.NotNil(t, registeredWallet.Version)

	return registeredWallet, keyPair
}

func registerWalletForMnemonicWithoutAssertion(t *testing.T, mnemonic string) (*model.Wallet, model.KeyPair, *resty.Response, error) {
	keyPair := crypto.GenerateKeys(t, mnemonic)
	publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	clientId := crypto.Sha3256(publicKeyBytes)
	walletRequest := model.Wallet{Id: clientId, PublicKey: keyPair.PublicKey.SerializeToHexStr()}

	registeredWallet, httpResponse, err := v1ClientPut(t, walletRequest)

	return registeredWallet, keyPair, httpResponse, err
}
