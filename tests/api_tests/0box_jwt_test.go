package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxJWT(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Create JWT token", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		_, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Refresh JWT token with user id, which differs from the one used by the given old JWT token", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)

		headers["X-App-Client-ID"] = client.X_APP_CLIENT_ID_A
		headers["X-App-User-ID"] = client.X_APP_USER_ID_A
		headers["X-App-Client-Key"] = client.X_APP_CLIENT_KEY_A
		headers["X-App-Client-Signature"] = client.X_APP_CLIENT_SIGNATURE_A

		_, response, err = zboxClient.RefreshJwtToken(t, jwtToken.JwtToken, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Refresh JWT token with incorrect old JWT token", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)

		_, response, err = zboxClient.RefreshJwtToken(t, "", headers)
		require.NoError(t, err)
		require.Equal(t, 500, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Refresh JWT token with user id, which equals to the one used by the given old JWT token", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)

		jwtToken, response, err = zboxClient.RefreshJwtToken(t, jwtToken.JwtToken, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)
	})
}
