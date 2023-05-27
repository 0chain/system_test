package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/model"

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

		storageMarker, response, err := zboxClient.CreateFreeStorage(t, zboxClient.DefaultMnemonic, walletName, description, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		decodedBytes, err := base64.StdEncoding.DecodeString(storageMarker.Marker)
		if err != nil {
			fmt.Println("Error decoding string:", err)
			return
		}

		// Converting bytes to string
		decodedString := string(decodedBytes)
		t.Logf("the decoded stringg is %v ", decodedString)
		// Printing the decoded string
		fmt.Println("Decoded string:", decodedString)

		// Removing whitespace

		// Unmarshaling into struct
		var markerResponse model.ZboxFreeStorageMarkerResponse
		err = json.Unmarshal([]byte(decodedString), &markerResponse)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}
		decodedBytes, err = base64.StdEncoding.DecodeString(markerResponse.Marker)
		if err != nil {
			fmt.Println("Error decoding string:", err)
			return
		}

		// Converting bytes to string
		decodedString = string(decodedBytes)
		var marker model.ZboxFreeStorageMarker
		err = json.Unmarshal([]byte(decodedString), &marker)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}
	})
}
func removeWhitespace(s string) string {
	return strings.ReplaceAll(s, " ", "")
}
