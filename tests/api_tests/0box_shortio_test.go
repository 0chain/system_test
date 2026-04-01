package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxShortIO tests the Short.io URL shortening integration.
func Test0BoxShortIO(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Create short link should respond", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		linkBody := map[string]interface{}{
			"original_url": "https://test.vult.network/share/test_auth_ticket",
			"title":        "Test shared file",
		}

		resp, err := zboxClient.CreateShortLink(t, headers, linkBody)
		require.NoError(t, err)
		// 200/201 = success, 400/403 = short.io not configured (acceptable on test chains)
		require.True(t, resp.StatusCode() != 404,
			"Short.io endpoint should exist (got 404). Output: %s", resp.String())
		t.Logf("CreateShortLink: %d — %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Create referral link should respond", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		referralBody := map[string]interface{}{
			"original_url": "https://test.vult.network/signup?ref=test_user",
			"title":        "Test referral",
		}

		resp, err := zboxClient.CreateReferralLink(t, headers, referralBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() != 404,
			"Referral endpoint should exist (got 404). Output: %s", resp.String())
		t.Logf("CreateReferralLink: %d — %s", resp.StatusCode(), resp.String())
	})
}
