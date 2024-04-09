package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"net/http" 
	"net/http/httptest" 
	"testing"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/chain"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/rest"
)


func TestCompareMPTAndEventsDBData(testSetup *testing.T) { 
	
	//t := test.NewSystemTest(testSetup)
	//createWallet(t)

    c := chain.GetServerChain()

	// Mock MPT server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        common.WithCORS(common.UserRateLimit(common.ToJSONResponse(c.GetNodeFromSCState)))(w, r)
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
		srh := NewStorageRestHandler(rh)
        switch r.URL.Path {
        case "/v1/screst/" + ADDRESS + "/getBlobber":
            common.WithCORS(common.UserRateLimit(common.ToJSONResponse(srh.GetBlobber)))(w, r)
        case "/v1/screst/" + ADDRESS + "/get_validator":
            common.WithCORS(common.UserRateLimit(common.ToJSONResponse(srh.GetBlobber)))(w, r)
		case "/v1/screst/" + ADDRESS + "/getblobbers":
			common.WithCORS(common.UserRateLimit(common.ToJSONResponse(srh.GetBlobbers)))(w, r)	        
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

}


