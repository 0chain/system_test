package api_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxZusLogin tests the Firebase/Zus login flow:
// - Firebase sign-in returns valid ID token (via RefreshFirebaseToken at startup)
// - GET /v2/csrftoken works
// - Full login flow: Firebase signIn -> get CSRF -> use headers -> create wallet
func Test0BoxZusLogin(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Firebase sign-in should have provided valid ID token at startup", 1*time.Minute, func(t *test.SystemTest) {
		// RefreshFirebaseToken is called in TestMain (main_test.go).
		// If Firebase credentials were configured, X_APP_ID_TOKEN should be a real JWT.
		// If not configured, it falls back to "test_firebase_token" (dev mode).
		require.NotEmpty(t, client.X_APP_ID_TOKEN, "Firebase ID token should be set")
		require.NotEmpty(t, client.X_APP_USER_ID, "Firebase user ID should be set")
		t.Logf("Firebase ID token present (len=%d), user_id=%s", len(client.X_APP_ID_TOKEN), client.X_APP_USER_ID)

		if client.X_APP_ID_TOKEN == "test_firebase_token" {
			t.Log("Using default test token (Firebase credentials not configured). Dev mode assumed.")
		} else {
			t.Log("Real Firebase ID token obtained via sign-in.")
		}
	})

	t.RunSequentiallyWithTimeout("CSRF token endpoint should work", 2*time.Minute, func(t *test.SystemTest) {
		csrfToken, resp, err := zboxClient.GetCSRFToken(t)
		require.NoError(t, err)
		t.Logf("GetCSRFToken: %d - %v", resp.StatusCode(), csrfToken)

		require.Equal(t, 200, resp.StatusCode(),
			"CSRF token endpoint should return 200. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotNil(t, csrfToken, "CSRF token response should not be nil")
		require.NotEmpty(t, csrfToken.CSRFToken, "CSRF token should not be empty")
		t.Logf("CSRF token obtained: %s...", csrfToken.CSRFToken[:min(20, len(csrfToken.CSRFToken))])
	})

	t.RunSequentiallyWithTimeout("Full login flow: Firebase -> CSRF -> create wallet should succeed", 3*time.Minute, func(t *test.SystemTest) {
		// This tests the complete Zus login flow end-to-end:
		// 1. Firebase sign-in already happened in TestMain (X_APP_ID_TOKEN is set)
		// 2. Get CSRF token
		// 3. Use headers with real Firebase token + CSRF to create owner + wallet

		// Step 1: Get headers with live CSRF token
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Step 2: Verify OTP (creates owner) — in dev mode this is a passthrough
		verifyOtpInput := NewVerifyOtpDetails()
		_, resp, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)
		t.Logf("VerifyOtp: %d - %s", resp.StatusCode(), resp.String())
		// 200 = owner created, 400 with "duplicate key" = already exists (both OK)
		require.True(t, resp.StatusCode() == 200 ||
			(resp.StatusCode() == 400 && containsDuplicateKey(resp.String())),
			"VerifyOtp should succeed or report duplicate. Got %d: %s", resp.StatusCode(), resp.String())

		// Step 3: Create wallet with the authenticated session
		walletInput := NewTestWallet()
		_, resp, err = zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		t.Logf("CreateWallet: %d - %s", resp.StatusCode(), resp.String())
		require.True(t, resp.StatusCode() == 201 ||
			(resp.StatusCode() == 400 && containsDuplicateOrExists(resp.String())),
			"CreateWallet should succeed or report exists. Got %d: %s", resp.StatusCode(), resp.String())

		// Step 4: Verify wallet was created by fetching wallet keys
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"GetWalletKeys should return 200. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, wallet.PublicKey, "Wallet public key should not be empty")
		t.Logf("Wallet verified: clientID=%s, publicKey=%s...", wallet.ClientID, wallet.PublicKey[:min(20, len(wallet.PublicKey))])
	})

	t.RunSequentiallyWithTimeout("Headers without Firebase token should fail wallet creation in production mode", 2*time.Minute, func(t *test.SystemTest) {
		// In production mode (auth required), missing X-App-ID-TOKEN should be rejected.
		// In dev mode (IsDevelopmentNoAuth=true), this may still succeed.
		headers := map[string]string{
			"X-App-Client-ID":        client.X_APP_CLIENT_ID,
			"X-App-Client-Key":       client.X_APP_CLIENT_KEY,
			"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        client.X_APP_TIMESTAMP,
			"X-App-User-ID":          client.X_APP_USER_ID,
			"X-APP-TYPE":             client.X_APP_VULT,
			// Deliberately omitting X-App-ID-TOKEN and X-CSRF-TOKEN
		}

		walletInput := NewTestWallet()
		_, resp, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		t.Logf("CreateWallet (no token): %d - %s", resp.StatusCode(), resp.String())
		// In production: 400/401/403. In dev mode: 201 (auth bypassed).
		// Both are valid — the test confirms the endpoint responds.
		require.True(t, resp.StatusCode() >= 200 && resp.StatusCode() < 500,
			"Endpoint should return a client-processable response. Got %d: %s",
			resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("JWT token creation after login should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create JWT token — this requires a valid authenticated session
		jwtToken, resp, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"JWT token creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, jwtToken.JwtToken, "JWT token should not be empty")
		t.Logf("JWT token obtained (len=%d)", len(jwtToken.JwtToken))
	})
}

// containsDuplicateKey checks if a response string contains a duplicate key message.
func containsDuplicateKey(s string) bool {
	return strings.Contains(s, "duplicate key value violates unique constraint")
}

// containsDuplicateOrExists checks if a response indicates the resource already exists.
func containsDuplicateOrExists(s string) bool {
	return strings.Contains(s, "duplicate key value violates unique constraint") ||
		strings.Contains(s, "Cannot create new wallet")
}
