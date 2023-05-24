//nolint:gocritic
//nolint:gocyclo
package api_tests

import (
	"encoding/hex"

	"github.com/0chain/system_test/internal/api/util/test"

	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

/*
Tests in here are skipped until the feature has been fixed
*/
func Test___BrokenScenariosRegisterWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Register wallet API call should be successful, ignoring invalid creation date", func(t *test.SystemTest) {
		mnemonic := crypto.GenerateMnemonics(t)
		expectedKeyPair := crypto.GenerateKeys(t, mnemonic)
		publicKeyBytes, _ := hex.DecodeString(expectedKeyPair.PublicKey.SerializeToHexStr())
		expectedClientId := crypto.Sha3256(publicKeyBytes)
		invalidCreationDate := -1

		walletRequest := model.Wallet{Id: expectedClientId, PublicKey: expectedKeyPair.PublicKey.SerializeToHexStr(), CreationDate: &invalidCreationDate}

		registeredWallet, httpResponse, err := apiClient.V1ClientPut(t, walletRequest, client.HttpOkStatus)

		require.Nil(t, err, "Unexpected error [%s] occurred creating wallet with http response [%s]", err, httpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", httpResponse)
		require.Equal(t, "200 OK", httpResponse.Status())
		require.Equal(t, registeredWallet.Id, expectedClientId)
		require.Equal(t, registeredWallet.PublicKey, expectedKeyPair.PublicKey)
		require.Greater(t, *registeredWallet.CreationDate, 0, "Creation date is an invalid value!")
		require.NotNil(t, registeredWallet.Version)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - client id invalid", func(t *test.SystemTest) {
		mnemonic := crypto.GenerateMnemonics(t)
		expectedKeyPair := crypto.GenerateKeys(t, mnemonic)
		walletRequest := model.Wallet{Id: "invalid", PublicKey: expectedKeyPair.PublicKey.SerializeToHexStr()}

		walletResponse, httpResponse, err := apiClient.V1ClientPut(t, walletRequest, client.HttpBadRequestStatus)

		require.Nil(t, walletResponse, "Expected returned wallet to be nil but was [%s] with http response [%s]", walletResponse, httpResponse)
		require.NotNil(t, err, "Expected error when creating wallet but was nil.")
		require.Equal(t, "400 Bad Request", httpResponse.Status())
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - public key invalid", func(t *test.SystemTest) {
		mnemonic := crypto.GenerateMnemonics(t)
		expectedKeyPair := crypto.GenerateKeys(t, mnemonic)
		publicKeyBytes, _ := hex.DecodeString(expectedKeyPair.PublicKey.SerializeToHexStr())
		clientId := crypto.Sha3256(publicKeyBytes)
		walletRequest := model.Wallet{Id: clientId, PublicKey: "invalid"}

		walletResponse, httpResponse, err := apiClient.V1ClientPut(t, walletRequest, client.HttpBadRequestStatus)

		require.Nil(t, walletResponse, "Expected returned wallet to be nil but was [%s] with http response [%s]", walletResponse, httpResponse)
		require.NotNil(t, err, "Expected error when creating wallet but was nil.")
		require.Equal(t, "400 Bad Request", httpResponse.Status())
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - empty json body", func(t *test.SystemTest) {
		walletRequest := model.Wallet{}
		walletResponse, httpResponse, err := apiClient.V1ClientPut(t, walletRequest, client.HttpBadRequestStatus)

		require.Nil(t, walletResponse, "Expected returned wallet to be nil but was [%s] with http response [%s]", walletResponse, httpResponse)
		require.NotNil(t, err, "Expected error when creating wallet but was nil.")
		require.Equal(t, "400 Bad Request", httpResponse.Status())
	})
}
