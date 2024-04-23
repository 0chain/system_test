package cli_tests

import (
	"testing"
	"net/url"
	"fmt"
	"strings"
	"encoding/json"
	"bytes"

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

type customDataMap map[string]interface{}

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
	
		// Extract the last segment from the URL path as the blobber name
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
	
		// Extract the last segment from the URL path as the blobber name
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

	// I - Test Case for Blobber 
	t.RunSequentially("Compare data in MPT with events DB for blobbers", func(t *test.SystemTest) {

		// Fetch all blobbers from Events DB via "/v1/screst/:sc_address/getblobbers" endpoint
		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)

		require.NoError(t, err, "Failed to fetch blobbers")
		require.Equal(t, 200, resp.StatusCode(), "Expected HTTP status code 200")
		require.NotEmpty(t, blobbers, "Blobbers list should not be empty")

		for _, blobber := range blobbers {
			t.Logf("Blobber ID: %s, URL: %s", blobber.ID, blobber.BaseURL)
			t.Logf("*** Blobber Response Body from Events DB ***")
			t.Log(blobber)
			t.Logf("***")
			// Fetch Blobbers from MPT data structure
			fullURL := fmt.Sprintf("%s%s?key=provider:%s", apiClient.HealthyServiceProviders.Sharders[0], client.SCStateGet, blobber.ID)
			t.Logf(fullURL)
			response, err := apiClient.HttpClient.R().Get(fullURL)
			require.NoError(t, err, "Failed to fetch data from blobber")
			require.NotNil(t, response, "Response from blobber should not be nil")

			//var blobberMPT model.SCRestGetBlobberResponse
			//err = json.Unmarshal([]byte(response.Body()), &blobberMPT)
			//var dataMap map[string]interface{}
			//err = json.Unmarshal(response.Body(), &dataMap)
			//require.NoError(t, err, "Failed to unmarshal response into map")

			var dataMap customDataMap
			// Unmarshal using the custom unmarshal logic.
			if err := json.Unmarshal([]byte(response.Body()), &dataMap); err != nil {
				t.Logf("Error unmarshalling JSON: %s", err)
			}

			t.Logf("*** Blobber Response Body from MPT datastructure ***")
			//t.Log(blobberMPT)
			t.Log(dataMap)
			t.Log(dataMap)
			t.Logf("***")

			providerMap, ok  := dataMap["Provider"].(map[string]interface{})
			if ok {
				t.Log("Retrieved provider node from MPT")
			}

			require.Equal(t, blobber.ID, providerMap["ID"], "Blobber ID does not match")
			require.Equal(t, blobber.BaseURL, dataMap.GetString("BaseURL"), "Blobber BaseURL does not match")
			require.Equal(t, blobber.Capacity, dataMap.GetInt64("Capacity"), "Blobber Capacity does not match")
			require.Equal(t, blobber.Allocated, dataMap.GetInt64("Allocated"), "Blobber Allocated does not match")
			//require.Equal(t, blobber.LastHealthCheck, providerMap["LastHealthCheck"].(int64), "Blobber LastHealthCheck does not match")
			require.Equal(t, blobber.PublicKey, dataMap.GetString("PublicKey"), "Blobber PublicKey does not match")
			//require.Equal(t, blobber.TotalStake, dataMap["TotalStake"], "Blobber TotalStake does not match")
			//require.Equal(t, blobber.SavedData, dataMap["SavedData"], "Blobber SavedData does not match")
			//require.Equal(t, blobber.ReadData, dataMap["ReadData"], "Blobber ReadData does not match")
			//require.Equal(t, blobber.ChallengesPassed, dataMap["ChallengesPassed"], "Blobber ChallengesPassed does not match")
			//require.Equal(t, blobber.ChallengesCompleted, dataMap["ChallengesCompleted"], "Blobber ChallengesCompleted does not match")
	
			
		}

		
	})

	//  II - Test Case for Sharders 
	t.RunSequentially("Compare data in MPT with events DB for Sharders", func(t *test.SystemTest) {
		sharders, resp, err := apiClient.V1SCRestGetAllSharders(t, client.HttpOkStatus)
		require.NoError(t, err, "Failed to fetch sharders")
		require.Equal(t, 200, resp.StatusCode(), "Expected HTTP status code 200")
		require.NotEmpty(t, sharders, "Sharders list should not be empty")

		for _, sharder := range sharders {
			sharderURL := fmt.Sprintf("%s%s?key=provider:%s", apiClient.HealthyServiceProviders.Sharders[0], client.SCStateGet, sharder.ID)
			response, err := apiClient.HttpClient.R().Get(sharderURL)
			require.NoError(t, err, "Failed to fetch data for sharder from MPT "+sharder.ID)
			t.Log(sharder)

			var dataMap customDataMap
			// Unmarshal using the custom unmarshal logic.
			if err := json.Unmarshal([]byte(response.Body()), &dataMap); err != nil {
				t.Logf("Error unmarshalling JSON: %s", err)
			}
			t.Log(dataMap)
			simpleNodeMap, ok  := dataMap["SimpleNode"].(map[string]interface{})
			if ok {
				t.Log("Retrieved simple node from MPT")
			}
			

		
			require.Equal(t, sharder.BuildTag, simpleNodeMap["BuildTag"], "Blobber ID does not match")
			require.Equal(t, sharder.Host, simpleNodeMap["Host"], "Blobber BaseURL does not match")
			require.Equal(t, sharder.LastHealthCheck,  simpleNodeMap["LastHealthCheck"].(int64), "Blobber Capacity does not match")
			require.Equal(t, sharder.LastSettingUpdateRound, simpleNodeMap["LastSettingUpdateRound"].(int64), "Blobber Allocated does not match")
			//require.Equal(t, sharder.LastHealthCheck, lastHealthCheck, "Blobber LastHealthCheck does not match")
			//require.Equal(t, blobber.TotalStake, dataMap["TotalStake"], "Blobber TotalStake does not match")
			//require.Equal(t, blobber.SavedData, dataMap["SavedData"], "Blobber SavedData does not match")
			//require.Equal(t, blobber.ReadData, dataMap["ReadData"], "Blobber ReadData does not match")
			//require.Equal(t, blobber.ChallengesPassed, dataMap["ChallengesPassed"], "Blobber ChallengesPassed does not match")
			//require.Equal(t, blobber.ChallengesCompleted, dataMap["ChallengesCompleted"], "Blobber ChallengesCompleted does not match")
	
				
			
		}
	})

	//  Test Case for Miners 
	t.RunSequentially("Compare data in MPT with events DB for Miners", func(t *test.SystemTest) {
		miners, resp, err := apiClient.V1SCRestGetAllMiners(t, client.HttpOkStatus)
		require.NoError(t, err, "Failed to fetch miners")
		require.Equal(t, 200, resp.StatusCode(), "Expected HTTP status code 200")
		require.NotEmpty(t, miners, "Miners list should not be empty")

		for _, miner := range miners {
			minerURL := fmt.Sprintf("%s%s?key=provider:%s", apiClient.HealthyServiceProviders.Sharders[0], client.SCStateGet, miner.ID)
			response, err := apiClient.HttpClient.R().Get(minerURL)
			require.NoError(t, err, "Failed to fetch data for miner "+miner.ID)

			var dataMap map[string]interface{}
			err = json.Unmarshal(response.Body(), &dataMap)
			require.NoError(t, err, "Failed to unmarshal response for miner "+miner.ID)

			
		}
	})



}


func (cdm *customDataMap) UnmarshalJSON(data []byte) error {
    
    temp := map[string]interface{}{}

    dec := json.NewDecoder(bytes.NewReader(data))
    dec.UseNumber() 
    
    if err := dec.Decode(&temp); err != nil {
        return err
    }

    // Convert json.Number into int64 or float64 where necessary.
    for key, value := range temp {
        switch v := value.(type) {
        case json.Number:
            if i, err := v.Int64(); err == nil {
                temp[key] = i
            } else if f, err := v.Float64(); err == nil { 
                temp[key] = f
            }
        }
    }

    *cdm = temp
    return nil
}


func (cdm customDataMap) GetString(key string) string {
    if val, ok := cdm[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    return ""
}

func (cdm customDataMap) GetInt64(key string) int64 {
    if val, ok := cdm[key]; ok {
        switch v := val.(type) {
        case int64:
            return v
        case json.Number:
            if i, err := v.Int64(); err == nil {
                return i
            }
        }
    }
    return 0
}

func (cdm customDataMap) GetFloat64(key string) float64 {
    if val, ok := cdm[key]; ok {
        switch v := val.(type) {
        case float64:
            return v
        case json.Number:
            if f, err := v.Float64(); err == nil {
                return f
            }
        }
    }
    return 0.0
}
