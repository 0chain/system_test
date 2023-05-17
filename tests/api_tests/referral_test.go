package api_tests

import (
	"testing"

	// "github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0Box_referral(testSetup *testing.T) {

	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Post referrals with correct CSRF should work properly")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Post referrals with correct CSRF should work properly", func(t *test.SystemTest) {

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

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		zboxRferral, response, err := zboxClient.GetReferralCode(t,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Len(t, zboxRferral.ReferrerCode, 14)

	})

	t.RunSequentially("GET referral count with correct CSRF should work properly before inserting referral", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		zboxRferral, response, err := zboxClient.GetReferralCount(t,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(0), zboxRferral.ReferralCount, "referral count should be 0 initially")
	})

	t.TestSetup("Autenticate with firebase", func() {
		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
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

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxRferral, response, err := zboxClient.GetReferralCode(t,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		description = "wallet created as part of " + t.Name()
		walletName = "wallet_name"
		firebaseToken = authenticateWithFirebase(t, "+91742873093")

		zboxWallet, response, err = zboxClient.PostWalletWithReferralCode(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			"+91742873093",
			"blimp",
			zboxRferral.ReferrerCode,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		zboxRferrals, responses, errs := zboxClient.GetReferralCount(t,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, errs)
		require.NotNil(t, zboxRferral)

		require.Equal(t, 200, responses.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(1), zboxRferrals.ReferralCount, "referral count should be 1 after inserting the referral")

	})

}
