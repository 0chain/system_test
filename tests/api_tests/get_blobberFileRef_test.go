package api_tests

import (
	"testing"

	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestBlobberFileRefs(t *testing.T) {
	t.Parallel()

	t.Run("Get file ref with allocation id, remote path and ref type should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		// TODO: replace with native "Upload API" call
		remoteFilePath := sdkClient.UploadFileWithSpecificName(t, allocationID)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		url := blobber.BaseURL
		keyPairSecond, _ := wallet.GetKeyPair()
		keyPair := crypto.GenerateKeys(wallet.Mnemonics)
		refType := "f"
		sign := encryption.Hash(allocation.Tx)

		clientSignature, _ := SignHash(sign, "bls0chain", []sys.KeyPair{sys.KeyPair{
			PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
			PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
		}})

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, keyPairSecond, allocationID, refType, clientSignature, remoteFilePath)
		require.NotNil(t, blobberFileRefRequest)
		blobberFileRefsResponse, httpStatusCode, err := apiClient.V1BlobberGetFileRefs(t, blobberFileRefRequest, client.HttpOkStatus)
		require.NotNil(t, blobberFileRefsResponse)
		require.NotNil(t, httpStatusCode)
		require.Nil(t, err)
	})
}

func getBlobberFileRefRequest(t *testing.T, url string, registeredWallet *model.Wallet, keyPair *model.KeyPair, allocationId string, refType string, clientSignature string, remotePath string) model.BlobberGetFileRefsRequest {
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

func SignHash(hash string, signatureScheme string, keys []sys.KeyPair) (string, error) {
	retSignature := ""
	for _, kv := range keys {
		ss := zcncrypto.NewSignatureScheme(signatureScheme)
		err := ss.SetPrivateKey(kv.PrivateKey)
		if err != nil {
			return "", err
		}

		if len(retSignature) == 0 {
			retSignature, err = ss.Sign(hash)
		} else {
			retSignature, err = ss.Add(retSignature, hash)
		}
		if err != nil {
			return "", err
		}
	}
	return retSignature, nil
}
