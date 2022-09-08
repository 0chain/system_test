package api_tests

import (
	"encoding/hex"
	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"strconv"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/go-resty/resty/v2" //nolint
	"github.com/stretchr/testify/require"
)

func TestRegisterWallet(t *testing.T) {
	t.Parallel()

	t.Run("Register wallet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonic(t)

		registeredWallet, keyPair, rawHttpResponse, err := registerWalletForMnemonicWithoutAssertion(t, mnemonic)

		publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
		expectedClientId := encryption.Hash(publicKeyBytes)
		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, rawHttpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", rawHttpResponse)
		require.Equal(t, endpoint.HttpOkStatus, rawHttpResponse.Status())
		require.Equal(t, registeredWallet.ClientID, expectedClientId)
		require.Equal(t, registeredWallet.ClientKey, keyPair.PublicKey.SerializeToHexStr())
		require.Greater(t, registeredWallet.MustConvertDateCreatedToInt(), 0, "Creation date is an invalid value!")
		require.NotNil(t, registeredWallet.Version)
	})
}

func registerWallet(t *testing.T) (*model.Wallet, *model.KeyPair) {
	mnemonic := crypto.GenerateMnemonic(t)

	return registerWalletForMnemonic(t, mnemonic)
}

func registerWalletForMnemonic(t *testing.T, mnemonic string) (*model.Wallet, *model.KeyPair) {
	registeredWallet, keyPair, httpResponse, err := registerWalletForMnemonicWithoutAssertion(t, mnemonic)

	publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	clientId := encryption.Hash(publicKeyBytes)

	require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, httpResponse)
	require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", httpResponse)
	require.Equal(t, endpoint.HttpOkStatus, httpResponse.Status())
	require.Equal(t, registeredWallet.ClientID, clientId)
	require.Equal(t, registeredWallet.ClientKey, keyPair.PublicKey.SerializeToHexStr())
	require.Greater(t, registeredWallet.MustConvertDateCreatedToInt(), 0, "Creation date is an invalid value!")
	require.NotNil(t, registeredWallet.Version)

	return registeredWallet, keyPair
}

func registerWalletForMnemonicWithoutAssertion(t *testing.T, mnemonic string) (*model.Wallet, *model.KeyPair, *resty.Response, error) { //nolint
	t.Logf("Registering wallet...")
	keyPair := crypto.GenerateKeys(t, mnemonic)
	publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	clientId := encryption.Hash(publicKeyBytes)
	walletRequest := model.ClientPutWalletRequest{Id: clientId, PublicKey: keyPair.PublicKey.SerializeToHexStr()}

	registeredWallet, httpResponse, err := v1ClientPut(t, walletRequest, endpoint.ConsensusByHttpStatus(endpoint.HttpOkStatus))

	wallet := &model.Wallet{
		ClientID:  registeredWallet.Id,
		ClientKey: registeredWallet.PublicKey,
		Keys: []*sys.KeyPair{{
			PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
			PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
		}},
		DateCreated: strconv.Itoa(*registeredWallet.CreationDate),
		Mnemonics:   mnemonic,
		Version:     registeredWallet.Version,
		Nonce:       registeredWallet.Nonce,
	}

	return wallet, keyPair, httpResponse, err
}
