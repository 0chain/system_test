package cli_tests

import (
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestOwnerUpdate(t *testing.T) {
	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
	require.Nil(t, err, "error fetching minerNodeDelegate nonce")
	nonceStr := strings.Split(ret[0], ":")[1]
	nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
	require.Nil(t, err, "error converting nonce to in")

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))
	newOwnerWallet, err := getWallet(t, configPath)
	require.Nil(t, err, "error fetching wallet")

	newOwnerName := escapedTestName(t)
	newRet, err := getNonceForWallet(t, configPath, newOwnerName, true)
	require.Nil(t, err, "error fetching minerNodeDelegate nonce")
	newNonceStr := strings.Split(newRet[0], ":")[1]
	newNonce, err := strconv.ParseInt(strings.Trim(newNonceStr, " "), 10, 64)
	require.Nil(t, err, "error converting nonce to in")

	t.Run("should allow update of owner: StorageSC", func(t *testing.T) {
		n := atomic.AddInt64(&nonce, 2)
		newN := atomic.AddInt64(&newNonce, 1)

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateStorageSCConfigWithNonce(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, newN, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateStorageSCConfigWithNonce(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, n-1, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getStorageSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		cfgAfter, _ := keyValuePairStringToMap(t, output)
		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// Updating config with old owner should fail
		output, err = updateStorageSCConfigWithNonce(t, scOwnerWallet, map[string]interface{}{
			"keys":   "max_read_price",
			"values": 99,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0])

	})

	t.Run("should allow update of owner: VestingSC", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		n := atomic.AddInt64(&nonce, 2)
		newN := atomic.AddInt64(&newNonce, 1)

		t.Cleanup(func() {
			output, err := updateVestingPoolSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, newN, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, n-1, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// should fail
		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   "max_destinations",
			"values": 25,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("should allow update of owner: MinerSC", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		n := atomic.AddInt64(&nonce, 2)
		newN := atomic.AddInt64(&newNonce, 1)

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateMinerSCConfigWithNonce(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, newN, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateMinerSCConfigWithNonce(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, n-1, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// Should fail update with old owner
		configKey := "interest_rate"
		newValue := "0.1"
		output, err = updateMinerSCConfigWithNonce(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("Should allow update owner: InterestSC", func(t *testing.T) {
		t.Skip("interest pool SC will be deprecated soon")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		n := atomic.AddInt64(&nonce, 2)
		newN := atomic.AddInt64(&newNonce, 1)

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateInterestPoolSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, newN, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		configKey := "min_lock"
		newValue := "8"

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, n-1, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "interest pool smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_variables: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("should allow update of owner: FaucetSC", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		n := atomic.AddInt64(&nonce, 2)
		newN := atomic.AddInt64(&newNonce, 1)

		newOwnerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateFaucetSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, newN, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateFaucetSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, n-1, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "faucet smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getFaucetSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// Should fail update with old owner
		configKey := "max_pour_amount"
		newValue := "15"
		output, err = updateFaucetSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})
}
