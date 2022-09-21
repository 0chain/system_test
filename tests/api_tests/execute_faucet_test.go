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

		wallet := apiClient.RegisterWalletWrapper(t)

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
		require.Equal(t, *tokenomics.IntToZCN(1), clientGetBalanceResponse.Balance)
	})
}
