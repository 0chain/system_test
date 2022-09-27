package api_tests

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/stretchr/testify/require"
	//nolint
)

func TestGetBlobberFileRefs(t *testing.T) {
	t.Parallel()

	t.Run("Get file ref with allocation id, remote path and ref type should work", func(t *testing.T) {
		t.Parallel()
		registeredWallet, keyPair := registerWallet(t)
		_, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*2)
		transactionResponse, _ := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		// making request at blobber01
		sysKeyPair := []sys.KeyPair{sys.KeyPair{
			PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
			PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
		}}
		// v1BlobberFileUpload
		url := "http://dev.0chain.net/blobber01"
		clientSignature, _ := SignHash(transactionResponse.Entity.Hash, "bls0chain", sysKeyPair)
		allocationId := transactionResponse.Entity.Hash // allocation id
		refType := "f"                                  // or can be "d"
		getBlobberFileUploadRequest(t, url, registeredWallet, keyPair, allocationId, refType, clientSignature)
		blobberFileRefRequest := getBlobberFileRefRequest(t, url, registeredWallet, keyPair, allocationId, refType, clientSignature)
		_, _, httpError := v1BlobberGetFileRefs(t, blobberFileRefRequest)
		require.NotNil(t, httpError)
		require.Nil(t, httpError)
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

func getBlobberFileUploadRequest(t *testing.T, url string, registeredWallet *model.Wallet, keyPair *model.KeyPair, allocationId string, refType string, clientSignature string) {
	t.Logf("get blobber file upload request object...")

	// generating random file
	filename := generateRandomTestFileName(t)
	file, err := createFileWithSize(filename, 1024)
	require.Nil(t, err)

	blobberUploadFileMeta := model.BlobberUploadFileMeta{
		ConnectionID: "connection_id", // need to find the connectionId
		FileName:     filename,
		FilePath:     "/",
		ActualHash:   "actual_hash",  // need actual_hash
		ContentHash:  "content_hash", // need to calculate content_hash
		MimeType:     "text/plain",
		ActualSize:   1024,
		IsFinal:      true,
	}
	blobberFileUploadFileRequest := model.BlobberUploadFileRequest{
		URL:             url,
		ClientID:        registeredWallet.ClientID,
		ClientKey:       registeredWallet.ClientKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		File:            file,
		Meta:            blobberUploadFileMeta,
	}
	fileUploadResponse, _, _ := v1BlobberFileUpload(t, blobberFileUploadFileRequest)
	require.NotNil(t, fileUploadResponse)
}

func generateRandomTestFileName(t *testing.T) string {
	path := strings.TrimSuffix(os.TempDir(), string(os.PathSeparator))

	//FIXME: Filenames longer than 100 characters are rejected see https://github.com/0chain/zboxcli/issues/249
	randomFilename := RandomAlphaNumericString(10)
	return fmt.Sprintf("%s%s%s_test.txt", path, string(os.PathSeparator), randomFilename)
}

func RandomAlphaNumericString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret)
}
