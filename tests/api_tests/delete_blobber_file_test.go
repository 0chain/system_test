package api_tests

import (
	client2 "github.com/0chain/gosdk/zboxcore/client"
	"github.com/0chain/gosdk/zboxcore/zboxutil"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeleteBlobberFile(t *testing.T) {
	t.Parallel()

	t.Run("delete function of blobber should delete a single blobber", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		usedBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, usedBlobberID, "Old blobber ID contains zero value")

		sign, err := crypto.SignHash(crypto.Sha3256([]byte(allocation.ID)), "bls0chain", []model.KeyPair{wallet.Keys})
		require.Nil(t, err)

		blobberUrl := getBlobberURL(usedBlobberID, allocation.Blobbers)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		blobberDeleteConnectionRequest := &model.BlobberDeleteConnectionRequest{
			URL:                blobberUrl,
			AllocationID:       allocation.ID,
			ClientKey:          wallet.PublicKey,
			ClientSignature:    sign,
			Path:               "/",
			BlobberID:          blobberId,
			ConnectionID:       zboxutil.NewConnectionId(),
			ClientID:           client2.GetClientID(),
			RequiredStatusCode: client.HttpOkStatus,
		}

		apiClient.DeleteBlobberFile(t, blobberDeleteConnectionRequest)

		// require.NotNil(t, resp)
		// require.Nil(t, err)
	})
}
