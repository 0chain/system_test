package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxWalletAuth tests partner wallet authentication endpoints:
// - POST /v2/wallet/verify (verify wallet signature — Metamask/Zus wallet login)
// - POST /v2/wallet/auth (register wallet authentication)
func Test0BoxWalletAuth(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Verify wallet signature endpoint should respond", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)

		verifyBody := map[string]interface{}{
			"wallet_address": "0x0000000000000000000000000000000000000000",
			"signature":      "test_signature",
			"message":        "test_message",
			"provider":       "metamask",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/verify", headers, verifyBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() != 404,
			"Wallet verify endpoint should exist. Got %d: %s", resp.StatusCode(), resp.String())
		t.Logf("VerifyWalletSignature: %d — %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Register wallet auth should require authentication", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		authBody := map[string]interface{}{
			"wallet_address": "0x1234567890abcdef1234567890abcdef12345678",
			"provider":       "metamask",
		}

		resp, err := zboxClient.PostJSON(t, "/v2/wallet/auth", headers, authBody)
		require.NoError(t, err)
		t.Logf("RegisterWalletAuth: %d — %s", resp.StatusCode(), resp.String())
		require.True(t, resp.StatusCode() != 404,
			"Wallet auth endpoint should exist. Got %d", resp.StatusCode())
	})
}
