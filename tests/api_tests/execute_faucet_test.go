package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExecuteFaucet(t *testing.T) {
	t.Parallel()

	t.Run("Execute Faucet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		faucetTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData()},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, faucetTransactionPutResponse)

		faucetTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: faucetTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, faucetTransactionGetConfirmationResponse)

		require.True(t, faucetTransactionPutResponse.Async)
		require.NotNil(t, faucetTransactionPutResponse.Entity)
		require.NotNil(t, faucetTransactionPutResponse.Entity.ChainId)
		require.Zero(t, faucetTransactionPutResponse.Entity.TransactionOutput)
		require.Zero(t, faucetTransactionPutResponse.Entity.TransactionStatus)

		require.Equal(t, faucetTransactionPutResponse.Request.Hash, faucetTransactionPutResponse.Entity.Hash)
		require.Equal(t, faucetTransactionPutResponse.Request.Version, faucetTransactionPutResponse.Entity.Version)
		require.Equal(t, faucetTransactionPutResponse.Request.ClientId, faucetTransactionPutResponse.Entity.ClientId)
		require.Equal(t, faucetTransactionPutResponse.Request.ToClientId, faucetTransactionPutResponse.Entity.ToClientId)
		require.Equal(t, faucetTransactionPutResponse.Request.PublicKey, faucetTransactionPutResponse.Entity.PublicKey)
		require.Equal(t, faucetTransactionPutResponse.Request.TransactionData, faucetTransactionPutResponse.Entity.TransactionData)
		require.Equal(t, faucetTransactionPutResponse.Request.TransactionValue, faucetTransactionPutResponse.Entity.TransactionValue)
		require.Equal(t, faucetTransactionPutResponse.Request.Signature, faucetTransactionPutResponse.Entity.Signature)
		require.Equal(t, faucetTransactionPutResponse.Request.CreationDate, faucetTransactionPutResponse.Entity.CreationDate)
		require.Equal(t, faucetTransactionPutResponse.Request.TransactionFee, faucetTransactionPutResponse.Entity.TransactionFee)
		require.Equal(t, faucetTransactionPutResponse.Request.TransactionType, faucetTransactionPutResponse.Entity.TransactionType)

		transactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: faucetTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, transactionGetConfirmationResponse)

		require.Equal(t, client.TxVersion, transactionGetConfirmationResponse.Version)
		require.NotNil(t, transactionGetConfirmationResponse.BlockHash)
		require.NotNil(t, transactionGetConfirmationResponse.PreviousBlockHash)
		require.Greater(t, transactionGetConfirmationResponse.CreationDate, int64(0))
		require.NotNil(t, transactionGetConfirmationResponse.MinerID)
		require.Greater(t, transactionGetConfirmationResponse.Round, int64(0))
		require.NotNil(t, transactionGetConfirmationResponse.Status)
		require.NotNil(t, transactionGetConfirmationResponse.RoundRandomSeed)
		require.NotNil(t, transactionGetConfirmationResponse.StateChangesCount)
		require.NotNil(t, transactionGetConfirmationResponse.MerkleTreeRoot)
		require.NotNil(t, transactionGetConfirmationResponse.MerkleTreePath)
		require.NotNil(t, transactionGetConfirmationResponse.ReceiptMerkleTreeRoot)
		require.NotNil(t, transactionGetConfirmationResponse.ReceiptMerkleTreePath)
		require.NotNil(t, transactionGetConfirmationResponse.Transaction.TransactionOutput)
		require.NotNil(t, transactionGetConfirmationResponse.Transaction.TxnOutputHash)

		clientGetBalanceResponse, resp, err := apiClient.V1ClientGetBalance(
			model.ClientGetBalanceRequest{
				ClientID: wallet.ClientID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, clientGetBalanceResponse)
		require.Equal(t, tokenomics.IntToZCN(1), clientGetBalanceResponse.Balance)
	})
}

//func confirmTransactionWithoutAssertion(t *testing.T, hash string, maxPollDuration time.Duration, consensusCategoriser client.ConsensusMetFunction) (*model.Confirmation, *resty.Response, error) { //nolint
//	t.Logf("Confirming transaction...")
//	confirmation, httpResponse, err := client.v1TransactionGetConfirmation(t, hash, consensusCategoriser)
//
//	wait.PoolImmediately(maxPollDuration, func() bool {
//		confirmation, httpResponse, err = client.v1TransactionGetConfirmation(t, hash, consensusCategoriser)
//
//		return httpResponse.StatusCode() == http.StatusOK
//	})
//
//	return confirmation, httpResponse, err
//}

//func getBalance(t *testing.T, clientId string) *model.Balance {
//	balance, httpResponse, err := getBalanceWithoutAssertion(t, clientId)
//
//	require.NotNil(t, balance, "Balance was unexpectedly nil! with http response [%s]", httpResponse)
//	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
//	require.Equal(t, client.HttpOkStatus, httpResponse.Status())
//
//	return balance
//}

//func getBalanceWithoutAssertion(t *testing.T, clientId string) (*model.Balance, *resty.Response, error) { //nolint
//	t.Logf("Getting balance...")
//	balance, httpResponse, err := client.v1ClientGetBalance(t, clientId, nil)
//	return balance, httpResponse, err
//}

//func executeFaucet(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair) (*model.TransactionResponse, *model.Confirmation) {
//t.Logf("Executing faucet...")
//
//faucetRequest := model.Transaction{
//	PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
//	TxnOutputHash:    "",
//	TransactionValue: 10000000000,
//	TransactionType:  1000,
//	TransactionFee:   0,
//	TransactionData:  "{\"name\":\"pour\",\"input\":{},\"name\":null}",
//	ToClientId:       client.FaucetSmartContractAddress,
//	CreationDate:     time.Now().Unix(),
//	ClientId:         wallet.ClientID,
//	Version:          "1.0",
//	TransactionNonce: wallet.Nonce + 1,
//}
//faucetTransaction := executeTransaction(t, &faucetRequest, keyPair)
//confirmation, _ := confirmTransaction(t, wallet, faucetTransaction.Entity, 2*time.Minute)

//	return faucetTransaction, confirmation
//}

//func executeTransactionWithoutAssertion(t *testing.T, txnRequest *model.Transaction, keyPair *model.KeyPair) (*model.TransactionResponse, *resty.Response, error) { //nolint
//	crypto.HashTransaction(txnRequest)
//	crypto.SignTransaction(txnRequest, keyPair)
//
//	transactionResponse, httpResponse, err := client.v1TransactionPut(t, txnRequest, nil)
//
//	return transactionResponse, httpResponse, err
//}
