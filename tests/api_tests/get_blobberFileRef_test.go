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

func TestBlobberFileRefs(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get file ref with allocation id, remote path with reftype as regular or updated should work")

	t.RunSequentially("Get file ref with allocation id, remote path with reftype as regular or updated should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus, resp)
		require.GreaterOrEqual(t, blobberFileRefsResponse.TotalPages, int(1))
		require.Equal(t, blobberFileRefsResponse.OffsetPath, remoteFilePath)
		require.Greater(t, len(blobberFileRefsResponse.Refs), int(0))
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.AllocationRoot)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.PrevAllocationRoot)
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.AllocationId, allocationID)
		require.Greater(t, blobberFileRefsResponse.LatestWriteMarker.Size, int(0))
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.BlobberId, blobberID)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Timestamp)
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.ClientId, sdkWallet.Id)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Signature)

		// request with refType as updated
		refType = "updated"
		blobberFileRefRequest = getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err = apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus)
		require.GreaterOrEqual(t, blobberFileRefsResponse.TotalPages, int(1))
		require.Equal(t, blobberFileRefsResponse.OffsetPath, remoteFilePath)
		require.Greater(t, len(blobberFileRefsResponse.Refs), int(0))
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.AllocationRoot)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.PrevAllocationRoot)
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.AllocationId, allocationID)
		require.Greater(t, blobberFileRefsResponse.LatestWriteMarker.Size, int(0))
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.BlobberId, blobberID)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Timestamp)
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.ClientId, sdkWallet.Id)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Signature)
	})

	t.RunSequentiallyWithTimeout("Get file ref with incorrect allocation id should fail", 90*time.Second, func(t *test.SystemTest) { // todo - too slow (70s)
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)
		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)

		require.NotZero(t, blobberID)
		remoteFilePath := "/temp" // no remote path as we don't have allocation now
		allocationID = "invalid-allocation-id"
		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with invalid remote file path should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		_, _ = sdkClient.UploadFile(t, allocationID)
		remoteFilePath := "/invalid-remote-file-path"
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with invalid refType should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := "invalid-ref-type"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with no path should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		_, _ = sdkClient.UploadFile(t, allocationID)
		remoteFilePath := ""
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := "invalid-ref-type"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with no refType should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := ""
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with no path and no refType should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		_, _ = sdkClient.UploadFile(t, allocationID)
		remoteFilePath := ""
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := ""
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with invalid client signature should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)
		refType := "regular"

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		clientSignature := "invalid-signature"
		blobberFileRefRequest := getBlobberFileRefRequest(url, sdkWallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with invalid client id should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		wallet := CopyWallet(sdkWallet)
		wallet.Id = "invalue-client-id"
		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.RunSequentially("Get file ref with invalid client key should fail", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath, _ := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/" + remoteFilePath
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, sdkWalletMnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		wallet := CopyWallet(sdkWallet)
		wallet.Id = "invalid-client-key"
		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, &blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})
}

func getBlobberFileRefRequest(url string, registeredsdkWallet *model.Wallet, allocationId, refType, clientSignature, remotePath string) model.BlobberGetFileRefsRequest {
	blobberFileRequest := model.BlobberGetFileRefsRequest{
		URL:             url,
		ClientID:        registeredsdkWallet.Id,
		ClientKey:       registeredsdkWallet.PublicKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		RefType:         refType,
		RemotePath:      remotePath,
	}
	return blobberFileRequest
}

func CopyWallet(wallet *model.Wallet) *model.Wallet {
	newWallet := &model.Wallet{
		Id:           wallet.Id,
		Version:      wallet.Version,
		PublicKey:    wallet.PublicKey,
		CreationDate: wallet.CreationDate,
		Nonce:        wallet.Nonce,
		Keys:         wallet.Keys,
	}
	return newWallet
}
