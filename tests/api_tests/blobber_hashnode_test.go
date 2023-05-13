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

// this test is working fine in local
// sharder01 was not available for this test so it failed
func TestHashnodeRoot(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get hashnode root from blobber for an empty allocation should work")

	t.RunSequentially("Get hashnode root from blobber for an empty allocation should work", func(t *test.SystemTest) {

		wallet := apiClient.CreateWallet(t)
		for i := 0; i < 2; i++ {
			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)
		}
		//OwnerBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		//t.Logf("owner balance: %v", OwnerBalance)
		//PrintBalance(t, ownerWallet, blobberOwnerWallet, wallet)
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
		for i := 0; i < 2; i++ {
			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)
		}
		OwnerBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("owner balance: %v", OwnerBalance)

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
		t.Logf("The rsponse is %v", restyResponse.String())
		require.NotNil(t, err)
		require.Equal(t, restyResponse.StatusCode(), 500)
		require.Nil(t, getBlobberResponse)
	})

	t.RunSequentially("Get hashnode root with bad signature should fail", func(t *test.SystemTest) {
		wallet := apiClient.CreateWallet(t)
		for i := 0; i < 2; i++ {
			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)
		}
		OwnerBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("owner balance: %v", OwnerBalance)

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
		t.Logf("The rsponse is %v", restyResponse.String())
		require.NotNil(t, err)
		require.NotNil(t, restyResponse) // The rsponse is bad request: invalid signature badsign
		require.Nil(t, getBlobberResponse)
	})

	// TODO: add a case for hasnoderoot of an allocation with a file in it.
}
