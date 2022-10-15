package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAtlusChimney(t *testing.T) {
	t.Parallel()

	t.Run("Get total minted tokens, should work", func(t *testing.T) {
		t.Parallel()

		getTotalMintedResponse, resp, err := apiClient.V1SharderGetTotalMinted(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, getTotalMintedResponse.TotalMinted, 0)
	})

	t.Run("Check if amount of total minted tokens changed after faucet execution, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		getTotalMintedResponse, resp, err := apiClient.V1SharderGetTotalMinted(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, getTotalMintedResponse.TotalMinted, 0)

		totalMintedBefore := getTotalMintedResponse.TotalMinted

		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		getTotalMintedResponse, resp, err = apiClient.V1SharderGetTotalMinted(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, getTotalMintedResponse.TotalMinted, 0)

		totalMintedAfter := getTotalMintedResponse.TotalMinted
		require.Greater(t, totalMintedAfter, totalMintedBefore)
	})

	t.Run("Get total total challenges, should work", func(t *testing.T) {
		t.Parallel()

		getTotalTotalChallengesResponse, resp, err := apiClient.V1SharderGetTotalTotalChallenges(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalTotalChallengesResponse.TotalTotalChallenges, 0)
	})

	t.Run("Check if amount of total total challenges changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)
		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)

			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

			getTotalTotalChallengesResponse, resp, err := apiClient.V1SharderGetTotalTotalChallenges(client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.GreaterOrEqual(t, getTotalTotalChallengesResponse.TotalTotalChallenges, 0)

			totalTotalChallengesBefore := getTotalTotalChallengesResponse.TotalTotalChallenges

			allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

			sdkClient.UploadFile(t, allocationID)

			var totalTotalChallengesAfter int

			wait.PoolImmediately(t, time.Minute*2, func() bool {
				getTotalTotalChallengesResponse, resp, err = apiClient.V1SharderGetTotalTotalChallenges(client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)

				totalTotalChallengesAfter = getTotalTotalChallengesResponse.TotalTotalChallenges

				return totalTotalChallengesAfter > totalTotalChallengesBefore
			})

			require.Greater(t, totalTotalChallengesAfter, totalTotalChallengesBefore)
		})
	})

	t.Run("Get total successful challenges, should work", func(t *testing.T) {
		t.Parallel()

		getTotalSuccessfulChallengesResponse, resp, err := apiClient.V1SharderGetTotalSuccessfulChallenges(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges, 0)
	})

	t.Run("Check if amount of total successful challenges changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)

			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

			getTotalSuccessfulChallengesResponse, resp, err := apiClient.V1SharderGetTotalSuccessfulChallenges(client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.GreaterOrEqual(t, getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges, 0)

			totalSuccessfulChallengesBefore := getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges

			allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

			sdkClient.UploadFile(t, allocationID)

			var totalSuccessfulChallengesAfter int

			wait.PoolImmediately(t, time.Minute*2, func() bool {
				getTotalSuccessfulChallengesResponse, resp, err = apiClient.V1SharderGetTotalSuccessfulChallenges(client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)

				totalSuccessfulChallengesAfter = getTotalSuccessfulChallengesResponse.TotalSuccessfulChallenges

				return totalSuccessfulChallengesAfter > totalSuccessfulChallengesBefore
			})

			require.Greater(t, totalSuccessfulChallengesAfter, totalSuccessfulChallengesBefore)
		})
	})

	t.Run("Get total allocated storage, should work", func(t *testing.T) {
		t.Parallel()

		getTotalAllocatedStorageResponse, resp, err := apiClient.V1SharderGetTotalAllocatedStorage(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalAllocatedStorageResponse.TotalAllocatedStorage, 0)
	})

	t.Run("Check if amount of total allocated storage changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)

			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

			getTotalAllocatedStorageResponse, resp, err := apiClient.V1SharderGetTotalAllocatedStorage(client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.GreaterOrEqual(t, getTotalAllocatedStorageResponse.TotalAllocatedStorage, 0)

			totalAllocatedStorageBefore := getTotalAllocatedStorageResponse.TotalAllocatedStorage

			allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

			sdkClient.UploadFile(t, allocationID)

			var totalAllocatedStorageAfter int

			wait.PoolImmediately(t, time.Minute*2, func() bool {
				getTotalAllocatedStorageResponse, resp, err = apiClient.V1SharderGetTotalAllocatedStorage(client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)

				totalAllocatedStorageAfter = getTotalAllocatedStorageResponse.TotalAllocatedStorage

				return totalAllocatedStorageAfter > totalAllocatedStorageBefore
			})

			require.Greater(t, totalAllocatedStorageAfter, totalAllocatedStorageBefore)
		})
	})

	t.Run("Get total staked, should work", func(t *testing.T) {
		t.Parallel()

		getTotalStakedResponse, resp, err := apiClient.V1SharderGetTotalStaked(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalStakedResponse.TotalStaked, 0)
	})

	t.Run("Check if amount of total staked changed after creating new allocation, should work", func(t *testing.T) {
		t.Parallel()

		getTotalStakedResponse, resp, err := apiClient.V1SharderGetTotalStaked(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalStakedResponse.TotalStaked, 0)
	})

	t.Run("Get total stored data, should work", func(t *testing.T) {
		t.Parallel()

		getTotalStoredDataResponse, resp, err := apiClient.V1SharderGetTotalStoredData(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalStoredDataResponse.TotalStoredData, 0)
	})

	t.Run("Check if total stored data changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)

			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

			getTotalStoredDataResponse, resp, err := apiClient.V1SharderGetTotalStoredData(client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.GreaterOrEqual(t, getTotalStoredDataResponse.TotalStoredData, 0)

			totalStoredDataBefore := getTotalStoredDataResponse.TotalStoredData

			allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

			sdkClient.UploadFile(t, allocationID)

			var totalStoredDataAfter int

			wait.PoolImmediately(t, time.Minute*2, func() bool {
				getTotalStoredDataResponse, resp, err = apiClient.V1SharderGetTotalStoredData(client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)

				totalStoredDataAfter = getTotalStoredDataResponse.TotalStoredData

				return totalStoredDataAfter > totalStoredDataBefore
			})

			require.Greater(t, totalStoredDataAfter, totalStoredDataBefore)
		})
	})

	t.Run("Get average write price, should work", func(t *testing.T) {
		t.Parallel()

		getAverageWritePriceResponse, resp, err := apiClient.V1SharderGetAverageWritePrice(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, getAverageWritePriceResponse.AverageWritePrice)
	})

	t.Run("Get total blobber capacity, should work", func(t *testing.T) {
		t.Parallel()

		getTotalBlobberCapacityResponse, resp, err := apiClient.V1SharderGetTotalBlobberCapacity(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, getTotalBlobberCapacityResponse.TotalBlobberCapacity, 0)
	})

	t.Run("Check if total blobber capacity changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.RegisterWalletForMnemonic(t, mnemonic)

		sdkClient.StartSession(func() {
			sdkClient.SetWallet(t, wallet, mnemonic)

			apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

			getTotalBlobberCapacityResponse, resp, err := apiClient.V1SharderGetTotalBlobberCapacity(client.HttpOkStatus)
			require.Nil(t, err)
			require.NotNil(t, resp)
			require.GreaterOrEqual(t, getTotalBlobberCapacityResponse.TotalBlobberCapacity, 0)

			totalBlobberCapacityBefore := getTotalBlobberCapacityResponse.TotalBlobberCapacity

			allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

			sdkClient.UploadFile(t, allocationID)

			var totalBlobberCapacityAfter int

			wait.PoolImmediately(t, time.Minute*2, func() bool {
				getTotalBlobberCapacityResponse, resp, err = apiClient.V1SharderGetTotalBlobberCapacity(client.HttpOkStatus)
				require.Nil(t, err)
				require.NotNil(t, resp)

				totalBlobberCapacityAfter = getTotalBlobberCapacityResponse.TotalBlobberCapacity

				return totalBlobberCapacityAfter < totalBlobberCapacityBefore
			})

			require.Less(t, totalBlobberCapacityAfter, totalBlobberCapacityBefore)
		})
	})

	t.Run("Get graph of blobber service charge of certain blobber, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (allocationBlobbers.Blobbers)[0]

		getGraphBlobberServiceChargeResponse, resp, err := apiClient.V1SharderGetGraphBlobberServiceCharge(
			model.GetGraphBlobberServiceChargeRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberServiceChargeResponse)
	})

	t.Run("Check if graph of blobber service charge changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (allocationBlobbers.Blobbers)[0]

		getGraphBlobberServiceChargeResponse, resp, err := apiClient.V1SharderGetGraphBlobberServiceCharge(
			model.GetGraphBlobberServiceChargeRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberServiceChargeResponse)
	})

	t.Run("Get graph of passed blobber challenges, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (allocationBlobbers.Blobbers)[0]

		getGraphBlobberChallengesPassed, resp, err := apiClient.V1SharderGetGraphBlobberChallengesPassed(
			model.GetGraphBlobberChallengesPassedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesPassed)
	})

	t.Run("Check if graph of passed blobber challenges changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (allocationBlobbers.Blobbers)[0]

		getGraphBlobberChallengesPassed, resp, err := apiClient.V1SharderGetGraphBlobberChallengesPassed(
			model.GetGraphBlobberChallengesPassedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesPassed)
	})

	t.Run("Get graph of completed blobber challenges, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberChallengesCompletedResponse, resp, err := apiClient.V1SharderGetGraphBlobberChallengesCompleted(
			model.GetGraphBlobberChallengesCompletedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesCompletedResponse)
	})

	t.Run("Check if graph of completed blobber challenges changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberChallengesCompletedResponse, resp, err := apiClient.V1SharderGetGraphBlobberChallengesCompleted(
			model.GetGraphBlobberChallengesCompletedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesCompletedResponse)
	})

	t.Run("Get graph of blobber inactive rounds, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberInactiveRounds, resp, err := apiClient.V1SharderGetGraphBlobberInactiveRounds(
			model.GetGraphBlobberInactiveRoundsRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberInactiveRounds)
	})

	t.Run("Check if graph of blobber inactive rounds changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, make([]*model.StorageNode, 0))

		getGraphBlobberInactiveRounds, resp, err := apiClient.V1SharderGetGraphBlobberInactiveRounds(
			model.GetGraphBlobberInactiveRoundsRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberInactiveRounds)
	})

	//echo -e "\naverges"
	//echo -e "\ngraph-blobber-write-price"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-write-price?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-capacity"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-capacity?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-allocated"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-allocated?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-saved-data"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-saved-data?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-read-data"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-read-data?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-offers-total"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-offers-total?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-unstake-total"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-unstake-total?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-total-stake"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-total-stake?data-points=17&id='$BLOBBERID
	//echo -e "\ngraph-blobber-challenges-open"
	//curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-open?data-points=17&id='$BLOBBERID
}
