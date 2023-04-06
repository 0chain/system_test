package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxAllocation(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
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
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
			zboxClient.DefaultAppType,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
			"blimp",
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
			zboxClient.DefaultAppType,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationId := "allocation id created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		allocationList, response, err := zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 1, "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, allocationId, allocationList[0].Id)
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
			zboxClient.DefaultAppType,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation created as part of " + t.Name()
		allocation_id := "allocation ID created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocation_id,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		_, response, err = zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, "1234567890")

		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, `{"error":"auth_failed: Authentication failed."}`, response.String())
	})
}
