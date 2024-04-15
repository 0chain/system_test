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

	var firebaseTokens []*model.FirebaseToken

	t.TestSetup("Autenticate with firebase", func() {
		firebaseTokens = append(firebaseTokens, authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber))
	})

	t.RunSequentially("Get referral code with correct CSRF and private auth should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseTokens[0].IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseTokens[0].IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", "userName")
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, "userName", zboxOwner.UserName, "owner name does not match expected")

		zboxRferral, response, err := zboxClient.GetReferralCode(t,
			csrfToken,
			firebaseTokens[0].IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Len(t, zboxRferral.ReferrerCode, 14, "length of referral code should be 14")
	})

	t.RunSequentially("Rank referrals with correct CSRF should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseTokens[0].IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseTokens[0].IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", "userName")
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, "userName", zboxOwner.UserName, "owner name does not match expected")

		zboxRferral, response, err := zboxClient.GetReferralRank(t,
			csrfToken,
			firebaseTokens[0].IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, int64(0), zboxRferral.UserCount, "referral count should be 0 initially")
		require.Equal(t, int64(0), zboxRferral.UserRank, "referral rank should be 0 initially")
	})

	t.RunSequentially("Create wallet for first time with the referral code should work", func(t *test.SystemTest) {
		teardown(t, firebaseTokens[0].IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		firebaseTokens[0] = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseTokens[0].IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseTokens[0].IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NotNil(t, zboxWallet)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxRferral, response, err := zboxClient.GetReferralCode(t,
			csrfToken,
			firebaseTokens[0].IdToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		description = "wallet created as part of " + t.Name()
		walletName = "wallet_name1"
		firebaseTokens = append(firebaseTokens, authenticateWithFirebase(t, "+919876543210"))
		referralMnemonic := "total today fortune output enjoy season desert tool transfer awkward post disease junk offer wedding wire brown broccoli size banana harsh stove raise skull"

		userName2 := "user_name2"

		zboxOwner, response, err = zboxClient.PostOwnerWithReferralCode(t, firebaseTokens[1].IdToken, csrfToken, "+919876543210", "blimp", userName2)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName2, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err = zboxClient.PostWalletWithReferralCode(t,
			referralMnemonic,
			walletName,
			description,
			firebaseTokens[1].IdToken,
			csrfToken,
			"+919876543210",
			"blimp",
			userName2,
			zboxRferral.ReferrerCode,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		zboxRferrals, responses, errs := zboxClient.GetReferralCount(t,
			csrfToken,
			firebaseTokens[0].IdToken,
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
	t.SetSmokeTests("Testing LeaderBoard")

	var firebaseTokens []*model.FirebaseToken

	t.TestSetup("Autenticate with firebase", func() {
		firebaseTokens = append(firebaseTokens, authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber), authenticateWithFirebase(t, "+919876543210"))
	})

	t.RunSequentially("Testing LeaderBoard", func(t *test.SystemTest) {
		teardown(t, firebaseTokens[0].IdToken, zboxClient.DefaultPhoneNumber)
		teardown(t, firebaseTokens[1].IdToken, "+919876543210")

		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseTokens[0].IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseTokens[0].IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)
		require.NotNil(t, zboxWallet)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxRferral, response, err := zboxClient.GetReferralCode(t,
			csrfToken,
			firebaseTokens[0].IdToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.NotNil(t, zboxRferral)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		description = "wallet created as part of " + t.Name()
		walletName = "wallet_name1"
		referralMnemonic := "total today fortune output enjoy season desert tool transfer awkward post disease junk offer wedding wire brown broccoli size banana harsh stove raise skull"
		userName2 := "user_name2"

		zboxOwner, response, err = zboxClient.PostOwnerWithReferralCode(t, firebaseTokens[1].IdToken, csrfToken, "+919876543210", "blimp", userName2)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName2, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err = zboxClient.PostWalletWithReferralCode(t,
			referralMnemonic,
			walletName,
			description,
			firebaseTokens[1].IdToken,
			csrfToken,
			"+919876543210",
			"blimp",
			userName2,
			zboxRferral.ReferrerCode,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		zboxRferrals, responses, errs := zboxClient.GetLeaderBoard(t,
			csrfToken,
			firebaseTokens[0].IdToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, errs)
		require.NotNil(t, zboxRferral)

		require.Equal(t, 200, responses.StatusCode(), "Failed to get LeaderBoard. Output: [%v]", responses.String())
		require.NotNil(t, zboxRferral)
		require.Equal(t, 1, zboxRferrals.TopUsers[0].Count, "User Score should be 1 Initially")
	})
}
