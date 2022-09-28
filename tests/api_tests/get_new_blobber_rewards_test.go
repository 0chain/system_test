package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewBlobberRewards(t *testing.T) {
	t.Parallel()

	t.Run("Check if a new added blobber as additional parity shard to allocation can receive rewards, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, "", allocationID, client.TxSuccessfulStatus)

		allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)

		apiClient.CreateStakePool(t, wallet, 3, newBlobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadSomeFile(t, allocationID)

		var rewards int64

		wait.PoolImmediately(t, time.Minute*2, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, newBlobberID)

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == wallet.ClientID {
					rewards = poolDelegateInfo.TotalReward
					break
				}
			}

			return rewards > int64(0)
		})
		require.Greater(t, rewards, int64(0))

		apiClient.CollectRewardWrapper(t, wallet, newBlobberID, 3, client.TxSuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.GreaterOrEqual(t, balanceAfter, balanceBefore+rewards)
	})
}

//hash, err := crypto.HashOfFileSHA256(newFile)
//require.Nil(t, err, "hash for new file is not created")

//newBlobberURL := getBlobberURL(newBlobberID, scRestGetAllocation.Blobbers)
//require.NotZero(t, newBlobberURL, "can't get URL of a new blobber")

//sign := crypto.Sha3256(allocation.Tx)

//signBLS, err := client.SignHash(sign, crypto.BLS0Chain, []sys.KeyPair{sys.KeyPair{
//	PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
//	PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
//}})
//require.Nil(t, err)
//blobberFileReferencePathRequest := model.BlobberGetFileReferencePathRequest{
//	URL:             newBlobberURL,
//	ClientID:        registeredWallet.ClientID,
//	ClientKey:       registeredWallet.ClientKey,
//	ClientSignature: signBLS,
//	AllocationID:    allocation.Tx,
//}
//blobberFileReferencePathResponse, restyResponse, err := api_client.v1BlobberGetFileReferencePath(t, blobberFileReferencePathRequest)
//require.Nil(t, err)
//require.NotNil(t, blobberFileReferencePathResponse)
//require.NotNil(t, restyResponse)
//require.Equal(t, api_client.HttpOkStatus, restyResponse.Status())
//
//_, err = blobberFileReferencePathResponse.GetDirTree(allocation.Tx)
//require.Nil(t, err)
//require.Greater(t, len(blobberFileReferencePathResponse.List), 0)
//
//blobberFileReferenceActualHash, ok := blobberFileReferencePathResponse.List[0].Meta["actual_file_hash"]
//require.True(t, ok)
//require.Equal(t, hash, blobberFileReferenceActualHash)
//
//blobberFileReferenceFileName, ok := blobberFileReferencePathResponse.List[0].Meta["name"]
//require.True(t, ok)
//require.Equal(t, fileName, blobberFileReferenceFileName)
//
//blobberFileReferenceActualSize, ok := blobberFileReferencePathResponse.List[0].Meta["actual_file_size"]
//require.True(t, ok)
//require.Equal(t, actualSize, int64(blobberFileReferenceActualSize.(float64)))
//
