package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxActivitySync tests activity sync and operation tracking through its observable effects.
// WebSocket /ws/sync cannot be tested via HTTP, so we verify that operations performed through
// the 0box API are tracked and persisted correctly by checking their results.
func Test0BoxActivitySync(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Wallet creation operation is tracked and persisted", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Create a wallet - the 0box middleware tracks this as an operation
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet creation should succeed")

		// Verify the creation operation was tracked: the wallet must exist and be retrievable
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Wallet should be retrievable after creation. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, wallet.PublicKey, "Wallet public key should be populated after tracked creation")
		require.Equal(t, headers["X-App-Client-Key"], wallet.PublicKey,
			"Wallet public key should match the client key used in creation")
		t.Logf("Wallet creation tracked: publicKey=%s, name=%s", wallet.PublicKey, wallet.Name)
	})

	t.RunSequentiallyWithTimeout("Wallet update operation is tracked and persisted", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Create wallet first
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet creation should succeed")

		// Perform a tracked update operation
		walletUpdate := map[string]string{
			"name":        "activity_tracked_wallet",
			"description": "wallet updated to verify activity tracking",
			"mnemonic":    "updated_mnemonic_for_tracking",
		}
		message, resp, err := zboxClient.UpdateWallet(t, headers, walletUpdate)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Wallet update should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.Equal(t, "updating wallet successful", message.Message)

		// Verify the update was persisted (confirms the operation was tracked end-to-end)
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "activity_tracked_wallet", wallet.Name,
			"Wallet name should reflect the tracked update operation")
		require.Equal(t, "wallet updated to verify activity tracking", wallet.Description,
			"Wallet description should reflect the tracked update operation")
		t.Logf("Wallet update tracked: name=%s, description=%s", wallet.Name, wallet.Description)
	})

	t.RunSequentiallyWithTimeout("Multiple sequential operations are all tracked", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Operation 1: Create wallet
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "Wallet creation should succeed")

		// Verify operation 1 was tracked
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "test_wallet_name", wallet.Name, "Initial wallet name should be set")

		// Operation 2: Update wallet name
		update1 := map[string]string{
			"name":        "first_update",
			"description": "first tracked update",
			"mnemonic":    "test_mnemonic",
		}
		msg, resp, err := zboxClient.UpdateWallet(t, headers, update1)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "updating wallet successful", msg.Message)

		// Verify operation 2 was tracked
		wallet, resp, err = zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "first_update", wallet.Name, "First update should be tracked")

		// Operation 3: Update wallet name again
		update2 := map[string]string{
			"name":        "second_update",
			"description": "second tracked update",
			"mnemonic":    "test_mnemonic",
		}
		msg, resp, err = zboxClient.UpdateWallet(t, headers, update2)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "updating wallet successful", msg.Message)

		// Verify operation 3 was tracked - final state reflects the last operation
		wallet, resp, err = zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "second_update", wallet.Name,
			"Final wallet name should reflect the last tracked operation")
		require.Equal(t, "second tracked update", wallet.Description,
			"Final description should reflect the last tracked operation")
		t.Logf("All 3 operations tracked: final name=%s", wallet.Name)
	})

	t.RunSequentiallyWithTimeout("Cross-app-type tracking: different app types see separate contexts", 3*time.Minute, func(t *test.SystemTest) {
		// Create wallet using vult app type
		vultHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, vultHeaders)

		err := Create0boxTestWallet(t, vultHeaders)
		require.NoError(t, err, "Vult wallet creation should succeed")

		// Verify vult wallet was tracked and persisted
		vultWallet, resp, err := zboxClient.GetWalletKeys(t, vultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Vult wallet should be retrievable. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, vultWallet.PublicKey, "Vult wallet should have a public key")
		t.Logf("Vult app wallet created and tracked: publicKey=%s", vultWallet.PublicKey)

		// Attempt to create wallet using blimp app type (same user, different app)
		// 0box tracks operations per app type. The same Firebase user already has a wallet
		// under vult, so creating under blimp for the same user tests cross-app tracking.
		blimpHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		_, resp, err = zboxClient.CreateWallet(t, blimpHeaders, NewTestWallet())
		require.NoError(t, err)
		// 0box rejects creating a second wallet for the same user under a different app type
		// because the user already has a wallet tracked under vult. This confirms app-type isolation.
		require.Equal(t, 400, resp.StatusCode(),
			"Blimp wallet creation for same user should be rejected (cross-app tracking). Got %d: %s",
			resp.StatusCode(), resp.String())
		t.Logf("Blimp app type tracked separately: creation rejected for existing user (status %d)", resp.StatusCode())

		// Verify bolt app type also sees the same cross-app tracking boundary
		boltHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BOLT)
		_, resp, err = zboxClient.CreateWallet(t, boltHeaders, NewTestWallet())
		require.NoError(t, err)
		require.Equal(t, 400, resp.StatusCode(),
			"Bolt wallet creation for same user should be rejected (cross-app tracking). Got %d: %s",
			resp.StatusCode(), resp.String())
		t.Logf("Bolt app type tracked separately: creation rejected for existing user (status %d)", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Zvault operations are tracked alongside 0box wallet", 3*time.Minute, func(t *test.SystemTest) {
		require.True(t, zvaultAvailable, "zvault must be available")

		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Operation 1: Create 0box wallet (tracked by 0box)
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet creation should succeed")

		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotEmpty(t, wallet.PublicKey)
		t.Logf("0box wallet created: publicKey=%s", wallet.PublicKey)

		// Operation 2: Get JWT and store in zvault (tracked by zvault)
		jwtToken, resp, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotEmpty(t, jwtToken.JwtToken)

		zvaultHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		resp, err = zvaultClient.Store(t, PRIVATE_KEY, MNEMONIC, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Zvault store should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		// Operation 3: Generate split key (tracked by zvault)
		resp, err = zvaultClient.GenerateSplitKey(t, CLIENT_ID_V, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Split key generation should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		// Verify both systems tracked their operations: 0box wallet + zvault keys
		wallet, resp, err = zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotEmpty(t, wallet.PublicKey, "0box wallet should still exist after zvault operations")

		keys, resp, err := zvaultClient.GetKeys(t, CLIENT_ID_V, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, keys.Keys, 1, "Zvault should have 1 split key from tracked operation")
		t.Logf("Both 0box and zvault operations tracked: wallet=%s, splitKeys=%d",
			wallet.PublicKey, len(keys.Keys))

		// Cleanup zvault
		resp, err = zvaultClient.Delete(t, CLIENT_ID_V, zvaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})
}
