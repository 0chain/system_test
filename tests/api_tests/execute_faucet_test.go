package api_tests

import (
	"encoding/hex"
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestExecuteFaucet(t *testing.T) {
	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()
		mnemonic := "offer crater property public myth middle crop boost shallow donkey icon lend decrease rifle smooth supply toilet method humor type elegant knock uncle more"
		registeredWallet, rawHttpResponse, err := registerWalletForMnemonic(t, mnemonic)
		require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, rawHttpResponse)
		require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", rawHttpResponse)

		keyPair := crypto.GenerateKeys(t, mnemonic)
		transactionPutResponse, rawHttpResponse, err := executeFaucet(t, registeredWallet.Id, keyPair)
		require.NotNil(t, transactionPutResponse, "Transaction execute response was unexpectedly nil! with http response [%s]", rawHttpResponse)

	})
}

func executeFaucet(t *testing.T, clientId string, keyPair model.KeyPair) (*model.Transaction, *resty.Response, error) {

	faucetRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 10000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  "{\"name\":\"pour\",\"input\":{},\"name\":null}",
		ToClientId:       FAUCET_SMART_CONTRACT_ADDRESS,
		CreationDate:     time.Now().Unix(),
		ClientId:         clientId,
		Version:          "1.0",
	}

	hash(&faucetRequest)
	sign(&faucetRequest, keyPair)

	transactionResponse, httpResponse, httpError := v1TransactionPut(t, faucetRequest)

	return transactionResponse, httpResponse, httpError
}

func hash(request *model.Transaction) {
	var transactionDataHash = crypto.Sha3256([]byte(request.TransactionData))

	var hashData = blankIfNil(request.CreationDate) + ":" +
		blankIfNil(request.ClientId) + ":" +
		blankIfNil(request.ToClientId) + ":" +
		blankIfNil(request.TransactionValue) + ":" +
		transactionDataHash

	var overallHash = crypto.Sha3256([]byte(hashData))
	request.Hash = overallHash
}

func sign(request *model.Transaction, pair model.KeyPair) {
	hashToSign, _ := hex.DecodeString(request.Hash)
	request.Signature = pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
}

func blankIfNil(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return fmt.Sprintf("%v", obj)
}
