package cli_tests

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestVestingPoolUpdateConfig(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	t.Run("should allow update of max_destinations", func(t *testing.T) {
		t.Parallel()

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
		require.Nil(t, err, "error fetching minerNodeDelegate nonce")
		nonceStr := strings.Split(ret[0], ":")[1]
		nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
		require.Nil(t, err, "error converting nonce to in")

		configKey := "max_destinations"
		newValue := "4"
		n := atomic.AddInt64(&nonce, 2)

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgBefore, _ := keyValuePairStringToMap(t, output)

		// ensure revert in config is run regardless of test result
		defer func() {
			oldValue := cfgBefore[configKey]
			output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": oldValue,
			}, n, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
			require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
		}()

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, n-1, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(t, output)
		require.Equal(t, newValue, cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	t.Run("update max_destinations to invalid value should fail", func(t *testing.T) {
		t.Parallel()

		ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
		require.Nil(t, err, "error fetching minerNodeDelegate nonce")
		nonceStr := strings.Split(ret[0], ":")[1]
		nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
		require.Nil(t, err, "error converting nonce to in")

		configKey := "max_destinations"
		newValue := "x"
		n := atomic.AddInt64(&nonce, 1)

		output, err := updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: value x cannot be converted to time.Duration, failing to set config key max_destinations",
			output[0], strings.Join(output, "\n"))
	})

	t.Run("update by non-smartcontract owner should fail", func(t *testing.T) {
		t.Parallel()

		configKey := "max_destinations"
		newValue := "4"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, escapedTestName(t), map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, 2, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with bad config key should fail", func(t *testing.T) {
		t.Parallel()

		configKey := "unknown_key"

		ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
		require.Nil(t, err, "error fetching minerNodeDelegate nonce")
		nonceStr := strings.Split(ret[0], ":")[1]
		nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
		require.Nil(t, err, "error converting nonce to in")

		n := atomic.AddInt64(&nonce, 1)
		output, err := updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": 1,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: config setting unknown_key not found", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with missing keys param should fail", func(t *testing.T) {
		t.Parallel()

		ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
		require.Nil(t, err, "error fetching minerNodeDelegate nonce")
		nonceStr := strings.Split(ret[0], ":")[1]
		nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
		require.Nil(t, err, "error converting nonce to in")

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))
		n := atomic.AddInt64(&nonce, 1)

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"values": 1,
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})

	t.Run("update with missing values param should fail", func(t *testing.T) {
		t.Parallel()

		ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
		require.Nil(t, err, "error fetching minerNodeDelegate nonce")
		nonceStr := strings.Split(ret[0], ":")[1]
		nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
		require.Nil(t, err, "error converting nonce to in")

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register SC owner wallet
		output, err = registerWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		n := atomic.AddInt64(&nonce, 1)
		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys": "max_destinations",
		}, n, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})
}

func getVestingPoolSCConfig(t *testing.T, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Retrieving vesting config...")

	cmd := "./zwallet vp-config --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateVestingPoolSCConfig(t *testing.T, walletName string, param map[string]interface{}, nonce int64, retry bool) ([]string, error) {
	t.Logf("Updating vesting config...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zwallet vp-update-config %s --silent --withNonce %v --wallet %s --configDir ./config --config %s",
		p,
		nonce,
		walletName+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
