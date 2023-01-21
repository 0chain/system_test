package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestDexState(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Create a DEX state with valid phone number should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PostDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, dexState)
	})

	t.RunSequentially("Create a DEX state with invalid phone number should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PostDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			"123456789",
		)
		require.Error(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Nil(t, dexState)
	})

	t.RunSequentially("Create a DEX state with invalid csrf token should fail", func(t *test.SystemTest) {
		dexState, response, err := zboxClient.PostDexState(t,
			firebaseToken.IdToken,
			"abcd",
			zboxClient.DefaultPhoneNumber,
		)
		require.Error(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Nil(t, dexState)
	})
}
