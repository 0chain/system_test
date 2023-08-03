package cli_tests

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/tokenomics"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

type BridgeConfig struct {
	Fields struct {
		BurnAddress        string `json:"burn_address"`
		MaxDelegates       string `json:"max_delegates"`
		MaxFee             string `json:"max_fee"`
		MaxStake           string `json:"max_stake"`
		MinAuthorizers     string `json:"min_authorizers"`
		MinBurn            string `json:"min_burn"`
		MinLock            string `json:"min_lock"`
		MinMint            string `json:"min_mint"`
		MinStake           string `json:"min_stake"`
		OwnerID            string `json:"owner_id"`
		AuthorizersPercent string `json:"percent_authorizers"`
	} `json:"fields"`
}

func TestZCNBridgeGlobalSettings(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("should allow update of min_mint_amount")

	output, err := executeFaucetWithTokensForWallet(t, zcnscOwner, configPath, 10.0)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	defaultParams := getDefaultConfig(t)

	t.RunSequentially("should allow update of min_mint_amount", func(t *test.SystemTest) {
		t.Cleanup(func() {
			_ = updateAndVerify(t, "min_mint", fmt.Sprintf("%v", tokenomics.ZcnToInt(defaultParams["min_mint"])))
		})
		cfgAfter := updateAndVerify(t, "min_mint", "1")

		resultInt, err := strconv.Atoi(cfgAfter["min_mint"])
		require.NoError(t, err)

		require.Equal(t, 10000000000, resultInt, "new value for config min_mint was not set")
	})

	t.RunSequentially("should allow update of min_burn_amount", func(t *test.SystemTest) {
		t.Cleanup(func() {
			_ = updateAndVerify(t, "min_burn", fmt.Sprintf("%v", tokenomics.ZcnToInt(defaultParams["min_burn"])))
		})
		cfgAfter := updateAndVerify(t, "min_burn", "2")

		resultInt, err := strconv.Atoi(cfgAfter["min_burn"])
		require.NoError(t, err)

		require.Equal(t, 20000000000, resultInt, "new value for config min_burn was not set")
	})

	t.RunSequentially("should allow update of min_stake_amount", func(t *test.SystemTest) {
		t.Cleanup(func() {
			_ = updateAndVerify(t, "min_stake", fmt.Sprintf("%v", tokenomics.ZcnToInt(defaultParams["min_stake"])))
		})
		cfgAfter := updateAndVerify(t, "min_stake", "3")

		resultInt, err := strconv.Atoi(cfgAfter["min_stake"])
		require.NoError(t, err)

		require.Equal(t, 30000000000, resultInt, "new value for config min_stake was not set")
	})

	t.RunSequentially("should allow update of max_fee", func(t *test.SystemTest) {
		t.Cleanup(func() {
			_ = updateAndVerify(t, "max_fee", "100")
		})
		cfgAfter := updateAndVerify(t, "max_fee", "4")

		resultInt, err := strconv.Atoi(cfgAfter["max_fee"])
		require.NoError(t, err)

		require.Equal(t, 40000000000, resultInt, "new value for config max_fee was not set")
	})

	t.RunSequentially("should allow update of percent_authorizers", func(t *test.SystemTest) {
		t.Cleanup(func() {
			_ = updateAndVerify(t, "percent_authorizers", fmt.Sprintf("%v", defaultParams["percent_authorizers"]))
		})
		cfgAfter := updateAndVerify(t, "percent_authorizers", "5")

		resultInt, err := strconv.Atoi(cfgAfter["percent_authorizers"])
		require.NoError(t, err)

		require.Equal(t, 5, resultInt, "new value for config percent_authorizers was not set")
	})

	t.RunSequentially("should allow update of min_authorizers", func(t *test.SystemTest) {
		t.Cleanup(func() {
			_ = updateAndVerify(t, "min_authorizers", fmt.Sprintf("%v", defaultParams["min_authorizers"]))
		})
		cfgAfter := updateAndVerify(t, "min_authorizers", "6")

		resultInt, err := strconv.Atoi(cfgAfter["min_authorizers"])
		require.NoError(t, err)

		require.Equal(t, 6, resultInt, "new value for config min_authorizers was not set")
	})

	t.RunSequentially("should allow update of burn_address", func(t *test.SystemTest) {
		t.Cleanup(func() {
			_ = updateAndVerify(t, "burn_address", "0000000000000000000000000000000000000000000000000000000000000000")
		})
		cfgAfter := updateAndVerify(t, "burn_address", "7")

		resultInt, err := strconv.Atoi(cfgAfter["burn_address"])
		require.NoError(t, err)

		require.Equal(t, 7, resultInt, "new value for config burn_address was not set")
	})

	t.RunSequentially("should allow update of owner_id", func(t *test.SystemTest) {
		newOwner := escapedTestName(t)

		output, err := createWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		t.Cleanup(func() {
			zcnscOwnerWallet, err := getWalletForName(t, configPath, zcnscOwner)
			require.Nil(t, err)
			_ = updateAndVerifyWithWallet(t, "owner_id", zcnscOwnerWallet.ClientID, newOwner)
		})

		cfgAfter := updateAndVerify(t, "owner_id", newOwnerWallet.ClientID)

		result := cfgAfter["owner_id"]
		require.NoError(t, err)

		require.Equal(t, newOwnerWallet.ClientID, result, "new value for config owner_id was not set")
	})
}

func getDefaultConfig(t *test.SystemTest) map[string]float64 {
	output, err := getZCNBridgeGlobalSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	_, cfgBefore := keyValuePairStringToMap(output)

	return cfgBefore
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
	return updateAndVerifyWithWallet(t, key, value, zcnscOwner)
}

func updateAndVerifyWithWallet(t *test.SystemTest, key, value, walletName string) map[string]string {
	params := createConfigParams(map[string]string{
		key: value,
	})

	output, err := updateZCNBridgeSCConfig(t, walletName, params, true)

	require.Nil(t, err, strings.Join(output, "\n"))
	require.Equal(t, 2, len(output), strings.Join(output, "\n"))
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
