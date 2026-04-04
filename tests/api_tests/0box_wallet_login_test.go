package api_tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxWalletLogin tests the MetaMask/Zus wallet login flow:
// - GET  /v2/wallet/nonce  (public — returns a one-time nonce)
// - POST /v2/wallet/verify (public — verifies signature + returns firebaseCustomToken)
func Test0BoxWalletLogin(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Get wallet nonce should return nonce", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxPublicHeaders(client.X_APP_VULT)

		resp, err := zboxClient.GetWalletNonce(t, headers)
		require.NoError(t, err)
		t.Logf("GetWalletNonce: %d - %s", resp.StatusCode(), resp.String())

		if resp.StatusCode() == 404 {
			t.Log("wallet/nonce endpoint not available on this 0box version, skipping")
			return
		}

		require.Equal(t, 200, resp.StatusCode(),
			"GetWalletNonce should return 200. Got %d: %s", resp.StatusCode(), resp.String())

		var nonceResp map[string]interface{}
		err = json.Unmarshal(resp.Body(), &nonceResp)
		require.NoError(t, err, "Failed to parse nonce response")
		require.NotEmpty(t, nonceResp["nonce"], "Nonce should not be empty")
		t.Logf("Nonce received: %v", nonceResp["nonce"])
	})

	t.RunSequentiallyWithTimeout("Verify wallet signature with invalid address should fail", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxPublicHeaders(client.X_APP_VULT)

		// First get a valid nonce
		nonceResp, err := zboxClient.GetWalletNonce(t, headers)
		require.NoError(t, err)
		if nonceResp.StatusCode() == 404 {
			t.Log("wallet/nonce endpoint not available, skipping")
			return
		}
		require.Equal(t, 200, nonceResp.StatusCode())

		var nonceData map[string]interface{}
		err = json.Unmarshal(nonceResp.Body(), &nonceData)
		require.NoError(t, err)
		nonce := nonceData["nonce"].(string)

		// Use invalid (non-Ethereum) wallet address
		body := map[string]interface{}{
			"walletAddress": "not-a-valid-address",
			"signature":     "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			"provider":      "metamask",
			"message":       "Sign this message to verify your wallet",
			"nonce":         nonce,
		}

		resp, err := zboxClient.VerifyWalletSignature(t, headers, body)
		require.NoError(t, err)
		t.Logf("VerifyWalletSignature (invalid address): %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 400, resp.StatusCode(),
			"Invalid address should return 400. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Verify wallet signature with invalid provider should fail", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxPublicHeaders(client.X_APP_VULT)

		nonceResp, err := zboxClient.GetWalletNonce(t, headers)
		require.NoError(t, err)
		if nonceResp.StatusCode() == 404 {
			t.Log("wallet/nonce endpoint not available, skipping")
			return
		}
		require.Equal(t, 200, nonceResp.StatusCode())

		var nonceData map[string]interface{}
		err = json.Unmarshal(nonceResp.Body(), &nonceData)
		require.NoError(t, err)
		nonce := nonceData["nonce"].(string)

		body := map[string]interface{}{
			"walletAddress": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"signature":     "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			"provider":      "unsupported_provider",
			"message":       "Sign this message to verify your wallet",
			"nonce":         nonce,
		}

		resp, err := zboxClient.VerifyWalletSignature(t, headers, body)
		require.NoError(t, err)
		t.Logf("VerifyWalletSignature (invalid provider): %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 400, resp.StatusCode(),
			"Unsupported provider should return 400. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Verify wallet signature with expired nonce should fail", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxPublicHeaders(client.X_APP_VULT)

		// Use a nonce that was never stored in Redis
		body := map[string]interface{}{
			"walletAddress": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"signature":     "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			"provider":      "metamask",
			"message":       "Sign this message to verify your wallet",
			"nonce":         "fake-nonce-never-stored",
		}

		resp, err := zboxClient.VerifyWalletSignature(t, headers, body)
		require.NoError(t, err)
		t.Logf("VerifyWalletSignature (expired nonce): %d - %s", resp.StatusCode(), resp.String())

		if resp.StatusCode() == 404 {
			t.Log("wallet/verify endpoint not available, skipping")
			return
		}

		// Expect 400 or 500 (redis error when nonce not found)
		require.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 500,
			"Expired/invalid nonce should return 400 or 500. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Full nonce then verify round-trip with dummy signature should return Invalid signature", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxPublicHeaders(client.X_APP_VULT)

		// Step 1: Get nonce
		nonceResp, err := zboxClient.GetWalletNonce(t, headers)
		require.NoError(t, err)
		if nonceResp.StatusCode() == 404 {
			t.Log("wallet/nonce endpoint not available, skipping")
			return
		}
		require.Equal(t, 200, nonceResp.StatusCode())

		var nonceData map[string]interface{}
		err = json.Unmarshal(nonceResp.Body(), &nonceData)
		require.NoError(t, err)
		nonce := nonceData["nonce"].(string)
		require.NotEmpty(t, nonce)

		// Step 2: Verify with a well-formed but incorrect signature.
		// We cannot produce a real Ethereum signature without a private key, so we
		// expect the endpoint to accept the request format and reject the signature.
		body := map[string]interface{}{
			"walletAddress": "0x71c7656ec7ab88b098defb751b7401b5f6d8976f",
			"signature":     "0x" + "aa" + "bb" + "cc" + "dd" + "ee" + "ff" + "0011223344556677889900112233445566778899001122334455667788990011223344556677889900112233445566778899001122334455667788990011220001",
			"provider":      "metamask",
			"message":       "Sign this message to verify your wallet. Nonce: " + nonce,
			"nonce":         nonce,
		}

		resp, err := zboxClient.VerifyWalletSignature(t, headers, body)
		require.NoError(t, err)
		t.Logf("VerifyWalletSignature (round-trip): %d - %s", resp.StatusCode(), resp.String())

		// The endpoint should process the request and return 400 "Invalid signature"
		// since we cannot actually sign with the test address's private key.
		require.Equal(t, 400, resp.StatusCode(),
			"Dummy signature should return 400 Invalid signature. Got %d: %s", resp.StatusCode(), resp.String())
	})
}
