package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// NewTestShareinfo builds a shareinfo map with an auth_ticket whose owner_id
// matches the current X_APP_CLIENT_ID. The auth ticket is constructed
// dynamically because X_APP_CLIENT_ID is set at runtime from a real on-chain
// wallet, and 0box validates that the owner_id exists on-chain.
func NewTestShareinfo() map[string]string {
	ticket := map[string]interface{}{
		"client_id":      client.X_APP_CLIENT_ID,
		"owner_id":       client.X_APP_CLIENT_ID,
		"allocation_id":  "e0c2cd2d5faaad13fc53733dd759749f2b2f01af46332009c678221a2c488153",
		"file_path_hash": "e724a2201e22653d32167fca1bf12b2e44bac6337d3ebdb7427fba4eecaa4c9d",
		"actual_file_hash": "1f112083f2405c395de51b7f13f9f795aa141c40f1d47d78c83a49930f2b9a34",
		"file_name":      "IMG_4874.PNG",
		"reference_type": "f",
		"expiration":     0,
		"timestamp":      1667218270,
		"encrypted":      false,
		"signature":      "339e55299b48e229dde902f8c965d15a940b2777c5d9327a4c79131b8a771e07",
	}
	ticketJSON, _ := json.Marshal(ticket)
	return map[string]string{
		"auth_ticket":     base64.StdEncoding.EncodeToString(ticketJSON),
		"share_info_type": "private",
	}
}

func Test0BoxShareinfo(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	// NOT parallel: shares identity with other 0box tests; concurrent Teardown deletes wallet rows.

	t.RunSequentially("Create shareinfo valid auth ticket should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

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

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		shareinfoData := NewTestShareinfo()
		shareinfoData["auth_ticket"] = "invalid_ticket"

		_, response, err := zboxClient.CreateShareInfo(t, headers, shareinfoData)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("get shareinfo valid auth ticket should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

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
