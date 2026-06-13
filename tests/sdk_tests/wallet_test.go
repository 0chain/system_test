package sdk_tests

import (
	"testing"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestWalletOperations(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Test core wallet operations")

	t.Run("Create wallet offline", func(t *test.SystemTest) {
		wallet, err := CreateWallet(t)
		require.NoError(t, err, "Should create wallet")
		AssertWalletValid(t, wallet)
		t.Logf("Created wallet: %s", wallet.ClientID)
	})

	t.Run("Recover wallet from mnemonic", func(t *test.SystemTest) {
		// First create a wallet to get a mnemonic
		wallet1, err := CreateWallet(t)
		require.NoError(t, err, "Should create wallet")
		require.NotEmpty(t, wallet1.Mnemonics, "Wallet should have mnemonics")

		// Recover wallet from mnemonic
		wallet2, err := CreateWalletFromMnemonic(t, wallet1.Mnemonics)
		require.NoError(t, err, "Should recover wallet from mnemonic")
		AssertWalletValid(t, wallet2)

		// Verify recovered wallet matches original
		require.Equal(t, wallet1.ClientID, wallet2.ClientID, "Client IDs should match")
		require.Equal(t, wallet1.ClientKey, wallet2.ClientKey, "Client keys should match")
		require.Equal(t, wallet1.Keys[0].PublicKey, wallet2.Keys[0].PublicKey, "Public keys should match")
	})

	t.Run("Wallet JSON serialization", func(t *test.SystemTest) {
		wallet, err := CreateWallet(t)
		require.NoError(t, err, "Should create wallet")

		// Convert to JSON
		walletJSON, err := GetWalletJSON(wallet)
		require.NoError(t, err, "Should serialize wallet to JSON")
		require.NotEmpty(t, walletJSON, "JSON should not be empty")

		t.Logf("Wallet JSON length: %d bytes", len(walletJSON))
	})

	// t.Run("Split key generation", func(t *test.SystemTest) {
	// 	wallet, err := CreateWallet(t)
	// 	require.NoError(t, err, "Should create wallet")
	// 	require.NotEmpty(t, wallet.Keys, "Wallet should have keys")

	// 	privateKey := wallet.Keys[0].PrivateKey

	// 	// Test split key generation with different split counts
	// 	for _, numSplits := range []int{2, 3, 4} {
	// 		splitKeys, err := zcncore.SplitKeys(privateKey, numSplits)
	// 		require.NoError(t, err, "Should split key into %d parts", numSplits)
	// 		require.NotEmpty(t, splitKeys, "Split keys should not be empty")
	// 		t.Logf("Split key into %d parts, result length: %d bytes", numSplits, len(splitKeys))
	// 	}
	// })

	t.Run("Encrypt and decrypt", func(t *test.SystemTest) {
		key := "test_encryption_key_123456789012"
		plaintext := "This is a secret message"

		// Encrypt
		ciphertext, err := zcncore.Encrypt(key, plaintext)
		require.NoError(t, err, "Should encrypt text")
		require.NotEmpty(t, ciphertext, "Ciphertext should not be empty")
		require.NotEqual(t, plaintext, ciphertext, "Ciphertext should differ from plaintext")

		// Decrypt
		decrypted, err := zcncore.Decrypt(key, ciphertext)
		require.NoError(t, err, "Should decrypt text")
		require.Equal(t, plaintext, decrypted, "Decrypted text should match original")

		t.Logf("Encrypted: %s -> %s", plaintext, ciphertext)
	})

	t.Run("Sign hash with wallet", func(t *test.SystemTest) {
		wallet, err := CreateWallet(t)
		require.NoError(t, err, "Should create wallet")
		require.NotEmpty(t, wallet.Keys, "Wallet should have keys")

		privateKey := wallet.Keys[0].PrivateKey
		// Use a valid hex string for the hash (64 chars = 32 bytes)
		hash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		// Sign hash
		signature, err := zcncore.SignWithKey(privateKey, hash)
		require.NoError(t, err, "Should sign hash")
		require.NotEmpty(t, signature, "Signature should not be empty")

		t.Logf("Signature: %s", signature)
	})

	t.Run("Set wallet info", func(t *test.SystemTest) {
		wallet, err := CreateWallet(t)
		require.NoError(t, err, "Should create wallet")

		// Set wallet as active
		err = SetWallet(t, wallet)
		require.NoError(t, err, "Should set wallet info")

		t.Logf("Set wallet %s as active", wallet.ClientID)
	})
}
