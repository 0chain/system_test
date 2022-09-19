//nolint:gocritic
//nolint:gocyclo
package api_tests

import (
	"encoding/hex"
	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/system_test/internal/api/util/client"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
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
		require.NotNil(t, err)

		expectedClientId := encryption.Hash(publicKeyBytes)
		invalidCreationDate := -1

		wallet, resp, err := apiClient.V1ClientPut(
			model.ClientPutRequest{
				ClientID:     expectedClientId,
				ClientKey:    expectedKeyPair.PublicKey.SerializeToHexStr(),
				CreationDate: &invalidCreationDate,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)
		require.Equal(t, wallet.ClientID, expectedClientId)
		require.Equal(t, wallet.ClientKey, expectedKeyPair.PublicKey.SerializeToHexStr())
		require.NotZero(t, wallet.DateCreated, "Creation date is an invalid value!")
		require.NotNil(t, wallet.Version)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - client id invalid", func(t *testing.T) {
		t.Parallel()

		mnemonics := crypto.GenerateMnemonics()
		expectedKeyPair := crypto.GenerateKeys(mnemonics)

		wallet, resp, err := apiClient.V1ClientPut(
			model.ClientPutRequest{
				ClientID:  "invalid",
				ClientKey: expectedKeyPair.PublicKey.SerializeToHexStr(),
			},
			client.HttpNotFoundStatus)
		require.NotNil(t, err)
		require.NotNil(t, resp)
		require.Nil(t, wallet)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - public key invalid", func(t *testing.T) {
		t.Parallel()

		mnemonics := crypto.GenerateMnemonics()
		expectedKeyPair := crypto.GenerateKeys(mnemonics)
		publicKeyBytes, err := hex.DecodeString(expectedKeyPair.PublicKey.SerializeToHexStr())
		require.NotNil(t, err)

		clientId := encryption.Hash(publicKeyBytes)

		wallet, resp, err := apiClient.V1ClientPut(
			model.ClientPutRequest{
				ClientID:  clientId,
				ClientKey: "invalid",
			},
			client.HttpNotFoundStatus)
		require.NotNil(t, err)
		require.NotNil(t, resp)
		require.Nil(t, wallet)
	})

	t.Run("Register wallet API call should be unsuccessful given an invalid request - empty json body", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(
			model.ClientPutRequest{},
			client.HttpNotFoundStatus)
		require.NotNil(t, err)
		require.NotNil(t, resp)
		require.Nil(t, wallet)
	})
}
