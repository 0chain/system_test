package cli_tests

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOwnerUpdate(t *testing.T) {
	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))
	newOwnerWallet, err := getWallet(t, configPath)
	require.Nil(t, err, "error fetching wallet")

	newOwnerName := escapedTestName(t)

	t.Run("should allow update of owner: StorageSC", func(t *testing.T) {
		t.Skip("Skip till fixed")
		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateStorageSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getStorageSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		cfgAfter, _ := keyValuePairStringToMap(t, output)
		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// Updating config with old owner should fail
		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   "max_read_price",
			"values": 99,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0])

	})

	t.Run("should allow update of owner: VestingSC", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateVestingPoolSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
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
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("should allow update of owner: MinerSC", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateMinerSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
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
		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("Should allow update owner: InterestSC", func(t *testing.T) {
		t.Skip("Interest pool is deprecated...")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateInterestPoolSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		configKey := "min_lock"
		newValue := "8"

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "interest pool smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_variables: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("should allow update of owner: FaucetSC", func(t *testing.T) {
		t.Skip("Skip till fixed")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateFaucetSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err = updateFaucetSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
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
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})
}
