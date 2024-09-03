package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestZvaultJWT(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Perform keys retrieval call with expired JWT token", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers = zvaultClient.NewZvaultHeaders("eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdF91c2VyX2lkX2FsdGVybmF0aXZlIiwiZXhwIjoxNzI1NDAwNzg4fQ.AoZeU7VfPuNntwnOpCjI5WMvSThNRIjgnJAmVfehYq4yOKq3DDXW6qKy8Q124r9WQaT-4pOMNvm3-LnUjYreRQ")

		_, response, err := zvaultClient.GetKeys(t, client.X_APP_CLIENT_ID, headers)
		require.Error(t, err)
		require.Equal(t, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Perform keys retrieval call with incorrect JWT token, which contains wrong client id", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		keys, response, err := zvaultClient.GetKeys(t, client.X_APP_CLIENT_ID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 0)
	})

	t.RunSequentially("Perform keys retrieval call with JWT token, which contains", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		keys, response, err := zvaultClient.GetKeys(t, client.X_APP_CLIENT_ID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 0)
	})
}
