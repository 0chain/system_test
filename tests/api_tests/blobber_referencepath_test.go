package api_tests

import (
	"testing"

	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestFileReferencePath(t *testing.T) {
	t.Run("Get file ref with allocation id, remote path should work", func(t *testing.T) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefPathRequest := newBlobberFileRefPathRequest(url, sdkWallet, allocationID, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefPaths(t, blobberFileRefPathRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus, resp)
		require.Equal(t, blobberFileRefsResponse.Ref.Path, remoteFilePath)
		require.Equal(t, blobberFileRefsResponse.Ref.Type, "f")

		// TODO add more assertions once there blobber endpoints are documented
	})

	t.Run("Get file ref with invalid allocation id should fail", func(t *testing.T) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		blobberUrl := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)
		blobberFileRefPathRequest := newBlobberFileRefPathRequest(blobberUrl, sdkWallet, "invalid_allocation_id", clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefPaths(t, blobberFileRefPathRequest, client.HttpOkStatus)
		// FIXME: error should be returned
		require.Nil(t, err)
		require.Empty(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)

		// TODO add more assertions once there blobber endpoints are documented
	})
}

func newBlobberFileRefPathRequest(url string, registeredsdkWallet *model.Wallet, allocationId, clientSignature, remotePath string) *model.BlobberFileRefPathRequest {
	blobberFileRefPathRequest := model.BlobberFileRefPathRequest{
		URL:             url,
		ClientID:        registeredsdkWallet.Id,
		ClientKey:       registeredsdkWallet.PublicKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		Path:            remotePath,
	}
	return &blobberFileRefPathRequest
}
