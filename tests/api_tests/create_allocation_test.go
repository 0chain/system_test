package api_tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-resty/resty/v2" //nolint

	"github.com/0chain/system_test/internal/api/model"
	"github.com/stretchr/testify/require"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucet(t, registeredWallet, keyPair)
		blobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 2147483648, 2, 2, time.Minute*5)
		blobberRequirements.Blobbers = blobbers
		transactionResponse, _ := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		allocation := getAllocation(t, transactionResponse.Entity.Hash)

		require.NotNil(t, allocation)
	})
}

func createAllocation(t *testing.T, wallet *model.Wallet, keyPair model.KeyPair, blobberRequirements model.BlobberRequirements) (*model.TransactionResponse, *model.Confirmation) {
	t.Logf("Creating allocation...")
	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "new_allocation_request", InputArgs: blobberRequirements})
	require.Nil(t, err)
	allocationRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 1000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  string(txnDataString),
		ToClientId:       STORAGE_SMART_CONTRACT_ADDRESS,
		CreationDate:     time.Now().Unix(),
		ClientId:         wallet.Id,
		Version:          "1.0",
		TransactionNonce: wallet.Nonce + 1,
	}

	allocationTransaction := executeTransaction(t, &allocationRequest, keyPair)
	confirmation, _ := confirmTransaction(t, wallet, allocationTransaction.Entity, 1*time.Minute)

	return allocationTransaction, confirmation
}

func getAllocation(t *testing.T, allocationId string) *model.Allocation {
	allocation, httpResponse, err := getAllocationWithoutAssertion(t, allocationId)

	require.NotNil(t, allocation, "Allocation was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())

	return allocation
}

func getAllocationWithoutAssertion(t *testing.T, allocationId string) (*model.Allocation, *resty.Response, error) { //nolint
	t.Logf("Retrieving allocation...")
	balance, httpResponse, err := v1ScrestAllocation(t, allocationId)
	return balance, httpResponse, err
}
