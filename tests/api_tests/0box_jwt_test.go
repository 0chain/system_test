package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"strconv"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	UserID            = "lWVZRhERosYtXR9MBJh5yJUtweI4"
	AlternativeUserID = "lWVZRhERosYtXR9MBJh5yJUtweI5"
)

func Test0BoxJWT(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Create JWT token with user id, which differs from the one used during session creation", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers["X-App-Client-ID"] = UserID

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		headers["X-App-Client-ID"] = AlternativeUserID
		headers["X-JWT-Session-ID"] = strconv.FormatInt(sessionID, 10)

		_, response, err = zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.NotEqual(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Create JWT token with user id, which equals to the one used during session creation", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers["X-App-Client-ID"] = UserID

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		headers["X-JWT-Session-ID"] = strconv.FormatInt(sessionID, 10)

		var jwtToken *model.ZboxJwtToken

		jwtToken, response, err = zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)
	})

	t.RunSequentially("Refresh JWT token with user id, which differs from the one used by the given old JWT token", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers["X-App-Client-ID"] = UserID

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		headers["X-JWT-Session-ID"] = strconv.FormatInt(sessionID, 10)

		var jwtToken *model.ZboxJwtToken

		jwtToken, response, err = zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)

		headers["X-App-Client-ID"] = AlternativeUserID

		jwtToken, response, err = zboxClient.RefreshJwtToken(t, jwtToken.JwtToken, headers)
		require.NoError(t, err)
		require.NotEqual(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Refresh JWT token with incorrect old JWT token", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers["X-App-Client-ID"] = UserID

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		headers["X-JWT-Session-ID"] = strconv.FormatInt(sessionID, 10)

		var jwtToken *model.ZboxJwtToken

		jwtToken, response, err = zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)

		jwtToken, response, err = zboxClient.RefreshJwtToken(t, "", headers)
		require.NoError(t, err)
		require.NotEqual(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Refresh JWT token with user id, which equals to the one used by the given old JWT token", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers["X-App-Client-ID"] = UserID

		sessionID, response, err := zboxClient.CreateJwtSession(t, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, int64(0), sessionID)

		headers["X-JWT-Session-ID"] = strconv.FormatInt(sessionID, 10)

		var jwtToken *model.ZboxJwtToken

		jwtToken, response, err = zboxClient.CreateJwtToken(t, sessionID, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)

		jwtToken, response, err = zboxClient.RefreshJwtToken(t, jwtToken.JwtToken, headers)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEmpty(t, jwtToken.JwtToken)
	})
}
