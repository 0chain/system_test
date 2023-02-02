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
		)

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

func Test0BoxAllocation(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	//t.SetTimeout(1 * time.Second)
	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	//FIXME some one broke list allocation previously it was working fine in pipeline
	t.RunSequentially("List allocation with zero allocation should work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationList, response, err := zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 0)
	})

	t.RunSequentially("Post allocation with invalid phonenumber should not work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		_, response, err = zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			"1234567890",
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("List allocation with existing allocation should work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		allocationList, response, err := zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 1, "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, zboxClient.DefaultAllocationId, allocationList[0].Id)
	})

	t.RunSequentially("List allocation with invalid phone number should not work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		_, response, err = zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, "1234567890")
		require.Error(t, err)
		//require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Post allocation with correct argument should work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)
	})

	t.RunSequentially("Post allocation with already existing allocation Id should not  work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		_, response, err = zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Get an allocation with allocation present should work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		allocation, response, err := zboxClient.GetAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, zboxClient.DefaultAllocationId, allocationName)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, zboxClient.DefaultAllocationId, allocation.Id)
	})

	t.RunSequentially("Get an allocation with allocation not present should not work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		_, response, err = zboxClient.GetAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, zboxClient.DefaultAllocationId, allocationName)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Update an allocation with allocation present should work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)
		updatedAllocationName := "update allocation name"
		allocationObjCreatedResponse, response, err = zboxClient.UpdateAllocation(t,
			zboxClient.DefaultAllocationId,
			updatedAllocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "updating allocation successful", allocationObjCreatedResponse.Message)

		allocation, response, err := zboxClient.GetAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, zboxClient.DefaultAllocationId, allocationName)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, zboxClient.DefaultAllocationId, allocation.Id)
		require.Equal(t, updatedAllocationName, allocation.Name)
	})

	t.RunSequentially("Update an allocation with allocation not present should not work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()

		updatedAllocationName := "update allocation name"
		allocationObjCreatedResponse, response, err := zboxClient.UpdateAllocation(t,
			zboxClient.DefaultAllocationId,
			updatedAllocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "updating allocation successful", allocationObjCreatedResponse.Message)
	})

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

	require.NotNil(t, csrfToken, "CSRF token container was nil!", response)
	require.NotNil(t, csrfToken.CSRFToken, "CSRF token was nil!", response)

	return csrfToken.CSRFToken
}

func authenticateWithFirebase(t *test.SystemTest, phoneNumber string) *model.FirebaseToken {
	session, response, err := zboxClient.FirebaseSendSms(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", phoneNumber)
	require.NoError(t, err, "Firebase send SMS failed: ", response.RawResponse)
	token, response, err := zboxClient.FirebaseCreateToken(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", session.SessionInfo)
	require.NoError(t, err, "Firebase create token failed: ", response.RawResponse)

	return token
}
