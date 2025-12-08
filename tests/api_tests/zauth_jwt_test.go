package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	CLIENT_ID         = "ec44cc15f31e82c9e6be24e52877dade5c6cb5a3ab7f7d326cf4f918afda5b77"
	CLIENT_KEY        = "afb16288dd4c6c86a91a39b959640ba2dbb9eb9a81887f3cf41224f30d5cf92047427c4c34a90c1d21b5f78b16463ce30d77a092de7b7e65db746d12a575890a"
	PUBLIC_KEY_A      = "95fc0341520d6211a6db46b3fbdd026edbbc20fa25175cfd9969ff925d8096216b0f36674bcee686caf2ab1314b6c0981493143d7ec9fd60f9db130fee4a0b8a"
	PRIVATE_KEY_A     = "0d38858f6b83c147529cf9d5f11be3d800742a5b910a40035a8f6b66161c8d11"
	PEER_PUBLIC_KEY   = "67d93b39403c90e1c99060d80f52980c6a348a77fbda6ba31d0a31463b29a81246f0c4e19c20a93542cf65a632f8aaeb853893fefb7d0683d72902634892508a"
	PEER_PUBLIC_KEY_A = ""
	EXPIRES_AT        = 0
)

func TestZauthJWT(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Perform keys retrieval call with expired JWT token", func(w *test.SystemTest) {
		headers := zauthClient.NewZauthHeaders(JWT_TOKEN, "")

		response, err := zauthClient.Setup(t, &model.SetupWallet{}, headers)
		require.NoError(t, err)
		require.Equal(t, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Perform wallet setup call with JWT token and remove with invalid JWT token", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		oldHeaders := zauthClient.NewZauthHeaders(jwtToken.JwtToken, "")

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID,
			ClientKey:     CLIENT_KEY,
			PublicKey:     PUBLIC_KEY_A,
			PrivateKey:    PRIVATE_KEY_A,
			PeerPublicKey: PEER_PUBLIC_KEY,
			ExpiredAt:     EXPIRES_AT,
		}, oldHeaders)
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

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, "")

		response, err = zauthClient.Delete(t, CLIENT_ID, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Perform wallet setup call with JWT token and remove with correct JWT token", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, "")

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID,
			ClientKey:     CLIENT_KEY,
			PublicKey:     PUBLIC_KEY_A,
			PrivateKey:    PRIVATE_KEY_A,
			PeerPublicKey: PEER_PUBLIC_KEY,
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}
