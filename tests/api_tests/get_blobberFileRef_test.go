package api_tests

import (
	"testing"
	"time"

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
		url := "http://dev.0chain.net/blobber01"
		allocationId := transactionResponse.Entity.Hash
		refType := "regular"
		blobberFileRefRequest := getBlobberFileRefRequest(t, url, registeredWallet, keyPair, allocationId, refType)
		_, _, httpError := v1BlobberGetFileRefs(t, blobberFileRefRequest)
		require.Nil(t, httpError)
	})
}

func getBlobberFileRefRequest(t *testing.T, url string, registeredWallet *model.Wallet, keyPair *model.KeyPair, allocationId string, refType string) model.BlobberGetFileRefsRequest {
	t.Logf("get blobber file request object...")
	blobberFileRequest := model.BlobberGetFileRefsRequest{
		URL:             url,
		ClientID:        registeredWallet.ClientID,
		ClientKey:       registeredWallet.ClientKey,
		ClientSignature: "",
		AllocationID:    allocationId,
		RefType:         refType,
	}
	return blobberFileRequest
}
