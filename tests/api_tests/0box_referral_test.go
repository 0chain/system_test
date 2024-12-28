package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxReferral(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Post referrals with correct CSRF should work properly")

	t.RunSequentially("Get referral code with owner should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		zboxReferral, response, err := zboxClient.GetReferralCode(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxReferral)
		require.Len(t, zboxReferral.ReferrerCode, 14, "length of referral code should be 14")
	})

	t.RunSequentially("Rank referrals with no referrer should work properly", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		zboxRferral, response, err := zboxClient.GetReferralRank(t, headers)
		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(0), zboxRferral.UserCount, "referral count should be 0 initially")
		require.Equal(t, int64(0), zboxRferral.UserRank, "referral rank should be 0 initially")
	})

	t.RunSequentially("Create wallet for first time with the referral code should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)
		referralHeaders := zboxClient.NewZboxHeaders_R(client.X_APP_BLIMP)
		Teardown(t, referralHeaders)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		zboxRferral, response, err := zboxClient.GetReferralCode(t, headers)
		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		verifyOtpInput := NewVerifyOtpDetails()
		verifyOtpInput["user_id"] = client.X_APP_USER_ID_R
		verifyOtpInput["username"] = "referred_user"
		verifyOtpInput["email"] = "dbiecougwbfvcsoo@gmail.com"
		verifyOtpInput["phone_number"] = "+15446424343"
		_, _, err = zboxClient.VerifyOtpDetails(t, referralHeaders, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxWallet, response, err := zboxClient.CreateWallet(t, referralHeaders, map[string]string{
			"name":    "referred_wallet",
			"refcode": zboxRferral.ReferrerCode,
		})
		require.NotNil(t, zboxWallet)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}

func Test0BoxReferralLeaderBoard(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Testing LeaderBoard")

	t.RunSequentially("Testing LeaderBoard", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)
		referralHeaders := zboxClient.NewZboxHeaders_R(client.X_APP_BLIMP)
		Teardown(t, referralHeaders)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		zboxRferral, response, err := zboxClient.GetReferralCode(t, headers)
		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		verifyOtpInput := NewVerifyOtpDetails()
		verifyOtpInput["user_id"] = client.X_APP_USER_ID_R
		verifyOtpInput["username"] = "referred_user"
		verifyOtpInput["email"] = "dbiecougwbfvcsoo@gmail.com"
		verifyOtpInput["phone_number"] = "+15446424343"
		_, response, err = zboxClient.VerifyOtpDetails(t, referralHeaders, verifyOtpInput)
		require.NoError(t, err)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxWallet, response, err := zboxClient.CreateWallet(t, referralHeaders, map[string]string{
			"name":    "referred_wallet",
			"refcode": zboxRferral.ReferrerCode,
		})
		require.NotNil(t, zboxWallet)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		referralLeaderBoard, response, err := zboxClient.GetLeaderBoard(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 1, len(referralLeaderBoard.TopUsers))
		require.Equal(t, 1, referralLeaderBoard.TopUsers[0].Count)
	})
}
