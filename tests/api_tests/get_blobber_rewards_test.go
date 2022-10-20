package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

func TestBlobberRewards(t *testing.T) {
	t.Parallel()

	t.Run("Check if blobber, which already exists in allocation as additional parity shard can receive rewards, should work", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behavior of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID, "Blobber ID contains zero value")

		apiClient.CreateStakePool(t, wallet, 3, blobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadFile(t, allocationID)

		var rewards int64

		wait.PoolImmediately(t, time.Minute*10, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, blobberID)

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == wallet.Id {
					rewards = poolDelegateInfo.Rewards
					break
				}
			}

			return rewards > 0
		})

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		apiClient.CollectRewards(t, wallet, blobberID, 3, client.TxSuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceAfter, balanceBefore+rewards)
	})

	t.Run("Check if the balance of the wallet has been changed without rewards being claimed, shouldn't work", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behavior of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID, "Blobber ID contains zero value")

		apiClient.CreateStakePool(t, wallet, 3, blobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadFile(t, allocationID)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		var rewards int64

		wait.PoolImmediately(t, time.Minute*10, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, blobberID)

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == wallet.Id {
					rewards = poolDelegateInfo.Rewards
					break
				}
			}

			return rewards > 0
		})

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceAfter, balanceBefore)
	})

	t.Run("Check if a new added blobber as additional parity shard to allocation can receive rewards, should work", func(t *testing.T) {
		t.Skip("Skipping due to sporadic behavior of api tests")
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.SetWallet(t, wallet, mnemonic)

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

		apiClient.CreateStakePool(t, wallet, 3, newBlobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadFile(t, allocationID)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		var rewards int64

		wait.PoolImmediately(t, time.Minute*20, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, newBlobberID)

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == wallet.Id {
					rewards = poolDelegateInfo.Rewards
					break
				}
			}

			return rewards > 0
		})

		apiClient.CollectRewards(t, wallet, newBlobberID, 3, client.TxSuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceAfter, balanceBefore+rewards)
	})
}
