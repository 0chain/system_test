package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/stretchr/testify/require"
)

func TestHashnodeRoot(t *testing.T) {
	t.Parallel()

	t.Run("Get hashnode root from blobber should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)

		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 1, 1, time.Minute*20)
		blobberRequirements.Blobbers = availableBlobbers
		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)
		require.NotNil(t, createAllocationTransactionResponse)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		blobberID := getFirstUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobberRequest := &model.BlobberGetHashnodeRequest{
			AllocationID:    allocation.ID,
			URL:             getBlobberURL(blobberID, allocation.Blobbers),
			ClientId:        registeredWallet.ClientID,
			ClientKey:       registeredWallet.ClientKey,
			ClientSignature: keyPair.PrivateKey.Sign(crypto.Sha3256([]byte(allocation.ID))).SerializeToHexStr(),
		}

		getBlobberResponse, restyResponse, err := v1BlobberGetHashNodeRoot(t, *blobberRequest)
		t.Log(getBlobberResponse, restyResponse, err)
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.NotNil(t, getBlobberResponse)
	})
}
