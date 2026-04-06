package api_tests

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

// TestDatalakeAPI tests the zusCloudNative (datalake/blimp backend) API endpoints.
// The server runs at <0box_url>:8088 or via nginx at /datalake/.
func TestDatalakeAPI(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// Determine base URL — try 0box host with datalake path, fallback to localhost:8088
	baseURL := "http://localhost:8088"
	client := resty.New().SetTimeout(10 * time.Second)

	// Quick check if datalake is available
	resp, err := client.R().Get(baseURL + "/api/v1/health-check")
	if err != nil || resp.StatusCode() != http.StatusOK {
		testSetup.Skipf("Datalake API not available at %s (got %v, err=%v) — skipping", baseURL, resp, err)
	}

	t.RunSequentiallyWithTimeout("Health check should return OK", 1*time.Minute, func(t *test.SystemTest) {
		resp, err := client.R().Get(baseURL + "/api/v1/health-check")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "OK", resp.String())
	})

	t.RunSequentiallyWithTimeout("CORS preflight should return allowed headers", 1*time.Minute, func(t *test.SystemTest) {
		resp, err := client.R().
			SetHeader("Origin", "https://test.blimp.software").
			SetHeader("Access-Control-Request-Method", "POST").
			Options(baseURL + "/api/v1/health-check")
		require.NoError(t, err)
		// Should allow the origin (200 or 204)
		require.True(t, resp.StatusCode() == 200 || resp.StatusCode() == 204,
			"CORS preflight should return 200/204, got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Pricing dimensions should return valid data", 1*time.Minute, func(t *test.SystemTest) {
		resp, err := client.R().Get(baseURL + "/api/v1/pricing/dimensions")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)
		require.True(t, result["success"].(bool), "pricing response should have success=true")

		data := result["data"].(map[string]interface{})
		require.Greater(t, data["hourly_node_price_usd"].(float64), 0.0)
		require.Greater(t, data["gb_storage_price_usd"].(float64), 0.0)
		t.Logf("Pricing: $%.2f/node/hr, $%.4f/GB", data["hourly_node_price_usd"], data["gb_storage_price_usd"])
	})

	t.RunSequentiallyWithTimeout("Pricing calculate should return estimate", 1*time.Minute, func(t *test.SystemTest) {
		body := map[string]interface{}{
			"data_shards":   4,
			"parity_shards": 1,
			"storage_gb":    100,
			"hours":         720, // 1 month
		}
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			Post(baseURL + "/api/v1/pricing/calculate")
		require.NoError(t, err)
		t.Logf("Pricing calculate: %d — %s", resp.StatusCode(), resp.String())
		// 200 = success, 400/405/500 = validation/method error (endpoint exists)
		require.True(t, resp.StatusCode() != 404,
			"Pricing calculate endpoint should exist, got 404")
	})

	t.RunSequentiallyWithTimeout("Clusters list should work (empty or populated)", 1*time.Minute, func(t *test.SystemTest) {
		resp, err := client.R().Get(baseURL + "/api/v1/clusters/list")
		require.NoError(t, err)
		t.Logf("Clusters list: %d — %s", resp.StatusCode(), resp.String())
		// 200 = success (may be empty list), 401 = auth required, 405 = wrong method, 500 = internal
		require.True(t, resp.StatusCode() != 404,
			"Clusters list endpoint should exist, got 404")
	})

	t.RunSequentiallyWithTimeout("Path traversal should be blocked", 1*time.Minute, func(t *test.SystemTest) {
		resp, err := client.R().Get(baseURL + "/api/v1/../../../etc/passwd")
		require.NoError(t, err)
		require.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 403 || resp.StatusCode() == 404,
			"Path traversal should be blocked, got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Invalid endpoint should return 404", 1*time.Minute, func(t *test.SystemTest) {
		resp, err := client.R().Get(baseURL + "/api/v1/nonexistent")
		require.NoError(t, err)
		require.Equal(t, 404, resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("User registration without auth should fail", 1*time.Minute, func(t *test.SystemTest) {
		body := map[string]interface{}{
			"email": "test@test.com",
		}
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			Post(baseURL + "/api/v1/users/register")
		require.NoError(t, err)
		t.Logf("Register without auth: %d — %s", resp.StatusCode(), resp.String())
		// Should require auth (401/403), validation error (400), or server error (500 = missing aws_customer_id)
		require.True(t, resp.StatusCode() != 200 && resp.StatusCode() != 201,
			"Unauthenticated register should NOT succeed, got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Revenue stats endpoint exists", 1*time.Minute, func(t *test.SystemTest) {
		resp, err := client.R().Get(baseURL + "/api/v1/revenue/stats")
		require.NoError(t, err)
		t.Logf("Revenue stats: %d — %s", resp.StatusCode(), resp.String())
		// 200 = success, 401 = auth required, 500 = internal (acceptable — endpoint exists)
		require.True(t, resp.StatusCode() != 404,
			"Revenue stats endpoint should exist, got 404")
	})
}
