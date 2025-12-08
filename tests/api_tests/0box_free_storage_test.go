package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/model"
	"gopkg.in/errgo.v2/errors"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxFreeStorage(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()
	t.SetSmokeTests("List allocation with zero allocation should work")

	t.RunSequentiallyWithTimeout("Create FreeStorage should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		storageMarker, response, err := zboxClient.CreateFreeStorage(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		marker, markerResponse, err := UnmarshalMarkerData(storageMarker)
		require.Nil(t, err)
		require.Equal(t, headers["X-App-Client-ID"], marker.Recipient)
		require.Equal(t, marker.Assigner, "0chain")
		require.Equal(t, markerResponse.RecipientPublicKey, headers["X-App-Client-Key"])
		require.Positive(t, marker.FreeTokens)
	})

	t.RunSequentially("Create FreeStorage without existing wallet should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		_, response, err := zboxClient.CreateFreeStorage(t, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}

func UnmarshalMarkerData(storage *model.ZboxFreeStorage) (model.ZboxFreeStorageMarker, model.ZboxFreeStorageMarkerResponse, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(storage.Marker)
	if err != nil {
		return model.ZboxFreeStorageMarker{}, model.ZboxFreeStorageMarkerResponse{}, errors.New("error decode free storage response")
	}

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
