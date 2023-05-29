package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxReferral(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Post referrals with correct CSRF should work properly")

	t.TestSetup("Autenticate with firebase", func() {
		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	})

	t.RunSequentially("Get referral code with correct CSRF and private auth should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		zboxRferral, response, err := zboxClient.GetReferralCode(t,
			csrfToken,
			firebaseToken.IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Len(t, zboxRferral.ReferrerCode, 14, "length of referral code should be 14")
	})

	t.RunSequentially("Rank referrals with correct CSRF should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		zboxRferral, response, err := zboxClient.GetReferralRank(t,
			csrfToken,
			firebaseToken.IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(0), zboxRferral.UserScore, "referral count should be 0 initially")
		require.Equal(t, int64(0), zboxRferral.UserRank, "referral rank should be 0 initially")
		require.Equal(t, zboxClient.DefaultPhoneNumber, zboxRferral.UserPhone, "phone should be same")
	})

	t.RunSequentially("Create wallet for first time with the referral code should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NotNil(t, zboxWallet)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxRferral, response, err := zboxClient.GetReferralCode(t,
			csrfToken,
			firebaseToken.IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)

		description = "wallet created as part of " + t.Name()
		walletName = "wallet_name1"
		firebaseToken = authenticateWithFirebase(t, "+919876543210")
		referralMnemonic := "total today fortune output enjoy season desert tool transfer awkward post disease junk offer wedding wire brown broccoli size banana harsh stove raise skull"

		zboxWallet, response, err = zboxClient.PostWalletWithReferralCode(t,
			referralMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			"+919876543210",
			"blimp",
			zboxRferral.ReferrerCode,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		zboxRferrals, responses, errs := zboxClient.GetReferralCount(t,
			csrfToken,
			firebaseToken.IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, errs)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, responses.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(1), zboxRferrals.ReferralCount, "referral count should be 1 after inserting the referral")
	})
}
