package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"gopkg.in/errgo.v2/errors"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

const (
	X_APP_CLIENT_ID        = "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9"
	X_APP_CLIENT_KEY       = "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96"
	X_APP_CLIENT_SIGNATURE = "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a"
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
		require.Equal(t, X_APP_CLIENT_ID, marker.Recipient)
		require.Equal(t, marker.Assigner, "0chain")
		require.Equal(t, markerResponse.RecipientPublicKey, X_APP_CLIENT_KEY)
		require.Positive(t, marker.FreeTokens)
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
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, response.String(), `{"error":"400: free storage for appType: blimp already used"}`)

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
	decodedString = string(decodedBytes) // nolint
	var marker model.ZboxFreeStorageMarker
	err = json.Unmarshal([]byte(decodedString), &marker)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error unmarshling marker")
	}
	return marker, markerResponse, nil

}

func (z *client.ZboxClient)checkStatus(t *test.SystemTest, fundingId, idToken, csrfToek, appType string)(bool){
	wait.PoolImmediately(t, time.Minute*2, func() bool {
		var zboxFundingResponse  model.ZboxFundingResponse ;
		zboxFundingResponse, resp, err := z.CheckFundingStatus(
			t,
			fundingId,
			idToken,
			csrfToken,
			z.DefaultPhoneNumber,
			appType,
			)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if zboxFundingResponse == model.ZboxFundingResponse{} {
			return false
		}

		return zboxFundingResponse.Funded == "true"
	})
}
