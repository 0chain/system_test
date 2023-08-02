package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/fatih/structs"
)

type BridgeConfig struct {
	Fields struct {
		BurnAddress          string `json:"burn_address"`
		AddAuthorizerCost    string `json:"cost.add-authorizer"`
		BurnCost             string `json:"cost.burn"`
		DeleteAuthorizerCost string `json:"cost.delete-authorizer"`
		MintCost             string `json:"cost.mint"`
		MaxDelegates         string `json:"max_delegates"`
		MaxFee               string `json:"max_fee"`
		MaxStake             string `json:"max_stake"`
		MinAuthorizers       string `json:"min_authorizers"`
		MinBurn              string `json:"min_burn"`
		MinLock              string `json:"min_lock"`
		MinMint              string `json:"min_mint"`
		MinStake             string `json:"min_stake"`
		OwnerID              string `json:"owner_id"`
		AuthorizersPercent   string `json:"percent_authorizers"`
	} `json:"fields"`
}

func TestZCNBridgeGlobalSettings(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("should allow update of min_mint_amount")

	output, err := executeFaucetWithTokensForWallet(t, zcnscOwner, configPath, 10.0)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	defaultParams := getDefaultConfig(t)

	t.Cleanup(func() {
		for _, defaultParam := range defaultParams {
			output, err := updateZCNBridgeSCConfig(t, zcnscOwner, createConfigParams(defaultParam), true)

			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			require.Equal(t, "faucet smart contract settings updated", output[0], strings.Join(output, "\n"))
			require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
		}
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

func getDefaultConfig(t *test.SystemTest) []map[string]string {
	output, err := getZCNBridgeGlobalSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	match := regexp.MustCompile(`{"fields":\s*({.*?})}`).FindAllString(strings.Join(output, " "), 1)[0]

	var resultRaw BridgeConfig
	err = json.Unmarshal([]byte(match), &resultRaw)
	require.Nil(t, err)

	var result []map[string]string

	fields := structs.Fields(resultRaw.Fields)
	for _, field := range fields {
		result = append(result, map[string]string{
			field.Tag("json"): field.Value().(string),
		})
	}

	return result
}

func testKey(t *test.SystemTest, key, value string) {
	cfgAfter := updateAndVerify(t, key, value)
	require.Equal(t, value, cfgAfter[key], "new value %s for config %s was not set", value, key)
}

func createConfigParams(params map[string]string) map[string]interface{} {
	var (
		keys   []string
		values []string
	)
	for k, v := range params {
		keys = append(keys, k)
		values = append(values, v)
	}

	return map[string]interface{}{
		"keys":   strings.Join(keys, ","),
		"values": strings.Join(values, ","),
	}
}

func updateAndVerify(t *test.SystemTest, key, value string) map[string]string {
	params := createConfigParams(map[string]string{
		key: value,
	})

	output, err := updateZCNBridgeSCConfig(t, zcnscOwner, params, true)

	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2, strings.Join(output, "\n"))
	require.Equal(t, "global settings updated", output[0], strings.Join(output, "\n"))
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
