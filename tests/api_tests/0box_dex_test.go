package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func NewTestDex() map[string]string {
	return map[string]string{
		"tx_hash":   "165f0f8e557c430929784035df7eeacf7a3ff795f10d76c8707409bba31cb617",
		"stage":     "mint",
		"reference": "test_reference",
	}
}

func Test0BoxDex(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Create dex should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		dexData := NewTestDex()

		_, response, err := zboxClient.CreateDexState(t, headers, dexData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		dex, response, err := zboxClient.GetDexState(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "mint", dex.Stage)
	})

	t.RunSequentially("Update dex should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		dexData := NewTestDex()

		_, response, err := zboxClient.CreateDexState(t, headers, dexData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		dexData["stage"] = "burn"
		_, response, err = zboxClient.UpdateDexState(t, headers, dexData)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		dex, response, err := zboxClient.GetDexState(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "burn", dex.Stage)
	})
}
