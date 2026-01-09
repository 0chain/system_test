package sdk_tests

import (
	"sync"
	"testing"

	"github.com/0chain/gosdk/core/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestMultiWalletManagement(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Test multi-wallet management and edge cases")

	t.Run("AddWallet and GetWalletByKey", func(t *test.SystemTest) {
		wallets, err := LoadWalletsFromPool(1)
		require.NoError(t, err)
		wallet := wallets[0]

		err = AddWalletToSDK(t, wallet)
		require.NoError(t, err)

		pubkey := GetWalletPublicKey(wallet)
		
		// Verify wallet exists
		retrieved := client.GetWalletByKey(pubkey)
		require.NotNil(t, retrieved, "Wallet should exist after adding")
		require.Equal(t, wallet.ClientID, retrieved.ClientID)

		t.Logf("Successfully verified AddWallet and GetWalletByKey")
	})

	t.Run("Concurrent wallet operations", func(t *test.SystemTest) {
		wallets, err := LoadWalletsFromPool(5)
		require.NoError(t, err)

		var wg sync.WaitGroup
		errors := make(chan error, len(wallets)*2)

		// Concurrently add wallets
		for i, wallet := range wallets {
			wg.Add(1)
			go func(w *Wallet, idx int) {
				defer wg.Done()
				if err := AddWalletToSDK(t, w); err != nil {
					errors <- err
				}
			}(wallet, i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			require.NoError(t, err, "Concurrent wallet addition should not error")
		}

		// Verify all wallets are accessible
		for _, wallet := range wallets {
			pubkey := GetWalletPublicKey(wallet)
			retrieved := client.GetWalletByKey(pubkey)
			require.NotNil(t, retrieved, "Wallet should be retrievable after concurrent add")
		}

		t.Logf("Successfully verified concurrent wallet operations")
	})

	t.Run("GetWalletByKey with non-existent key", func(t *test.SystemTest) {
		fakeKey := "nonexistent_key_12345"
		retrieved := client.GetWalletByKey(fakeKey)
		require.Nil(t, retrieved, "GetWalletByKey should return nil for non-existent key")
		
		t.Logf("Successfully verified non-existent key handling")
	})

	t.Run("PublicKey lookup with specific key", func(t *test.SystemTest) {
		wallets, err := LoadWalletsFromPool(1)
		require.NoError(t, err)
		wallet := wallets[0]

		err = AddWalletToSDK(t, wallet)
		require.NoError(t, err)

		pubkey := GetWalletPublicKey(wallet)
		
		// Lookup using the key
		retrievedKey := client.PublicKey(pubkey)
		require.Equal(t, wallet.ClientKey, retrievedKey, "PublicKey should return correct client key")

		t.Logf("Successfully verified PublicKey lookup with specific key")
	})

	t.Run("Id lookup with specific key", func(t *test.SystemTest) {
		wallets, err := LoadWalletsFromPool(1)
		require.NoError(t, err)
		wallet := wallets[0]

		err = AddWalletToSDK(t, wallet)
		require.NoError(t, err)

		pubkey := GetWalletPublicKey(wallet)
		
		// Lookup using the key
		retrievedID := client.Id(pubkey)
		require.Equal(t, wallet.ClientID, retrievedID, "Id should return correct client ID")

		t.Logf("Successfully verified Id lookup with specific key")
	})

	t.Run("IsWalletSplit with specific key", func(t *test.SystemTest) {
		wallets, err := LoadWalletsFromPool(1)
		require.NoError(t, err)
		wallet := wallets[0]

		err = AddWalletToSDK(t, wallet)
		require.NoError(t, err)

		pubkey := GetWalletPublicKey(wallet)
		
		isSplit, err := client.IsWalletSplit(pubkey)
		require.NoError(t, err)
		require.False(t, isSplit, "Test wallets should not be split-key wallets")

		t.Logf("Successfully verified IsWalletSplit with specific key")
	})
}
