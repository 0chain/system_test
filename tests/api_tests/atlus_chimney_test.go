package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/google/go-cmp/cmp"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

func TestAtlusChimney(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Get total minted tokens, should work", func(t *test.SystemTest) {
		getTotalMintedResponse, resp, err := apiClient.V1SharderGetTotalMinted(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getTotalMintedResponse)
	})

	t.RunWithTimeout("Check if amount of total minted tokens changed after a blobber receives rewards, should work", time.Minute*10, func(t *test.SystemTest) {
		t.Skip()
		sdkClient.StartSession(func() {
			apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

			blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
			require.NotZero(t, blobberID, "Blobber ID contains zero value")

			getTotalMintedResponse, resp, err := apiClient.V1SharderGetTotalMinted(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, getTotalMintedResponse, 0)

			totalMintedBefore := getTotalMintedResponse

			apiClient.CreateStakePool(t, sdkWallet, 3, blobberID, client.TxSuccessfulStatus)

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
			apiClient.CollectRewards(t, sdkWallet, blobberID, 3, client.TxSuccessfulStatus)

			walletBalance = apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
			balanceAfter := walletBalance.Balance

			require.Equal(t, balanceAfter, balanceBefore+rewards)

			wait.PoolImmediately(t, time.Minute*2, func() bool {
				getTotalMintedResponse, resp, err = apiClient.V1SharderGetTotalMinted(t, client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, getTotalMintedResponse, 0)

				return *getTotalMintedResponse > *totalMintedBefore
			})
		})
	})

	t.RunWithTimeout("Check if amount of total minted tokens changed after minting zcn tokens, should work", time.Minute*10, func(t *test.SystemTest) {
		t.Skip("Skip till authorizers are deployed correctly")
		sdkClient.StartSession(func() {
			getTotalMintedResponse, resp, err := apiClient.V1SharderGetTotalMinted(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, getTotalMintedResponse, 0)

			totalMintedBefore := getTotalMintedResponse

			burnTicketHash := sdkClient.BurnWZCN(t, 1)
			sdkClient.MintZCN(t, burnTicketHash)

			wait.PoolImmediately(t, time.Minute*5, func() bool {
				getTotalMintedResponse, resp, err = apiClient.V1SharderGetTotalMinted(t, client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, getTotalMintedResponse, 0)

				return *getTotalMintedResponse > *totalMintedBefore
			})
		})
	})

	t.Run("Get total total challenges, should work", func(t *test.SystemTest) {
		t.Skip()
		getTotalTotalChallengesResponse, resp, err := apiClient.V1SharderGetTotalTotalChallenges(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalTotalChallengesResponse.TotalTotalChallenges, 0)
	})

	t.Run("Check if amount of total total challenges changed after file uploading, should work", func(t *test.SystemTest) {
		t.Skip()
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		getTotalTotalChallengesResponse, resp, err := apiClient.V1SharderGetTotalTotalChallenges(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalTotalChallengesResponse.TotalTotalChallenges, 0)

		totalTotalChallengesBefore := getTotalTotalChallengesResponse.TotalTotalChallenges

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)
			sdkClient.UploadFile(t, allocationID)
		})

		var totalTotalChallengesAfter int

		wait.PoolImmediately(t, time.Minute*2, func() bool {
			getTotalTotalChallengesResponse, resp, err = apiClient.V1SharderGetTotalTotalChallenges(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)

			totalTotalChallengesAfter = getTotalTotalChallengesResponse.TotalTotalChallenges

			return totalTotalChallengesAfter > totalTotalChallengesBefore
		})

		require.Greater(t, totalTotalChallengesAfter, totalTotalChallengesBefore)
	})

	t.Run("Get total successful challenges, should work", func(t *test.SystemTest) {
		t.Skip()
		getTotalSuccessfulChallengesResponse, resp, err := apiClient.V1SharderGetTotalSuccessfulChallenges(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges, 0)
	})

	t.Run("Check if amount of total successful challenges changed after file uploading, should work", func(t *test.SystemTest) {
		t.Skip()
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		getTotalSuccessfulChallengesResponse, resp, err := apiClient.V1SharderGetTotalSuccessfulChallenges(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges, 0)

		totalSuccessfulChallengesBefore := getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)
			sdkClient.UploadFile(t, allocationID)
		})

		var totalSuccessfulChallengesAfter int

		wait.PoolImmediately(t, time.Minute*2, func() bool {
			getTotalSuccessfulChallengesResponse, resp, err = apiClient.V1SharderGetTotalSuccessfulChallenges(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)

			totalSuccessfulChallengesAfter = getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges

			return totalSuccessfulChallengesAfter > totalSuccessfulChallengesBefore
		})

		require.Greater(t, totalSuccessfulChallengesAfter, totalSuccessfulChallengesBefore)
	})

	t.Run("Get total allocated storage, should work", func(t *test.SystemTest) {
		t.Skip()
		getTotalAllocatedStorageResponse, resp, err := apiClient.V1SharderGetTotalAllocatedStorage(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalAllocatedStorageResponse.TotalAllocatedStorage, 0)
	})

	t.Run("Check if amount of total allocated storage changed after file uploading, should work", func(t *test.SystemTest) {
		t.Skip()
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		getTotalAllocatedStorageResponse, resp, err := apiClient.V1SharderGetTotalAllocatedStorage(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalAllocatedStorageResponse.TotalAllocatedStorage, 0)

		totalAllocatedStorageBefore := getTotalAllocatedStorageResponse.TotalAllocatedStorage

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)
			sdkClient.UploadFile(t, allocationID)
		})

		var totalAllocatedStorageAfter int

		wait.PoolImmediately(t, time.Minute*2, func() bool {
			getTotalAllocatedStorageResponse, resp, err = apiClient.V1SharderGetTotalAllocatedStorage(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)

			totalAllocatedStorageAfter = getTotalAllocatedStorageResponse.TotalAllocatedStorage

			return totalAllocatedStorageAfter > totalAllocatedStorageBefore
		})

		require.Greater(t, totalAllocatedStorageAfter, totalAllocatedStorageBefore)
	})

	t.Run("Get total staked, should work", func(t *test.SystemTest) {
		t.Skip()
		getTotalStakedResponse, resp, err := apiClient.V1SharderGetTotalStaked(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalStakedResponse.TotalStaked, 0)
	})

	t.Run("Check if amount of total staked changed after creating new allocation, should work", func(t *test.SystemTest) {
		t.Skip()
		getTotalStakedResponse, resp, err := apiClient.V1SharderGetTotalStaked(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalStakedResponse.TotalStaked, 0)
	})

	t.Run("Get total stored data, should work", func(t *test.SystemTest) {
		t.Skip()
		getTotalStoredDataResponse, resp, err := apiClient.V1SharderGetTotalStoredData(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalStoredDataResponse.TotalStoredData, 0)
	})

	t.Run("Check if total stored data changed after file uploading, should work", func(t *test.SystemTest) {
		t.Skip()
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		getTotalStoredDataResponse, resp, err := apiClient.V1SharderGetTotalStoredData(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalStoredDataResponse.TotalStoredData, 0)

		totalStoredDataBefore := getTotalStoredDataResponse.TotalStoredData

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)
			sdkClient.UploadFile(t, allocationID)
		})

		var totalStoredDataAfter int

		wait.PoolImmediately(t, time.Minute*2, func() bool {
			getTotalStoredDataResponse, resp, err = apiClient.V1SharderGetTotalStoredData(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)

			totalStoredDataAfter = getTotalStoredDataResponse.TotalStoredData

			return totalStoredDataAfter > totalStoredDataBefore
		})

		require.Greater(t, totalStoredDataAfter, totalStoredDataBefore)
	})

	t.Run("Get average write price, should work", func(t *test.SystemTest) {
		t.Skip()
		getAverageWritePriceResponse, resp, err := apiClient.V1SharderGetAverageWritePrice(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, getAverageWritePriceResponse.AverageWritePrice)
	})

	t.Run("Get total blobber capacity, should work", func(t *test.SystemTest) {
		t.Skip()
		getTotalBlobberCapacityResponse, resp, err := apiClient.V1SharderGetTotalBlobberCapacity(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalBlobberCapacityResponse.TotalBlobberCapacity, 0)
	})

	t.Run("Check if total blobber capacity changed after file uploading, should work", func(t *test.SystemTest) {
		t.Skip()
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		getTotalBlobberCapacityResponse, resp, err := apiClient.V1SharderGetTotalBlobberCapacity(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalBlobberCapacityResponse.TotalBlobberCapacity, 0)

		totalBlobberCapacityBefore := getTotalBlobberCapacityResponse.TotalBlobberCapacity

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)
			sdkClient.UploadFile(t, allocationID)
		})

		var totalBlobberCapacityAfter int

		wait.PoolImmediately(t, time.Minute*2, func() bool {
			getTotalBlobberCapacityResponse, resp, err = apiClient.V1SharderGetTotalBlobberCapacity(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)

			totalBlobberCapacityAfter = getTotalBlobberCapacityResponse.TotalBlobberCapacity

			return totalBlobberCapacityAfter < totalBlobberCapacityBefore
		})

		require.Less(t, totalBlobberCapacityAfter, totalBlobberCapacityBefore)
	})

	t.Run("Get graph of blobber service charge of certain blobber, should work", func(t *test.SystemTest) {
		t.Skip()
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberServiceChargeResponse, resp, err := apiClient.V1SharderGetGraphBlobberServiceCharge(
			t,
			model.GetGraphBlobberServiceChargeRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberServiceChargeResponse)
	})

	t.Run("Check if graph of blobber service charge changed after file uploading, should work", func(t *test.SystemTest) {
		t.Skip()
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberServiceChargeResponse, resp, err := apiClient.V1SharderGetGraphBlobberServiceCharge(
			t,
			model.GetGraphBlobberServiceChargeRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberServiceChargeResponse)
	})

	t.Run("Get graph of blobber inactive rounds, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		var getGraphBlobberInactiveRoundsResponse *model.GetGraphBlobberInactiveRoundsResponse
		getGraphBlobberInactiveRoundsResponse, resp, err = apiClient.V1SharderGetGraphBlobberInactiveRounds(
			t,
			model.GetGraphBlobberInactiveRoundsRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberInactiveRoundsResponse)
	})

	t.Run("Get graph of completed blobber challenges, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		var getGraphBlobberChallengesCompletedResponse *model.GetGraphBlobberChallengesCompletedResponse
		getGraphBlobberChallengesCompletedResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesCompleted(
			t,
			model.GetGraphBlobberChallengesCompletedRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesCompletedResponse)
	})

	t.RunWithTimeout("Check if graph of completed blobber challenges changed after file uploading, should work", time.Minute*10, func(t *test.SystemTest) {
		sdkClient.StartSession(func() {
			apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)
			sdkClient.UploadFile(t, allocationID)

			getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, getCurrentRoundResponse)

			currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

			var getGraphBlobberChallengesCompletedResponse *model.GetGraphBlobberChallengesCompletedResponse
			getGraphBlobberChallengesCompletedResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesCompleted(
				t,
				model.GetGraphBlobberChallengesCompletedRequest{
					DataPoints: 17,
					BlobberID:  blobberID,
					To:         currentRoundString,
				},
				client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotZero(t, *getGraphBlobberChallengesCompletedResponse)

			getGraphBlobberChallengesCompletedResponseBefore := *getGraphBlobberChallengesCompletedResponse

			wait.PoolImmediately(t, time.Minute*5, func() bool {
				getGraphBlobberChallengesCompletedResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesCompleted(
					t,
					model.GetGraphBlobberChallengesCompletedRequest{
						DataPoints: 17,
						BlobberID:  blobberID,
						To:         currentRoundString,
					},
					client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)
				require.NotZero(t, *getGraphBlobberChallengesCompletedResponse)

				return !cmp.Equal(*getGraphBlobberChallengesCompletedResponse, getGraphBlobberChallengesCompletedResponseBefore)
			})
		})
	})

	t.Run("Get graph of passed blobber challenges, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		getGraphBlobberChallengesPassed, resp, err := apiClient.V1SharderGetGraphBlobberChallengesPassed(
			t,
			model.GetGraphBlobberChallengesPassedRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesPassed)
	})

	t.RunWithTimeout("Check if graph of passed blobber challenges changed after file uploading, should work", time.Minute*10, func(t *test.SystemTest) {
		sdkClient.StartSession(func() {
			apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)
			sdkClient.UploadFile(t, allocationID)

			getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, getCurrentRoundResponse)

			currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

			var getGraphBlobberChallengesPassedResponse *model.GetGraphBlobberChallengesPassedResponse
			getGraphBlobberChallengesPassedResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesPassed(
				t,
				model.GetGraphBlobberChallengesPassedRequest{
					DataPoints: 17,
					BlobberID:  blobberID,
					To:         currentRoundString,
				},
				client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotZero(t, *getGraphBlobberChallengesPassedResponse)

			getGraphBlobberChallengesPassedResponseBefore := *getGraphBlobberChallengesPassedResponse

			wait.PoolImmediately(t, time.Minute*5, func() bool {
				getGraphBlobberChallengesPassedResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesPassed(
					t,
					model.GetGraphBlobberChallengesPassedRequest{
						DataPoints: 17,
						BlobberID:  blobberID,
						To:         currentRoundString,
					},
					client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)
				require.NotZero(t, *getGraphBlobberChallengesPassedResponse)

				return !cmp.Equal(*getGraphBlobberChallengesPassedResponse, getGraphBlobberChallengesPassedResponseBefore)
			})
		})
	})

	t.Run("Get graph of opened challenges of a certain blobber, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		var getGraphBlobberChallengesOpenResponse *model.GetGraphBlobberChallengesOpenedResponse
		getGraphBlobberChallengesOpenResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesOpened(
			t,
			model.GetGraphBlobberChallengesOpenedRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesOpenResponse)
	})

	t.RunWithTimeout("Check if graph of opened blobber challenges changed after file uploading, should work", time.Minute*10, func(t *test.SystemTest) {
		sdkClient.StartSession(func() {
			apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)
			sdkClient.UploadFile(t, allocationID)

			getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, getCurrentRoundResponse)

			currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

			var getGraphBlobberChallengesOpenedResponse *model.GetGraphBlobberChallengesOpenedResponse
			getGraphBlobberChallengesOpenedResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesOpened(
				t,
				model.GetGraphBlobberChallengesOpenedRequest{
					DataPoints: 17,
					BlobberID:  blobberID,
					To:         currentRoundString,
				},
				client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotZero(t, *getGraphBlobberChallengesOpenedResponse)

			getGraphBlobberChallengesOpenedResponseBefore := *getGraphBlobberChallengesOpenedResponse

			wait.PoolImmediately(t, time.Minute*5, func() bool {
				getGraphBlobberChallengesOpenedResponse, resp, err = apiClient.V1SharderGetGraphBlobberChallengesOpened(
					t,
					model.GetGraphBlobberChallengesOpenedRequest{
						DataPoints: 17,
						BlobberID:  blobberID,
						To:         currentRoundString,
					},
					client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)
				require.NotZero(t, *getGraphBlobberChallengesOpenedResponse)

				return !cmp.Equal(*getGraphBlobberChallengesOpenedResponse, getGraphBlobberChallengesOpenedResponseBefore)
			})
		})
	})

	t.Run("Get graph of write prices of a certain blobber, should work", func(t *test.SystemTest) {
		t.Skip()
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberWritePriceResponse, resp, err := apiClient.V1SharderGetGraphBlobberWritePrice(
			t,
			model.GetGraphBlobberWritePriceRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberWritePriceResponse)
	})

	t.Run("Get graph of capacity of a certain blobber, should work", func(t *test.SystemTest) {
		t.Skip()
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberCapacityResponse, resp, err := apiClient.V1SharderGetGraphBlobberCapacity(
			t,
			model.GetGraphBlobberCapacityRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberCapacityResponse)
	})

	t.Run("Get graph of allocated storage of a certain blobber, should work", func(t *test.SystemTest) {
		t.Skip()
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberAllocatedResponse, resp, err := apiClient.V1SharderGetGraphBlobberAllocated(
			t,
			model.GetGraphBlobberAllocatedRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberAllocatedResponse)
	})

	t.Run("Get graph of all saved data of a certain blobber, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		getGraphBlobberSavedDataResponse, resp, err := apiClient.V1SharderGetGraphBlobberSavedData(
			t,
			model.GetGraphBlobberSavedDataRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, *getGraphBlobberSavedDataResponse)
	})

	t.RunWithTimeout("Check if a graph of saved data of a certain blobber changes after file upload, should work", time.Minute*10, func(t *test.SystemTest) {
		sdkClient.StartSession(func() {
			apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			sdkClient.UploadFile(t, allocationID)

			getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, getCurrentRoundResponse)

			currentRoundTwiceString := getCurrentRoundResponse.CurrentRoundTwiceToString()

			getGraphBlobberSavedDataResponse, resp, err := apiClient.V1SharderGetGraphBlobberSavedData(
				t,
				model.GetGraphBlobberSavedDataRequest{
					DataPoints: 17,
					BlobberID:  blobberID,
					To:         currentRoundTwiceString,
				},
				client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, *getGraphBlobberSavedDataResponse)
			getGraphBlobberSavedDataResponseBefore := *getGraphBlobberSavedDataResponse

			wait.PoolImmediately(t, time.Minute*10, func() bool {
				getGraphBlobberSavedDataResponse, resp, err = apiClient.V1SharderGetGraphBlobberSavedData(
					t,
					model.GetGraphBlobberSavedDataRequest{
						DataPoints: 17,
						BlobberID:  blobberID,
						To:         currentRoundTwiceString,
					},
					client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, *getGraphBlobberSavedDataResponse)

				return !cmp.Equal(*getGraphBlobberSavedDataResponse, getGraphBlobberSavedDataResponseBefore)
			})
		})
	})

	t.Run("Get graph of read data of a certain blobber, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundTwiceString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		getGraphBlobberReadDataResponse, resp, err := apiClient.V1SharderGetGraphBlobberReadData(
			t,
			model.GetGraphBlobberReadDataRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundTwiceString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, *getGraphBlobberReadDataResponse)
	})

	t.Run("Check if a graph of read data of a certain blobber changes after file upload, should work", func(t *test.SystemTest) {
		t.Skip()
		//t.Skip("Skip until fixed")
		t.Parallel()

		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		sdkClient.StartSession(func() {
			fileName := sdkClient.UploadFile(t, allocationID)
			sdkClient.DownloadFile(t, allocationID, fileName)
		})

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundTwiceString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		getGraphBlobberReadDataResponse, resp, err := apiClient.V1SharderGetGraphBlobberReadData(
			t,
			model.GetGraphBlobberReadDataRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundTwiceString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, *getGraphBlobberReadDataResponse)
		getGraphBlobberReadDataResponseBefore := *getGraphBlobberReadDataResponse

		wait.PoolImmediately(t, time.Minute*10, func() bool {
			getGraphBlobberReadDataResponse, resp, err = apiClient.V1SharderGetGraphBlobberReadData(
				t,
				model.GetGraphBlobberReadDataRequest{
					DataPoints: 17,
					BlobberID:  blobberID,
					To:         currentRoundTwiceString,
				},
				client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, *getGraphBlobberReadDataResponse)

			return !cmp.Equal(*getGraphBlobberReadDataResponse, getGraphBlobberReadDataResponseBefore)
		})
	})

	t.Run("Get graph of total offers of a certain blobber, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundTwiceString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		var getGraphBlobberOffersTotalResponse *model.GetGraphBlobberOffersTotalResponse
		getGraphBlobberOffersTotalResponse, resp, err = apiClient.V1SharderGetGraphBlobberOffersTotal(
			t,
			model.GetGraphBlobberOffersTotalRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundTwiceString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberOffersTotalResponse)
	})

	t.RunWithTimeout("Check if a graph of total offers of a certain blobber changes after stake pool creation, should work", time.Minute*10, func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundTwiceString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		var getGraphBlobberOffersTotalResponse *model.GetGraphBlobberOffersTotalResponse
		getGraphBlobberOffersTotalResponse, resp, err = apiClient.V1SharderGetGraphBlobberOffersTotal(
			t,
			model.GetGraphBlobberOffersTotalRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundTwiceString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberOffersTotalResponse)

		getGraphBlobberOffersTotalResponseBefore := *getGraphBlobberOffersTotalResponse

		apiClient.CreateStakePool(t, wallet, 3, blobberID, client.TxSuccessfulStatus)

		wait.PoolImmediately(t, time.Minute*5, func() bool {
			getGraphBlobberOffersTotalResponse, resp, err = apiClient.V1SharderGetGraphBlobberOffersTotal(
				t,
				model.GetGraphBlobberOffersTotalRequest{
					DataPoints: 17,
					BlobberID:  blobberID,
					To:         currentRoundTwiceString,
				},
				client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotZero(t, *getGraphBlobberOffersTotalResponse)

			return !cmp.Equal(*getGraphBlobberOffersTotalResponse, getGraphBlobberOffersTotalResponseBefore)
		})
	})

	t.Run("Get graph of unstaked tokens of a certain blobber, should work", func(t *test.SystemTest) {
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getCurrentRoundResponse)

		currentRoundTwiceString := getCurrentRoundResponse.CurrentRoundTwiceToString()

		var getGraphBlobberUnstakeTotalResponse *model.GetGraphBlobberUnstakeTotalResponse
		getGraphBlobberUnstakeTotalResponse, resp, err = apiClient.V1SharderGetGraphBlobberUnstakeTotal(
			t,
			model.GetGraphBlobberUnstakeTotalRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
				To:         currentRoundTwiceString,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberUnstakeTotalResponse)
	})

	t.RunWithTimeout("Check if a graph of unstaked tokens of a certain blobber changes after stake pool unstaked, should work", time.Minute*10, func(t *test.SystemTest) {
		t.Skip("Skip until fixed")
		sdkClient.StartSession(func() {
			apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			getCurrentRoundResponse, resp, err := apiClient.GetCurrentRound(t, client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, getCurrentRoundResponse)

			currentRoundTwiceString := getCurrentRoundResponse.CurrentRoundTwiceToString()

			var getGraphBlobberUnstakeTotalResponse *model.GetGraphBlobberUnstakeTotalResponse
			getGraphBlobberUnstakeTotalResponse, resp, err = apiClient.V1SharderGetGraphBlobberUnstakeTotal(
				t,
				model.GetGraphBlobberUnstakeTotalRequest{
					DataPoints: 17,
					BlobberID:  blobberID,
					To:         currentRoundTwiceString,
				},
				client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.NotZero(t, *getGraphBlobberUnstakeTotalResponse)

			getGraphBlobberUnstakeTotalResponseBefore := *getGraphBlobberUnstakeTotalResponse

			apiClient.CreateStakePool(t, sdkWallet, 3, blobberID, client.TxSuccessfulStatus)

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
			apiClient.CollectRewards(t, sdkWallet, blobberID, 3, client.TxSuccessfulStatus)

			walletBalance = apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
			balanceAfter := walletBalance.Balance

			require.Equal(t, balanceAfter, balanceBefore+rewards)

			wait.PoolImmediately(t, time.Minute*5, func() bool {
				getGraphBlobberUnstakeTotalResponse, resp, err = apiClient.V1SharderGetGraphBlobberUnstakeTotal(
					t,
					model.GetGraphBlobberUnstakeTotalRequest{
						DataPoints: 17,
						BlobberID:  blobberID,
						To:         currentRoundTwiceString,
					},
					client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)
				require.NotZero(t, *getGraphBlobberUnstakeTotalResponse)

				return !cmp.Equal(*getGraphBlobberUnstakeTotalResponse, getGraphBlobberUnstakeTotalResponseBefore)
			})
		})
	})

	t.Run("Get graph of staked tokens of a certain blobber, should work", func(t *test.SystemTest) {
		t.Skip()
		wallet := apiClient.RegisterWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		blobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberTotalStakeResponse, resp, err := apiClient.V1SharderGetGraphBlobberTotalStake(
			t,
			model.GetGraphBlobberTotalStakeRequest{
				DataPoints: 17,
				BlobberID:  blobberID,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberTotalStakeResponse)
	})
}
