package api_tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

func TestBlobberRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Check if blobber, which already exists in allocation as additional parity shard can receive rewards, should work")

	t.RunSequentially("Check if blobber, which already exists in allocation as additional parity shard can receive rewards, should work", func(t *test.SystemTest) {

		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID, "Blobber ID contains zero value")

		apiClient.CreateStakePool(t, sdkWallet, 3, blobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadFile(t, allocationID)

		var rewards int64

		wait.PoolImmediately(t, time.Minute*10, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, blobberID, "3")

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == sdkWallet.Id {
					rewards = poolDelegateInfo.Rewards
					break
				}
			}

			return rewards > 0
		})

		walletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		fmt.Println("Balance before: ", balanceBefore)

		walletBalance = apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		balanceBefore = walletBalance.Balance

		fmt.Println("Balance before: ", balanceBefore)

		_, fee := apiClient.CollectRewards(t, sdkWallet, blobberID, 3, client.TxSuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		fmt.Println("rewards: ", rewards)
		fmt.Println("fee: ", fee)
		fmt.Println("balanceBefore: ", balanceBefore)
		fmt.Println("balanceAfter: ", balanceAfter)

		require.Equal(t, balanceAfter, balanceBefore+rewards-fee)
	})

	t.RunSequentially("Check if the balance of the wallet has been changed without rewards being claimed, shouldn't work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID, "Blobber ID contains zero value")

		apiClient.CreateStakePool(t, sdkWallet, 3, blobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadFile(t, allocationID)

		walletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		var rewards int64

		wait.PoolImmediately(t, time.Minute*10, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, blobberID, "3")

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == sdkWallet.Id {
					rewards = poolDelegateInfo.Rewards
					break
				}
			}

			return rewards > 0
		})

		walletBalance = apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceAfter, balanceBefore)
	})

	t.RunSequentiallyWithTimeout("Check if a new added blobber as additional parity shard to allocation can receive rewards, should work", 3*time.Minute, func(t *test.SystemTest) {
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 9.0, client.TxSuccessfulStatus)
		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 9.0, client.TxSuccessfulStatus) // Needs more tokens

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbers(t, sdkWallet, newBlobberID, "", allocationID, client.TxSuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore+1
		})
		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore+1)

		apiClient.CreateStakePool(t, sdkWallet, 3, newBlobberID, client.TxSuccessfulStatus)

		// TODO: replace with native "Upload API" call
		sdkClient.UploadFile(t, allocationID)

		walletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		balanceBefore := walletBalance.Balance

		var rewards int64

		wait.PoolImmediately(t, time.Minute*2, func() bool {
			stakePoolInfo := apiClient.GetStakePoolStat(t, newBlobberID, "3")

			for _, poolDelegateInfo := range stakePoolInfo.Delegate {
				if poolDelegateInfo.DelegateID == sdkWallet.Id {
					rewards = poolDelegateInfo.Rewards
					break
				}
			}

			return rewards > 0
		})

		collecRewardTxn, fee := apiClient.CollectRewards(t, sdkWallet, newBlobberID, 3, client.TxSuccessfulStatus)
		formmattedTxnOutput := strings.ReplaceAll(collecRewardTxn.Transaction.TransactionOutput, `\"`, `"`)

		collectRewardTxnOutput := model.RewardTransactionOutput{}

		err := json.Unmarshal([]byte(formmattedTxnOutput), &collectRewardTxnOutput)
		require.Nil(t, err)

		walletBalance = apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		balanceAfter := walletBalance.Balance

		require.Equal(t, balanceBefore-fee+collectRewardTxnOutput.Amount, balanceAfter)
	})
}
