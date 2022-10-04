package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAtlusChimney(t *testing.T) {
	t.Parallel()

	t.Run("Get total minted tokens, should work", func(t *testing.T) {
		t.Parallel()

		getTotalMintedResponse, resp, err := apiClient.V1SharderGetTotalMinted(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getTotalMintedResponse)
	})

	t.Run("Check if amount of total minted tokens changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		getTotalMintedResponse, resp, err := apiClient.V1SharderGetTotalMinted(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getTotalMintedResponse)
	})

	t.Run("Get total total challenges, should work", func(t *testing.T) {
		t.Parallel()

		getTotalTotalChallengesResponse, resp, err := apiClient.V1SharderGetTotalTotalChallenges(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getTotalTotalChallengesResponse)
	})

	t.Run("Check if amount of total total challenges changed after file uploading, should work", func(t *testing.T) {
		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		getTotalTotalChallengesResponse, resp, err := apiClient.V1SharderGetTotalTotalChallenges(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getTotalTotalChallengesResponse)
	})

	t.Run("Get total successful challenges, should work", func(t *testing.T) {
		//total-successful-challenges
	})

	t.Run("Check if amount of total successful challenges changed after file uploading, should work", func(t *testing.T) {
		//total-successful-challenges
	})

	t.Run("Get total allocated storage, should work", func(t *testing.T) {
		//total-allocated-storage
	})

	t.Run("Check if amount of total allocated storage changed after file uploading, should work", func(t *testing.T) {
		//total-allocated-storage
	})

	t.Run("Get total staked, should work", func(t *testing.T) {
		//total-staked
		t.Parallel()

		getTotalStakedResponse, resp, err := apiClient.V1SharderGetTotalStaked(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getTotalStakedResponse)
	})

	t.Run("Check if amount of total staked changed after creating new allocation, should work", func(t *testing.T) {
		//total-staked
		t.Parallel()

		getTotalStakedResponse, resp, err := apiClient.V1SharderGetTotalStaked(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getTotalStakedResponse)
	})

	t.Run("Get total stored data, should work", func(t *testing.T) {
		t.Parallel()

		getTotalStoredDataResponse, resp, err := apiClient.V1SharderGetTotalStoredData(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getTotalStoredDataResponse)
	})

	t.Run("Check if total stored data changed after file uploading, should work", func(t *testing.T) {
		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		getTotalStoredDataResponse, resp, err := apiClient.V1SharderGetTotalStoredData(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, getTotalStoredDataResponse.TotalStoredData)

		totalStoredDataBefore := getTotalStoredDataResponse.TotalStoredData

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		sdkClient.UploadSomeFile(t, allocationID)

		getTotalStoredDataResponse, resp, err = apiClient.V1SharderGetTotalStoredData(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, getTotalStoredDataResponse.TotalStoredData)

		totalStoredDataAfter := getTotalStoredDataResponse.TotalStoredData

		require.Greater(t, totalStoredDataAfter, totalStoredDataBefore)
	})

	t.Run("Get average write price, should work", func(t *testing.T) {
		t.Parallel()

		getAverageWritePriceResponse, resp, err := apiClient.V1SharderGetAverageWritePrice(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getAverageWritePriceResponse)
	})

	t.Run("Get total blobber capacity, should work", func(t *testing.T) {
		t.Parallel()

		getAverageWritePriceResponse, resp, err := apiClient.V1SharderGetTotalBlobberCapacity(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, getAverageWritePriceResponse)
	})

	t.Run("Check if total blobber capacity changed after file uploading, should work", func(t *testing.T) {
		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		sdkClient.SetWallet(wallet)

		//TODO: upload some data

		//TODO: check blobber capacity before uploading
		getAverageWritePriceResponse, resp, err := apiClient.V1SharderGetTotalBlobberCapacity(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getAverageWritePriceResponse)
		//TODO: check blobber capacity after uploading
	})

	t.Run("Get graph of blobber service charge of certain blobber, should work", func(t *testing.T) {
		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberServiceChargeResponse, resp, err := apiClient.V1SharderGetGraphBlobberServiceCharge(
			model.GetGraphBlobberServiceChargeRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberServiceChargeResponse)

		//graph-blobber-service-charge?data-points=17&id='$BLOBBERID
	})

	t.Run("Check if graph of blobber service charge changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberServiceChargeResponse, resp, err := apiClient.V1SharderGetGraphBlobberServiceCharge(
			model.GetGraphBlobberServiceChargeRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberServiceChargeResponse)

		//graph-blobber-service-charge?data-points=17&id='$BLOBBERID
	})

	t.Run("Get graph of passed blobber challenges, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberChallengesPassed, resp, err := apiClient.V1SharderGetGraphBlobberChallengesPassed(
			model.GetGraphBlobberChallengesPassedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesPassed)

		//graph-blobber-challenges-passed?data-points=17&id='$BLOBBERID
	})

	t.Run("Check if graph of passed blobber challenges changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberChallengesPassed, resp, err := apiClient.V1SharderGetGraphBlobberChallengesPassed(
			model.GetGraphBlobberChallengesPassedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesPassed)

		//graph-blobber-challenges-passed?data-points=17&id='$BLOBBERID
	})

	t.Run("Get graph of completed blobber challenges, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberChallengesCompletedResponse, resp, err := apiClient.V1SharderGetGraphBlobberChallengesCompleted(
			model.GetGraphBlobberChallengesCompletedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesCompletedResponse)

		//graph-blobber-challenges-completed?data-points=17&id='$BLOBBERID
	})

	t.Run("Check if graph of completed blobber challenges changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberChallengesCompletedResponse, resp, err := apiClient.V1SharderGetGraphBlobberChallengesCompleted(
			model.GetGraphBlobberChallengesCompletedRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberChallengesCompletedResponse)

		//graph-blobber-challenges-completed?data-points=17&id='$BLOBBERID
	})

	t.Run("Get graph of blobber inactive rounds, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberInactiveRounds, resp, err := apiClient.V1SharderGetGraphBlobberInactiveRounds(
			model.GetGraphBlobberInactiveRoundsRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberInactiveRounds)

		//graph-blobber-inactive-rounds?data-points=17&id='$BLOBBERID
	})

	t.Run("Check if graph of blobber inactive rounds changed after file uploading, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		blobberId := (*allocationBlobbers.Blobbers)[0]

		getGraphBlobberInactiveRounds, resp, err := apiClient.V1SharderGetGraphBlobberInactiveRounds(
			model.GetGraphBlobberInactiveRoundsRequest{
				DataPoints: 17,
				BlobberID:  blobberId,
			},
			client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, *getGraphBlobberInactiveRounds)

		//graph-blobber-inactive-rounds?data-points=17&id='$BLOBBERID
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
