package api_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxReferral(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	// NOT parallel: shares identity with other 0box tests; concurrent Teardown deletes wallet rows.
	t.SetSmokeTests("Post referrals with correct CSRF should work properly")

	t.RunSequentially("Get referral code with owner should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		zboxReferral, response, err := zboxClient.GetReferralCode(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxReferral)
		require.Len(t, zboxReferral.ReferrerCode, 14, "length of referral code should be 14")
	})

	t.RunSequentially("Rank referrals with no referrer should work properly", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		zboxRferral, response, err := zboxClient.GetReferralRank(t, headers)
		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(0), zboxRferral.UserCount, "referral count should be 0 initially")
		require.Equal(t, int64(0), zboxRferral.UserRank, "referral rank should be 0 initially")
	})

	t.RunSequentially("Create wallet for first time with the referral code should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		referralHeaders := zboxClient.NewZboxHeaders_RWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, referralHeaders)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		zboxRferral, response, err := zboxClient.GetReferralCode(t, headers)
		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Clear the shared HTTP client's cookies to isolate the _R user's session
		// from the primary user's authenticated session, then get fresh CSRF token.
		zboxClient.ClearCookies()
		referralHeaders = zboxClient.NewZboxHeaders_RWithCSRF(t, client.X_APP_BLIMP)

		verifyOtpInput := NewVerifyOtpDetails()
		verifyOtpInput["user_id"] = client.X_APP_USER_ID_R
		verifyOtpInput["firebase_token"] = client.X_APP_ID_TOKEN_R
		verifyOtpInput["username"] = fmt.Sprintf("ref_user_%d", time.Now().UnixNano())
		// Use the _R Firebase email if available; fall back to a generated email
		// so the required Email field is never empty (0box rejects empty Email).
		refEmail := client.GetFirebaseEmail_R()
		if refEmail == "" {
			refEmail = fmt.Sprintf("ref_user_%d@test.com", time.Now().UnixNano())
		}
		verifyOtpInput["email"] = refEmail
		verifyOtpInput["phone_number"] = fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
		_, response, err = zboxClient.VerifyOtpDetails(t, referralHeaders, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxWallet, response, err := zboxClient.CreateWallet(t, referralHeaders, map[string]string{
			"name":    "referred_wallet",
			"refcode": zboxRferral.ReferrerCode,
		})
		require.NoError(t, err)
		require.NotNil(t, zboxWallet)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}

func Test0BoxReferralLeaderBoard(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Testing LeaderBoard")

	t.RunSequentially("Testing LeaderBoard", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		referralHeaders := zboxClient.NewZboxHeaders_RWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, referralHeaders)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		zboxRferral, response, err := zboxClient.GetReferralCode(t, headers)
		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Clear the shared HTTP client's cookies to isolate the _R user's session
		// from the primary user's authenticated session, then get fresh CSRF token.
		zboxClient.ClearCookies()
		referralHeaders = zboxClient.NewZboxHeaders_RWithCSRF(t, client.X_APP_BLIMP)

		verifyOtpInput := NewVerifyOtpDetails()
		verifyOtpInput["user_id"] = client.X_APP_USER_ID_R
		verifyOtpInput["firebase_token"] = client.X_APP_ID_TOKEN_R
		verifyOtpInput["username"] = fmt.Sprintf("ref_user_%d", time.Now().UnixNano())
		// Use the _R Firebase email if available; fall back to a generated email
		// so the required Email field is never empty (0box rejects empty Email).
		refEmail := client.GetFirebaseEmail_R()
		if refEmail == "" {
			refEmail = fmt.Sprintf("ref_user_%d@test.com", time.Now().UnixNano())
		}
		verifyOtpInput["email"] = refEmail
		verifyOtpInput["phone_number"] = fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
		_, response, err = zboxClient.VerifyOtpDetails(t, referralHeaders, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxWallet, response, err := zboxClient.CreateWallet(t, referralHeaders, map[string]string{
			"name":    "referred_wallet",
			"refcode": zboxRferral.ReferrerCode,
		})
		require.NoError(t, err)
		require.NotNil(t, zboxWallet)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Clear cookies again to switch back to primary user's session for leaderboard check
		zboxClient.ClearCookies()
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		referralLeaderBoard, response, err := zboxClient.GetLeaderBoard(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 1, len(referralLeaderBoard.TopUsers))
		require.Equal(t, 1, referralLeaderBoard.TopUsers[0].Count)
	})
}
