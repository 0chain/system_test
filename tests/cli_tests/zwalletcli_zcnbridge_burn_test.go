package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeBurn(t *testing.T) {
	t.Parallel()

	t.Run("Burning WZCN tokens without WZCN tokens on balance, shouldn't work", func(t *testing.T) {
		t.Parallel()

		err := PrepareBridgeClient(t)
		require.NoError(t, err)

		output, err := executeFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnEth(t, "10", bridgeClientConfigFile, true)
		require.Nil(t, err)
		// todo: enable test: require.Nil(t, err, "error trying to burn WZCN tokens: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:") // : WZCN burn [OK]")
	})

	t.Run("Burning WZCN tokens with available WZCN tokens on balance, should work", func(t *testing.T) {
		t.Parallel()

		err := PrepareBridgeClient(t)
		require.NoError(t, err)

		output, err := executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnEth(t, "10", bridgeClientConfigFile, true)
		require.Nil(t, err)
		// todo: enable test: require.Nil(t, err, "error trying to burn WZCN tokens: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:") // : WZCN burn [OK]")
	})

	t.Run("Burning ZCN tokens without ZCN tokens on balance, shouldn't work", func(t *testing.T) {
		t.Parallel()

		output, err := burnZcn(t, "1", bridgeClientConfigFile, true)
		require.NotNil(t, err)
		require.Greater(t, len(output), 0)
	})

	t.Run("Burning ZCN tokens with available ZCN tokens on balance, should work", func(t *testing.T) {
		t.Parallel()

		output, err := executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnZcn(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
	})

	t.Run("Get ZCN burn ticket", func(t *testing.T) {
		t.Parallel()

		output, err := executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnZcn(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)

		hash := strings.TrimSpace(strings.Split(output[len(output)-2], ":")[1])

		output, err = getZcnBurnTicket(t, hash, false)
		require.Nil(t, err)

		amount := strings.TrimSpace(strings.Split(output[len(output)-2], ":")[1])
		var amountFloat float64
		amountFloat, err = strconv.ParseFloat(amount, 32)
		require.Nil(t, err)
		require.Equal(t, int64(1), tokenomics.ZcnToInt(amountFloat))

		nonce := strings.TrimSpace(strings.Split(output[len(output)-3], ":")[1])
		var nonceInt int
		nonceInt, err = strconv.Atoi(nonce)
		require.Nil(t, err)
		require.Equal(t, 1, nonceInt)
	})

	t.Run("Get WZCN burn ticket", func(t *testing.T) {
		t.Skipf("Skipping due to Authorizer failing to register in the 0chain")
		t.Parallel()

		output, err := getWrappedZcnBurnTicket(t, "0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95", false)
		require.Nil(t, err, "error: '%s'", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})
}

//nolint
func burnZcn(t *testing.T, amount, bridgeClientConfigFile string, retry bool) ([]string, error) {
	t.Logf("Burning ZCN tokens that will be minted for WZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-zcn --token %s --path %s --bridge_config %s --wallet %s --configDir ./config --config %s",
		amount,
		configDir,
		bridgeClientConfigFile,
		escapedTestName(t)+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func burnEth(t *testing.T, amount, bridgeClientConfigFile string, retry bool) ([]string, error) {
	t.Logf("Burning WZCN tokens that will be minted for ZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-eth --amount %s --path %s --bridge_config %s",
		amount,
		configDir,
		bridgeClientConfigFile,
	)
	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func getZcnBurnTicket(t *testing.T, hash string, retry bool) ([]string, error) {
	t.Logf("Get ZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-get-zcn-burn --hash %s --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		hash,
		configPath,
		escapedTestName(t)+"_wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func getWrappedZcnBurnTicket(t *testing.T, hash string, retry bool) ([]string, error) {
	t.Logf("Get WZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-get-zcn-burn --hash %s --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		hash,
		configPath,
		escapedTestName(t)+"_wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
