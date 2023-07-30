package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZCNBridgeGlobalSettings(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("should allow update of min_mint_amount")

	defaultConfig := getDefaultConfig(t)

	t.Cleanup(func() {
		output, err := updateZCNBridgeSCConfig(t, zcnscOwner, defaultConfig, true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "faucet smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
	})

	t.RunSequentially("should allow update of min_mint_amount", func(t *test.SystemTest) {
		testKey(t, "min_mint", "1")
	})

	t.RunSequentially("should allow update of min_burn_amount", func(t *test.SystemTest) {
		testKey(t, "min_burn", "2")
	})

	t.RunSequentially("should allow update of min_stake_amount", func(t *test.SystemTest) {
		testKey(t, "min_stake", "3")
	})

	t.RunSequentially("should allow update of max_fee", func(t *test.SystemTest) {
		testKey(t, "max_fee", "4")
	})

	t.RunSequentially("should allow update of percent_authorizers", func(t *test.SystemTest) {
		testKey(t, "percent_authorizers", "5")
	})

	t.RunSequentially("should allow update of min_authorizers", func(t *test.SystemTest) {
		testKey(t, "min_authorizers", "6")
	})

	t.RunSequentially("should allow update of burn_address", func(t *test.SystemTest) {
		testKey(t, "burn_address", "7")
	})

	t.RunSequentially("should allow update of owner_id", func(t *test.SystemTest) {
		testKey(t, "owner_id", "8")
	})
}

func getDefaultConfig(t *test.SystemTest) map[string]interface{} {
	output, err := getZCNBridgeGlobalSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	defaultConfig, _ := keyValuePairStringToMap(output)

	result := make(map[string]interface{})
	for k, v := range defaultConfig {
		result[k] = v
	}

	return result
}

func testKey(t *test.SystemTest, key, value string) {
	cfgAfter := updateAndVerify(t, key, value)
	require.Equal(t, value, cfgAfter[key], "new value %s for config %s was not set", value, key)
}

func updateAndVerify(t *test.SystemTest, key, value string) map[string]string {
	output, err := updateZCNBridgeSCConfig(t, zcnscOwner, map[string]interface{}{
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

func getZCNBridgeGlobalSCConfig(t *test.SystemTest, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Log("Retrieving zcnc bridge global config...")

	cmd :=
		"./zwallet bridge-config --silent --wallet " +
			zcnscOwner +
			"_wallet.json --configDir ./config --config " +
			cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateZCNBridgeSCConfig(t *test.SystemTest, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
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
