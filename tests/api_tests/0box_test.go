package api_tests

import (
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"
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

	t.RunSequentially("Get empty user info should not work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		userInfo, response, err := zboxClient.GetUserInfo(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, userInfo)
		require.Equal(t, "", userInfo.Username, "output not as expected", response.String())
		require.Equal(t, "", userInfo.Biography, "output not as expected", response.String()) //FIXME: should be null
		require.NotNil(t, userInfo.Avatar, "output not as expected", response.String())       //FIXME: should be null
		require.Equal(t, "", userInfo.Avatar.LargeLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.Avatar.MedLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.Avatar.SmallLoc, "output not as expected", response.String())
		require.NotNil(t, userInfo.BackgroundImage, "output not as expected", response.String()) //FIXME: should be null
		require.Equal(t, "", userInfo.BackgroundImage.LargeLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.BackgroundImage.MedLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.BackgroundImage.SmallLoc, "output not as expected", response.String())
		require.NotNil(t, userInfo.CreatedAt, "output not as expected", response.String()) // FIXME: weird that this is present on a blank object
	})

	t.RunSequentially("Create User Info Biography should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		success, response, err := zboxClient.PostUserInfoBiography(t, "bio from "+escapedTestName(t), firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, success)
		require.Equal(t, "biography saved", success.Success, "output not as expected", response.String())
	})

	t.RunSequentially("Create User Info Avatar should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		avatarImagePath := escapedTestName(t) + "avatar.png"
		generateImage(t, avatarImagePath)
		success, response, err := zboxClient.PostUserInfoAvatar(t, avatarImagePath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, success)
		require.Equal(t, "avatar saved", success.Success, "output not as expected", response.String())
	})

	t.RunSequentially("Create User Info background image should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		backgroundImagePath := escapedTestName(t) + "background.png"
		generateImage(t, backgroundImagePath)

		success, response, err := zboxClient.PostUserInfoBackgroundImage(t, backgroundImagePath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, success)
		require.Equal(t, "background image saved", success.Success, "output not as expected", response.String())
	})

	t.RunSequentially("Create User Info username should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		username := cliutils.RandomAlphaNumericString(10)

		usernameResponse, response, err := zboxClient.PutUsername(t, username, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, username)
		require.Equal(t, username, usernameResponse.Username, "output not as expected", response.String())
	})

	t.RunSequentially("Get fully populated user info from username should work", func(t *test.SystemTest) {
		t.Skip("skip till fixed")
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		username := cliutils.RandomAlphaNumericString(10)
		_, _, err := zboxClient.PutUsername(t, username, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		bio := "bio from " + escapedTestName(t)
		_, _, err = zboxClient.PostUserInfoBiography(t, bio, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		avatarImagePath := escapedTestName(t) + "avatar.png"
		generateImage(t, avatarImagePath)
		_, _, err = zboxClient.PostUserInfoAvatar(t, avatarImagePath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		thumbnailPath := escapedTestName(t) + "background.png"
		generateImage(t, thumbnailPath)
		_, _, err = zboxClient.PostUserInfoBackgroundImage(t, thumbnailPath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		userInfo, response, err := zboxClient.GetUserInfo(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, userInfo)
		require.Equal(t, username, userInfo.Username, "output not as expected", response.String())
		require.Equal(t, bio, userInfo.Biography, "output not as expected", response.String())
		require.NotNil(t, userInfo.Avatar, "output not as expected", response.String())
		require.NotNil(t, userInfo.CreatedAt, "output not as expected", response.String())
		require.NotNil(t, userInfo.BackgroundImage, "output not as expected", response.String())
	})
	// FIXME: Missing field description does not match field name (Pascal case instead of snake case)
	// [{ClientID  required } {PublicKey  required } {Timestamp  required } {TokenInput  required } {AppType  required } {PhoneNumber  required }]
}

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

		_, _, err = zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, "1234567890")
		require.Error(t, err)
		// require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		// I guess this a bug. Will discuss it and fix it
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

	t.Run("Test /v2/graph-write-price parameters", func(t *test.SystemTest) {
		// should fail for invalid parameters
		_, resp, err := zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: "AX", To: "20", DataPoints: "5" })
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid from param")

		_, resp, err = zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: "10", To: "AX", DataPoints: "5" })
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid to param")

		_, resp, err = zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: "10", To: "20", DataPoints: "AX" })
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid data-points query param")

		// should fail for invalid parameters (end - start < points + 1)
		_, resp, err = zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: "10000", To: "10010", DataPoints: "10" })
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "there must be at least one interval")

		// should fail for invalid parameters (end < start)
		_, resp, err = zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: "10000", To: "1000", DataPoints: "10" })
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "to 1000 less than from 10000")

		// should succeed in case of 1 point
		data, resp, err := zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ DataPoints: "1" })
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		require.Greater(t, (*data)[0], int64(0))

		// should succeed in case of multiple points
		minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
		require.NoError(t, err)
		latestRound := minerStats.LastFinalizedRound
		data, resp, err = zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: strconv.FormatInt(latestRound - int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10" })
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 10, len([]int64(*data)))


	})

	t.RunWithTimeout("test /v2/graph-write-price behavior", 2 * time.Minute, func(t *test.SystemTest) {
		// Should increase when increasing write price of one of the blobbers
		data, resp, err := zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ DataPoints: "1" })
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		wpBefore, wpAfter := (*data)[0], int64(0)

		// apiClient.ExecuteFaucet(t, blobberOwnerWallet, client.TxSuccessfulStatus)
		targetBlobber, resp, err := apiClient.V1SCRestGetRandomBlobber(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotNil(t, targetBlobber)

		targetBlobber.Terms.WritePrice *= 2
		apiClient.UpdateBlobber(t, blobberOwnerWallet, &model.SCRestGetBlobberResponse{
			ID: targetBlobber.ID,
			Terms: targetBlobber.Terms,
		}, client.TxSuccessfulStatus)

		wait.PoolImmediately(t, 2 * time.Minute, func() bool {
			minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
			require.NoError(t, err)
			latestRound := minerStats.LastFinalizedRound
			data, resp, err = zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: strconv.FormatInt(latestRound - int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10" })
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 10, len([]int64(*data)))
			wpAfter = (*data)[9]
			return wpAfter > wpBefore
		})

		wpBefore = wpAfter

		// Should decrease when decreasing write price of one of the blobbers
		targetBlobber.Terms.WritePrice /= 2
		apiClient.UpdateBlobber(t, blobberOwnerWallet, &model.SCRestGetBlobberResponse{
			ID: targetBlobber.ID,
			Terms: targetBlobber.Terms,
		}, client.TxSuccessfulStatus)

		wait.PoolImmediately(t, 2 * time.Minute, func() bool {
			minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
			require.NoError(t, err)
			latestRound := minerStats.LastFinalizedRound
			data, resp, err = zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{ From: strconv.FormatInt(latestRound - int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10" })
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 10, len([]int64(*data)))
			wpAfter = (*data)[9]
			return wpAfter > wpBefore
		})

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

func escapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func generateImage(t *test.SystemTest, localpath string) {
	//nolint
	thumbnailBytes, _ := base64.StdEncoding.DecodeString(`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`)
	err := os.WriteFile(localpath, thumbnailBytes, os.ModePerm)
	require.Nil(t, err, "failed to generate thumbnail", err)
}
