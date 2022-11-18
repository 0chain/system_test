package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeBurn(t *testing.T) {
	t.Parallel()

	t.Run("Burning WZCN tokens", func(t *testing.T) {
		t.Parallel()

		err := PrepareBridgeClient(t)
		require.NoError(t, err)

		output, _ := burnEth(t, "10", bridgeClientConfigFile, true)
		// todo: enable test: require.Nil(t, err, "error trying to burn WZCN tokens: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:") // : WZCN burn [OK]")
	})

	// todo: enable test
	t.Run("Burning ZCN tokens", func(t *testing.T) {
		t.Parallel()

		output, err := burnZcn(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err, "error trying to burn ZCN tokens: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification successful")
	})

	t.Run("Get ZCN burn ticket", func(t *testing.T) {
		//t.Skipf("Skipping due to context deadline error when burning ZCN tokens")
		t.Parallel()

		output, err := getZcnBurnTicket(t, "0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95", false)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification [OK]")
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
