package api_tests

import (
	"fmt"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBlobberTokenAccounting(t *testing.T) {
	t.Parallel()

	t.Run("Token accounting of added blobber as additional parity shard to allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWalletWrapper(t)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucetWrapper(t, wallet)

		walletBalance := apiClient.GetWalletBalanceWrapper(t, wallet)
		balanceBefore := walletBalance.Balance

		allocationBlobbers := apiClient.GetAllocationBlobbersWrapper(t, wallet)
		//stakePoolID, _, err := sdk.StakePoolLock(sdk.ProviderBlobber, (*allocationBlobbers.Blobbers)[0], uint64(*tokenomics.IntToZCN(0.1)), 1000)
		//require.Nil(t, err)

		allocationID := apiClient.CreateAllocationWrapper(t, wallet, allocationBlobbers)

		allocation := apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbersWrapper(t, wallet, newBlobberID, "", allocationID)

		allocation = apiClient.GetAllocationWrapper(t, allocationID)
		numberOfBlobbersAfter := len(allocation.Blobbers)
		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)

		//{"async":true,"entity":{"hash":"7d2e88bf421df2c2117d06ebfc2c1a987a924129be3c2e03484bbc2ea5ed4099","version":"1.0","client_id":"d800aa309d0791f35d449ba44e3dd499a10d1b4a026a3046e4480075a4b31aba","public_key":"c1f161b4f75dd86a04a9406d366f9a6169dde9ef95b7935e999af54a13d3de07dc0371e5190ebfdce1c907691d19b3109d4e9619224c0d54dbfc0e700b5d6119","to_client_id":"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7","chain_id":"0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe","transaction_data":"{\"name\":\"stake_pool_lock\",\"input\":{\"provider_type\":3,\"provider_id\":\"302b49678daf652e0e9e409863d6072b0333816d6cf289bad27ae6f2157a60fc\"}}","transaction_value":5000000000,"signature":"88d7e47353709f97464ac27050bedb3abbeab7456d535122112ab3c6b1ecd485","creation_date":1664093977,"transaction_fee":1000,"transaction_nonce":4,"transaction_type":1000,"txn_output_hash":"","transaction_status":0}}
		//_, _, err := sdk.StakePoolLock(sdk.ProviderBlobber, newBlobberID, uint64(*tokenomics.IntToZCN(0.5)), 1000)
		//require.Nil(t, err)
		//{"async":true,"entity":{"hash":"e773ac1e0bc73bd041018b86f2253830763ac23083513dbbba3d896183a3fd59","version":"1.0","client_id":"813d636b2cb7d69ca925add4456c90021c3565802c59d9bf1957cdbc5f18ffcf","public_key":"d951a73c3bc72110d8a5c8739e03f62849202f41937142e1bca1d6e79a41c817bd4ccc5de03d2fa0a06d11b17a7687b8f0dff30dbe31d251630ab2ca06e5ca92","to_client_id":"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7","chain_id":"0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe","transaction_data":"{\"name\":\"stake_pool_lock\",\"input\":{\"provider_type\":3,\"provider_id\":\"302b49678daf652e0e9e409863d6072b0333816d6cf289bad27ae6f2157a60fc\"}}","transaction_value":5000000000,"signature":"5b12a6dd7d6af2d24fb90c1674e0323691e6778ad47e8271f6ae22167b18db9b","creation_date":1664094106,"transaction_fee":1000,"transaction_nonce":3,"transaction_type":1000,"txn_output_hash":"","transaction_status":0}}
		stakePoolID := apiClient.CreateStakePoolWrapper(t, wallet, sdk.ProviderBlobber, newBlobberID)
		fmt.Println(stakePoolID)

		sdkClient.UploadSomeFile(t, allocationID)
		wait.PoolImmediately(time.Minute*2, func() bool { return false })

		stakePoolInfo, err := sdk.GetStakePoolInfo(newBlobberID)
		require.Nil(t, err)

		//stakePoolInfo := apiClient.GetStakePoolStatWrapper(t, newBlobberID)

		var rewards int64
		for _, poolDelegateInfo := range stakePoolInfo.Delegate {
			if string(poolDelegateInfo.DelegateID) == wallet.ClientID {
				rewards = int64(poolDelegateInfo.TotalReward)
				break
			}
		}

		fmt.Println(rewards)

		require.Greater(t, rewards, int64(0))

		//apiClient.CollectRewardWrapper(t, wallet, newBlobberID, stakePoolID, 3)
		_, _, err = sdk.CollectRewards(newBlobberID, 3)
		require.Nil(t, err)

		walletBalance = apiClient.GetWalletBalanceWrapper(t, wallet)
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
