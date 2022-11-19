package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

// This will test the following config

// ZCNSConfig config both for GlobalNode and AuthorizerNode
// type ZCNSConfig struct {
//	MinMintAmount      state.Balance `json:"min_mint_amount"`
//	MinBurnAmount      state.Balance `json:"min_burn_amount"`
//	MinStakeAmount     state.Balance `json:"min_stake_amount"`
//	MaxFee             state.Balance `json:"max_fee"`
//	PercentAuthorizers float64       `json:"percent_authorizers"`
//	MinAuthorizers     int64         `json:"min_authorizers"`
//	BurnAddress        string        `json:"burn_address"`
//	OwnerId            datastore.Key `json:"owner_id"`
// }

var (
	configKey string
	newValue  string
)

func TestZCNBridgeGlobalSettings(testSetup *testing.T) {
	t := test.SystemTest{T: testSetup}

	if _, err := os.Stat("./config/" + zcnscOwner + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+zcnscOwner+"_wallet.json")
	}

	// unused wallet, just added to avoid having the creating new wallet outputs
	output, err := registerWallet(testSetup, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	// register SC owner wallet
	output, err = registerWalletForName(testSetup, configPath, zcnscOwner)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	// get global config
	output, err = getZCNBridgeGlobalSCConfig(testSetup, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	cfgBefore, _ := keyValuePairStringToMap(output)

	// ensure revert in config is run regardless of test result
	defer func() {
		oldValue := cfgBefore[configKey]
		output, err = updateZCNBridgeSCConfig(testSetup, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": oldValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "faucet smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
	}()

	t.Run("should allow update of min_mint_amount", func(t *testing.T) {
		testKey(t, "min_mint_amount", "1")
	})

	t.Run("should allow update of min_burn_amount", func(t *testing.T) {
		testKey(t, "min_burn_amount", "2")
	})

	t.Run("should allow update of min_stake_amount", func(t *testing.T) {
		testKey(t, "min_stake_amount", "3")
	})

	t.Run("should allow update of max_fee", func(t *testing.T) {
		testKey(t, "max_fee", "4")
	})

	t.Run("should allow update of percent_authorizers", func(t *testing.T) {
		testKey(t, "percent_authorizers", "5")
	})

	t.Run("should allow update of min_authorizers", func(t *testing.T) {
		testKey(t, "min_authorizers", "6")
	})

	t.Run("should allow update of burn_address", func(t *testing.T) {
		testKey(t, "burn_address", "7")
	})

	t.Run("should allow update of owner_id", func(t *testing.T) {
		testKey(t, "owner_id", "8")
	})
}

func testKey(t *testing.T, key, value string) {
	cfgAfter := updateAndVerify(t, key, value)
	require.Equal(t, newValue, cfgAfter[key], "new value %s for config %s was not set", value, key)
}

func updateAndVerify(t *testing.T, key, value string) map[string]string {
	output, err := updateZCNBridgeSCConfig(t, scOwnerWallet, map[string]interface{}{
		"keys":   key,
		"values": value,
	}, true)

	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2, strings.Join(output, "\n"))
	require.Equal(t, "zcnsc smart contract settings updated", output[0], strings.Join(output, "\n"))
	require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

	output, err = getZCNBridgeGlobalSCConfig(t, configPath, true)

	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	cfgAfter, _ := keyValuePairStringToMap(output)
	return cfgAfter
}

func getZCNBridgeGlobalSCConfig(t *testing.T, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Log("Retrieving zcnc bridge global config...")

	cmd :=
		"./zwallet bridge-config --silent --wallet " +
			escapedTestName(t) +
			"_wallet.json --configDir ./config --config " +
			cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateZCNBridgeSCConfig(t *testing.T, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Log("Updating zcnsc bridge global config...")

	cmd := fmt.Sprintf(
		"./zwallet bridge-config-update %s --silent --wallet %s --configDir ./config --config %s",
		createParams(param),
		walletName+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
