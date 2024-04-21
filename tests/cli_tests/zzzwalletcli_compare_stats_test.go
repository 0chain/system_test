package cli_tests

import (
	"testing"
	"net/url"
	"fmt"
	"strings"
	"encoding/json"

	"github.com/0chain/system_test/internal/api/util/test"
	//"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/stretchr/testify/require"
	//"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	//resty "github.com/go-resty/resty/v2"
	
)

var (
		apiClient *client.APIClient
		
)

func TestCompareMPTAndEventsDBData(testSetup *testing.T) { 


	t := test.NewSystemTest(testSetup)
	createWallet(t)

	t.Log("Default Config File ",configPath)

	parsedConfig := config.Parse("./config/"+configPath)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)

	// Fetch base URLs for all sharders
	sharderBaseURLs := make(map[string]string)

	for _, sharderURL := range apiClient.HealthyServiceProviders.Sharders {

		parsedURL, err := url.Parse(sharderURL)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			continue
		}
	
		// Extract the last segment from the URL path as the blobber ID
		segments := strings.Split(parsedURL.Path, "/")
		sharderID := segments[len(segments)-1]
		sharderBaseURLs[sharderID] = sharderURL

		t.Log("Fetched blobber URL:", sharderURL)


	}

	// Fetch base URLs for all miners
	minerBaseURLs := make(map[string]string)

	for _, minerURL := range apiClient.HealthyServiceProviders.Miners {

		parsedURL, err := url.Parse(minerURL)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			continue
		}
	
		// Extract the last segment from the URL path as the blobber ID
		segments := strings.Split(parsedURL.Path, "/")
		minerID := segments[len(segments)-1]
		minerBaseURLs[minerID] = minerURL

		t.Log("Fetched miner URL:", minerURL)


	}	

	// Fetch base URLs for all blobbers
	blobberBaseURLs := make(map[string]string)

	for _, blobberURL := range apiClient.HealthyServiceProviders.Blobbers {

		parsedURL, err := url.Parse(blobberURL)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			continue
		}
	
		// Extract the last segment from the URL path as the blobber ID
		segments := strings.Split(parsedURL.Path, "/")
		blobberID := segments[len(segments)-1]
		blobberBaseURLs[blobberID] = blobberURL

		t.Log("Fetched blobber URL:", blobberURL)


	}


	t.RunSequentially("Compare data in MPT with events DB for blobbers", func(t *test.SystemTest) {



		// Fetch Blobbers from Events DB
		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)

		require.NoError(t, err, "Failed to fetch blobbers")
		require.Equal(t, 200, resp.StatusCode(), "Expected HTTP status code 200")
		require.NotEmpty(t, blobbers, "Blobbers list should not be empty")

		for _, blobber := range blobbers {
			t.Logf("Blobber ID: %s, URL: %s", blobber.ID, blobber.BaseURL)
			t.Logf("***")
			t.Log(blobber)
			t.Logf("***")
			// Fetch Blobbers from MPT data structure
			fullURL := fmt.Sprintf("%s%s?key=provider:%s", apiClient.HealthyServiceProviders.Sharders[0], client.SCStateGet, blobber.ID)
			t.Logf(fullURL)
			response, err := apiClient.HttpClient.R().Get(fullURL)
			require.NoError(t, err, "Failed to fetch data from blobber")
			require.NotNil(t, response, "Response from blobber should not be nil")

			var dataMap map[string]interface{}
			err = json.Unmarshal(response.Body(), &dataMap)
			require.NoError(t, err, "Failed to unmarshal response into map")
			t.Log(dataMap)

			providerData, ok := dataMap["Provider"].(map[string]interface{})
			if !ok {
				t.Error("Provider data is missing or not in expected format")
				continue 
			}

			require.Equal(t, blobber.ID, providerData["ID"].(string), "Blobber ID does not match")
			require.Equal(t, blobber.BaseURL, dataMap["BaseURL"], "Blobber BaseURL does not match")
			require.Equal(t, blobber.Capacity, dataMap["Capacity"], "Blobber Capacity does not match")
			require.Equal(t, blobber.Allocated, dataMap["Allocated"], "Blobber Allocated does not match")
			require.Equal(t, blobber.LastHealthCheck, providerData["LastHealthCheck"].(string), "Blobber LastHealthCheck does not match")
			//require.Equal(t, blobber.TotalStake, dataMap["TotalStake"], "Blobber TotalStake does not match")
			require.Equal(t, blobber.SavedData, dataMap["SavedData"], "Blobber SavedData does not match")
			require.Equal(t, blobber.ReadData, dataMap["ReadData"], "Blobber ReadData does not match")
			//require.Equal(t, blobber.ChallengesPassed, dataMap["ChallengesPassed"], "Blobber ChallengesPassed does not match")
			//require.Equal(t, blobber.ChallengesCompleted, dataMap["ChallengesCompleted"], "Blobber ChallengesCompleted does not match")
	
			
		}

		
	})



}