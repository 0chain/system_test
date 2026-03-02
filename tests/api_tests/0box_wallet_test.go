package api_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func NewTestWallet() map[string]string {
	return map[string]string{
		"name":        "test_wallet_name",
		"description": "test_wallet_description",
		"mnemonic":    "test_mnemonic",
	}
}

func Create0boxTestWallet(t *test.SystemTest, headers map[string]string) error {
	// Clean state first via SQL teardown
	Teardown(t, headers)

	verifyOtpInput := NewVerifyOtpDetails()
	_, resp, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
	if err != nil {
		return fmt.Errorf("VerifyOtpDetails failed: %w", err)
	}
	// 200 = owner created, 400 with "duplicate key" = owner already exists (both OK)
	if resp.StatusCode() != 200 {
		if strings.Contains(resp.String(), "duplicate key value violates unique constraint") {
			t.Log("Owner already exists, proceeding to wallet creation")
		} else {
			return fmt.Errorf("VerifyOtpDetails returned %d: %s", resp.StatusCode(), resp.String())
		}
	}
	walletInput := NewTestWallet()
	_, resp, err = zboxClient.CreateWallet(t, headers, walletInput)
	if err != nil {
		return fmt.Errorf("CreateWallet failed: %w", err)
	}
	// 201 = wallet created; 400 with "duplicate key" or "Cannot create new wallet" = wallet already exists (both OK).
	// 0box wraps the internal duplicate key as "500: Cannot create new wallet" in its 400 response.
	if resp.StatusCode() != 201 {
		if strings.Contains(resp.String(), "duplicate key value violates unique constraint") ||
			strings.Contains(resp.String(), "Cannot create new wallet") {
			t.Log("Wallet already exists, proceeding")
		} else {
			return fmt.Errorf("CreateWallet returned %d: %s", resp.StatusCode(), resp.String())
		}
	}
	return nil
}

func Test0BoxWallet(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	t.RunSequentially("create wallet without owner should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("create wallet without existing wallet should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
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

	t.RunSequentially("create wallet with existing wallet should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		_, response, err = zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("create wallet with existing wallet another apptype should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// 0box no longer allows creating a second wallet for the same user
		// with a different apptype. It returns 400 "Cannot create new wallet".
		newHeaders := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_CHIMNEY)
		_, response, err = zboxClient.CreateWallet(t, newHeaders, walletInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("create wallet with existing wallet same apptype should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers["X-App-Client-ID"] = "new_client_id"
		_, response, err = zboxClient.CreateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("update wallet with existing wallet should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
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
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)

		walletInput := NewTestWallet()
		message, response, err := zboxClient.UpdateWallet(t, headers, walletInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "no wallet was updated for these details", message.Message)
	})
}
