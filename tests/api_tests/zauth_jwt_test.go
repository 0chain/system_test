package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const PEER_PUBLIC_KEY = ""

func TestZauthJWT(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Perform keys retrieval call with expired JWT token", func(w *test.SystemTest) {
		headers := zauthClient.NewZauthHeaders(JWT_TOKEN, "")

		_, response, err := zauthClient.Setup(t, &model.SetupWallet{}, headers)
		require.Error(t, err)
		require.Equal(t, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Perform wallet setup call with JWT token and remove with invalid JWT token", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		oldHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, splitWallet)

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, "")

		_, response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      splitWallet.ClientID,
			ClientKey:     splitWallet.ClientKey,
			PublicKey:     keys.Keys[0].PublicKey,
			PrivateKey:    keys.Keys[0].PrivateKey,
			PeerPublicKey: splitWallet.PeerPublicKey,
			ExpiredAt:     keys.Keys[0].ExpiresAt,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers["X-App-User-ID"] = client.X_APP_USER_ID_A

		sessionID, response, err = zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		jwtToken, response, err = zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, "")

		response, err = zauthClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	// t.RunSequentially("Perform keys retrieval call with JWT token, containing user id with present split key", func(w *test.SystemTest) {
	// 	headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
	// 	Teardown(t, headers)

	// 	sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	// 	require.NotEqual(t, int64(0), sessionID)

	// 	jwtToken, response, err := zboxClient.CreateJwtToken(t, sessionID, headers)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	// 	headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

	// 	splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, headers)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	// 	require.NotNil(t, splitWallet)

	// 	keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	// 	require.Len(t, keys.Keys, 1)
	// 	require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

	// 	response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	// })
}
