package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/wait"
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
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, "", allocationID, client.TxSuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore+1
		})
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore+1)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadSomeFile(t, allocationID)
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

		blobberFileRefRequest := getBlobberFileRefRequest(t, url, wallet, keyPairSecond, allocationID, refType, clientSignature)
		require.NotNil(t, blobberFileRefRequest)
	})
}

func getBlobberFileRefRequest(t *testing.T, url string, registeredWallet *model.Wallet, keyPair *model.KeyPair, allocationId string, refType string, clientSignature string) model.BlobberGetFileRefsRequest {
	t.Logf("get blobber file request object...")
	blobberFileRequest := model.BlobberGetFileRefsRequest{
		URL:             url,
		ClientID:        registeredWallet.ClientID,
		ClientKey:       registeredWallet.ClientKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		RefType:         refType,
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
