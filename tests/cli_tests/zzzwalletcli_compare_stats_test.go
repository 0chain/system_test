package cli_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	//"github.com/stretchr/testify/require"
	//"github.com/0chain/system_test/tests/tokenomics_tests/utils"
)

var (
		apiClient *client.APIClient
)

func TestCompareMPTAndEventsDBData(testSetup *testing.T) { 


	t := test.NewSystemTest(testSetup)
	wallet := createWallet(t)
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	//sharderBaseUrl := utils.GetSharderUrl(t)
	t.Log("Default Config File ",configPath)

	//var scRestShardersResponse *model.SCRestGetMinersShardersResponse

	parsedConfig := config.Parse("./config/"+configPath)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)

	/*
	sharders, resp, err := apiClient.V1SCRestGetAllSharders(t, client.HttpOkStatus)

	require.NoError(t, err, "Failed to fetch sharders")
	require.Equal(t, 200, resp.StatusCode(), "Expected HTTP status code 200")
	require.NotEmpty(t, sharders, "Sharders list should not be empty")
	t.Log(sharders[0].Host)
*/
	
	scStateGetResponse, resp, err = fetchMPTdata(testSetup, wallet, StorageScAddress)

	t.RunSequentially("Compare data in MPT with events DB for blobbers", func(t *test.SystemTest) {

		// Blobbers
		for _, blobber := range apiClient.HealthyServiceProviders.Blobbers  { 
			t.Logf("Blobber : %s", blobber)
			fetchAndCompareProviderData(testSetup, blobber, "blobber") 
		}			

	})

	t.RunSequentially("Compare data in MPT with events DB for Sharders", func(t *test.SystemTest) {

		// Sharders
		for _, sharder := range apiClient.HealthyServiceProviders.Sharders  { 
			t.Logf("sharder : %s", sharder)
			fetchAndCompareProviderData(testSetup, sharder, "sharder", client.GetSharders) 
		}		

	})	

	t.RunSequentially("Compare data in MPT with events DB for Miners", func(t *test.SystemTest) {
		
		// Miners
		for _, miner := range apiClient.HealthyServiceProviders.Miners  { 
			t.Logf("miner : %s", miner)
			fetchAndCompareProviderData(testSetup, miner, StorageScAddress, "miner", client.GetMiners) 
		}			

	})	

	//url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/blobber_ids")



	// ref code for retrieving blobber individual URLs
	//sharders := getShardersList(t)
	//sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

}

func fetchAndCompareProviderData(t *testing.T, block, provider, providerType string) {
	t.Logf("Fetch state of provider : %s ", providerType)

	if(providerType == "blobber") {
		var scRestGetBlobbersResponse *model.SCRestGetBlobbersResponse
		scRestGetBlobbersResponse = provider
	}
}
	

func fetchMPTdata(t *testing.T, wallet, StorageScAddress string) {

	scStateGetResponse, resp, err := apiClient.V1SharderGetSCState(
		t,
		model.SCStateGetRequest{
			SCAddress: client.StorageSmartContractAddress,
			Key:       wallet.Id,
		},
		client.HttpOkStatus)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, scStateGetResponse)	

	return 	scStateGetResponse, resp, err
}
