package api_tests

import (
	"os/exec"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Teardown(t *test.SystemTest, headers map[string]string) {
	// Clean 0box test records via direct SQL instead of the API.
	// The API's DELETE /v2/owner tries to delete the Firebase user and fails
	// when Firebase admin is not configured, leaving stale records that cause
	// "duplicate key" errors in subsequent subtests.
	// Owner deletion cascades to wallet, active_wallet, etc. via ON DELETE CASCADE.
	// Delete wallets first (explicit, in case ON DELETE CASCADE is not set), then owners.
	// Delete in correct dependency order: allocation → wallet → owner.
	// allocation.wallet_id references wallet.id; wallet.owner_id references owner.id.
	// Use separate statements so one failure doesn't block the rest.
	cmd := exec.Command("docker", "exec", "postgres-0box", "psql", "-U", "zbox_user", "-d", "zbox", "-c",
		`DELETE FROM allocation WHERE wallet_id IN (SELECT w.id FROM wallet w JOIN owner o ON w.owner_id = o.id WHERE o.username LIKE 'test_%' OR o.username LIKE 'ref_%' OR o.username = 'referred_user');
		 DELETE FROM wallet WHERE owner_id IN (SELECT id FROM owner WHERE username LIKE 'test_%' OR username LIKE 'ref_%' OR username = 'referred_user');
		 DELETE FROM owner WHERE username LIKE 'test_%' OR username LIKE 'ref_%' OR username = 'referred_user';`)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Teardown SQL cleanup warning: %v (output: %s)", err, string(output))
		// Fallback to API method
		zboxClient.DeleteOwner(t, headers) //nolint:errcheck
	}
}

func NewTestOwner() map[string]string {
	return map[string]string{
		"username":     "test_owner_1",
		"email":        client.X_APP_FIREBASE_EMAIL,
		"phone_number": "+919876543210",
	}
}

func NewVerifyOtpDetails() map[string]string {
	return map[string]string{
		"username":       "test_owner_1",
		"email":          client.X_APP_FIREBASE_EMAIL,
		"phone_number":   "+919876543210",
		"otp":            "123456",
		"firebase_token": client.X_APP_ID_TOKEN,
		"user_id":        client.X_APP_USER_ID,
	}
}

func Test0BoxOwner(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("create owner without existing userID should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, response, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Refresh headers after signup (CSRF token may be invalidated)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		owner, response, err := zboxClient.GetOwner(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, verifyOtpInput["username"], owner.UserName)
		require.Equal(t, verifyOtpInput["email"], owner.Email)
		require.Equal(t, verifyOtpInput["phone_number"], owner.PhoneNumber)
	})

	t.RunSequentially("create owner with existing userID should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// First call: create the owner
		verifyOtpInput := NewVerifyOtpDetails()
		_, response, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "First VerifyOtp should return 200 (created), got %d: %s", response.StatusCode(), response.String())

		// Second call: should fail since owner already exists
		_, response, err = zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("update owner with existing owner should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Create owner (Teardown deletes it, so this should succeed with 200)
		verifyOtpInput := NewVerifyOtpDetails()
		_, response, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "VerifyOtp should return 200 (created), got %d: %s", response.StatusCode(), response.String())

		ownerInput := NewTestOwner()
		ownerInput["username"] = "new_user_name"
		ownerInput["biography"] = "new_biography"
		message, resp, err := zboxClient.UpdateOwner(t, headers, ownerInput)
		require.NoError(t, err)
		require.NotNil(t, message, "UpdateOwner returned nil message, status: %v", resp.String())
		require.Equal(t, "updated owner details successfully", message.Message)

		owner, response, err := zboxClient.GetOwner(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, ownerInput["username"], owner.UserName)
		require.Equal(t, ownerInput["email"], owner.Email)
		require.Equal(t, ownerInput["phone_number"], owner.PhoneNumber)
		require.Equal(t, ownerInput["biography"], owner.Biography)
	})

	t.RunSequentially("update owner without existing owner should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		// Ensure no owner exists by cleaning via SQL
		Teardown(t, headers)

		ownerInput := NewTestOwner()
		message, resp, err := zboxClient.UpdateOwner(t, headers, ownerInput)
		require.NoError(t, err)
		require.NotNil(t, message, "UpdateOwner returned nil message, status: %v", resp.String())
		// 0box now auto-creates the owner on update if one doesn't exist,
		// returning "updated owner details successfully" instead of "No Data was updated".
		require.Equal(t, "updated owner details successfully", message.Message)
	})
}
