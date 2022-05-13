package cli_tests

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestOwnerUpdate(t *testing.T) {
	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	t.Run("should allow update of owner: StorageSC", func(t *testing.T) {
		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet")

		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))

		cliutils.Wait(t, 1*time.Minute)

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

		t.Cleanup(func() {
			output, err := updateStorageSCConfig(t, escapedTestName(t), map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			cliutils.Wait(t, 1*time.Minute)
		})
	})

	t.Run("should allow update of owner: VestingSC", func(t *testing.T) {
		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
		require.Nil(t, err, "error fetching minerNodeDelegate nonce")
		nonceStr := strings.Split(ret[0], ":")[1]
		nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
		require.Nil(t, err, "error converting nonce to in")

		n := atomic.AddInt64(&nonce, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, n, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))

		cliutils.Wait(t, 1*time.Minute)

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)

		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// should fail
		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   "max_destinations",
			"values": 25,
		}, n+1, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))

		t.Cleanup(func() {
			ret, err := getNonceForWallet(t, configPath, escapedTestName(t), true)
			require.Nil(t, err, "error fetching wallet nonce")
			nonceStr := strings.Split(ret[0], ":")[1]
			nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
			require.Nil(t, err, "error converting nonce to in")
			n := atomic.AddInt64(&nonce, 1)
			output, err := updateVestingPoolSCConfig(t, escapedTestName(t), map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, n, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			cliutils.Wait(t, 1*time.Minute)
		})
	})

	t.Run("should allow update of owner: MinerSC", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))

		cliutils.Wait(t, 1*time.Minute)

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

		t.Cleanup(func() {
			output, err := updateMinerSCConfig(t, escapedTestName(t), map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			cliutils.Wait(t, 1*time.Minute)
		})
	})

	t.Run("Should allow update owner: InterestSC", func(t *testing.T) {
		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		configKey := "min_lock"
		newValue := "8"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "interest pool smart contract settings updated", output[0], strings.Join(output, "\n"))

		cliutils.Wait(t, 1*time.Minute)

		output, err = updateInterestPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_variables: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))

		t.Cleanup(func() {
			output, err := updateInterestPoolSCConfig(t, escapedTestName(t), map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			cliutils.Wait(t, 1*time.Minute)
		})
	})

	t.Run("should allow update of owner: FaucetSC", func(t *testing.T) {
		ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
		require.Nil(t, err, "error fetching minerNodeDelegate nonce")
		nonceStr := strings.Split(ret[0], ":")[1]
		nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
		require.Nil(t, err, "error converting nonce to in")

		n := atomic.AddInt64(&nonce, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		output, err = updateFaucetSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, n, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "faucet smart contract settings updated", output[0], strings.Join(output, "\n"))

		cliutils.Wait(t, 1*time.Minute)

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
		}, n+1, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))

		t.Cleanup(func() {
			ret, err := getNonceForWallet(t, configPath, escapedTestName(t), true)
			require.Nil(t, err, "error fetching wallet nonce")
			nonceStr := strings.Split(ret[0], ":")[1]
			nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
			require.Nil(t, err, "error converting nonce to in")
			n := atomic.AddInt64(&nonce, 1)
			output, err := updateFaucetSCConfig(t, escapedTestName(t), map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, n, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			cliutils.Wait(t, 1*time.Minute)
		})
	})
}
