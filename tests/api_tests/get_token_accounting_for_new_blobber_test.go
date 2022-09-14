package api_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/zboxcore/client"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
	"time"
)

func TestBlobberTokenAccounting(t *testing.T) {
	t.Parallel()

	t.Run("Token accounting of added blobber as additional parity shard to allocation, should work", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)

		err := sdk.InitStorageSDK(registeredWallet.String(), "https://bolt.devnet-0chain.net/dns", "", "bls0chain", nil, int64(registeredWallet.Nonce))
		require.Nil(t, err)

		conf.InitClientConfig(&conf.Config{
			BlockWorker:             "https://bolt.devnet-0chain.net/dns",
			SignatureScheme:         "bls0chain",
			MinSubmit:               50,
			MinConfirmation:         50,
			ConfirmationChainLength: 3,
		})

		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		fmt.Println(getBalance(t, registeredWallet.ClientID).Balance)

		//poolId, _, err := sdk.StakePoolLock("", uint64(tokenomics.IntToZCN(0.1)), 0)
		//require.Nil(t, err)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 4000, 2, 1, time.Minute*20)
		blobberRequirements.Blobbers = availableBlobbers
		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

		fmt.Println(getBalance(t, registeredWallet.ClientID).Balance)

		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(availableBlobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		createStakePoolModel := getCreateStakePool(newBlobberID)
		createStakePoolTransactionResponse, confirmation := createStakePool(t, registeredWallet, keyPair, createStakePoolModel)
		require.NotNil(t, createStakePoolTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		poolId := createStakePoolTransactionResponse.Entity.Hash

		balanceBeforeAllocationUpdate := getBalance(t, registeredWallet.ClientID)
		require.NotNil(t, balanceBeforeAllocationUpdate)

		allocationUpdate := getAllocationUpdate(allocation.ID, newBlobberID, "")
		updateAllocationTransactionResponse, confirmation := updateAllocation(t, registeredWallet, keyPair, allocationUpdate)
		require.NotNil(t, updateAllocationTransactionResponse)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		balanceAfterAllocationUpdate := getBalance(t, registeredWallet.ClientID)
		require.NotNil(t, balanceAfterAllocationUpdate)

		allocation = getAllocation(t, allocation.Tx)
		require.NotNil(t, allocation)

		_, _, err = sdk.WritePoolLock(allocation.ID, uint64(tokenomics.IntToZCN(0.1)), 0)
		require.Nil(t, err)

		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Greater(t, numberOfBlobbersAfter, numberOfBlobbersBefore)

		const fileName = "test"
		const filePath = "/test"
		const actualSize int64 = 1024
		const metaType = "application/octet-stream"

		newFile, err := createFileWithSize(fileName, actualSize)
		require.Nil(t, err, "new file is not created")

		hash, err := crypto.HashOfFileSHA256(newFile)
		require.Nil(t, err, "hash for new file is not created")

		newBlobberURL := getBlobberURL(newBlobberID, allocation.Blobbers)
		require.NotZero(t, newBlobberURL, "can't get URL of a new blobber")

		sign := encryption.Hash(allocation.Tx)

		signBLS, err := client.SignHash(sign, crypto.BLS0Chain, []sys.KeyPair{sys.KeyPair{
			PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
			PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
		}})
		require.Nil(t, err)

		fileMeta := sdk.FileMeta{
			Path:       filepath.Base(filePath),
			ActualSize: actualSize,
			MimeType:   metaType,
			RemoteName: fileName,
			RemotePath: filePath,
		}

		sdkAllocation, err := sdk.GetAllocation(allocation.ID)
		require.Nil(t, err)

		stakePoolInfoBeforeUpload, err := sdk.GetStakePoolInfo(newBlobberID)
		require.Nil(t, err)

		chunkedUpload, err := sdk.CreateChunkedUpload(config.MustGetHomeDir(), sdkAllocation,
			fileMeta, newFile, false, false)
		require.Nil(t, err)
		require.Nil(t, chunkedUpload.Start())

		err = sdkAllocation.CommitMetaTransaction(filePath, "Upload", "", "", nil, new(model.StubStatusBar))
		require.Nil(t, err)

		balanceAfterFileUpload := getBalance(t, registeredWallet.ClientID)
		require.NotNil(t, balanceAfterFileUpload)

		blobberFileReferencePathRequest := model.BlobberGetFileReferencePathRequest{
			URL:             newBlobberURL,
			ClientID:        registeredWallet.ClientID,
			ClientKey:       registeredWallet.ClientKey,
			ClientSignature: signBLS,
			AllocationID:    allocation.Tx,
		}
		blobberFileReferencePathResponse, restyResponse, err := api_client.v1BlobberGetFileReferencePath(t, blobberFileReferencePathRequest)
		require.Nil(t, err)
		require.NotNil(t, blobberFileReferencePathResponse)
		require.NotNil(t, restyResponse)
		require.Equal(t, api_client.HttpOkStatus, restyResponse.Status())

		_, err = blobberFileReferencePathResponse.GetDirTree(allocation.Tx)
		require.Nil(t, err)
		require.Greater(t, len(blobberFileReferencePathResponse.List), 0)

		blobberFileReferenceActualHash, ok := blobberFileReferencePathResponse.List[0].Meta["actual_file_hash"]
		require.True(t, ok)
		require.Equal(t, hash, blobberFileReferenceActualHash)

		blobberFileReferenceFileName, ok := blobberFileReferencePathResponse.List[0].Meta["name"]
		require.True(t, ok)
		require.Equal(t, fileName, blobberFileReferenceFileName)

		blobberFileReferenceActualSize, ok := blobberFileReferencePathResponse.List[0].Meta["actual_file_size"]
		require.True(t, ok)
		require.Equal(t, actualSize, int64(blobberFileReferenceActualSize.(float64)))

		fmt.Println(newBlobberID, poolId)
		time.Sleep(time.Second * 30)

		_, _, err = sdk.CollectRewards(newBlobberID, poolId, sdk.ProviderBlobber)
		require.Nil(t, err)

		stakePoolInfoAfterUpload, err := sdk.GetStakePoolInfo(newBlobberID)
		require.Nil(t, err)

		fmt.Println(stakePoolInfoBeforeUpload.Rewards)
		fmt.Println(stakePoolInfoAfterUpload.Rewards)

		for _, poolDelegateInfo := range stakePoolInfoBeforeUpload.Delegate {
			if string(poolDelegateInfo.DelegateID) == registeredWallet.ClientID {
				fmt.Println(poolDelegateInfo)
				break
			}
		}

		for _, poolDelegateInfo := range stakePoolInfoAfterUpload.Delegate {
			if string(poolDelegateInfo.DelegateID) == registeredWallet.ClientID {
				fmt.Println(poolDelegateInfo)
				break
			}
		}
	})
}

func getCreateStakePool(blobberId string) *model.CreateStakePoolRequest {
	return &model.CreateStakePoolRequest{BlobberID: blobberId}
}

func createStakePool(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair, createStakePool *model.CreateStakePoolRequest) (*model.TransactionResponse, *model.Confirmation) {
	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "stake_pool_lock", InputArgs: createStakePool})
	require.Nil(t, err)

	updateAllocationRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 1000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  string(txnDataString),
		ToClientId:       api_client.StorageSmartContractAddress,
		CreationDate:     time.Now().Unix(),
		ClientId:         wallet.ClientID,
		Version:          "1.0",
		TransactionNonce: wallet.Nonce + 1,
	}

	updateBlobberTransaction := executeTransaction(t, &updateAllocationRequest, keyPair)
	confirmation, restyResponse := confirmTransaction(t, wallet, updateBlobberTransaction.Entity, time.Minute)
	require.NotNil(t, restyResponse)

	return updateBlobberTransaction, confirmation

}
