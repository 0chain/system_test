package api_tests

//
//import (
//	"encoding/json"
//	"fmt"
//	"strconv"
//	"testing"
//
//	"github.com/0chain/system_test/internal/api/model"
//	"github.com/0chain/system_test/internal/api/util/test"
//	"github.com/stretchr/testify/require"
//)
//

//func Test0Box_share_info(testSetup *testing.T) {
//	// todo: These tests are sequential and start with Teardown as they all share a common phone number
//	t := test.NewSystemTest(testSetup)
//	t.SetSmokeTests("Post ShareInfo with correct AuthTicket should work properly")
//
//	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
//
//	t.RunSequentially("Post ShareInfo with correct AuthTicket should work properly", func(t *test.SystemTest) {
//		Teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//		description := "wallet created as part of " + t.Name()
//		walletName := "wallet_name"
//		userName := "user_name"
//
//		zboxOwner, response, err := zboxClient.CreateOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxOwner)
//		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")
//
//		zboxWallet, response, err := zboxClient.PostWallet(t,
//			zboxClient.DefaultMnemonic,
//			walletName,
//			description,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"blimp",
//			userName,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxWallet)
//		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
//
//		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			zboxClient.DefaultAuthTicket,
//		)
//
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Nil(t, err)
//		require.NotNil(t, shareInfoDeletionMssg)
//
//		shareMessage := "Massege created as a part of " + t.Name()
//		fromInfo := "FromInfo created as a part of " + t.Name()
//		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
//			zboxClient.DefaultAuthTicket,
//			shareMessage,
//			fromInfo,
//			zboxClient.DefaultRecieverId,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, shareInfoSuccessMssg)
//		require.Equal(t, "shareinfo added successfully", shareInfoSuccessMssg.Message)
//
//		shareInfoData, response, err := zboxClient.GetShareInfo(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, shareInfoData)
//		require.Equal(t, shareInfoData.Message, "Data is present for the given details")
//		require.Len(t, shareInfoData.Data, 1)
//		require.Equal(t, shareInfoData.Data[0].Message, shareMessage)
//		require.Equal(t, shareInfoData.Data[0].Receiver, zboxClient.DefaultRecieverId)
//	})
//
//	t.RunSequentially("Post ShareInfo with Incorrect AuthTicket should work properly", func(t *test.SystemTest) {
//		Teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//		description := "wallet created as part of " + t.Name()
//		walletName := "wallet_name"
//		userName := "user_name"
//
//		zboxOwner, response, err := zboxClient.CreateOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxOwner)
//		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")
//
//		zboxWallet, response, err := zboxClient.PostWallet(t,
//			zboxClient.DefaultMnemonic,
//			walletName,
//			description,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"blimp",
//			userName,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxWallet)
//		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
//
//		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			zboxClient.DefaultAuthTicket,
//		)
//
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Nil(t, err)
//		require.NotNil(t, shareInfoDeletionMssg)
//
//		shareMessage := "Massege created as a part of " + t.Name()
//		fromInfo := "FromInfo created as a part of " + t.Name()
//		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
//			"abc",
//			shareMessage,
//			fromInfo,
//			zboxClient.DefaultRecieverId,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//
//		errorString := `{"error":"illegal base64 data at input byte 0"}`
//		require.NoError(t, err)
//		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Equal(t, shareInfoSuccessMssg.Message, "")
//		require.Equal(t, errorString, response.String())
//	})
//
//	t.RunSequentially("Delete ShareInfo without AUthTicket should not work properly", func(t *test.SystemTest) {
//		Teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//		description := "wallet created as part of " + t.Name()
//		walletName := "wallet_name"
//		userName := "user_name"
//
//		zboxOwner, response, err := zboxClient.CreateOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxOwner)
//		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")
//
//		zboxWallet, response, err := zboxClient.PostWallet(t,
//			zboxClient.DefaultMnemonic,
//			walletName,
//			description,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"blimp",
//			userName,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxWallet)
//		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
//
//		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			zboxClient.DefaultAuthTicket,
//		)
//
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Nil(t, err)
//		require.NotNil(t, shareInfoDeletionMssg)
//
//		shareMessage := "Massege created as a part of " + t.Name()
//		fromInfo := "FromInfo created as a part of " + t.Name()
//		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
//			zboxClient.DefaultAuthTicket,
//			shareMessage,
//			fromInfo,
//			zboxClient.DefaultRecieverId,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, shareInfoSuccessMssg)
//		require.Equal(t, "shareinfo added successfully", shareInfoSuccessMssg.Message, "Error adding ShareInfo")
//
//		shareInfoDeletionMssg, response, err = zboxClient.DeleteShareInfo(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"",
//		)
//		errorString := `{"error":"invalid params: pass atleast one of lookuphash or authticket"}`
//		require.NoError(t, err)
//		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Equal(t, shareInfoDeletionMssg.Message, "")
//		require.Equal(t, errorString, response.String())
//	})
//
//	t.RunSequentially("Delete ShareInfo with correct parameter should work properly", func(t *test.SystemTest) {
//		Teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//		description := "wallet created as part of " + t.Name()
//		walletName := "wallet_name"
//		userName := "user_name"
//
//		zboxOwner, response, err := zboxClient.CreateOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxOwner)
//		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")
//
//		zboxWallet, response, err := zboxClient.PostWallet(t,
//			zboxClient.DefaultMnemonic,
//			walletName,
//			description,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"blimp",
//			userName,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxWallet)
//		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
//
//		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			zboxClient.DefaultAuthTicket,
//		)
//
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Nil(t, err)
//		require.NotNil(t, shareInfoDeletionMssg)
//
//		shareMessage := "Massege created as a part of " + t.Name()
//		fromInfo := "FromInfo created as a part of " + t.Name()
//		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
//			zboxClient.DefaultAuthTicket,
//			shareMessage,
//			fromInfo,
//			zboxClient.DefaultRecieverId,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, shareInfoSuccessMssg)
//		require.Equal(t, "shareinfo added successfully", shareInfoSuccessMssg.Message, "Error adding shareInfo")
//
//		shareInfoDeletionMssg, response, err = zboxClient.DeleteShareInfo(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			zboxClient.DefaultAuthTicket,
//		)
//
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Nil(t, err)
//		require.NotNil(t, shareInfoDeletionMssg)
//		require.Equal(t, shareInfoDeletionMssg.Message, "deleting shareinfo successful", "Error deleting ShareInfo")
//	})
//}
//

//
//func TestDexState(testSetup *testing.T) {
//	t := test.NewSystemTest(testSetup)
//	t.SetSmokeTests("Create a DEX state with valid phone number should work")
//
//	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
//
//	postData := map[string]string{
//		"stage":     "burn",
//		"reference": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
//	}
//
//	updateData := map[string]string{
//		"stage":     "burn",
//		"reference": "{\"test_2\":\"test1\", \"test4\":\"test3\"}",
//	}
//
//	// POST DEX STATE
//	t.RunSequentially("Create a DEX state with valid phone number should work", func(t *test.SystemTest) {
//		Teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//		description := "wallet created as part of " + t.Name()
//		walletName := "wallet_name"
//		userName := "user_name"
//
//		zboxOwner, response, err := zboxClient.CreateOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxOwner)
//		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")
//
//		zboxWallet, response, err := zboxClient.PostWallet(t,
//			zboxClient.DefaultMnemonic,
//			walletName,
//			description,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"blimp",
//			userName,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxWallet)
//		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
//
//		dexState, response, err := zboxClient.PostDexState(t,
//			postData,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, dexState)
//	})
//
//	t.RunSequentially("Create a DEX state with invalid csrf token should fail", func(t *test.SystemTest) {
//		dexState, response, err := zboxClient.PostDexState(t,
//			postData,
//			firebaseToken.IdToken,
//			"abcd",
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 400, response.StatusCode())
//		require.Empty(t, dexState)
//	})
//
//	t.RunSequentially("Create a DEX state with invalid field should fail", func(t *test.SystemTest) {
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//
//		wrongData := map[string]string{
//			"stage":        "burn",
//			"refe3r72t981": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
//		}
//
//		dexState, response, err := zboxClient.PostDexState(t,
//			wrongData,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 400, response.StatusCode())
//		require.Empty(t, dexState)
//	})
//
//	t.RunSequentially("Create a DEX state 2 times with same phone number should fail", func(t *test.SystemTest) {
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//
//		dexState, response, err := zboxClient.PostDexState(t,
//			postData,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 400, response.StatusCode())
//		require.Empty(t, dexState)
//	})
//
//	// GET DEX STATE
//	t.RunSequentially("Get DEX state with valid phone number should work", func(t *test.SystemTest) {
//		Teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//		description := "wallet created as part of " + t.Name()
//		walletName := "wallet_name"
//		userName := "user_name"
//
//		zboxOwner, response, err := zboxClient.CreateOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxOwner)
//		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")
//
//		zboxWallet, response, err := zboxClient.PostWallet(t,
//			zboxClient.DefaultMnemonic,
//			walletName,
//			description,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"blimp",
//			userName,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxWallet)
//		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
//
//		data := map[string]string{
//			"stage":     "burn",
//			"reference": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
//		}
//
//		_, response, err = zboxClient.PostDexState(t,
//			data,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//
//		dexState, response, err := zboxClient.GetDexState(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, dexState)
//		require.Equal(t, postData["stage"], dexState.Stage)
//		require.Equal(t, postData["reference"], dexState.Reference)
//	})
//
//	// UPDATE DEX STATE
//	t.RunSequentially("Update DEX state with valid phone number should work", func(t *test.SystemTest) {
//		Teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//		description := "wallet created as part of " + t.Name()
//		walletName := "wallet_name"
//		userName := "user_name"
//
//		zboxOwner, response, err := zboxClient.CreateOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxOwner)
//		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")
//
//		zboxWallet, response, err := zboxClient.PostWallet(t,
//			zboxClient.DefaultMnemonic,
//			walletName,
//			description,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//			"blimp",
//			userName,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.NotNil(t, zboxWallet)
//		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
//
//		data := map[string]string{
//			"stage":     "burn",
//			"reference": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
//		}
//
//		_, response, err = zboxClient.PostDexState(t,
//			data,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//
//		dexState, response, err := zboxClient.GetDexState(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Equal(t, postData["reference"], dexState.Reference)
//
//		dexState, response, err = zboxClient.PutDexState(t,
//			updateData,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 200, response.StatusCode())
//		require.NotNil(t, dexState)
//
//		// get dex state
//		dexState, response, err = zboxClient.GetDexState(t,
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//
//		require.NoError(t, err)
//		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
//		require.Equal(t, updateData["reference"], dexState.Reference)
//	})
//
//	t.RunSequentially("Update DEX state with invalid data should fail", func(t *test.SystemTest) {
//		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
//
//		dexState, response, err := zboxClient.PutDexState(t,
//			map[string]string{
//				"stage": "burn",
//			},
//			firebaseToken.IdToken,
//			csrfToken,
//			zboxClient.DefaultPhoneNumber,
//		)
//		require.NoError(t, err)
//		require.Equal(t, 400, response.StatusCode())
//		require.Empty(t, dexState)
//	})
//}
//
//func Teardown(t *test.SystemTest, headers map[string]string) {
//	t.Logf("Tearing down existing test data")
//
//	message, response, err := zboxClient.DeleteOwner(t, clientId, clientKey, clientSignature, idToken, csrfToken, phoneNumber)
//	println(message, response, err)
//}
//
//func teardownFCM(t *test.SystemTest, idToken, phoneNumber string) {
//	t.Logf("Tearing down existing fcm test data for [%v]", phoneNumber)
//	csrfToken := createCsrfToken(t, phoneNumber)
//
//	_, _, err := zboxClient.DeleteFCMToken(t, idToken, csrfToken, phoneNumber)
//	require.NoError(t, err, "Cleanup failed for 0box FCM Token")
//}
//
//func createCsrfToken(t *test.SystemTest, phoneNumber string) string {
//	csrfToken, response, err := zboxClient.CreateCSRFToken(t, phoneNumber)
//	require.NoError(t, err, "CSRF token creation failed with output: %v and error %v ", response, err)
//
//	require.NotNil(t, csrfToken, "CSRF token container was nil!", response)
//	require.NotNil(t, csrfToken.CSRFToken, "CSRF token was nil!", response)
//
//	return csrfToken.CSRFToken
//}
//
//func authenticateWithFirebase(t *test.SystemTest, phoneNumber string) *model.FirebaseToken {
//	session, response, err := zboxClient.FirebaseSendSms(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", phoneNumber)
//	require.NoError(t, err, "Firebase send SMS failed: ", response.RawResponse)
//	token, response, err := zboxClient.FirebaseCreateToken(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", session.SessionInfo)
//	require.NoError(t, err, "Firebase create token failed: ", response.RawResponse)
//
//	return token
//}
