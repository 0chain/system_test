//nolint:gocritic
//nolint:gocyclo
package api_tests

import (
	"encoding/hex"
	"github.com/0chain/system_test/internal/api/util/client"
	"testing"

	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

/*
Tests in here are skipped until the feature has been fixed
*/
func Test___BrokenScenariosRegisterWallet(t *testing.T) {
	t.Skip()
	t.Parallel()

	t.Run("Register wallet API call should be successful, ignoring invalid creation date", func(t *testing.T) {
		t.Parallel()

		mnemonics := crypto.GenerateMnemonics()
		expectedKeyPair := crypto.GenerateKeys(mnemonics)
		publicKeyBytes, err := hex.DecodeString(expectedKeyPair.PublicKey.SerializeToHexStr())
		require.Nil(t, err)

		expectedClientId := crypto.Sha3256(publicKeyBytes)
		invalidCreationDate := -1

		apiClient.RegisterWalletWithAssertions(t, expectedClientId, expectedKeyPair.PublicKey.SerializeToHexStr(), &invalidCreationDate, false, client.HttpOkStatus)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - client id invalid", func(t *testing.T) {
		t.Parallel()

		mnemonics := crypto.GenerateMnemonics()
		expectedKeyPair := crypto.GenerateKeys(mnemonics)

		apiClient.RegisterWalletWithAssertions(t, "invalid", expectedKeyPair.PublicKey.SerializeToHexStr(), nil, false, client.HttpOkStatus)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - public key invalid", func(t *testing.T) {
		t.Parallel()

		mnemonics := crypto.GenerateMnemonics()
		expectedKeyPair := crypto.GenerateKeys(mnemonics)
		publicKeyBytes, err := hex.DecodeString(expectedKeyPair.PublicKey.SerializeToHexStr())
		require.Nil(t, err)

		clientId := crypto.Sha3256(publicKeyBytes)

		wallet := apiClient.RegisterWallet(t, clientId, "invalid", nil, false, client.HttpNotFoundStatus)

		require.NotEqual(t, wallet.ClientID, clientId)

		keyPair, err := wallet.GetKeyPair()
		require.Nil(t, err)
		require.NotEqual(t, wallet.ClientKey, keyPair.PublicKey)

		_, err = wallet.ConvertDateCreatedToInt()
		require.NotNil(t, err)

		require.Zero(t, wallet.Version)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - empty json body", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, false, client.HttpOkStatus)

		require.NotZero(t, wallet.ClientID)
		require.Zero(t, wallet.ClientKey)

		dateCreated, err := wallet.ConvertDateCreatedToInt()
		require.Nil(t, err)
		require.NotZero(t, dateCreated)

		require.NotZero(t, wallet.Version)
	})
}
