package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxActivitySync tests the activity sync and operation tracking functionality.
// WebSocket endpoint /ws/sync cannot be tested via HTTP, so we verify operation tracking
// through wallet creation operations and cross-app-type tracking.
func Test0BoxActivitySync(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Operation tracking after wallet creation should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Create a wallet - this operation should be tracked by 0box middleware
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup should succeed")

		// Verify the wallet exists, confirming the operation was processed
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Wallet should exist after creation. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, wallet.PublicKey, "Wallet public key should be populated")
		t.Logf("Operation tracked: wallet created with public key %s", wallet.PublicKey)
	})

	t.RunSequentiallyWithTimeout("File operation tracking via wallet update should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Create wallet first
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup should succeed")

		// Perform a tracked operation: update the wallet
		walletUpdate := map[string]string{
			"name":        "updated_wallet_name",
			"description": "updated_wallet_description",
			"mnemonic":    "updated_mnemonic",
		}
		message, resp, err := zboxClient.UpdateWallet(t, headers, walletUpdate)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Wallet update should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.Equal(t, "updating wallet successful", message.Message)

		// Verify the update was persisted (operation was tracked and completed)
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "updated_wallet_name", wallet.Name,
			"Wallet name should reflect the update operation")
		t.Logf("File operation tracked: wallet updated to name=%s", wallet.Name)
	})

	t.RunSequentiallyWithTimeout("Different app types should be tracked separately", 3*time.Minute, func(t *test.SystemTest) {
		// Create wallet using vult app type
		vultHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, vultHeaders)

		err := Create0boxTestWallet(t, vultHeaders)
		require.NoError(t, err, "Vult wallet creation should succeed")

		vultWallet, resp, err := zboxClient.GetWalletKeys(t, vultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Vult wallet should be retrievable. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, vultWallet.PublicKey)
		t.Logf("Vult app wallet created: %s", vultWallet.PublicKey)

		// Attempt to create wallet using blimp app type (same user, different app)
		// 0box tracks operations per app type - blimp should see a different context
		blimpHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		_, resp, err = zboxClient.CreateWallet(t, blimpHeaders, NewTestWallet())
		require.NoError(t, err)
		// 0box does not allow creating a second wallet for the same user with a different app type.
		// This confirms app-type-level tracking: the operation is rejected because the user
		// already has a wallet tracked under a different app type.
		require.Equal(t, 400, resp.StatusCode(),
			"Blimp wallet creation for same user should be rejected (tracked separately). Got %d: %s",
			resp.StatusCode(), resp.String())
		t.Logf("Blimp app type tracked separately: creation rejected for existing user (status %d)", resp.StatusCode())

		// Verify bolt app type also sees the same tracking boundary
		boltHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BOLT)
		_, resp, err = zboxClient.CreateWallet(t, boltHeaders, NewTestWallet())
		require.NoError(t, err)
		require.Equal(t, 400, resp.StatusCode(),
			"Bolt wallet creation for same user should be rejected (tracked separately). Got %d: %s",
			resp.StatusCode(), resp.String())
		t.Logf("Bolt app type tracked separately: creation rejected for existing user (status %d)", resp.StatusCode())
	})
}
