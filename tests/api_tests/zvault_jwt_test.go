package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

//nolint:gosec // test-only dummy JWT token, not a real credential
const JWT_TOKEN = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdF91c2VyX2lkX2FsdGVybmF0aXZlIiwiZXhwIjoxNzI1NDAwNzg4fQ.AoZeU7VfPuNntwnOpCjI5WMvSThNRIjgnJAmVfehYq4yOKq3DDXW6qKy8Q124r9WQaT-4pOMNvm3-LnUjYreRQ"

func TestZvaultJWT(testSetup *testing.T) {
	require.True(testSetup, zvaultAvailable, "zvault service must be available")
	if !firebaseTokenValid {
		testSetup.Skip("Firebase authentication not configured")
	}
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Perform keys retrieval call with expired JWT token", func(w *test.SystemTest) {
		headers := zvaultClient.NewZvaultHeaders(JWT_TOKEN)

		_, response, err := zvaultClient.GetKeys(w, client.X_APP_CLIENT_ID, headers)
		require.Error(w, err)
		require.Equal(w, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Perform wallets retrieval call with JWT token, containing user id, for which there are no keys", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(w, client.X_APP_BLIMP)
		Teardown(w, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(w, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(w, headers)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		oldHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		var generateWalletResponse *model.GenerateWalletResponse

		generateWalletResponse, response, err = zvaultClient.GenerateSplitWallet(w, oldHeaders)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Ensure cleanup even if test skips or fails
		defer func() {
			if generateWalletResponse != nil {
				zvaultClient.Delete(w, generateWalletResponse.ClientID, oldHeaders)
			}
		}()

		response, err = zvaultClient.GenerateSplitKey(w, generateWalletResponse.ClientID, oldHeaders)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zboxClient.NewZboxHeadersWithCSRF(w, client.X_APP_BLIMP)
		Teardown(w, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(w, client.X_APP_BLIMP)

		headers["X-App-Client-ID"] = client.X_APP_CLIENT_ID_A
		headers["X-App-User-ID"] = client.X_APP_USER_ID_A
		headers["X-App-Client-Key"] = client.X_APP_CLIENT_KEY_A
		headers["X-App-Client-Signature"] = client.X_APP_CLIENT_SIGNATURE_A

		jwtToken, response, err = zboxClient.CreateJwtToken(w, headers)
		require.NoError(w, err)
		require.NotEqual(w, 401, response.StatusCode(), "Alternative user JWT creation failed (user may not exist in Firebase)")
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		var keys *model.GetKeyResponse

		keys, response, err = zvaultClient.GetKeys(w, generateWalletResponse.ClientID, headers)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(w, keys.Keys, 0)
	})

	t.RunSequentially("Perform wallets retrieval call with JWT token, containing user id with present split key", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(w, client.X_APP_BLIMP)
		Teardown(w, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(w, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(w, headers)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		var generateWalletResponse *model.GenerateWalletResponse

		generateWalletResponse, response, err = zvaultClient.GenerateSplitWallet(w, headers)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zvaultClient.GenerateSplitKey(w, generateWalletResponse.ClientID, headers)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		var keys *model.GetKeyResponse

		keys, response, err = zvaultClient.GetKeys(w, generateWalletResponse.ClientID, headers)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(w, keys.Keys, 1)
		require.Equal(w, keys.Keys[0].ClientID, keys.Keys[0].ClientID)

		response, err = zvaultClient.Delete(w, generateWalletResponse.ClientID, headers)
		require.NoError(w, err)
		require.Equal(w, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}
