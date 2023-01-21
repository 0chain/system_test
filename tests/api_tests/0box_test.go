package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0Box(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Create a wallet with valid phone number should work", func(t *test.SystemTest) {
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
		) // github.com/0chain/system_test.git

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		// require.Equal(t, description, zboxWallet.Description, "Description does not match expected") // FIXME: Description is not persisted see: https://github.com/0chain/0box/issues/377
	})

	t.RunSequentially("List wallet should work with zero wallets", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		wallets, response, err := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, wallets)
		require.Equal(t, 0, len(wallets.Data), "More wallets present than expected")
	})

	t.RunSequentially("List wallet should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
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

	// FIXME: Missing field description does not match field name (Pascal case instead of snake case)
	// [{ClientID  required } {PublicKey  required } {Timestamp  required } {TokenInput  required } {AppType  required } {PhoneNumber  required }]
}

func teardown(t *test.SystemTest, idToken, phoneNumber string) {
	t.Logf("Tearing down existing test data for [%v]", phoneNumber)
	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	wallets, _, _ := zboxClient.ListWalletKeys(t, idToken, csrfToken, phoneNumber) // This endpoint used instead of list wallet as list wallet doesn't return the required data

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
