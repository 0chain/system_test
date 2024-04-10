package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func NewTestWallet() map[string]string {
	return map[string]string{
		"name":        "test_wallet_name",
		"description": "test_wallet_description",
		"mnemonic":    "test_mnemonic",
	}
}

func Create0boxTestWallet(t *test.SystemTest, headers map[string]string) error {
	ownerInput := NewTestOwner()
	_, _, err := zboxClient.CreateOwner(t, headers, ownerInput)
	if err != nil {
		return err
	}
	walletInput := NewTestWallet()
	_, _, err = zboxClient.CreateWallet(t, headers, walletInput)
	if err != nil {
		return err
	}
	return nil
}

func Test0BoxWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("create wallet without owner should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("create wallet without existing wallet should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		ownerInput := NewTestOwner()
		_, _, err := zboxClient.CreateOwner(t, headers, ownerInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		wallet, response, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, walletInput["name"], wallet.Name)
		require.Equal(t, walletInput["mnemonic"], wallet.Mnemonic)
		require.Equal(t, headers["X-App-Client-Key"], wallet.PublicKey)
		require.Equal(t, walletInput["description"], wallet.Description)
	})

	t.RunSequentially("create wallet with existing wallet blimp should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		ownerInput := NewTestOwner()
		_, _, err := zboxClient.CreateOwner(t, headers, ownerInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		_, response, err = zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("create wallet with existing wallet bolt should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BOLT)
		Teardown(t, headers)

		ownerInput := NewTestOwner()
		_, _, err := zboxClient.CreateOwner(t, headers, ownerInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers["X-App-Client-ID"] = "new_client_id"
		_, response, err = zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		res, response, err := zboxClient.GetWalletList(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 2, len(res.Data))
	})

	t.RunSequentially("update wallet with existing wallet should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		ownerInput := NewTestOwner()
		_, _, err := zboxClient.CreateOwner(t, headers, ownerInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		walletInput["name"] = "new_name"
		walletInput["mnemonic"] = "new_mnemonic"
		message, response, err := zboxClient.UpdateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "updating wallet successful", message.Message)

		wallet, response, err := zboxClient.GetWalletKeys(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, walletInput["name"], wallet.Name)
		require.Equal(t, walletInput["mnemonic"], wallet.Mnemonic)
		require.Equal(t, headers["X-App-Client-Key"], wallet.PublicKey)
		require.Equal(t, walletInput["description"], wallet.Description)
	})

	t.RunSequentially("update wallet without existing wallet should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		ownerInput := NewTestOwner()
		_, _, err := zboxClient.CreateOwner(t, headers, ownerInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		message, response, err := zboxClient.UpdateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "no wallet was updated for these details", message.Message)
	})
}
