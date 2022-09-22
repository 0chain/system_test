package api_tests

import (
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/interaction"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
	"time"
)

func TestBlobberTokenAccounting(t *testing.T) {
	//t.Skip()
	t.Parallel()

	t.Run("Token accounting of added blobber as additional parity shard to allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet, resp, err := apiClient.V1ClientPut(model.ClientPutRequest{}, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, wallet)

		sdkClient.SetWallet(wallet)

		faucetTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData(),
				Value:           tokenomics.IntToZCN(3)},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, faucetTransactionPutResponse)

		faucetTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: faucetTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, faucetTransactionGetConfirmationResponse)

		clientGetBalanceResponse, resp, err := apiClient.V1ClientGetBalance(
			model.ClientGetBalanceRequest{
				ClientID: wallet.ClientID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, clientGetBalanceResponse)

		balanceBefore := clientGetBalanceResponse.Balance

		scRestGetAllocationBlobbersResponse, resp, err := apiClient.V1SCRestGetAllocationBlobbers(
			&model.SCRestGetAllocationBlobbersRequest{
				ClientID:  wallet.ClientID,
				ClientKey: wallet.ClientKey,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocationBlobbersResponse)

		createAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      client.StorageSmartContractAddress,
				TransactionData: model.NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse),
				Value:           tokenomics.IntToZCN(0.1),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createAllocationTransactionPutResponse)

		createAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createAllocationTransactionGetConfirmationResponse)

		scRestGetAllocation, resp, err := apiClient.V1SCRestGetAllocation(
			model.SCRestGetAllocationRequest{
				AllocationID: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocation)

		numberOfBlobbersBefore := len(scRestGetAllocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(scRestGetAllocationBlobbersResponse.Blobbers, scRestGetAllocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		createStakePoolTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewCreateStackPoolTransactionData(
					model.CreateStakePoolRequest{
						BlobberID: newBlobberID,
					}),
				Value: tokenomics.IntToZCN(0.5)},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createStakePoolTransactionPutResponse)

		createStakePoolTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: createStakePoolTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createStakePoolTransactionGetConfirmationResponse)

		poolId := createStakePoolTransactionPutResponse.Entity.Hash

		//balanceBeforeAllocationUpdate := getBalance(t, registeredWallet.ClientID)
		//require.NotNil(t, balanceBeforeAllocationUpdate)

		updateAllocationTransactionPutResponse, resp, err := apiClient.V1TransactionPut(
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: client.StorageSmartContractAddress,
				TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
					ID:              scRestGetAllocation.ID,
					AddBlobberId:    newBlobberID,
					RemoveBlobberId: "",
				}),
				Value: tokenomics.IntToZCN(0.1),
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionPutResponse)

		updateAllocationTransactionGetConfirmationResponse, resp, err := apiClient.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus,
			client.TxSuccessfulStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, updateAllocationTransactionGetConfirmationResponse)

		scRestGetAllocation, resp, err = apiClient.V1SCRestGetAllocation(
			model.SCRestGetAllocationRequest{
				AllocationID: createAllocationTransactionPutResponse.Entity.Hash,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocation)

		numberOfBlobbersAfter := len(scRestGetAllocation.Blobbers)

		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)

		const fileName = "test"
		const filePath = "/test"
		const actualSize int64 = 1024
		const metaType = "application/octet-stream"

		newFile, err := interaction.CreateFile(fileName, actualSize)
		require.Nil(t, err, "new file is not created")

		//hash, err := crypto.HashOfFileSHA256(newFile)
		//require.Nil(t, err, "hash for new file is not created")

		//newBlobberURL := getBlobberURL(newBlobberID, scRestGetAllocation.Blobbers)
		//require.NotZero(t, newBlobberURL, "can't get URL of a new blobber")

		//sign := encryption.Hash(allocation.Tx)

		//signBLS, err := client.SignHash(sign, crypto.BLS0Chain, []sys.KeyPair{sys.KeyPair{
		//	PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
		//	PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
		//}})
		//require.Nil(t, err)

		fileMeta := sdk.FileMeta{
			Path:       filepath.Base(filePath),
			ActualSize: actualSize,
			MimeType:   metaType,
			RemoteName: fileName,
			RemotePath: filePath,
		}

		//TODO: replace sdk part to nature API calls

		sdkAllocation, err := sdk.GetAllocation(createAllocationTransactionPutResponse.Entity.Hash)
		require.Nil(t, err)

		chunkedUpload, err := sdk.CreateChunkedUpload(config.MustGetHomeDir(), sdkAllocation,
			fileMeta, newFile, false, false)
		require.Nil(t, err)
		require.Nil(t, chunkedUpload.Start())

		err = sdkAllocation.CommitMetaTransaction(filePath, "Upload", "", "", nil, new(model.StubStatusBar))
		require.Nil(t, err)

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
		//fmt.Println(newBlobberID, poolId)
		time.Sleep(time.Second * 30)

		stakePoolInfo, err := sdk.GetStakePoolInfo(newBlobberID)
		require.Nil(t, err)

		var rewards int64
		for _, poolDelegateInfo := range stakePoolInfo.Delegate {
			if string(poolDelegateInfo.DelegateID) == wallet.ClientID {
				rewards = int64(poolDelegateInfo.TotalReward)
				break
			}
		}

		require.Greater(t, rewards, int64(0))

		_, _, err = sdk.CollectRewards(newBlobberID, poolId, sdk.ProviderBlobber)
		require.Nil(t, err)

		clientGetBalanceResponse, resp, err = apiClient.V1ClientGetBalance(
			model.ClientGetBalanceRequest{
				ClientID: wallet.ClientID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, clientGetBalanceResponse)

		balanceAfter := clientGetBalanceResponse.Balance
		require.GreaterOrEqual(t, balanceAfter, balanceBefore+rewards)
	})
}
