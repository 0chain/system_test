package api_tests

import (
	"encoding/hex"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExecuteFaucet(t *testing.T) {
	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()
		mnemonic := crypto.GenerateMnemonic(t)
		registeredWallet, rawHttpResponse, err := registerWalletForMnemonic(t, mnemonic)
		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, rawHttpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", rawHttpResponse)

		keyPair := crypto.GenerateKeys(t, mnemonic)
		transactionPutResponse, rawHttpResponse, err := executeFaucet(t, keyPair)
		require.NotNil(t, transactionPutResponse, "Transaction execute response was unexpectedly nil! with http response [%s]", rawHttpResponse)

	})
}

func executeFaucet(t *testing.T, keyPair model.KeyPair) (*model.Transaction, *resty.Response, error) {
	publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey)
	walletRequest := model.Transaction{PublicKey: string(publicKeyBytes)}

	transactionResponse, httpResponse, httpError := v1TransactionPut(t, walletRequest)

	return transactionResponse, httpResponse, httpError
}
