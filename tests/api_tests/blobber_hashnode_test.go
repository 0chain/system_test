package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestHashnodeRoot(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get hashnode root from blobber for an empty allocation should work")

	t.RunSequentially("Get hashnode root from blobber for an empty allocation should work", func(t *test.SystemTest) {
		wallet := apiClient.CreateWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
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

	t.RunSequentiallyWithTimeout("Get hashnode root for non-existent allocation should fail", 90*time.Second, func(t *test.SystemTest) { //TODO: why is this so slow (67s) ?
		wallet := apiClient.CreateWallet(t)
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
		require.Equal(t, 500, restyResponse.StatusCode())
		require.Contains(t, string(restyResponse.Body()), "too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders")
		require.Nil(t, getBlobberResponse)
	})

	t.RunSequentially("Get hashnode root with bad signature should fail", func(t *test.SystemTest) {
		wallet := apiClient.CreateWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
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
		require.Equal(t, "bad request: invalid signature badsign\n", string(restyResponse.Body()))
		require.Nil(t, getBlobberResponse)
	})

	// TODO: add a case for hasnoderoot of an allocation with a file in it.
}
