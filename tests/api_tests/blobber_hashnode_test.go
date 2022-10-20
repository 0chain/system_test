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

	t.Run("Get hashnode root from blobber for an empty allocation should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		usedBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, usedBlobberID, "Old blobber ID contains zero value")

		sign, err := crypto.SignHashUsingSignatureScheme(crypto.Sha3256([]byte(allocation.ID)), "bls0chain", []*model.KeyPair{wallet.Keys})
		require.Nil(t, err)

		blobberUrl := getBlobberURL(usedBlobberID, allocation.Blobbers)

		blobberRequest := &model.BlobberGetHashnodeRequest{
			AllocationID:    allocation.ID,
			URL:             blobberUrl,
			ClientId:        wallet.Id,
			ClientKey:       wallet.PublicKey,
			ClientSignature: sign,
		}

		getBlobberResponse, restyResponse, err := apiClient.V1BlobberGetHashNodeRoot(t, blobberRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, restyResponse)
		require.NotNil(t, getBlobberResponse)
		require.Equal(t, getBlobberResponse.AllocationID, allocationID)
		require.Equal(t, getBlobberResponse.Type, "d")
		require.Equal(t, getBlobberResponse.Path, "/")
	})

	t.Run("Get hashnode root for non-existent allocation should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationID := "badallocation"

		blobberUrl := apiClient.HealthyServiceProviders.Blobbers[0]

		sign, err := crypto.SignHashUsingSignatureScheme(crypto.Sha3256([]byte(allocationID)), "bls0chain", []*model.KeyPair{wallet.Keys})
		require.Nil(t, err)

		blobberRequest := &model.BlobberGetHashnodeRequest{
			AllocationID:    allocationID,
			URL:             blobberUrl,
			ClientId:        wallet.Id,
			ClientKey:       wallet.PublicKey,
			ClientSignature: sign,
		}

		getBlobberResponse, restyResponse, err := apiClient.V1BlobberGetHashNodeRoot(t, blobberRequest, client.HttpOkStatus)
		require.NotNil(t, err)
		require.Nil(t, restyResponse)
		require.Nil(t, getBlobberResponse)
	})

	t.Run("Get hashnode root with bad signature should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		usedBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, usedBlobberID, "Old blobber ID contains zero value")

		sign := "badsign"

		blobberUrl := getBlobberURL(usedBlobberID, allocation.Blobbers)

		blobberRequest := &model.BlobberGetHashnodeRequest{
			AllocationID:    allocation.ID,
			URL:             blobberUrl,
			ClientId:        wallet.Id,
			ClientKey:       wallet.PublicKey,
			ClientSignature: sign,
		}

		getBlobberResponse, restyResponse, err := apiClient.V1BlobberGetHashNodeRoot(t, blobberRequest, client.HttpOkStatus)
		require.NotNil(t, err)
		require.Nil(t, restyResponse)
		require.Nil(t, getBlobberResponse)
	})

	// TODO: add a case for hasnoderoot of an allocation with a file in it.
}
