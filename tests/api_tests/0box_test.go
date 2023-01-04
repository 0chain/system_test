package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test0Box(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	var csrfToken string
	t.Run("Test setup", func(t *test.SystemTest) {
		token := generateCsrfToken(t, parsedConfig.ZboxPhoneNumber)
		require.NotNil(t, token, "CSRF response was nil!")
		csrfToken = token.CSRFToken
		require.NotNil(t, csrfToken, "CSRF token was nil!")
	})

}

func generateCsrfToken(t *test.SystemTest, phoneNumber string) *model.CSRFToken {
	authenticateWithFirebase(t, phoneNumber)
	token, response, err := apiClient.CreateCSRFToken(t, phoneNumber)
	require.NoError(t, err, "CSRF token creation failed with output: %v and error %v ", response, err)

	return token
}

func authenticateWithFirebase(t *test.SystemTest, phoneNumber string) *model.FirebaseToken {
	session, response, err := apiClient.FirebaseSendSms(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", phoneNumber)
	require.NoError(t, err, "Firebase send SMS failed: ", response.RawResponse)
	token, response, err := apiClient.FirebaseCreateToken(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", session.SessionInfo)
	require.NoError(t, err, "Firebase create token failed: ", response.RawResponse)

	return token
}
