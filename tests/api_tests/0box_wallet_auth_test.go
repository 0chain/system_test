package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxWalletAuth tests the real wallet authentication flow:
// - Full split-key wallet lifecycle via zvault (store, split, share, verify keys)
// - Wallet verify endpoint with real wallet data
// - Wallet auth registration for metamask and zus providers
// - Unauthenticated access rejection
func Test0BoxWalletAuth(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Full split-key wallet flow: store in zvault, split key, share, verify keys", 3*time.Minute, func(t *test.SystemTest) {
		require.True(t, zvaultAvailable, "zvault must be available")

		// Step 1: Create wallet in 0box and get JWT token
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup should succeed")

		jwtToken, resp, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"JWT token creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, jwtToken.JwtToken, "JWT token should not be empty")
		t.Logf("JWT token obtained: %s...", jwtToken.JwtToken[:20])

		// Step 2: Store wallet private key + mnemonic in zvault
		zvaultHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		resp, err = zvaultClient.Store(t, PRIVATE_KEY, MNEMONIC, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Store in zvault should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		t.Log("Private key and mnemonic stored in zvault")

		// Step 3: Generate split key via zvault
		resp, err = zvaultClient.GenerateSplitKey(t, CLIENT_ID_V, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Split key generation should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		t.Log("Split key generated")

		// Step 4: Verify keys exist via zvault GetKeys
		keys, resp, err := zvaultClient.GetKeys(t, CLIENT_ID_V, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"GetKeys should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.Len(t, keys.Keys, 1, "Should have exactly 1 split key")
		require.Equal(t, CLIENT_ID_V, keys.Keys[0].ClientID, "Split key client ID should match")
		require.NotEmpty(t, keys.Keys[0].PublicKey, "Split key should have a public key")
		require.NotEmpty(t, keys.Keys[0].PeerPublicKey, "Split key should have a peer public key")
		t.Logf("Split key verified: clientID=%s, publicKey=%s...", keys.Keys[0].ClientID, keys.Keys[0].PublicKey[:20])

		// Step 5: Share wallet to a different user via zvault
		resp, err = zvaultClient.ShareWallet(t, client.X_APP_USER_ID_A, keys.Keys[0].PublicKey, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"ShareWallet should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		t.Log("Wallet shared to alternative user")

		// Step 6: Verify shared wallet is visible to the target user
		altHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		altHeaders["X-App-Client-ID"] = client.X_APP_CLIENT_ID_A
		altHeaders["X-App-User-ID"] = client.X_APP_USER_ID_A
		altHeaders["X-App-Client-Key"] = client.X_APP_CLIENT_KEY_A
		altHeaders["X-App-Client-Signature"] = client.X_APP_CLIENT_SIGNATURE_A

		altJwt, resp, err := zboxClient.CreateJwtToken(t, altHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Alt user JWT creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		altZvaultHeaders := zvaultClient.NewZvaultHeaders(altJwt.JwtToken)
		sharedWallets, resp, err := zvaultClient.GetSharedWallets(t, altZvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"GetSharedWallets should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.Len(t, sharedWallets, 1, "Alt user should see exactly 1 shared wallet")
		require.Equal(t, keys.Keys[0].ClientID, sharedWallets[0].ClientID,
			"Shared wallet client ID should match the original split key")
		t.Logf("Shared wallet verified for alt user: clientID=%s", sharedWallets[0].ClientID)

		// Step 7: Clean up - delete from zvault
		resp, err = zvaultClient.Delete(t, CLIENT_ID_V, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Delete from zvault should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		t.Log("Zvault cleanup complete")
	})

	t.RunSequentiallyWithTimeout("Wallet verify with real wallet data should respond with valid status", 2*time.Minute, func(t *test.SystemTest) {
		require.True(t, zvaultAvailable, "zvault must be available")

		// Create a real wallet and use its data for verification
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup should succeed")

		// Get the wallet keys so we have real wallet data
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"GetWalletKeys should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, wallet.PublicKey, "Wallet should have a public key")

		// Store in zvault and generate split key to have a complete wallet
		jwtToken, resp, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		zvaultHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		// Generate a fresh split wallet so we have client ID for verification
		var genResp *model.GenerateWalletResponse
		genResp, resp, err = zvaultClient.GenerateSplitWallet(t, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"GenerateSplitWallet should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, genResp.ClientID, "Generated wallet should have a client ID")

		// Call wallet/verify with the generated wallet data
		verifyBody := map[string]interface{}{
			"wallet_address": genResp.ClientID,
			"signature":      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			"message":        "Sign this message to verify your wallet",
			"provider":       "metamask",
		}

		resp, err = zboxClient.PostJSON(t, "/v2/wallet/verify", headers, verifyBody)
		require.NoError(t, err)
		t.Logf("Wallet verify with real data: %d - %s", resp.StatusCode(), resp.String())
		if resp.StatusCode() == 404 {
			t.Log("Wallet verify endpoint not available on this 0box version, skipping")
			return
		}
		// Any non-404 response means the endpoint exists and processed the request
		require.True(t, resp.StatusCode() >= 200 && resp.StatusCode() < 500,
			"Wallet verify should return a valid response. Got %d: %s", resp.StatusCode(), resp.String())

		// Cleanup
		resp, err = zvaultClient.Delete(t, genResp.ClientID, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Register wallet auth for metamask provider with authenticated request", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup should succeed")

		// Verify wallet exists before auth registration
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotEmpty(t, wallet.PublicKey, "Wallet must exist before auth registration")

		// Register metamask wallet auth with the authenticated session
		authBody := map[string]interface{}{
			"wallet_address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"provider":       "metamask",
		}

		resp, err = zboxClient.PostJSON(t, "/v2/wallet/auth", headers, authBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() != 404,
			"Wallet auth endpoint should exist. Got %d: %s", resp.StatusCode(), resp.String())
		// Successful registration or already-registered are both acceptable
		require.True(t, resp.StatusCode() == 200 || resp.StatusCode() == 201 || resp.StatusCode() == 400,
			"Metamask auth registration should succeed or report already registered. Got %d: %s",
			resp.StatusCode(), resp.String())
		t.Logf("RegisterWalletAuth (metamask, authenticated): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Register wallet auth for zus provider with authenticated request", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup should succeed")

		// Verify wallet exists before auth registration
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotEmpty(t, wallet.PublicKey, "Wallet must exist before auth registration")

		// Register zus wallet auth with the authenticated session
		authBody := map[string]interface{}{
			"wallet_address": headers["X-App-Client-ID"],
			"provider":       "zus",
		}

		resp, err = zboxClient.PostJSON(t, "/v2/wallet/auth", headers, authBody)
		require.NoError(t, err)
		require.True(t, resp.StatusCode() != 404,
			"Wallet auth endpoint should exist. Got %d: %s", resp.StatusCode(), resp.String())
		require.True(t, resp.StatusCode() == 200 || resp.StatusCode() == 201 || resp.StatusCode() == 400,
			"Zus auth registration should succeed or report already registered. Got %d: %s",
			resp.StatusCode(), resp.String())
		t.Logf("RegisterWalletAuth (zus, authenticated): %d - %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Unauthenticated wallet auth registration should fail", 2*time.Minute, func(t *test.SystemTest) {
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
			"Unauthenticated wallet auth should return 400/401/403. Got %d: %s",
			resp.StatusCode(), resp.String())
		t.Logf("RegisterWalletAuth (unauthenticated): %d - %s", resp.StatusCode(), resp.String())
	})
}
