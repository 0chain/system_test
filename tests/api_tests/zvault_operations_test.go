package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestZvaultOperations(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Retrieve split keys for default client id, should be empty", func(w *test.SystemTest) {
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

	t.RunSequentially("Create split wallet", func(t *test.SystemTest) {
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

		splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
	})
}
