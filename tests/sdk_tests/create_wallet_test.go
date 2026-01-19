package sdk_tests

import (
	"encoding/json"
	"testing"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestCreateWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	//Basic success
	t.RunSequentially("Successfully create wallet string", func(t *test.SystemTest) {
		walletStr, err := zcncore.CreateWalletOffline()
		require.NoError(t, err, "should create wallet without error")
		require.NotEmpty(t, walletStr, "wallet string should not be empty")

	})
	// Valid Json Structure
	t.RunSequentially("Returns valid json", func(t *test.SystemTest) {
		walletStr, err := zcncore.CreateWalletOffline()

		var wallet map[string]interface{}
		json.Unmarshal([]byte(walletStr), &wallet)
		require.NoError(t, err, "Wallet should be valid Json")

		// Check top-level fields
		require.Contains(t, wallet, "client_id", "Missing client_id")
		require.Contains(t, wallet, "client_key", "Missing client_id")
		require.Contains(t, wallet, "keys", "Missing keys")
		require.Contains(t, wallet, "mnemonics", "Missing mnemonic")
		require.Contains(t, wallet, "version", "Missing version")
		require.Contains(t, wallet, "date_created", "Missing date_created")
		require.Contains(t, wallet, "nonce", "Missing nonce")
		require.Contains(t, wallet, "is_split", "Missing is_split")

		// Verify is_split is boolean false
		require.Equal(t, wallet["is_split"].(bool), false, "is_split is not set to false ")

		// Check keys array exists and has content
		keys, ok := wallet["keys"].([]interface{})
		require.True(t, ok, "keys should be an array")
		require.NotEmpty(t, keys, "should have at least one key pair")

		// Check first key has public and private key
		firstKey := keys[0].(map[string]interface{})
		require.Contains(t, firstKey, "public_key")
		require.Contains(t, firstKey, "private_key")

		// Verify keys are not empty
		require.NotEmpty(t, firstKey["public_key"])
		require.NotEmpty(t, firstKey["private_key"])

	})

	t.RunSequentially("Creates unique wallets", func(t *test.SystemTest) {
		const numWallets = 5
		seen := make(map[string]bool)

		for i := 0; i < numWallets; i++ {
			wallet, err := zcncore.CreateWalletOffline()
			require.NoError(t, err, "should create wallet %d", i+1)

			// Check if we've seen this wallet before
			require.False(t, seen[wallet], "wallet %d is duplicate", i+1)

			// Mark as seen
			seen[wallet] = true
		}

		require.Equal(t, numWallets, len(seen), "should have %d unique wallets", numWallets)
		t.Logf("Created and verified %d unique wallets", numWallets)
	})

}
