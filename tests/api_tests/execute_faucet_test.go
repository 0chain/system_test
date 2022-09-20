package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/0chain/system_test/internal/api/util/wait"
	resty "github.com/go-resty/resty/v2" //nolint
	"net/http"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestExecuteFaucet(t *testing.T) {
	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)

		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		balance := getBalance(t, registeredWallet.ClientID)
		require.Equal(t, tokenomics.IntToZCN(1), balance.Balance)
	})
}

func confirmTransaction(t *testing.T, wallet *model.Wallet, sentTransaction model.Transaction, maxPollDuration time.Duration) (*model.Confirmation, *resty.Response) { //nolint
	consensus := func(response *resty.Response, resolvedObject interface{}) bool { //nolint
		confirmation := resolvedObject.(**model.Confirmation)
		return (*confirmation) != nil && (*confirmation).Transaction.TransactionStatus == 1
	}

	confirmation, httpResponse, err := confirmTransactionWithoutAssertion(t, sentTransaction.Hash, maxPollDuration, consensus)

	require.NotNil(t, confirmation, "Confirmation was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred confirming transaction with http response [%s]", err, httpResponse)
	require.Equal(t, endpoint.HttpOkStatus, httpResponse.Status())
	require.Equal(t, "1.0", confirmation.Version, "version did not match expected")
	require.Equal(t, sentTransaction.Hash, confirmation.Hash, "hash did not match expected")
	require.NotNil(t, confirmation.BlockHash)
	require.NotNil(t, confirmation.PreviousBlockHash)
	require.Greater(t, confirmation.CreationDate, int64(0))
	require.NotNil(t, confirmation.MinerID)
	require.Greater(t, confirmation.Round, int64(0))
	require.NotNil(t, confirmation.Status)
	require.NotNil(t, confirmation.RoundRandomSeed)
	require.NotNil(t, confirmation.StateChangesCount)
	require.NotNil(t, confirmation.MerkleTreeRoot)
	require.NotNil(t, confirmation.MerkleTreePath)
	require.NotNil(t, confirmation.ReceiptMerkleTreeRoot)
	require.NotNil(t, confirmation.ReceiptMerkleTreePath)
	require.NotNil(t, confirmation.Transaction.TransactionOutput)
	require.NotNil(t, confirmation.Transaction.TxnOutputHash)

	assertTransactionEquals(t, &sentTransaction, confirmation.Transaction)

	wallet.Nonce++

	return confirmation, httpResponse
}

func confirmTransactionWithoutAssertion(t *testing.T, hash string, maxPollDuration time.Duration, consensusCategoriser endpoint.ConsensusMetFunction) (*model.Confirmation, *resty.Response, error) { //nolint
	t.Logf("Confirming transaction...")
	var (
		confirmation *model.Confirmation
		httpResponse *resty.Response //nolint
		err          error
	)

	wait.PoolImmediately(maxPollDuration, func() bool {
		confirmation, httpResponse, err = v1TransactionGetConfirmation(t, hash, consensusCategoriser)

		return httpResponse.StatusCode() == http.StatusOK
	})

	return confirmation, httpResponse, err
}

func getBalance(t *testing.T, clientId string) *model.Balance {
	balance, httpResponse, err := getBalanceWithoutAssertion(t, clientId)

	require.NotNil(t, balance, "Balance was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
	require.Equal(t, endpoint.HttpOkStatus, httpResponse.Status())

	return balance
}

func getBalanceWithoutAssertion(t *testing.T, clientId string) (*model.Balance, *resty.Response, error) { //nolint
	t.Logf("Getting balance...")
	balance, httpResponse, err := v1ClientGetBalance(t, clientId, nil)
	return balance, httpResponse, err
}

func executeFaucet(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair) (*model.TransactionResponse, *model.Confirmation) {
	t.Logf("Executing faucet...")

	faucetRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 10000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  "{\"name\":\"pour\",\"input\":{},\"name\":null}",
		ToClientId:       endpoint.FaucetSmartContractAddress,
		CreationDate:     time.Now().Unix(),
		ClientId:         wallet.ClientID,
		Version:          "1.0",
		TransactionNonce: wallet.Nonce + 1,
	}
	faucetTransaction := executeTransaction(t, &faucetRequest, keyPair)
	confirmation, _ := confirmTransaction(t, wallet, faucetTransaction.Entity, 5*time.Minute)

	return faucetTransaction, confirmation
}

func executeTransaction(t *testing.T, txnRequest *model.Transaction, keyPair *model.KeyPair) *model.TransactionResponse {
	transactionResponse, httpResponse, err := executeTransactionWithoutAssertion(t, txnRequest, keyPair)

	require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, httpResponse)
	require.NotNil(t, transactionResponse, "Registered wallet was unexpectedly nil! with http response [%s]", httpResponse)
	require.Equal(t, endpoint.HttpOkStatus, httpResponse.Status())
	require.True(t, transactionResponse.Async)
	require.NotNil(t, transactionResponse.Entity, "Transaction entity was unexpectedly nil! with http response [%s]", httpResponse)
	require.NotNil(t, transactionResponse.Entity.ChainId)
	require.Equal(t, "", txnRequest.TransactionOutput)
	assertTransactionEquals(t, txnRequest, &transactionResponse.Entity)
	require.Equal(t, 0, txnRequest.TransactionStatus)

	return transactionResponse
}

func assertTransactionEquals(t *testing.T, txnRequest *model.Transaction, transactionResponse *model.Transaction) {
	require.Equal(t, txnRequest.Hash, transactionResponse.Hash)
	require.Equal(t, txnRequest.Version, transactionResponse.Version)
	require.Equal(t, txnRequest.ClientId, transactionResponse.ClientId)
	require.Equal(t, txnRequest.ToClientId, transactionResponse.ToClientId)
	require.Equal(t, txnRequest.PublicKey, transactionResponse.PublicKey)
	require.Equal(t, txnRequest.TransactionData, transactionResponse.TransactionData)
	require.Equal(t, txnRequest.TransactionValue, transactionResponse.TransactionValue)
	require.Equal(t, txnRequest.Signature, transactionResponse.Signature)
	require.Equal(t, txnRequest.CreationDate, transactionResponse.CreationDate)
	require.Equal(t, txnRequest.TransactionFee, transactionResponse.TransactionFee)
	require.Equal(t, txnRequest.TransactionType, transactionResponse.TransactionType)
}

func executeTransactionWithoutAssertion(t *testing.T, txnRequest *model.Transaction, keyPair *model.KeyPair) (*model.TransactionResponse, *resty.Response, error) { //nolint
	crypto.HashTransaction(txnRequest)
	crypto.SignTransaction(t, txnRequest, keyPair)

	transactionResponse, httpResponse, err := v1TransactionPut(t, txnRequest, nil)

	return transactionResponse, httpResponse, err
}
