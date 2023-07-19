package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxReferral(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Post referrals with correct CSRF should work properly")

	var firebaseToken *model.FirebaseToken
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

		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

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

		require.Equal(t, 200, responses.StatusCode(), "Failed to get Referral Count. Output: [%v]", responses.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(1), zboxRferrals.ReferralCount, "referral count should be 1 after inserting the referral")
	})
}

func Test0BoxReferralLeaderBoard(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Post referrals with correct CSRF should work properly")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	t.TestSetup("Autenticate with firebase", func() {
		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	})

	t.RunSequentially("Testing LeaderBoard", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		teardown(t, firebaseToken.IdToken, "+919876543210")

		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

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
		teardown(t, firebaseToken.IdToken, "+919876543210")

		description = "wallet created as part of " + t.Name()
		walletName = "wallet_name1"

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

		zboxRferrals, responses, errs := zboxClient.GetLeaderBoard(t,
			csrfToken,
			firebaseToken.IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, errs)
		require.NotNil(t, zboxRferral)

		require.Equal(t, 200, responses.StatusCode(), "Failed to get LeaderBoard. Output: [%v]", responses.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(1), zboxRferrals.Total, "LeaderBoard should contain 1 User")
		require.Equal(t, int64(1), zboxRferrals.Users[0].Rank, "User Rank should be 1 Initially")
		require.Equal(t, int64(1), zboxRferrals.Users[0].Score, "User Score should be 1 Initially")
	})
}
