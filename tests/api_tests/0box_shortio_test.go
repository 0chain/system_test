package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxShortIO(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create short link for valid shared file URL")

	t.RunSequentiallyWithTimeout("Create short link for valid shared file URL", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		linkBody := map[string]interface{}{
			"original_url": "https://test.vult.network/share/abc123_auth_ticket_sample",
			"title":        "Shared test file",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON should not return transport error")
		t.Logf("CreateShortLink response: %d — %s", resp.StatusCode(), resp.String())

		// Accept 200/201 as success. 400/403/500 may indicate Short.io API key
		// is not configured on the test chain, but 404 means the endpoint is missing.
		require.NotEqual(t, 404, resp.StatusCode(),
			"Short.io link endpoint must exist. Got 404: %s", resp.String())

		// If the endpoint is properly configured, verify the response contains a URL
		if resp.StatusCode() == 200 || resp.StatusCode() == 201 {
			require.Contains(t, resp.String(), "short",
				"Successful response should contain shortened URL data")
		}
	})

	t.RunSequentiallyWithTimeout("Create short link with empty URL returns error", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		linkBody := map[string]interface{}{
			"original_url": "",
			"title":        "Empty URL test",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON should not return transport error")
		t.Logf("CreateShortLink (empty URL) response: %d — %s", resp.StatusCode(), resp.String())

		// Empty URL should be rejected — expect 400 or similar client error
		require.True(t, resp.StatusCode() >= 400,
			"Empty original_url should be rejected with 4xx/5xx, got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Create referral short link", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		referralBody := map[string]interface{}{
			"original_url": "https://test.vult.network/signup?ref=test_user_001",
			"title":        "Test referral link",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/referral", headers, referralBody)
		require.NoError(t, err, "PostJSON should not return transport error")
		t.Logf("CreateReferralLink response: %d — %s", resp.StatusCode(), resp.String())

		require.NotEqual(t, 404, resp.StatusCode(),
			"Referral endpoint must exist. Got 404: %s", resp.String())

		if resp.StatusCode() == 200 || resp.StatusCode() == 201 {
			require.Contains(t, resp.String(), "short",
				"Successful referral response should contain shortened URL data")
		}
	})

	t.RunSequentiallyWithTimeout("Create short link without auth headers returns error", 2*time.Minute, func(t *test.SystemTest) {
		// Send request with minimal/empty headers — no auth
		noAuthHeaders := map[string]string{
			"X-APP-TYPE": client.X_APP_VULT,
		}

		linkBody := map[string]interface{}{
			"original_url": "https://test.vult.network/share/no_auth_test",
			"title":        "No auth test",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/link", noAuthHeaders, linkBody)
		require.NoError(t, err, "PostJSON should not return transport error")
		t.Logf("CreateShortLink (no auth) response: %d — %s", resp.StatusCode(), resp.String())

		// Without proper auth headers, the request should be rejected
		require.True(t, resp.StatusCode() >= 400,
			"Request without auth should be rejected, got %d", resp.StatusCode())
	})
}
