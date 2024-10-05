package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	PRIVATE_KEY   = "04e57640c99339b7ca75e880722788d5d18d5127b37deabcab0ad39e40b84b1e"
	PRIVATE_KEY_I = "04e57640c99339b7ca75e880722788d5d18d5127b37deabcab0ad39e40"
	PUBLIC_KEY    = "be63f802120e6b164d8df7f2941cc9ed81046e93337862cbdab17fe87a763e0d04846a09491f6dda642289f7651e0df1eda63e3272b2df27dadda31bc1463f91"
	MNEMONIC      = "cupboard raven slush easily author profit argue second evolve autumn tool favorite spider version found raw laptop donkey unlock address minimum truck tilt calm"
	MNEMONIC_A    = "toss ocean track betray donate pioneer broken seek glare two force decade master invite harvest oval thing few leaf toss avoid immense orbit ribbon"
)

func TestZvaultOperations(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Retrieve split keys for default client id, should be empty", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		keys, response, err := zvaultClient.GetKeys(t, client.X_APP_CLIENT_ID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 0)
	})

	t.RunSequentially("Retrieve master wallets for default client id, should be empty", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		wallets, response, err := zvaultClient.GetWallets(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, wallets, 0)
	})

	t.RunSequentially("Retrieve shared wallets for default client id, should be empty", func(w *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		sharedWallets, response, err := zvaultClient.GetSharedWallets(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, sharedWallets, 0)
	})

	t.RunSequentially("Generate split wallet", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Store previously created private key without mnemonic", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.Store(t, PRIVATE_KEY, "", headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		wallets, response, err := zvaultClient.GetWallets(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, wallets, 1)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Store invalid private key with correct mnemonic", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		_, response, err = zvaultClient.Store(t, PRIVATE_KEY_I, MNEMONIC, headers)
		require.Error(t, err)
		require.Equal(t, 500, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Store previously created private key with correct mnemonic", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.Store(t, PRIVATE_KEY, MNEMONIC, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		wallets, response, err := zvaultClient.GetWallets(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, wallets, 1)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Store previously created private key with invalid mnemonic", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.Store(t, PRIVATE_KEY, MNEMONIC_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		wallets, response, err := zvaultClient.GetWallets(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, wallets, 1)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Generate split key with previously stored private key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.Store(t, PRIVATE_KEY, MNEMONIC, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		splitKey, response, err := zvaultClient.GenerateSplitKey(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, splitKey)

		keys, response, err = zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 2)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Revoke not existing split wallet", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		response, err = zvaultClient.Revoke(t, client.X_APP_CLIENT_ID, PUBLIC_KEY, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Revoke generated split wallet, when only one split wallet is available", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		response, err = zvaultClient.Revoke(t, splitWallet.ClientID, keys.Keys[0].PublicKey, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err = zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)
		require.True(t, keys.Keys[0].IsRevoked)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Delete not existing master wallet", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		response, err = zvaultClient.Delete(t, client.X_APP_CLIENT_ID, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Share split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		oldHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		response, err = zvaultClient.ShareWallet(t, client.X_APP_USER_ID_A, keys.Keys[0].PublicKey, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		headers["X-App-Client-ID"] = client.X_APP_CLIENT_ID_A
		headers["X-App-User-ID"] = client.X_APP_USER_ID_A
		headers["X-App-Client-Key"] = client.X_APP_CLIENT_KEY_A
		headers["X-App-Client-Signature"] = client.X_APP_CLIENT_SIGNATURE_A

		jwtToken, response, err = zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		sharedWallets, response, err := zvaultClient.GetSharedWallets(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, sharedWallets, 1)
		require.Equal(t, sharedWallets[0].ClientID, splitWallet.ClientID)
		require.Equal(t, sharedWallets[0].PeerPublicKey, splitWallet.PeerPublicKey)
		require.Equal(t, sharedWallets[0].PublicKey, keys.Keys[0].PublicKey)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, oldHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Share not existing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		response, err = zvaultClient.ShareWallet(t, client.X_APP_USER_ID_A, PUBLIC_KEY, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Share split key to the initial split key owner", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

		splitWallet, response, err := zvaultClient.GenerateSplitWallet(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		keys, response, err := zvaultClient.GetKeys(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, keys.Keys, 1)
		require.Equal(t, keys.Keys[0].ClientID, splitWallet.ClientID)

		response, err = zvaultClient.ShareWallet(t, client.X_APP_USER_ID, keys.Keys[0].PublicKey, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		sharedWallets, response, err := zvaultClient.GetSharedWallets(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, sharedWallets, 1)
		require.Equal(t, sharedWallets[0].ClientID, splitWallet.ClientID)
		require.Equal(t, sharedWallets[0].PeerPublicKey, splitWallet.PeerPublicKey)
		require.Equal(t, sharedWallets[0].PublicKey, keys.Keys[0].PublicKey)

		response, err = zvaultClient.Delete(t, splitWallet.ClientID, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}
