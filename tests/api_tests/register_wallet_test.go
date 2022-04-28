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
		expectedPublicKey, _ := crypto.GenerateKeys(t, mnemonic)
		publicKeyBytes, _ := hex.DecodeString(expectedPublicKey)
		expectedClientId := crypto.Sha3256(publicKeyBytes)

		registeredWallet, rawHttpResponse, err := registerWalletForMnemonic(t, mnemonic)

		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, rawHttpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", rawHttpResponse)
		require.Equal(t, "200 OK", rawHttpResponse.Status())
		require.Equal(t, registeredWallet.Id, expectedClientId)
		require.Equal(t, registeredWallet.PublicKey, expectedPublicKey)
		require.Greater(t, *registeredWallet.CreationDate, 0, "Creation date is an invalid value!")
		require.NotNil(t, registeredWallet.Version)
	})
}

func registerNewWallet(t *testing.T) (*model.Wallet, *resty.Response, error) {
	mnemonic := crypto.GenerateMnemonic(t)
	return registerWalletForMnemonic(t, mnemonic)
}

func registerWalletForMnemonic(t *testing.T, mnemonic string) (*model.Wallet, *resty.Response, error) {
	publicKeyHex, _ := crypto.GenerateKeys(t, mnemonic)
	publicKeyBytes, _ := hex.DecodeString(publicKeyHex)
	clientId := crypto.Sha3256(publicKeyBytes)
	walletRequest := model.Wallet{Id: clientId, PublicKey: publicKeyHex}

	walletResponse, httpResponse, httpError := v1ClientPut(t, walletRequest)

	return walletResponse, httpResponse, httpError
}
