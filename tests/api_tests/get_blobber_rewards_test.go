package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBlobberRewards(t *testing.T) {
	t.Parallel()

	t.Run("Check if blobber, which already exists in allocation as additional parity shard can receive rewards, should work", func(t *testing.T) {
		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID, "Blobber ID contains zero value")

		apiClient.CreateStakePool(t, wallet, 3, blobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadSomeFile(t, allocationID)

		var rewards int64

		wait.PoolImmediately(t, time.Minute*10, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, blobberID)

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {

				if poolDelegateInfo.DelegateID == wallet.ClientID {
					rewards = poolDelegateInfo.TotalReward
					break
				}
			}

			return rewards > int64(0)
		})
		require.Greater(t, rewards, int64(0))

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		apiClient.CollectReward(t, wallet, blobberID, 3, client.TxSuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceAfter, balanceBefore+rewards)
	})

	t.Run("Check if a blobber, which is not used for allocation can receive rewards, shouldn't work", func(t *testing.T) {
		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		apiClient.CreateStakePool(t, wallet, 3, newBlobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadSomeFile(t, allocationID)

		var rewards int64

		wait.PoolImmediately(t, time.Minute*10, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, newBlobberID)

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == wallet.ClientID {
					rewards = poolDelegateInfo.TotalReward
					break
				}
			}

			return rewards > int64(0)
		})
		require.Zero(t, rewards)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		apiClient.CollectReward(t, wallet, newBlobberID, 3, client.TxSuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceAfter, balanceBefore+rewards)
	})

	t.Run("Check if a new added blobber as additional parity shard to allocation can receive rewards, should work", func(t *testing.T) {
		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

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
		sdkClient.UploadSomeFile(t, allocationID)

		var rewards int64

		wait.PoolImmediately(t, time.Minute*10, func() bool {
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

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		apiClient.CollectReward(t, wallet, newBlobberID, 3, client.TxSuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceAfter, balanceBefore+rewards)
	})
}
