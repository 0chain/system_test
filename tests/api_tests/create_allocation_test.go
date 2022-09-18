package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
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

		scRestGetAllocationBlobbersResponse, resp, err := apiClient.V1SCRestGetAllocationBlobbers(
			&model.SCRestGetAllocationBlobbersRequest{
				ClientID:  wallet.ClientID,
				ClientKey: wallet.ClientKey,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocationBlobbersResponse)

		createAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.StorageSmartContractAddress,
				TransactionData: model.NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createAllocationTransactionPutResponse)

		createAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createAllocationTransactionGetConfirmationResponse)

		scRestGetAllocation, resp, err := apiClient.V1SCRestGetAllocation(
			model.SCRestGetAllocationRequest{
				AllocationID: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocation)
	})
}

//
//func createAllocation(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair, blobberRequirements model.BlobberRequirements) (*model.TransactionResponse, *model.Confirmation) {
//	t.Logf("Creating allocation...")
//	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "new_allocation_request", InputArgs: blobberRequirements})
//	require.Nil(t, err)
//	allocationRequest := model.Transaction{
//		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
//		TxnOutputHash:    "",
//		TransactionValue: 1000000000,
//		TransactionType:  1000,
//		TransactionFee:   0,
//		TransactionData:  string(txnDataString),
//		ToClientId:       api_client.StorageSmartContractAddress,
//		CreationDate:     time.Now().Unix(),
//		ClientId:         wallet.ClientID,
//		Version:          "1.0",
//		TransactionNonce: wallet.Nonce + 1,
//	}
//
//	allocationTransaction := executeTransaction(t, &allocationRequest, keyPair)
//	confirmation, _ := confirmTransaction(t, wallet, allocationTransaction.Entity, 2*time.Minute)
//
//	return allocationTransaction, confirmation
//}

//
//func getAllocation(t *testing.T, allocationId string) *model.Allocation {
//	allocation, httpResponse, err := getAllocationWithoutAssertion(t, allocationId)
//
//	require.NotNil(t, allocation, "Allocation was unexpectedly nil! with http response [%s]", httpResponse)
//	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
//	require.Equal(t, api_client.HttpOkStatus, httpResponse.Status())
//
//	return allocation
//}
//
//func getAllocationWithoutAssertion(t *testing.T, allocationId string) (*model.Allocation, *resty.Response, error) { //nolint
//	t.Logf("Retrieving allocation...")
//	balance, httpResponse, err := api_client.v1ScrestAllocation(t, allocationId, nil)
//	return balance, httpResponse, err
//}
