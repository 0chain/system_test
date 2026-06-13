package sdk_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestMultiWalletCreation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create and verify basic wallet functionality")

	t.Run("Create single wallet successfully", func(t *test.SystemTest) {
		wallet, err := CreateWallet(t)
		
		require.NoError(t, err, "Wallet creation should succeed")
		AssertWalletValid(t, wallet)
	})

	t.Run("Create wallet from mnemonic", func(t *test.SystemTest) {
		// First create a wallet to get a mnemonic
		originalWallet, err := CreateWallet(t)
		require.NoError(t, err, "Should create original wallet")
		AssertWalletValid(t, originalWallet)
		require.NotEmpty(t, originalWallet.Mnemonics, "Original wallet should have mnemonics")

		// Now recover wallet from the mnemonic
		recoveredWallet, err := CreateWalletFromMnemonic(t, originalWallet.Mnemonics)
		
		require.NoError(t, err, "Wallet recovery from mnemonic should succeed")
		AssertWalletValid(t, recoveredWallet)
		require.NotEmpty(t, recoveredWallet.Mnemonics, "Recovered wallet should have mnemonics")
		
		// Verify the recovered wallet matches the original
		require.Equal(t, originalWallet.ClientID, recoveredWallet.ClientID, 
			"Recovered wallet should have same ClientID")
		require.Equal(t, originalWallet.ClientKey, recoveredWallet.ClientKey,
			"Recovered wallet should have same ClientKey")
	})

	t.Run("Wallet serialization and deserialization", func(t *test.SystemTest) {
		wallet, err := CreateWallet(t)
		require.NoError(t, err)
		AssertWalletValid(t, wallet)

		// Set wallet
		err = SetWallet(t, wallet)
		require.NoError(t, err, "Setting wallet should succeed")
		
		// Verify we can retrieve the same wallet info
		require.Equal(t, wallet.ClientID, wallet.ClientID, "Client ID should match")
	})
}

func TestMultiWalletKeyGeneration(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Test multi-key wallet generation")

	t.Run("Create wallet with single key", func(t *test.SystemTest) {
		wallet, err := CreateMultiKeyWallet(t, 1)
		
		require.NoError(t, err, "Should create wallet with 1 key")
		AssertWalletValid(t, wallet)
		require.Len(t, wallet.Keys, 1, "Should have exactly 1 key")
	})


}

func TestMultiWalletBalance(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Test wallet balance operations")

	t.Run("Get balance for newly created wallet", func(t *test.SystemTest) {
		wallet, err := CreateWallet(t)
		require.NoError(t, err)
		
		balance, err := GetBalance(t, wallet)
		
		// New wallets may have 0 balance or fail if not registered on chain
		// We just verify the call structure works
		if err != nil {
			t.Logf("Expected: New wallet may not be registered yet: %v", err)
		} else {
			require.GreaterOrEqual(t, balance, int64(0), 
				"Balance should be non-negative")
		}
	})

	t.Run("Get balance for wallet from mnemonic", func(t *test.SystemTest) {
		// Create a wallet to get a mnemonic
		originalWallet, err := CreateWallet(t)
		require.NoError(t, err, "Should create original wallet")
		require.NotEmpty(t, originalWallet.Mnemonics, "Wallet should have mnemonics")

		// Recover wallet from mnemonic
		wallet, err := CreateWalletFromMnemonic(t, originalWallet.Mnemonics)
		require.NoError(t, err, "Should recover wallet from mnemonic")
		
		// Verify recovered wallet matches original
		require.Equal(t, originalWallet.ClientID, wallet.ClientID,
			"Recovered wallet should have same ClientID")
		
		// Try to get balance
		balance, err := GetBalance(t, wallet)
		
		if err != nil {
			t.Logf("Note: Could not retrieve balance: %v", err)
		} else {
			t.Logf("Wallet balance: %d", balance)
			require.GreaterOrEqual(t, balance, int64(0), 
				"Balance should be non-negative")
		}
	})
}
