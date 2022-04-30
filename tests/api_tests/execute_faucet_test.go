package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

func TestExecuteFaucet(t *testing.T) {
	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)

		executeFaucet(t, registeredWallet.Id, keyPair)
		balance := getBalance(t, registeredWallet.Id)
		require.Equal(t, util.IntToZCN(1), balance.Balance)
	})
}

func confirmTransaction(t *testing.T, hash string, maxPollDuration time.Duration) (*model.Confirmation, *resty.Response) {
	confirmation, httpResponse, err := confirmTransactionWithoutAssertion(t, hash, maxPollDuration)

	require.NotNil(t, confirmation, "Confirmation was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred confirming transaction with http response [%s]", err, httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())

	return confirmation, httpResponse
}

func confirmTransactionWithoutAssertion(t *testing.T, hash string, maxPollDuration time.Duration) (*model.Confirmation, *resty.Response, error) {
	confirmation, httpResponse, err := v1TransactionGetConfirmation(t, hash)

	startPollTime := time.Now()
	for httpResponse.StatusCode() != 200 && time.Since(startPollTime) < maxPollDuration {
		t.Logf("Confirmation for txn hash [%s] failed. Will poll until specified duration [%s] has been reached...", hash, maxPollDuration)
		time.Sleep(maxPollDuration / 20)
		confirmation, httpResponse, err = v1TransactionGetConfirmation(t, hash)
	}
	return confirmation, httpResponse, err
}

func getBalance(t *testing.T, clientId string) *model.Balance {
	balance, httpResponse, err := getBalanceWithoutAssertion(t, clientId)

	require.NotNil(t, balance, "Balance was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())

	return balance
}

func getBalanceWithoutAssertion(t *testing.T, clientId string) (*model.Balance, *resty.Response, error) {
	balance, httpResponse, err := v1ClientGetBalance(t, clientId)
	return balance, httpResponse, err
}

func executeFaucet(t *testing.T, clientId string, keyPair model.KeyPair) *model.TransactionResponse {
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

	faucetTransaction := executeTransaction(t, &faucetRequest, keyPair)
	confirmTransaction(t, faucetTransaction.Entity.Hash, 1*time.Minute)

	return faucetTransaction
}

func executeTransaction(t *testing.T, txnRequest *model.Transaction, keyPair model.KeyPair) *model.TransactionResponse {
	transactionResponse, httpResponse, err := executeTransactionWithoutAssertion(t, txnRequest, keyPair)

	require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, httpResponse)
	require.NotNil(t, transactionResponse, "Registered wallet was unexpectedly nil! with http response [%s]", httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())
	require.True(t, transactionResponse.Async)
	require.NotNil(t, transactionResponse.Entity, "Transaction entity was unexpectedly nil! with http response [%s]", httpResponse)
	require.Equal(t, txnRequest.Hash, transactionResponse.Entity.Hash)
	require.Equal(t, txnRequest.Version, transactionResponse.Entity.Version)
	require.Equal(t, txnRequest.ClientId, transactionResponse.Entity.ClientId)
	require.Equal(t, txnRequest.ToClientId, transactionResponse.Entity.ToClientId)
	require.NotNil(t, transactionResponse.Entity.ChainId)
	require.Equal(t, txnRequest.PublicKey, transactionResponse.Entity.PublicKey)
	require.Equal(t, txnRequest.TransactionData, transactionResponse.Entity.TransactionData)
	require.Equal(t, txnRequest.TransactionValue, transactionResponse.Entity.TransactionValue)
	require.Equal(t, txnRequest.Signature, transactionResponse.Entity.Signature)
	require.Equal(t, txnRequest.CreationDate, transactionResponse.Entity.CreationDate)
	require.Equal(t, txnRequest.TransactionFee, transactionResponse.Entity.TransactionFee)
	require.Equal(t, txnRequest.TransactionType, transactionResponse.Entity.TransactionType)
	require.Equal(t, txnRequest.TransactionOutput, transactionResponse.Entity.TransactionOutput)
	require.Equal(t, txnRequest.TxnOutputHash, transactionResponse.Entity.TxnOutputHash)
	require.Equal(t, txnRequest.TransactionStatus, transactionResponse.Entity.TransactionStatus)

	return transactionResponse
}

func executeTransactionWithoutAssertion(t *testing.T, txnRequest *model.Transaction, keyPair model.KeyPair) (*model.TransactionResponse, *resty.Response, error) {
	crypto.Hash(txnRequest)
	crypto.Sign(txnRequest, keyPair)

	transactionResponse, httpResponse, err := v1TransactionPut(t, txnRequest)

	return transactionResponse, httpResponse, err
}
