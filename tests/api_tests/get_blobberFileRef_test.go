package api_tests

import (
	"testing"

	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestBlobberFileRefs(t *testing.T) {
	t.Parallel()

	t.Run("Get file ref with allocation id, remote path with reftype as regular or updated should work", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
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
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.ClientId, wallet.Id)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Signature)

		// request with refType as updated
		refType = "updated"
		blobberFileRefRequest = getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err = apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
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
		require.Equal(t, blobberFileRefsResponse.LatestWriteMarker.ClientId, wallet.Id)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Signature)
	})

	t.Run("Get file ref with incorrect allocation id should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)

		require.NotZero(t, blobberID)
		remoteFilePath := "/temp" // no remote path as we don't have allocation now
		allocationID = "invalid-allocation-id"
		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with invalid remote file path should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = "/invalid-remote-file-path"
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with invalid refType should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := "invalid-ref-type"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with no path should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = ""
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := "invalid-ref-type"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with no refType should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := ""
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with no path and no refType should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		remoteFilePath = ""
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := ""
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with invalid client signature should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)
		refType := "regular"

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		clientSignature := "invalid-signature"
		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with invalid client id should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		wallet.Id = "invalue-client-id"
		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})

	t.Run("Get file ref with invalid client key should fail", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behaviour of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(t, mnemonic)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature := crypto.SignHexString(t, sign, &keyPair.PrivateKey)

		wallet.Id = "invalid-client-key"
		blobberFileRefRequest := getBlobberFileRefRequest(url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpBadRequestStatus)
	})
}

func getBlobberFileRefRequest(url string, registeredWallet *model.Wallet, allocationId string, refType string, clientSignature string, remotePath string) model.BlobberGetFileRefsRequest {
	blobberFileRequest := model.BlobberGetFileRefsRequest{
		URL:             url,
		ClientID:        registeredWallet.Id,
		ClientKey:       registeredWallet.PublicKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		RefType:         refType,
		RemotePath:      remotePath,
	}
	return blobberFileRequest
}
