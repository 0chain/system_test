package api_tests

import (
	"encoding/hex"
	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterWallet(t *testing.T) {
	t.Parallel()

	t.Run("Register wallet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		require.NotZero(t, len(wallet.Keys))

		publicKeyBytes, err := hex.DecodeString(wallet.RawKeys.PublicKey.SerializeToHexStr())
		require.Nil(t, err)

		require.Equal(t, wallet.ClientID, encryption.Hash(publicKeyBytes))
		require.Equal(t, wallet.ClientKey, wallet.MustGetKeyPair().PublicKey)
		require.NotZero(t, wallet.MustConvertDateCreatedToInt(), "creation date is an invalid value!")
		require.NotZero(t, wallet.Version)
	})
}
