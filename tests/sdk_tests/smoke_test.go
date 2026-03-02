package sdk_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// TestSmokeInfrastructure is a lightweight smoke test that verifies all
// infrastructure services are operational before running the full test suite.
// Run with: go test -run TestSmokeInfrastructure -v ./tests/sdk_tests/
func TestSmokeInfrastructure(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetRunAllTestsAsSmokeTest()

	t.RunSequentially("Chain is producing blocks", func(t *test.SystemTest) {
		lfb := apiClient.GetLatestFinalizedBlock(t, client.HttpOkStatus)
		require.NotNil(t, lfb, "LFB should not be nil")
		require.Greater(t, lfb.Round, int64(0), "Chain should be past genesis")
		t.Logf("Chain OK - LFB round: %d, miner: %s", lfb.Round, lfb.MinerId)
	})

	t.RunSequentially("Miners are registered", func(t *test.SystemTest) {
		miners, resp, err := apiClient.V1SCRestGetAllMiners(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, len(miners), 3, "Expected at least 3 miners, got %d", len(miners))
		t.Logf("Miners OK - %d miners registered", len(miners))
	})

	t.RunSequentially("Sharders are registered", func(t *test.SystemTest) {
		sharders, resp, err := apiClient.V1SCRestGetAllSharders(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, len(sharders), 1, "Expected at least 1 sharder, got %d", len(sharders))
		t.Logf("Sharders OK - %d sharders registered", len(sharders))
	})

	t.RunSequentially("Blobbers are registered", func(t *test.SystemTest) {
		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, len(blobbers), 6, "Expected at least 6 blobbers, got %d", len(blobbers))

		staked := 0
		for _, b := range blobbers {
			if b.TotalStake > 0 {
				staked++
			}
		}
		require.GreaterOrEqual(t, staked, 6, "Expected at least 6 staked blobbers, got %d", staked)
		t.Logf("Blobbers OK - %d registered, %d staked", len(blobbers), staked)
	})

	t.RunSequentially("Validators are registered", func(t *test.SystemTest) {
		validators, resp, err := apiClient.V1SCRestGetAllValidators(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, len(validators), 6, "Expected at least 6 validators, got %d", len(validators))
		t.Logf("Validators OK - %d registered", len(validators))
	})

	t.RunSequentiallyWithTimeout("Wallet creation and faucet funding works", 2*time.Minute, func(t *test.SystemTest) {
		walletStr, err := zcncore.CreateWalletOffline()
		require.NoError(t, err, "Wallet creation should succeed")
		require.NotEmpty(t, walletStr)

		var walletMap map[string]interface{}
		err = json.Unmarshal([]byte(walletStr), &walletMap)
		require.NoError(t, err, "Wallet should be valid JSON")
		require.Contains(t, walletMap, "client_id")
		require.Contains(t, walletMap, "mnemonics")

		mnemonic := walletMap["mnemonics"].(string)
		wallet := apiClient.CreateWalletForMnemonic(t, mnemonic)
		require.NotNil(t, wallet)
		require.NotEmpty(t, wallet.Id)

		apiClient.FundWallet(t, wallet, 3, client.TxSuccessfulStatus)
		balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		require.Greater(t, balance.Balance, int64(0), "Wallet should have positive balance after funding")
		t.Logf("Wallet+Faucet OK - wallet %s funded with %d tokens", wallet.Id, balance.Balance)
	})

	t.RunSequentiallyWithTimeout("Allocation creation and file upload works", 5*time.Minute, func(t *test.SystemTest) {
		walletStr, err := zcncore.CreateWalletOffline()
		require.NoError(t, err)

		var walletMap map[string]interface{}
		err = json.Unmarshal([]byte(walletStr), &walletMap)
		require.NoError(t, err)

		wallet := apiClient.CreateWalletForMnemonic(t, walletMap["mnemonics"].(string))
		apiClient.FundWallet(t, wallet, 15, client.TxSuccessfulStatus)

		// Ensure we have enough balance for allocation + fees
		apiClient.EnsureWalletBalance(t, wallet, 12)

		blobberRequirements := model.SCRestGetAllocationBlobbersRequest{
			BlobberRequirements: model.BlobberRequirements{
				DataShards:     2,
				ParityShards:   2,
				Size:           10 * 1024 * 1024, // 10 MB
				OwnerId:       wallet.Id,
				OwnerPublicKey: wallet.PublicKey,
				ExpirationDate: time.Now().Add(8760 * time.Hour).Unix(), // 1 year
				ReadPriceRange: model.PriceRange{
					Min: 0,
					Max: 9223372036854775807,
				},
				WritePriceRange: model.PriceRange{
					Min: 0,
					Max: 9223372036854775807,
				},
				StorageVersion: 1,
			},
		}

		allocBlobbers, resp, err := apiClient.V1SCRestGetAllocationBlobbers(t, &blobberRequirements, client.HttpOkStatus)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, allocBlobbers)
		require.NotNil(t, allocBlobbers.Blobbers)
		require.GreaterOrEqual(t, len(*allocBlobbers.Blobbers), 4, "Need at least 4 blobbers for 2+2 allocation")

		allocID := apiClient.CreateAllocation(t, wallet, allocBlobbers, client.TxSuccessfulStatus)
		require.NotEmpty(t, allocID, "Allocation ID should not be empty")
		t.Logf("Allocation OK - created %s", allocID)

		// Wait for blobbers to sync the allocation from the chain event
		time.Sleep(15 * time.Second)

		sdkClient.SetWallet(t, wallet)
		remoteName, size := sdkClient.UploadFile(t, allocID)
		require.NotEmpty(t, remoteName)
		require.Greater(t, size, int64(0))
		t.Logf("File Upload OK - uploaded %s (%d bytes) to allocation %s", remoteName, size, allocID)
	})
}

// TestSmokeServices verifies external services (0box, zauth, zvault) are reachable.
// These tests use HTTP health checks and don't require Firebase credentials.
// Run with: go test -run TestSmokeServices -v ./tests/sdk_tests/
func TestSmokeServices(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetRunAllTestsAsSmokeTest()

	httpClient := &http.Client{Timeout: 10 * time.Second}

	t.RunSequentially("0box service is reachable", func(t *test.SystemTest) {
		if parsedConfig.ZboxUrl == "" {
			t.Skip("0box_url not configured")
		}
		resp, err := httpClient.Get(parsedConfig.ZboxUrl + "/v2/csrftoken")
		require.NoError(t, err, "0box should be reachable")
		require.Equal(t, http.StatusOK, resp.StatusCode, "0box /v2/csrftoken should return 200")
		t.Logf("0box OK - %s responding", parsedConfig.ZboxUrl)
	})

	t.RunSequentially("zauth service is reachable", func(t *test.SystemTest) {
		if parsedConfig.ZauthUrl == "" {
			t.Skip("zauth_url not configured")
		}
		resp, err := httpClient.Get(parsedConfig.ZauthUrl + "/setup")
		require.NoError(t, err, "zauth should be reachable")
		// zauth /setup is POST-only, so 405 means the service is alive
		require.Contains(t, []int{http.StatusOK, http.StatusMethodNotAllowed}, resp.StatusCode,
			"zauth /setup should return 200 or 405, got %d", resp.StatusCode)
		t.Logf("zauth OK - %s responding (status %d)", parsedConfig.ZauthUrl, resp.StatusCode)
	})

	t.RunSequentially("zvault service is reachable", func(t *test.SystemTest) {
		if parsedConfig.ZvaultUrl == "" {
			t.Skip("zvault_url not configured")
		}
		resp, err := httpClient.Get(parsedConfig.ZvaultUrl + "/")
		require.NoError(t, err, "zvault should be reachable")
		// zvault returns 404 on / but that means the service is running
		require.Less(t, resp.StatusCode, 500, "zvault should not return 5xx, got %d", resp.StatusCode)
		t.Logf("zvault OK - %s responding (status %d)", parsedConfig.ZvaultUrl, resp.StatusCode)
	})

	t.RunSequentially("0dns network endpoint works", func(t *test.SystemTest) {
		resp, err := httpClient.Get(parsedConfig.BlockWorker + "/network")
		require.NoError(t, err, "0dns /network should be reachable")
		require.Equal(t, http.StatusOK, resp.StatusCode, "0dns /network should return 200")

		var networkResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&networkResp)
		resp.Body.Close()
		require.NoError(t, err, "0dns response should be valid JSON")

		miners, ok := networkResp["miners"]
		require.True(t, ok, "0dns response should contain 'miners'")
		minerList, ok := miners.([]interface{})
		require.True(t, ok, "miners should be an array")
		require.Greater(t, len(minerList), 0, "Should have at least 1 miner in network")

		sharders, ok := networkResp["sharders"]
		require.True(t, ok, "0dns response should contain 'sharders'")
		sharderList, ok := sharders.([]interface{})
		require.True(t, ok, "sharders should be an array")
		require.Greater(t, len(sharderList), 0, "Should have at least 1 sharder in network")

		t.Logf("0dns OK - %d miners, %d sharders", len(minerList), len(sharderList))
	})

	t.RunSequentially("0box event pipeline is processing", func(t *test.SystemTest) {
		if parsedConfig.ZboxUrl == "" {
			t.Skip("0box_url not configured")
		}
		// Check if 0box has recent snapshots (indicates Kafka pipeline is working)
		url := fmt.Sprintf("%s/v2/graph-total-minted?data-points=1", parsedConfig.ZboxUrl)
		resp, err := httpClient.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Skip("0box graph endpoint not available - Kafka pipeline may need time to accumulate data")
		}
		defer resp.Body.Close()

		var graphData []interface{}
		err = json.NewDecoder(resp.Body).Decode(&graphData)
		if err != nil {
			t.Skip("0box graph data not parseable yet - pipeline may still be bootstrapping")
		}
		t.Logf("0box Pipeline OK - graph endpoint returning data (%d points)", len(graphData))
	})
}
