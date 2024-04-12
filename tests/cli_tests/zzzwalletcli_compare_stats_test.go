package cli_tests

import (
	"fmt"
	"testing"
	"io"
	"github.com/0chain/system_test/internal/api/util/test"
	"net/http" 
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"

)


func TestCompareMPTAndEventsDBData(testSetup *testing.T) { 


	t := test.NewSystemTest(testSetup)
	createWallet(t)
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/blobber_ids")

	// Iterate over each provider type 
	for _, provider := range []string{"blobber", "sharder", "miner"} 
	{ 
		fetchAndLogProviderData(t, sharderBaseUrl, StorageScAddress, provider) 
	}	
	// ref code for retrieving blobber individual URLs
	//sharders := getShardersList(t)
	//sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]
	

	t.Log("URL : ", url)

	resp, err := http.Get(url) 

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	responseBody := string(body)
	t.Log("Response Body: ", responseBody)

	blobberIDs := strings.Split(responseBody, ",")

	for _, blobberID := range blobberIDs {

		url := fmt.Sprintf("%s/blobber/%s", sharderBaseUrl, blobberID)
		t.Log("Request URL: ", url)
	

		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Failed to fetch data for blobber %s: %v", blobberID, err)
		}
		defer resp.Body.Close()
	
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response for blobber %s: %v", blobberID, err)
		}
	
		t.Logf("Response for blobber %s: %s", blobberID, string(body))
	}

}

func fetchAndLogProviderData(t *testing.T, baseURL, StorageScAddress, providerType string) {
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

