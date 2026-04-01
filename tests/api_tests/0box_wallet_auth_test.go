package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxWalletAuth tests partner wallet authentication endpoints:
// - POST /v2/wallet/verify (verify wallet signature - Metamask/Zus wallet login)
// - POST /v2/wallet/auth (register wallet authentication)
func Test0BoxWalletAuth(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Verify wallet signature with valid Ethereum address should respond", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)

		verifyBody := map[string]interface{}{
			"wallet_address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"signature":      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			"message":        "Sign this message to verify your wallet",
			"provider":       "metamask",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/verify", headers, verifyBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() != 404,
			"Wallet verify endpoint should exist. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("VerifyWalletSignature (valid address): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Verify wallet signature with empty address should return 400", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)

		verifyBody := map[string]interface{}{
			"wallet_address": "",
			"signature":      "test_signature",
			"message":        "test_message",
			"provider":       "metamask",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/verify", headers, verifyBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 422,
			"Empty wallet address should return 400 or 422. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("VerifyWalletSignature (empty address): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Verify wallet signature with invalid address should return 400", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)

		verifyBody := map[string]interface{}{
			"wallet_address": "not_a_valid_address",
			"signature":      "invalid_sig",
			"message":        "test_message",
			"provider":       "metamask",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/verify", headers, verifyBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 422,
			"Invalid wallet address should return 400 or 422. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("VerifyWalletSignature (invalid address): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Verify wallet signature with wrong provider should return 400", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)

		verifyBody := map[string]interface{}{
			"wallet_address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"signature":      "0xabcdef1234567890",
			"message":        "test_message",
			"provider":       "unsupported_provider",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/verify", headers, verifyBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 422,
			"Unsupported provider should return 400 or 422. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("VerifyWalletSignature (wrong provider): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Register wallet auth for metamask provider should respond", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		authBody := map[string]interface{}{
			"wallet_address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"provider":       "metamask",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/auth", headers, authBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() != 404,
			"Wallet auth endpoint should exist. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("RegisterWalletAuth (metamask): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Register wallet auth for zus provider should respond", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		authBody := map[string]interface{}{
			"wallet_address": "0x1234567890abcdef1234567890abcdef12345678",
			"provider":       "zus",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/auth", headers, authBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() != 404,
			"Wallet auth endpoint should exist. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("RegisterWalletAuth (zus): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Register wallet auth without authentication should return 401 or 403", 2*time.Minute, func(t *test.SystemTest) {
		// Use minimal headers without proper auth credentials
		headers := map[string]string{
			"X-App-Type": client.X_APP_VULT,
		}

		authBody := map[string]interface{}{
			"wallet_address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"provider":       "metamask",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/auth", headers, authBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 401 || resp.StatusCode() == 403,
			"Unauthenticated wallet auth should return 400/401/403. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("RegisterWalletAuth (no auth): %d - %s", resp.StatusCode(), resp.String())
	})
}
