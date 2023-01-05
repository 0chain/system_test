package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test0Box(testSetup *testing.T) {
	//todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Create a wallet with valid phone number should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		csrfToken, response, err := zboxClient.CreateCSRFToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		zboxWallet, response, err := zboxClient.PostWallet(t,
			"613ed9fb5b9311f6f22080eb1db69b2e786c990706c160faf1f9bdd324fd909bc640ad6a3a44cb4248ddcd92cc1fabf66a69ac4eb38a102b984b98becb0674db7d69c5727579d5f756bb8c333010866d4d871dae1b7032d6140db897e4349f60f94f1eb14a3b7a14a489226a1f35952472c9b2b13e3698523a8be2dcba91c344f55da17c21c403543d82fe5a32cb0c8133759ab67c31f1405163a2a255ec270b1cca40d9f236e007a3ba8f6be4eaeaad10376c5f224bad45c597d85a3b8b984f46c597f6cf561405bd0b0007ac6833cfff408aeb51c0d2fX",
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		//require.Equal(t, description, zboxWallet.Description, "Description does not match expected") //FIXME: Description is not persisted
	})

	t.RunSequentially("List wallet should work with zero wallets", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		csrfToken, response, err := zboxClient.CreateCSRFToken(t, zboxClient.DefaultPhoneNumber)

		wallets, response, err := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, wallets)
		require.Equal(t, 0, len(wallets.Data), "More wallets present than expected")
	})

	t.RunSequentially("List wallet should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		csrfToken, response, err := zboxClient.CreateCSRFToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err = zboxClient.PostWallet(t,
			"613ed9fb5b9311f6f22080eb1db69b2e786c990706c160faf1f9bdd324fd909bc640ad6a3a44cb4248ddcd92cc1fabf66a69ac4eb38a102b984b98becb0674db7d69c5727579d5f756bb8c333010866d4d871dae1b7032d6140db897e4349f60f94f1eb14a3b7a14a489226a1f35952472c9b2b13e3698523a8be2dcba91c344f55da17c21c403543d82fe5a32cb0c8133759ab67c31f1405163a2a255ec270b1cca40d9f236e007a3ba8f6be4eaeaad10376c5f224bad45c597d85a3b8b984f46c597f6cf561405bd0b0007ac6833cfff408aeb51c0d2fX",
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		wallets, response, err := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, wallets)
		require.Equal(t, 1, len(wallets.Data), "Expected 1 wallet only to be present")
	})

	//FIXME: Missing field description does not match field name (Pascal case instead of snake case)
	// [{ClientID  required } {PublicKey  required } {Timestamp  required } {TokenInput  required } {AppType  required } {PhoneNumber  required }]
}

func teardown(t *test.SystemTest, idToken, phoneNumber string) {
	t.Logf("Tearing down existing test data for [%v]", phoneNumber)
	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	//wallets, _, _ := zboxClient.ListWallets(t, idToken, csrfToken, phoneNumber) //FIXME: list wallets endpoint returns the client id in place of the wallet id, and does not return the wallet id at all so we need to use list keys instead
	wallets, _, _ := zboxClient.ListWalletKeys(t, idToken, csrfToken, phoneNumber)

	if wallets != nil {
		t.Logf("Found [%v] existing wallets for [%v]", len(wallets), phoneNumber)
		for _, wallet := range wallets {
			message, response, err := zboxClient.DeleteWallet(t, wallet.WalletId, idToken, csrfToken, phoneNumber)
			println(message, response, err)
		}
	} else {
		t.Logf("No wallets found for [%v] teardown", phoneNumber)
	}
}

func createCsrfToken(t *test.SystemTest, phoneNumber string) string {
	csrfToken, response, err := zboxClient.CreateCSRFToken(t, phoneNumber)
	require.NoError(t, err, "CSRF token creation failed with output: %v and error %v ", response, err)

	require.NotNil(t, csrfToken, "CSRF token was nil!")
	require.NotNil(t, csrfToken, "id token was nil!")

	return csrfToken
}

func authenticateWithFirebase(t *test.SystemTest, phoneNumber string) *model.FirebaseToken {
	session, response, err := zboxClient.FirebaseSendSms(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", phoneNumber)
	require.NoError(t, err, "Firebase send SMS failed: ", response.RawResponse)
	token, response, err := zboxClient.FirebaseCreateToken(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", session.SessionInfo)
	require.NoError(t, err, "Firebase create token failed: ", response.RawResponse)

	return token
}
