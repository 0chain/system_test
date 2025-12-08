package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func NewTestShareinfo() map[string]string {
	return map[string]string{
		"auth_ticket": "eyJjbGllbnRfaWQiOiIiLCJvd25lcl9pZCI6ImNhYWU1YTlkNDhiMWEwY2QwMWE1ZGE5ODI4MDdkN2FkNmZjYzhhODM2N2I3OWM2YWRiZTQ4ZTdjNjMyNTQ0ZjIiLCJhbGxvY2F0aW9uX2lkIjoiZTBjMmNkMmQ1ZmFhYWQxM2ZjNTM3MzNkZDc1OTc0OWYyYjJmMDFhZjQ2MzMyMDA5YzY3ODIyMWEyYzQ4ODE1MyIsImZpbGVfcGF0aF9oYXNoIjoiZTcyNGEyMjAxZTIyNjUzZDMyMTY3ZmNhMWJmMTJiMmU0NGJhYzYzMzdkM2ViZGI3NDI3ZmJhNGVlY2" +
			"FhNGM5ZCIsImFjdHVhbF9maWxlX2hhc2giOiIxZjExMjA4M2YyNDA1YzM5NWRlNTFiN2YxM2Y5Zjc5NWFhMTQxYzQwZjFkNDdkNzhjODNhNDk5MzBmMmI5YTM0IiwiZmlsZV9uYW1lIjoiSU1HXzQ4NzQuUE5HIiwicmVmZXJlbmNlX3R5cGUiOiJmIiwiZXhwaXJhdGlvbiI6MCwidGltZXN0YW1wIjoxNjY3MjE4MjcwLCJlbmNyeXB0ZWQiOmZhbHNlLCJzaWduYXR1cmUiOiIzMzllNTUyOTliNDhlMjI5ZGRlOTAyZjhjOTY1ZDE1YTk0MGIyNzc3YzVkOTMyN2E0Yzc5MTMxYjhhNzcxZTA3In0=",
	}
}

func Test0BoxShareinfo(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Create shareinfo valid auth ticket should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		shareinfoData := NewTestShareinfo()

		shareinfoResponse, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "shareinfo added successfully", shareinfoResponse.Message)

		_, _, err = zboxClient.DeleteShareinfo(t, headers, shareinfoData["auth_ticket"])
		require.NoError(t, err)
	})

	t.RunSequentially("Create shareinfo invalid auth ticket should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		shareinfoData := NewTestShareinfo()
		shareinfoData["auth_ticket"] = "invalid_ticket"

		_, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("get shareinfo valid auth ticket should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		shareinfoData := NewTestShareinfo()

		_, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		shareinfoSharedResponse, response, err := zboxClient.GetShareInfoShared(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 1, len(shareinfoSharedResponse.Data))
		require.Equal(t, client.X_APP_CLIENT_ID, shareinfoSharedResponse.Data[0].Receiver)

		hareinfoReceivedResponse, response, err := zboxClient.GetShareInfoReceived(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 1, len(hareinfoReceivedResponse.Data))
		require.Equal(t, client.X_APP_CLIENT_ID, hareinfoReceivedResponse.Data[0].ClientID)

		_, _, err = zboxClient.DeleteShareinfo(t, headers, shareinfoData["auth_ticket"])
		require.NoError(t, err)
	})
}
