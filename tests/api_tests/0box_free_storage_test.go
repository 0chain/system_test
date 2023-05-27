package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"gopkg.in/errgo.v2/errors"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxFreeStorage(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("List allocation with zero allocation should work")

	var firebaseToken *model.FirebaseToken
	t.TestSetup("Autenticate with firebase", func() {
		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	})

	t.RunSequentially("Create FreeStorage should work", func(t *test.SystemTest) {
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

		storageMarker, response, err := zboxClient.CreateFreeStorage(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp")
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		marker, markerResponse, err := UnmarshalMarkerData(storageMarker)
		require.Nil(t, err)
		require.Equal(t, marker.Assigner, zboxClient.DefaultRecieverId)
		require.Equal(t, markerResponse.RecipientPublicKey, zboxClient.DefaultPhoneNumber)
	})

	t.RunSequentially("Create FreeStorage should not work more than once", func(t *test.SystemTest) {
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

		_, response, err = zboxClient.CreateFreeStorage(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp")
		require.NotNil(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	})

}

func UnmarshalMarkerData(storage *model.ZboxFreeStorage) (model.ZboxFreeStorageMarker, model.ZboxFreeStorageMarkerResponse, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(storage.Marker)
	if err != nil {
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error decode free storage response")
	}

	// Converting bytes to string
	decodedString := string(decodedBytes)
	var markerResponse model.ZboxFreeStorageMarkerResponse
	err = json.Unmarshal([]byte(decodedString), &markerResponse)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error unmarshaling storage response")
	}

	decodedBytes, err = base64.StdEncoding.DecodeString(markerResponse.Marker)
	if err != nil {
		fmt.Println("Error decoding string:", err)
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error decoding marker")
	}
	decodedString = string(decodedBytes)
	var marker model.ZboxFreeStorageMarker
	err = json.Unmarshal([]byte(decodedString), &marker)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error unmarshling marker")
	}
	return marker, markerResponse, nil

}
