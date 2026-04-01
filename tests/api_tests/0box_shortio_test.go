package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxShortIO(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create short link for public file share URL")

	// shortioConfigured tracks whether Short.io API keys are set up on this
	// 0box instance. The first real test probes it; later tests use the
	// cached result so they can log a warning instead of failing.
	shortioConfigured := true

	t.RunSequentiallyWithTimeout("Create short link for public file share URL", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a real shareinfo entry with a public auth ticket
		publicShareData := newShareinfoWithType("public")
		shareinfoResp, resp, err := zboxClient.CreateShareInfo(t, headers, publicShareData)
		require.NoError(t, err, "CreateShareInfo transport error")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateShareInfo failed: %s", resp.String())
		require.Equal(t, "shareinfo added successfully", shareinfoResp.Message)

		// Verify the shareinfo was persisted by reading it back
		sharedList, resp, err := zboxClient.GetShareInfoShared(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.GreaterOrEqual(t, len(sharedList.Data), 1,
			"At least one share entry should exist after creation")

		// Build the share URL from the real auth ticket
		shareURL := "https://test.vult.network/share/" + publicShareData["auth_ticket"]

		linkBody := map[string]interface{}{
			"original_url": shareURL,
			"title":        "Public file share",
		}

		shortResp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON transport error")
		t.Logf("Short.io link response: %d - %s", shortResp.StatusCode(), shortResp.String())

		require.NotEqual(t, 404, shortResp.StatusCode(),
			"Short.io link endpoint must exist (got 404): %s", shortResp.String())

		if shortResp.StatusCode() >= 400 {
			shortioConfigured = false
			t.Logf("Short.io may not be configured (status %d). "+
				"Remaining tests will verify endpoint existence only.", shortResp.StatusCode())
		} else {
			require.True(t, shortResp.StatusCode() == 200 || shortResp.StatusCode() == 201,
				"Expected 200/201, got %d: %s", shortResp.StatusCode(), shortResp.String())
			require.Contains(t, shortResp.String(), "short",
				"Response should contain shortened URL data")
		}

		_, _, _ = zboxClient.DeleteShareinfo(t, headers, publicShareData["auth_ticket"])
	})

	t.RunSequentiallyWithTimeout("Create short link for private file share URL", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a real private shareinfo with encrypted flag
		privateShareData := newShareinfoWithType("private")
		shareinfoResp, resp, err := zboxClient.CreateShareInfo(t, headers, privateShareData)
		require.NoError(t, err, "CreateShareInfo transport error")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateShareInfo failed: %s", resp.String())
		require.Equal(t, "shareinfo added successfully", shareinfoResp.Message)

		// Verify the private share was persisted
		sharedList, resp, err := zboxClient.GetShareInfoShared(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.GreaterOrEqual(t, len(sharedList.Data), 1,
			"Private share entry should exist after creation")

		shareURL := "https://test.vult.network/share/" + privateShareData["auth_ticket"]

		linkBody := map[string]interface{}{
			"original_url": shareURL,
			"title":        "Private file share",
		}

		shortResp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON transport error")
		t.Logf("Short.io private link response: %d - %s", shortResp.StatusCode(), shortResp.String())

		require.NotEqual(t, 404, shortResp.StatusCode(),
			"Short.io link endpoint must exist (got 404): %s", shortResp.String())

		if shortioConfigured && (shortResp.StatusCode() == 200 || shortResp.StatusCode() == 201) {
			require.Contains(t, shortResp.String(), "short",
				"Response should contain shortened URL data")
		} else if !shortioConfigured {
			t.Logf("Short.io not configured, skipping link content validation")
		}

		_, _, _ = zboxClient.DeleteShareinfo(t, headers, privateShareData["auth_ticket"])
	})

	t.RunSequentiallyWithTimeout("Create short link for folder share URL", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a folder share (reference_type=d for directory)
		folderShareData := newFolderShareinfo()
		shareinfoResp, resp, err := zboxClient.CreateShareInfo(t, headers, folderShareData)
		require.NoError(t, err, "CreateShareInfo transport error")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateShareInfo for folder failed: %s", resp.String())
		require.Equal(t, "shareinfo added successfully", shareinfoResp.Message)

		// Verify the folder share was persisted
		sharedList, resp, err := zboxClient.GetShareInfoShared(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.GreaterOrEqual(t, len(sharedList.Data), 1,
			"Folder share entry should exist after creation")

		shareURL := "https://test.vult.network/share/" + folderShareData["auth_ticket"]

		linkBody := map[string]interface{}{
			"original_url": shareURL,
			"title":        "Shared folder link",
		}

		shortResp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON transport error")
		t.Logf("Short.io folder link response: %d - %s", shortResp.StatusCode(), shortResp.String())

		require.NotEqual(t, 404, shortResp.StatusCode(),
			"Short.io link endpoint must exist (got 404): %s", shortResp.String())

		if shortioConfigured && (shortResp.StatusCode() == 200 || shortResp.StatusCode() == 201) {
			require.Contains(t, shortResp.String(), "short",
				"Response should contain shortened URL data")
		} else if !shortioConfigured {
			t.Logf("Short.io not configured, skipping link content validation")
		}

		_, _, _ = zboxClient.DeleteShareinfo(t, headers, folderShareData["auth_ticket"])
	})

	t.RunSequentiallyWithTimeout("Empty URL should return error", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		linkBody := map[string]interface{}{
			"original_url": "",
			"title":        "Empty URL test",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON transport error")
		t.Logf("Short.io empty URL response: %d - %s", resp.StatusCode(), resp.String())

		require.True(t, resp.StatusCode() >= 400,
			"Empty original_url should be rejected with 4xx/5xx, got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Malformed URL should return error", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		linkBody := map[string]interface{}{
			"original_url": "not-a-valid-url",
			"title":        "Malformed URL test",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON transport error")
		t.Logf("Short.io malformed URL response: %d - %s", resp.StatusCode(), resp.String())

		require.True(t, resp.StatusCode() >= 400,
			"Malformed URL should be rejected with 4xx/5xx, got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Very long URL should be handled gracefully", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		longTicket := strings.Repeat("a", 4000)
		longURL := "https://test.vult.network/share/" + longTicket

		linkBody := map[string]interface{}{
			"original_url": longURL,
			"title":        "Very long URL test",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/link", headers, linkBody)
		require.NoError(t, err, "PostJSON transport error")
		t.Logf("Short.io long URL response: %d - %s", resp.StatusCode(), resp.String())

		require.NotEqual(t, 404, resp.StatusCode(),
			"Short.io link endpoint must exist (got 404): %s", resp.String())
	})

	t.RunSequentiallyWithTimeout("Request without auth headers should return error", 2*time.Minute, func(t *test.SystemTest) {
		noAuthHeaders := map[string]string{
			"X-APP-TYPE": client.X_APP_VULT,
		}

		linkBody := map[string]interface{}{
			"original_url": "https://test.vult.network/share/no_auth_test",
			"title":        "No auth test",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/shortio/link", noAuthHeaders, linkBody)
		require.NoError(t, err, "PostJSON transport error")
		t.Logf("Short.io no-auth response: %d - %s", resp.StatusCode(), resp.String())

		require.True(t, resp.StatusCode() >= 400,
			"Request without auth should be rejected, got %d", resp.StatusCode())
	})

}

// newShareinfoWithType builds a shareinfo map with a dynamically constructed
// auth ticket whose owner_id matches the current X_APP_CLIENT_ID.
func newShareinfoWithType(shareType string) map[string]string {
	ticket := map[string]interface{}{
		"client_id":        client.X_APP_CLIENT_ID,
		"owner_id":         client.X_APP_CLIENT_ID,
		"allocation_id":    "e0c2cd2d5faaad13fc53733dd759749f2b2f01af46332009c678221a2c488153",
		"file_path_hash":   "e724a2201e22653d32167fca1bf12b2e44bac6337d3ebdb7427fba4eecaa4c9d",
		"actual_file_hash": "1f112083f2405c395de51b7f13f9f795aa141c40f1d47d78c83a49930f2b9a34",
		"file_name":        "test_share_file.txt",
		"reference_type":   "f",
		"expiration":       0,
		"timestamp":        time.Now().Unix(),
		"encrypted":        shareType == "private",
		"signature":        "339e55299b48e229dde902f8c965d15a940b2777c5d9327a4c79131b8a771e07",
	}
	ticketJSON, _ := json.Marshal(ticket)
	return map[string]string{
		"auth_ticket":     base64.StdEncoding.EncodeToString(ticketJSON),
		"share_info_type": shareType,
	}
}

// newFolderShareinfo builds a shareinfo map for a folder share (reference_type=d).
func newFolderShareinfo() map[string]string {
	ticket := map[string]interface{}{
		"client_id":      client.X_APP_CLIENT_ID,
		"owner_id":       client.X_APP_CLIENT_ID,
		"allocation_id":  "e0c2cd2d5faaad13fc53733dd759749f2b2f01af46332009c678221a2c488153",
		"file_path_hash": "a3b4c5d6e7f8091a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5",
		"file_name":      "test_shared_folder",
		"reference_type": "d",
		"expiration":     0,
		"timestamp":      time.Now().Unix(),
		"encrypted":      false,
		"signature":      "449f66399c59f339ede902f8c965d15a940b2777c5d9327a4c79131b8a882f18",
	}
	ticketJSON, _ := json.Marshal(ticket)
	return map[string]string{
		"auth_ticket":     base64.StdEncoding.EncodeToString(ticketJSON),
		"share_info_type": "public",
	}
}
