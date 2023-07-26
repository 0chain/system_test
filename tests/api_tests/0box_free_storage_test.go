package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"gopkg.in/errgo.v2/errors"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	X_APP_CLIENT_ID          = "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9"
	X_APP_CLIENT_KEY         = "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96"
	X_APP_CLIENT_SIGNATURE   = "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a"
	X_APP_CLIENT_ID_R        = "3fb9694ebf47b5a51c050025d9c807c3319a05499b1eb980bbb9f1e27e119c9f"
	X_APP_CLIENT_KEY_R       = "9a8a960db2dd93eb35f26e8f7e84976349064cae3246da23abd575f05e7ed31bd90726cfcc960e017a9246d080f5419ada219d03758c370208c5b688e5ec7a9c"
	X_APP_CLIENT_SIGNATURE_R = "6b710d015b9e5e4734c08ac2de79ffeeeb49e53571cce8f71f21e375e5eca916"
)

func Test0BoxFreeStorage(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("List allocation with zero allocation should work")

	var firebaseToken *model.FirebaseToken
	t.TestSetup("Autenticate with firebase", func() {
		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	})

	t.RunSequentiallyWithTimeout("Create FreeStorage should work", 3*time.Minute, func(t *test.SystemTest) {
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

		fundingId := strconv.Itoa(storageMarker.FundidngId)
		require.Equal(t, "1", fundingId)
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

		storageMarker, response, _ := zboxClient.CreateFreeStorage(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp")
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		marker, markerResponse, err := UnmarshalMarkerData(storageMarker)
		require.Nil(t, err)
		require.Equal(t, X_APP_CLIENT_ID, marker.Recipient)
		require.Equal(t, marker.Assigner, "0chain")
		require.Equal(t, markerResponse.RecipientPublicKey, X_APP_CLIENT_KEY)
		require.Positive(t, marker.FreeTokens)
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
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error unmarshaling storage response")
	}

	decodedBytes, err = base64.StdEncoding.DecodeString(markerResponse.Marker)
	if err != nil {
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error decoding marker")
	}
	decodedString = string(decodedBytes) // nolint
	var marker model.ZboxFreeStorageMarker
	err = json.Unmarshal([]byte(decodedString), &marker)
	if err != nil {
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error unmarshling marker")
	}
	return marker, markerResponse, nil
}
