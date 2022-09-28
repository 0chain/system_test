package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestHashnodeRoot(t *testing.T) {
	t.Parallel()

	t.Run("Get hashnode root from blobber should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		sign, err := crypto.SignHash(crypto.Sha3256([]byte(allocation.ID)), "bls0chain", []model.KeyPair{*keyPair})
		require.Nil(t, err)

		blobberRequest := &model.BlobberGetHashnodeRequest{
			AllocationID:    allocation.ID,
			URL:             getBlobberURL(newBlobberID, allocation.Blobbers),
			ClientId:        wallet.ClientID,
			ClientKey:       wallet.ClientKey,
			ClientSignature: sign,
		}

		getBlobberResponse, restyResponse, err := apiClient.V1BlobberGetHashNodeRoot(t, *blobberRequest)
		t.Log(getBlobberResponse, restyResponse, err)
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.NotNil(t, getBlobberResponse)
	})
}
