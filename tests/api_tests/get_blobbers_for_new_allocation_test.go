package api_tests

import (
	"encoding/json"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetBlobbersForNewAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Alloc blobbers API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()
		registeredWallet, keyPair := registerWallet(t)
		blobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 2147483648, 2, 2, 3600000000000, time.Minute*2)
		println("", blobbers)
		json.Marshal(blobberRequirements)
	})

	t.Run("Alloc blobbers API call should fail gracefully given valid request", func(t *testing.T) {
		t.Parallel()
		t.Skip("FIXME: lack of field validation leads to error")
		blobbers, _, _ := v1ScrestAllocBlobbers(t, "{}")
		println(blobbers)
	})

}

func getBlobbersMatchingRequirements(t *testing.T, wallet *model.Wallet, keyPair model.KeyPair, size int64, dataShards int64, parityShards int64, maxChallengeCompletionTime int64, expiresIn time.Duration) (*[]string, model.BlobberRequirements) {
	blobbers, blobberRequirements, httpResponse, err := getBlobbersMatchingRequirementsWithoutAssertion(t, wallet, keyPair, size, dataShards, parityShards, maxChallengeCompletionTime, expiresIn)

	require.NotNil(t, blobbers, "Allocation was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())

	return blobbers, blobberRequirements
}

func getBlobbersMatchingRequirementsWithoutAssertion(t *testing.T, wallet *model.Wallet, keyPair model.KeyPair, size int64, dataShards int64, parityShards int64, maxChallengeCompletionTime int64, expiresIn time.Duration) (*[]string, model.BlobberRequirements, *resty.Response, error) { //nolint
	blobberRequirements := model.BlobberRequirements{
		Size:                       size,
		DataShards:                 dataShards,
		ParityShards:               parityShards,
		ExpirationDate:             time.Now().Add(expiresIn).Unix(),
		MaxChallengeCompletionTime: 3600000000000,
		ReadPriceRange: model.PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		WritePriceRange: model.PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		OwnerId:        wallet.Id,
		OwnerPublicKey: keyPair.PublicKey.SerializeToHexStr(),
	}

	allocationData, err := json.Marshal(blobberRequirements)
	require.Nil(t, err)

	blobbers, httpResponse, err := v1ScrestAllocBlobbers(t, string(allocationData))
	return blobbers, blobberRequirements, httpResponse, err
}
