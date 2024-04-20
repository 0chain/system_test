package cli_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/stretchr/testify/require"
	//"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	resty "github.com/go-resty/resty/v2"
)

var (
		apiClient *client.APIClient
)

func TestCompareMPTAndEventsDBData(testSetup *testing.T) { 


	t := test.NewSystemTest(testSetup)
	createWallet(t)
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
	//var resp *model.SCRestGetBlobbersResponse
	scStateGetResponse, resp, err := fetchMPTdata(t, StorageScAddress)
	t.Log(scStateGetResponse)
	t.Log(err)

	t.RunSequentially("Compare data in MPT with events DB for blobbers", func(t *test.SystemTest) {

		t.Log(apiClient.HealthyServiceProviders.Blobbers)

		// Blobbers
		for _, blobber := range apiClient.HealthyServiceProviders.Blobbers  { 
			t.Log(blobber)
			fetchAndCompareProviderData(testSetup, resp, blobber, "blobber")
			require.Equal(t, blobber, "blobber01") 
		}			

	})
/*
	t.RunSequentially("Compare data in MPT with events DB for Sharders", func(t *test.SystemTest) {

		// Sharders
		for _, sharder := range apiClient.HealthyServiceProviders.Sharders  { 
			t.Logf("sharder : %s", sharder)
			fetchAndCompareProviderData(testSetup, resp, sharder, "sharder") 
		}		

	})	

	t.RunSequentially("Compare data in MPT with events DB for Miners", func(t *test.SystemTest) {
		
		// Miners
		for _, miner := range apiClient.HealthyServiceProviders.Miners  { 
			t.Logf("miner : %s", miner)
			fetchAndCompareProviderData(testSetup, resp, miner, "miner") 
		}			

	})	
*/
	//url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/blobber_ids")



	// ref code for retrieving blobber individual URLs
	//sharders := getShardersList(t)
	//sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

}
	
func fetchAndLogProviderData(t *testing.T, baseURL, StorageScAddress, providerType string) {
	/*
		t.Logf("Fetch state of provider : %s ", providerType)

	if(providerType == "blobber") {
		var scRestGetBlobbersResponse *model.SCRestGetBlobbersResponse
		scRestGetBlobbersResponse = provider
		t.Log(scRestGetBlobbersResponse)
	}
	*/

	// Fetch the list of provider IDs
	url := fmt.Sprintf("%s/v1/screst/%s/%s_ids", baseURL, StorageScAddress, providerType)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to fetch %s list: %v", providerType, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body for %s list: %v", providerType, err)
	}

	// Split the response body to get individual provider IDs
	providerIDs := strings.Split(string(body), ",")
	for _, id := range providerIDs {
		fetchAndLogProviderDetails(t, baseURL, providerType, id)
	}
}

func fetchAndLogProviderDetails(t *testing.T, baseURL, providerType, id string) {
	// URL to fetch details for the specific provider
	detailURL := fmt.Sprintf("%s/%s/%s", baseURL, providerType, id)
	t.Log("Request URL: ", detailURL)

	resp, err := http.Get(detailURL)
	if err != nil {
		t.Fatalf("Failed to fetch data for %s %s: %v", providerType, id, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response for %s %s: %v", providerType, id, err)
	}

	t.Logf("Response for %s %s: %s", providerType, id, string(body))
}

func fetchMPTdata(testSetup *test.SystemTest, StorageScAddress string) (*model.SCStateGetResponse, *resty.Response, error){

	scStateGetResponse, resp, err := apiClient.V1SharderGetSCState(testSetup,
		model.SCStateGetRequest{
			SCAddress: client.StorageSmartContractAddress,
			Key:       walletIdx,
		},
		client.HttpOkStatus)

	require.Nil(testSetup, err)
	require.NotNil(testSetup, resp)
	require.NotNil(testSetup, scStateGetResponse)	

	return scStateGetResponse, resp, err
}
