package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxAllocation(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("List allocation with zero allocation should work")

	var firebaseToken *model.FirebaseToken
	t.TestSetup("Autenticate with firebase", func() {
		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	})

	t.RunSequentially("List allocation with zero allocation should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
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
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
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
		require.Equal(t, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("List allocation with existing allocation should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID1"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
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
		require.Equal(t, allocationID, allocationList[0].Id)
	})

	t.RunSequentially("Post allocation with correct argument should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationID := "allocationID2"
		allocationType := "direct_storage"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)
	})

	t.RunSequentially("Post allocation with correct argument for vult should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"vult",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID3"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"vult",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)
	})

	t.RunSequentially("Post multiple allocation for vult should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"vult",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation name"
		allocationDescription := "second allocation description created as part of " + t.Name()
		allocationType := "s3_server"
		allocation_id := "new allocation for vult"
		_, _, err = zboxClient.PostAllocation(t,
			allocation_id,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"vult",
		)
		require.NoError(t, err)
		_, response, err = zboxClient.PostAllocation(t,
			allocation_id,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"vult",
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, `{"error":"400: allocation already exists for appType: vult"}`, response.String())
	})

	t.RunSequentially("Post multiple allocation for blimp should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID4"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		allocationName = "second allocation"
		allocationDescription = "second allocation description created as part of " + t.Name()
		allocationType = "s3_server"
		allocation_id := "new allocation id for blimp"
		allocationObjCreatedResponse, response, err = zboxClient.PostAllocation(t,
			allocation_id,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)
	})

	t.RunSequentially("Post multiple allocation for chalk should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"chalk",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID5"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"chalk",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		allocationName = "second allocation"
		allocationDescription = "second allocation description created as part of " + t.Name()
		allocationType = "s3_server"
		allocation_id := "new allocation for chalk"
		allocationObjCreatedResponse, response, err = zboxClient.PostAllocation(t,
			allocation_id,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"chalk",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)
	})

	t.RunSequentially("Post allocation for chimney should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"chimney",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID6"
		_, response, err = zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"chimney",
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, `{"error":"400: allocation creation not allowed for appType: chimney"}`, response.String())
	})

	t.RunSequentially("Post allocation for bolt should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"bolt",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID7"
		_, response, err = zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"bolt",
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, `{"error":"400: allocation creation not allowed for appType: bolt"}`, response.String())
	})

	t.RunSequentially("Post allocation for invalid app type should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID8"
		_, response, err = zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"abc",
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Equal(t, `{"error":{"code":"invalid_header","msg":"invalid application type."}}`, response.String())
	})

	t.RunSequentially("Post allocation with already existing allocation Id should not  work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID9"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		_, response, err = zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Get an allocation with allocation present should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID10"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

		allocation, response, err := zboxClient.GetAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, allocationID, allocationName)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, allocationID, allocation.Id)
	})

	t.RunSequentially("Get an allocation with allocation not present should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		_, response, err = zboxClient.GetAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, zboxClient.DefaultAllocationId, allocationName)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Update an allocation with allocation present should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID11"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationID,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)
		updatedAllocationName := "update allocation name"
		allocationObjCreatedResponse, response, err = zboxClient.UpdateAllocation(t,
			allocationID,
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

		allocation, response, err := zboxClient.GetAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, allocationID, allocationName)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, allocationID, allocation.Id)
		require.Equal(t, updatedAllocationName, allocation.Name)
	})

	t.RunSequentially("Update an allocation with allocation not present should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAppType,
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationID := "allocationID12"

		updatedAllocationName := "update allocation name"
		allocationObjCreatedResponse, response, err := zboxClient.UpdateAllocation(t,
			allocationID,
			updatedAllocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "no allocation was updated for these details", allocationObjCreatedResponse.Message)
	})
}
