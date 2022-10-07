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
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus)
		require.GreaterOrEqual(t, blobberFileRefsResponse.TotalPages, int(1))
		require.NotNil(t, blobberFileRefsResponse.OffsetPath)
		require.Greater(t, len(blobberFileRefsResponse.Refs), int(0))
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.AllocationRoot)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.PrevAllocationRoot)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.AllocationId)
		require.Greater(t, blobberFileRefsResponse.LatestWriteMarker.Size, int(0))
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.BlobberId)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Timestamp)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.ClientId)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Signature)

		// request with refType as updated
		refType = "updated"
		blobberFileRefRequest = getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err = apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus)
		require.GreaterOrEqual(t, blobberFileRefsResponse.TotalPages, int(1))
		require.NotNil(t, blobberFileRefsResponse.OffsetPath)
		require.Greater(t, len(blobberFileRefsResponse.Refs), int(0))
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.AllocationRoot)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.PrevAllocationRoot)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.AllocationId)
		require.Greater(t, blobberFileRefsResponse.LatestWriteMarker.Size, int(0))
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.BlobberId)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Timestamp)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.ClientId)
		require.NotNil(t, blobberFileRefsResponse.LatestWriteMarker.Signature)
	})

	t.Run("Get file ref with incorrect allocation id should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

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
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with invalid remote file path should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		remoteFilePath = "/invalid-remote-file-path"
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with invalid refType should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "invalid-ref-type"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with no path should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		remoteFilePath = ""
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "invalid-ref-type"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with no refType should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := ""
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with no path and no refType should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		remoteFilePath = ""
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := ""
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with invalid client signature should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)
		refType := "regular"

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		clientSignature := "invalid-signature"
		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with invalid client id should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		wallet.ClientID = "invalue-client-id"
		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})

	t.Run("Get file ref with invalid client key should fail", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadSomeFile(t, allocationID)
		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "regular"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := crypto.SignHash(sign, keyPair)

		wallet.ClientKey = "invalid-client-key"
		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, allocationID, refType, clientSignature, remoteFilePath)
		blobberFileRefsResponse, resp, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, blobberFileRefsResponse)
		require.Equal(t, resp.StatusCode(), client.HttpNotFoundStatus)
	})
}

func getBlobberFileRefRequest(t *testing.T, url string, registeredWallet *model.Wallet, allocationId string, refType string, clientSignature string, remotePath string) model.BlobberGetFileRefsRequest {
	t.Logf("get blobber file request object...")
	blobberFileRequest := model.BlobberGetFileRefsRequest{
		URL:             url,
		ClientID:        registeredWallet.ClientID,
		ClientKey:       registeredWallet.ClientKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		RefType:         refType,
		RemotePath:      remotePath,
	}
	return blobberFileRequest
}
