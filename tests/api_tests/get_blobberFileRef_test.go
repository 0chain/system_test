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
		refType := "f"
		// func v1BlobberFileUpload(t *testing.T, blobberUploadFileRequest model.BlobberUploadFileRequest)
		// fileUploadStats, _, _ = v1BlobberFileUpload(t)
		// refType = "d"
		getBlobberFileUploadRequest(t, url, registeredWallet, keyPair, allocationId, refType, clientSignature)
		// require.NotNil()

		// ending the function here with panic
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
	// size := 1024
	filename := generateRandomTestFileName(t)
	file, err := createFileWithSize(filename, 1024)
	// blobberUploadFileRequest
	// File
	require.Nil(t, err)
	blobberFileUploadFileRequest := model.BlobberUploadFileRequest{
		URL:             url,
		ClientID:        registeredWallet.ClientID,
		ClientKey:       registeredWallet.ClientKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		File:            file,
	}
	fileUploadResponse, _, _ := v1BlobberFileUpload(t, blobberFileUploadFileRequest)
	require.NotNil(t, fileUploadResponse)
	// t.Logf(" this is upload file response %v", fileUploadResponse)
	// t.Logf(" this is whole file ------------------------------------------------------------------------------------------------ ", file)
	// require.Nil(t, err)
	// blobberFileMeta :=
	// type BlobberUploadFileMeta struct {
	// 	ConnectionID string `json:"connection_id" validation:"required"`
	// 	FileName     string `json:"filename" validation:"required"`
	// 	FilePath     string `json:"filepath" validation:"required"`
	// 	ActualHash   string `json:"actual_hash,omitempty" validation:"required"`
	// 	ContentHash  string `json:"content_hash" validation:"required"`
	// 	MimeType     string `json:"mimetype" validation:"required"`
	// 	ActualSize   int64  `json:"actual_size,omitempty" validation:"required"`
	// 	IsFinal      bool   `json:"is_final" validation:"required"`
	// }

	// blobberUploadMeta := model.BlobberUploadFileMeta{
	// 	ConnectionID: "",
	// 	FileName:     "",
	// 	FilePath:     "",
	// 	ActualHash:   "",
	// 	ContentHash:  "",
	// 	MimeType:     "",
	// 	ActualSize:   0,
	// 	IsFinal:      false,
	// }

	// blobberFileUploadFileRequest := model.BlobberUploadFileRequest{
	// 	URL:             url,
	// 	ClientID:        registeredWallet.ClientID,
	// 	ClientKey:       registeredWallet.ClientKey,
	// 	ClientSignature: clientSignature,
	// 	AllocationID:    allocationId,
	// }
	// BlobberUploadFileRequest
}

// func createFileWithSize(name string, size int64) error {
// 	buffer := make([]byte, size)
// 	rand.Read(buffer) //nolint:gosec,revive
// 	return os.WriteFile(name, buffer, os.ModePerm)
// }

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
