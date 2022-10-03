package api_tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/stretchr/testify/require"
)

func TestUpdateBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Update blobber in allocation without correct delegated client, shouldn't work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)

		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		blobberRequirements.Blobbers = availableBlobbers
		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status)
		require.NotNil(t, createAllocationTransactionResponse)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		blobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		getBlobberResponse, restyResponse, err := v1SCRestGetBlobber(t, blobberID, endpoint.ConsensusByHttpStatus(endpoint.HttpOkStatus))
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.NotNil(t, getBlobberResponse)
		require.NotEqual(t, registeredWallet.ClientID, getBlobberResponse.StakePoolSettings.DelegateWallet)

		updateBlobberTransactionResponse, confirmation := updateBlobber(t, registeredWallet, keyPair, getBlobberResponse)
		require.Equal(t, endpoint.TxUnsuccessfulStatus, confirmation.Status)
		require.NotNil(t, updateBlobberTransactionResponse)
	})
}

func updateBlobber(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair, blobberUpdate *model.GetBlobberResponse) (*model.TransactionResponse, *model.Confirmation) {
	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "update_blobber_settings", InputArgs: blobberUpdate})
	require.Nil(t, err)

	updateAllocationRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 1000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  string(txnDataString),
		ToClientId:       endpoint.StorageSmartContractAddress,
		CreationDate:     time.Now().Unix(),
		ClientId:         wallet.ClientID,
		Version:          "1.0",
		TransactionNonce: wallet.Nonce + 1,
	}

	updateBlobberTransaction := executeTransaction(t, &updateAllocationRequest, keyPair)
	confirmation, restyResponse := confirmTransaction(t, wallet, updateBlobberTransaction.Entity, time.Minute)
	require.NotNil(t, restyResponse)

	return updateBlobberTransaction, confirmation
}
