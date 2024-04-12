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
	/*
	t := test.NewSystemTest(testSetup)
	createWallet(t)
	mptBaseURL := ""

	// Blobber 
	response = baseURL+ SCStateGet
	urlBuilder := NewURLBuilder().SetPath(SCStateGet)

	scStateGetResponse, resp, err := apiClient.V1SharderGetSCState(
		t,
		model.SCStateGetRequest{
			SCAddress: client.FaucetSmartContractAddress,
			Key:       wallet.Id,
		},
		client.HttpOkStatus)

		urlBuilder := NewURLBuilder().
		SetPath(SCStateGet)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			FormData: map[string]string{
				"sc_address": scStateGetRequest.SCAddress,
				"key":        scStateGetRequest.Key,
			},
			Dst:                &scStateGetResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		SharderServiceProvider)

	return scStateGetResponse, resp, err
	
	http.HandleFunc("/v1/scstate/get", common.WithCORS(common.UserRateLimit(common.ToJSONResponse(c.GetNodeFromSCState))))
*/
	/*
	// Mock MPT server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{"key": "value"}` 
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
    }))
    defer server.Close()

    resp, err := http.Get(server.URL + "/v1/scstate/get")
    if err != nil {
        t.Fatalf("Failed to make request: %v", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        t.Fatalf("Failed to read response: %v", err)
    }

	// Mock EventsDB server

	dbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

        switch r.URL.Path {
        case "/v1/screst/" + ADDRESS + "/getBlobber":
			w.Header().Set("Content-Type", "application/json")
			response := `{"key": "value"}` 
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
        case "/v1/screst/" + ADDRESS + "/get_validator":
			w.Header().Set("Content-Type", "application/json")
			response := `{"key": "value"}` 
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		case "/v1/screst/" + ADDRESS + "/getblobbers":
			w.Header().Set("Content-Type", "application/json")
			response := `{"key": "value"}` 
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))      
        }
    }))
    defer dbServer.Close()

    // Fetch blobber list
    resp, err := http.Get(server.URL + "/v1/screst/" + ADDRESS + "/getblobbers")
    if err != nil {
        t.Fatalf("Failed to fetch blobber list: %v", err)
    }
    defer resp.Body.Close()

    blobberListBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        t.Fatalf("Failed to read blobber list response: %v", err)
    }

    var blobbers []BlobberData
    if err := json.Unmarshal(blobberListBody, &blobbers); err != nil {
        t.Fatalf("Failed to unmarshal blobber list: %v", err)
    }

    for _, blobber := range blobbers {
        individualResp, err := http.Get(server.URL + "/v1/screst/" + ADDRESS + "/getBlobber?id=" + blobber.ID)
        if err != nil {
            t.Errorf("Failed to fetch data for blobber %s: %v", blobber.ID, err)
            continue
        }
        defer individualResp.Body.Close()
    }
*/

	t := test.NewSystemTest(testSetup)
	createWallet(t)
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/blobber_ids")

	// ref code for retrieving blobber individual URLs
	//sharders := getShardersList(t)
	//sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]
	

	t.Log("URL : ", url)

	resp, err := http.Get(url) //nolint:gosec

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	responseBody := string(body)
	t.Log("Response Body: ", responseBody)

}


