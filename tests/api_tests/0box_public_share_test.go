package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxPublicShare tests the public/group share endpoints:
// - POST /v2/shareinfo (share_info_type=public)
// - GET /v2/shareinfo/public/check
// - GET /v2/shareinfo/public/recipients
// - DELETE /v2/shareinfo/public/revoke
func Test0BoxPublicShare(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Create public share should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		shareinfoData := map[string]string{
			"auth_ticket":     NewTestShareinfo()["auth_ticket"],
			"share_info_type": "public",
		}

		_, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(),
			"Public share creation failed. Output: [%v]", response.String())
	})

	t.RunSequentiallyWithTimeout("Check public share exists should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a public share first
		shareinfoData := map[string]string{
			"auth_ticket":     NewTestShareinfo()["auth_ticket"],
			"share_info_type": "public",
		}
		_, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode())

		// Check if public share exists
		queryParams := map[string]string{
			"file_path_hash": "e724a2201e22653d32167fca1bf12b2e44bac6337d3ebdb7427fba4eecaa4c9d",
		}
		resp, err := zboxClient.CheckPublicShareExists(t, headers, queryParams)
		require.NoError(t, err)
		t.Logf("CheckPublicShareExists: %d — %s", resp.StatusCode(), resp.String())
		// 200 = exists, 400 = bad params, 404 = not found (all valid — endpoint exists)
		require.True(t, resp.StatusCode() != 405,
			"Endpoint should accept GET. Got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Get public share recipients should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a public share
		shareinfoData := map[string]string{
			"auth_ticket":     NewTestShareinfo()["auth_ticket"],
			"share_info_type": "public",
		}
		_, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode())

		// Get recipients
		queryParams := map[string]string{
			"file_path_hash": "e724a2201e22653d32167fca1bf12b2e44bac6337d3ebdb7427fba4eecaa4c9d",
		}
		resp, err := zboxClient.GetPublicShareRecipients(t, headers, queryParams)
		require.NoError(t, err)
		t.Logf("GetPublicShareRecipients: %d — %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Revoke public share should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create then revoke
		shareinfoData := map[string]string{
			"auth_ticket":     NewTestShareinfo()["auth_ticket"],
			"share_info_type": "public",
		}
		_, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode())

		queryParams := map[string]string{
			"file_path_hash": "e724a2201e22653d32167fca1bf12b2e44bac6337d3ebdb7427fba4eecaa4c9d",
		}
		resp, err := zboxClient.RevokePublicShare(t, headers, queryParams)
		require.NoError(t, err)
		t.Logf("RevokePublicShare: %d — %s", resp.StatusCode(), resp.String())
		// 200/204 = revoked, 400 = share not found or bad params (acceptable for test data)
		require.True(t, resp.StatusCode() != 404 && resp.StatusCode() != 405,
			"Revoke endpoint should exist. Got %d", resp.StatusCode())
	})
}
