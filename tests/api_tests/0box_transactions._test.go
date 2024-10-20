package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxTransactions(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("get paginated transactions list while creating pit id", 1*time.Minute, func(t *test.SystemTest) {
		txnsData, resp, err := zboxClient.GetTransactionsList(t, "")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotNil(t, txnsData, "Nil transaction response received")
		require.NotEmpty(t, txnsData.PitId, "")
		require.NotEmpty(t, txnsData.Transactions, "No transactions data received")
		txnDataByHash, resp, err := apiClient.V1TransactionGetConfirmation(t,
			model.TransactionGetConfirmationRequest{
				Hash: txnsData.Transactions[0].Hash,
			},
			client.HttpOkStatus)

		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotNil(t, txnDataByHash, "Nil transaction response received with hash request")
		txnFromZbox := txnsData.Transactions[0]
		txnByHash := txnDataByHash.Transaction
		require.NotNil(t, txnByHash, "No transaction received with hash request")
		require.Equal(t, txnFromZbox.Hash, txnByHash.Hash)
		require.Equal(t, txnFromZbox.BlockHash, txnDataByHash.BlockHash)
		require.Equal(t, txnFromZbox.Round, txnDataByHash.Round)
		require.Equal(t, txnFromZbox.Version, txnByHash.Version)
		require.Equal(t, txnFromZbox.ClientId, txnByHash.ClientId)
		require.Equal(t, txnFromZbox.ToClientId, txnByHash.ToClientId)
		require.Equal(t, txnFromZbox.TransactionData, txnByHash.TransactionData)
		require.Equal(t, txnFromZbox.TransactionOutput, txnByHash.TransactionOutput)
		require.Equal(t, txnFromZbox.TransactionType, txnByHash.TransactionType)
		require.Equal(t, txnFromZbox.Fee, txnByHash.TransactionFee)
		require.Equal(t, txnFromZbox.Nonce, txnByHash.TransactionNonce)
		require.Equal(t, txnFromZbox.Status, txnByHash.TransactionStatus)
		require.Equal(t, txnFromZbox.Signature, txnByHash.Signature)
		require.Equal(t, txnFromZbox.Value, txnByHash.TransactionValue)
		require.Equal(t, txnFromZbox.OutputHash, txnByHash.TxnOutputHash)
		require.Equal(t, txnFromZbox.CreationDate/int64(1e9), txnByHash.CreationDate)

		pitId := txnsData.PitId
		txnsData, resp, err = zboxClient.GetTransactionsList(t, pitId)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotNil(t, txnsData, "Nil transaction response received")

		require.NotEmpty(t, txnsData.PitId, "")
		require.NotEmpty(t, txnsData.Transactions, "No transactions data received")
		txnDataByHash, resp, err = apiClient.V1TransactionGetConfirmation(t,
			model.TransactionGetConfirmationRequest{
				Hash: txnsData.Transactions[0].Hash,
			},
			client.HttpOkStatus)

		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotNil(t, txnDataByHash, "Nil transaction response received with hash request")
		txnFromZbox = txnsData.Transactions[0]
		txnByHash = txnDataByHash.Transaction
		require.NotNil(t, txnByHash, "No transaction received with hash request")
		require.Equal(t, txnFromZbox.Hash, txnByHash.Hash)
		require.Equal(t, txnFromZbox.BlockHash, txnDataByHash.BlockHash)
		require.Equal(t, txnFromZbox.Round, txnDataByHash.Round)
		require.Equal(t, txnFromZbox.Version, txnByHash.Version)
		require.Equal(t, txnFromZbox.ClientId, txnByHash.ClientId)
		require.Equal(t, txnFromZbox.ToClientId, txnByHash.ToClientId)
		require.Equal(t, txnFromZbox.TransactionData, txnByHash.TransactionData)
		require.Equal(t, txnFromZbox.TransactionOutput, txnByHash.TransactionOutput)
		require.Equal(t, txnFromZbox.TransactionType, txnByHash.TransactionType)
		require.Equal(t, txnFromZbox.Fee, txnByHash.TransactionFee)
		require.Equal(t, txnFromZbox.Nonce, txnByHash.TransactionNonce)
		require.Equal(t, txnFromZbox.Status, txnByHash.TransactionStatus)
		require.Equal(t, txnFromZbox.Signature, txnByHash.Signature)
		require.Equal(t, txnFromZbox.Value, txnByHash.TransactionValue)
		require.Equal(t, txnFromZbox.OutputHash, txnByHash.TxnOutputHash)
		require.Equal(t, txnFromZbox.CreationDate/int64(1e9), txnByHash.CreationDate)
	})
}
