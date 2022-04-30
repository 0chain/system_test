package api_tests

import (
	"strconv"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucet(t, registeredWallet.Id, keyPair)
		transactionResponse := createAllocation(t, registeredWallet.Id, keyPair)
		allocation := getAllocation(t, transactionResponse.Entity.Hash)

		require.Nil(t, allocation)
	})
}

func createAllocation(t *testing.T, clientId string, keyPair model.KeyPair) *model.TransactionResponse {
	allocationRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 1000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  "{\"name\":\"new_allocation_request\",\"input\":{\"data_shards\":2,\"expiration_date\":" + strconv.FormatInt(time.Now().Add(time.Hour*1).Unix(), 10) + ",\"max_challenge_completion_time\":3600000000000,\"owner_id\":\"" + clientId + "\",\"owner_public_key\":\"" + keyPair.PublicKey.SerializeToHexStr() + "\",\"parity_shards\":2,\"read_price_range\":{\"min\":0,\"max\":9223372036854775807},\"size\":2147483648,\"write_price_range\":{\"min\":0,\"max\":9223372036854775807}}}",
		ToClientId:       STORAGE_SMART_CONTRACT_ADDRESS,
		CreationDate:     time.Now().Unix(),
		ClientId:         clientId,
		Version:          "1.0",
	}

	allocationTransaction := executeTransaction(t, &allocationRequest, keyPair)
	confirmTransaction(t, allocationTransaction.Entity.Hash, 1*time.Minute)

	return allocationTransaction
}

func getAllocation(t *testing.T, allocationId string) *model.Allocation {
	allocation, httpResponse, err := getAllocationWithoutAssertion(t, allocationId)

	require.NotNil(t, allocation, "Balance was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())

	return allocation
}

func getAllocationWithoutAssertion(t *testing.T, allocationId string) (*model.Allocation, *resty.Response, error) {
	balance, httpResponse, err := v1ScrestAllocation(t, allocationId)
	return balance, httpResponse, err
}
