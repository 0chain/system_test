package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestObjectTree(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get object tree with allocation id, remote path should work")

	t.RunSequentially("Get object tree with allocation id, remote path should work", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, wallet.Mnemonics)
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberObjectTreeRequest := newBlobberObjectTreeRequest(url, wallet, allocationID, clientSignature, remoteFilePath)
		blobberObjectTreeResponse, resp, err := apiClient.V1BlobberObjectTree(t, blobberObjectTreeRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberObjectTreeResponse)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus, resp)
		require.Equal(t, blobberObjectTreeResponse.Ref.Path, remoteFilePath)
		require.Equal(t, blobberObjectTreeResponse.Ref.Type, "f")

		// TODO add more assertions once there blobber endpoints are documented
	})

	t.RunSequentially("Get file ref for empty allocation should work", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		remoteFilePath := "/"

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, wallet.Mnemonics)
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberObjectTreeRequest := newBlobberObjectTreeRequest(url, wallet, allocationID, clientSignature, remoteFilePath)
		blobberObjectTreeResponse, resp, err := apiClient.V1BlobberObjectTree(t, blobberObjectTreeRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberObjectTreeResponse)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus, resp)
		require.NotNil(t, blobberObjectTreeResponse.Ref)

		// TODO add more assertions once there blobber endpoints are documented
	})

	t.RunSequentiallyWithTimeout("Get file ref with invalid allocation id should fail", 90*time.Second, func(t *test.SystemTest) { //TODO: Why is this so slow?  (69s)
		wallet := initialisedWallets[walletIdx]
		walletIdx++

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		blobberUrl := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, wallet.Mnemonics)
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)
		blobberObjectTreeRequest := newBlobberObjectTreeRequest(blobberUrl, wallet, "invalid_allocation_id", clientSignature, remoteFilePath)
		blobberObjectTreeResponse, resp, err := apiClient.V1BlobberObjectTree(t, blobberObjectTreeRequest, client.HttpOkStatus)
		// FIXME: error should be returned
		require.Nil(t, err)
		require.Empty(t, blobberObjectTreeResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with invalid sign should fail", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		blobberUrl := blobber.BaseURL

		blobberObjectTreeRequest := newBlobberObjectTreeRequest(blobberUrl, wallet, allocation.ID, "invalid_signature", remoteFilePath)
		blobberObjectTreeResponse, resp, err := apiClient.V1BlobberObjectTree(t, blobberObjectTreeRequest, client.HttpOkStatus)
		// FIXME: error should be returned
		require.Nil(t, err)
		require.Empty(t, blobberObjectTreeResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with invalid remotepath should fail", func(t *test.SystemTest) {
		wallet := initialisedWallets[walletIdx]
		walletIdx++

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		blobberUrl := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, wallet.Mnemonics)
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)
		blobberObjectTreeRequest := newBlobberObjectTreeRequest(blobberUrl, wallet, allocation.ID, clientSignature, "invalid_path")
		blobberObjectTreeResponse, resp, err := apiClient.V1BlobberObjectTree(t, blobberObjectTreeRequest, client.HttpOkStatus)
		// FIXME: error should be returned
		require.Nil(t, err)
		require.Empty(t, blobberObjectTreeResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})
}

func newBlobberObjectTreeRequest(url string, registeredwallet *model.Wallet, allocationId, clientSignature, remotePath string) *model.BlobberObjectTreeRequest {
	blobberObjectTreeRequest := model.BlobberObjectTreeRequest{
		URL:             url,
		ClientID:        registeredwallet.Id,
		ClientKey:       registeredwallet.PublicKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		Path:            remotePath,
	}
	return &blobberObjectTreeRequest
}
