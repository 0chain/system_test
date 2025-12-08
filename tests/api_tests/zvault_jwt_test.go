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
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Perform keys retrieval call with expired JWT token", func(w *test.SystemTest) {
		headers := zvaultClient.NewZvaultHeaders(JWT_TOKEN)

		_, response, err := zvaultClient.GetKeys(t, client.X_APP_CLIENT_ID, headers)
		require.Error(t, err)
		require.Equal(t, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Perform wallets retrieval call with JWT token, containing user id, for which there are no keys", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		oldHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		var generateWalletResponse *model.GenerateWalletResponse

		generateWalletResponse, response, err = zvaultClient.GenerateSplitWallet(t, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zvaultClient.GenerateSplitKey(t, generateWalletResponse.ClientID, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		headers["X-App-Client-ID"] = client.X_APP_CLIENT_ID_A
		headers["X-App-User-ID"] = client.X_APP_USER_ID_A
		headers["X-App-Client-Key"] = client.X_APP_CLIENT_KEY_A
		headers["X-App-Client-Signature"] = client.X_APP_CLIENT_SIGNATURE_A

		jwtToken, response, err = zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		var keys *model.GetKeyResponse

		keys, response, err = zvaultClient.GetKeys(t, generateWalletResponse.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 0)

		response, err = zvaultClient.Delete(t, generateWalletResponse.ClientID, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Perform wallets retrieval call with JWT token, containing user id with present split key", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		var generateWalletResponse *model.GenerateWalletResponse

		generateWalletResponse, response, err = zvaultClient.GenerateSplitWallet(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zvaultClient.GenerateSplitKey(t, generateWalletResponse.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		var keys *model.GetKeyResponse

		keys, response, err = zvaultClient.GetKeys(t, generateWalletResponse.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, keys.Keys[0].ClientID)

		response, err = zvaultClient.Delete(t, generateWalletResponse.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}
