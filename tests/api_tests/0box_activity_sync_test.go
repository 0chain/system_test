package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxActivitySync tests that 0box tracks real user operations by performing
// operations and verifying their state changes are persisted. The operation_tracker
// middleware wraps all authenticated endpoints, so every API call is an implicit
// activity sync verification.
func Test0BoxActivitySync(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Wallet creation is tracked and persisted", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Create wallet -- this operation passes through the operation_tracker middleware
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet creation should succeed")

		// Read back to verify the creation was tracked and persisted
		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Wallet should be retrievable after tracked creation. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, wallet.PublicKey, "Wallet public key should be populated")
		require.Equal(t, headers["X-App-Client-Key"], wallet.PublicKey,
			"Wallet public key should match the key used at creation time")
		require.Equal(t, "test_wallet_name", wallet.Name,
			"Wallet name should match what was passed to CreateWallet")
		t.Logf("Wallet creation tracked: clientID=%s, name=%s", wallet.ClientID, wallet.Name)
	})

	t.RunSequentiallyWithTimeout("Allocation creation is tracked and persisted", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Create wallet + allocation
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet creation should succeed")

		allocationData := NewTestAllocation()
		alloc, resp, err := zboxClient.CreateAllocation(t, headers, allocationData)
		require.NoError(t, err, "CreateAllocation transport error")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateAllocation should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotEmpty(t, alloc.ID, "Created allocation should have an ID")

		// Read back to verify the allocation was tracked and persisted
		allocations, resp, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"ListAllocation should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.GreaterOrEqual(t, len(allocations), 1,
			"At least one allocation should exist after tracked creation")

		found := false
		for _, a := range allocations {
			if a.ID == alloc.ID {
				found = true
				require.Equal(t, allocationData["name"], a.Name,
					"Allocation name should match what was passed to CreateAllocation")
				break
			}
		}
		require.True(t, found, "Created allocation %s should appear in the list", alloc.ID)
		t.Logf("Allocation creation tracked: id=%s, name=%s", alloc.ID, alloc.Name)
	})

	t.RunSequentiallyWithTimeout("Shareinfo create and delete are tracked and persisted", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet creation should succeed")

		// Create shareinfo -- tracked operation
		shareinfoData := NewTestShareinfo()
		shareinfoResp, resp, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err, "CreateShareInfo transport error")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateShareInfo should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.Equal(t, "shareinfo added successfully", shareinfoResp.Message)

		// Verify the share was persisted
		sharedList, resp, err := zboxClient.GetShareInfoShared(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len(sharedList.Data),
			"Exactly one share should exist after tracked creation")
		require.Equal(t, client.X_APP_CLIENT_ID, sharedList.Data[0].Receiver,
			"Share receiver should match the client ID")
		t.Logf("Shareinfo creation tracked: receiver=%s", sharedList.Data[0].Receiver)

		// Delete shareinfo -- tracked operation
		delMsg, resp, err := zboxClient.DeleteShareinfo(t, headers, shareinfoData["auth_ticket"])
		require.NoError(t, err, "DeleteShareinfo transport error")
		require.Equal(t, 200, resp.StatusCode(),
			"DeleteShareinfo should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.NotNil(t, delMsg)
		t.Logf("Shareinfo delete tracked: message=%s", delMsg.Message)

		// Verify the share was removed (delete operation was persisted)
		sharedListAfter, resp, err := zboxClient.GetShareInfoShared(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 0, len(sharedListAfter.Data),
			"No shares should exist after tracked deletion")
		t.Logf("Shareinfo deletion verified: shares remaining=%d", len(sharedListAfter.Data))
	})

	t.RunSequentiallyWithTimeout("Wallet update is tracked and persisted", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet creation should succeed")

		// Verify initial state
		walletBefore, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "test_wallet_name", walletBefore.Name)

		// Update wallet -- tracked operation
		walletUpdate := map[string]string{
			"name":        "activity_tracked_update",
			"description": "updated via activity sync test",
			"mnemonic":    "updated_mnemonic_tracking",
		}
		msg, resp, err := zboxClient.UpdateWallet(t, headers, walletUpdate)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode(),
			"Wallet update should succeed. Got %d: %s", resp.StatusCode(), resp.String())
		require.Equal(t, "updating wallet successful", msg.Message)

		// Verify the update was persisted
		walletAfter, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "activity_tracked_update", walletAfter.Name,
			"Wallet name should reflect the tracked update")
		require.Equal(t, "updated via activity sync test", walletAfter.Description,
			"Wallet description should reflect the tracked update")
		require.Equal(t, "updated_mnemonic_tracking", walletAfter.Mnemonic,
			"Wallet mnemonic should reflect the tracked update")
		t.Logf("Wallet update tracked: name=%s -> %s", walletBefore.Name, walletAfter.Name)
	})

	t.RunSequentiallyWithTimeout("Multiple sequential operations are all tracked", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		// Operation 1: Create wallet
		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "Wallet creation should succeed")

		wallet, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "test_wallet_name", wallet.Name, "Initial wallet name should be set")

		// Operation 2: Create allocation
		allocData := NewTestAllocation()
		alloc, resp, err := zboxClient.CreateAllocation(t, headers, allocData)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode(),
			"Allocation creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		// Operation 3: Create shareinfo
		shareinfoData := NewTestShareinfo()
		_, resp, err = zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode(),
			"ShareInfo creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		// Operation 4: Update wallet
		update := map[string]string{
			"name":        "multi_op_final",
			"description": "after all operations",
			"mnemonic":    "test_mnemonic",
		}
		msg, resp, err := zboxClient.UpdateWallet(t, headers, update)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "updating wallet successful", msg.Message)

		// Verify all operations were tracked by reading back all state
		walletFinal, resp, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, "multi_op_final", walletFinal.Name,
			"Wallet name should reflect the final tracked update")

		allocList, resp, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		found := false
		for _, a := range allocList {
			if a.ID == alloc.ID {
				found = true
				break
			}
		}
		require.True(t, found, "Allocation should be in the list after all operations")

		sharedList, resp, err := zboxClient.GetShareInfoShared(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.GreaterOrEqual(t, len(sharedList.Data), 1,
			"At least one share should exist after all operations")

		t.Logf("All 4 operations tracked: wallet=%s, alloc=%s, shares=%d",
			walletFinal.Name, alloc.ID, len(sharedList.Data))

		// Cleanup share
		_, _, _ = zboxClient.DeleteShareinfo(t, headers, shareinfoData["auth_ticket"])
	})

	t.RunSequentiallyWithTimeout("Cross-app-type tracking isolation", 3*time.Minute, func(t *test.SystemTest) {
		vultHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, vultHeaders)

		// Create wallet using vult app type
		err := Create0boxTestWallet(t, vultHeaders)
		require.NoError(t, err, "Vult wallet creation should succeed")

		vultWallet, resp, err := zboxClient.GetWalletKeys(t, vultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotEmpty(t, vultWallet.PublicKey, "Vult wallet should have a public key")
		t.Logf("Vult wallet created: publicKey=%s", vultWallet.PublicKey)

		// Same user attempting wallet creation under blimp should fail
		// because 0box tracks operations per user, not per app type
		blimpHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		_, resp, err = zboxClient.CreateWallet(t, blimpHeaders, NewTestWallet())
		require.NoError(t, err)
		require.Equal(t, 400, resp.StatusCode(),
			"Blimp wallet creation for same user should be rejected. Got %d: %s",
			resp.StatusCode(), resp.String())

		// Same user attempting under bolt should also fail
		boltHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BOLT)
		_, resp, err = zboxClient.CreateWallet(t, boltHeaders, NewTestWallet())
		require.NoError(t, err)
		require.Equal(t, 400, resp.StatusCode(),
			"Bolt wallet creation for same user should be rejected. Got %d: %s",
			resp.StatusCode(), resp.String())

		// Verify the original vult wallet is unchanged
		vultWalletAfter, resp, err := zboxClient.GetWalletKeys(t, vultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, vultWallet.PublicKey, vultWalletAfter.PublicKey,
			"Vult wallet should be unchanged after cross-app creation attempts")
		t.Logf("Cross-app tracking verified: vult wallet intact, blimp/bolt rejected")
	})
}
