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
		require.Contains(t, output[len(output)-1], "Verification:") //: WZCN burn [OK]")
	})

	// todo: enable test
	t.Run("Burning ZCN tokens", func(t *testing.T) {
		t.Skipf("Skipping due to transaction execution errr (context deadline error)")
		t.Parallel()

		output, err := burnZcn(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err, "error trying to burn ZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification successful")
	})
}

//nolint
func burnZcn(t *testing.T, amount, bridgeClientConfigFile string, retry bool) ([]string, error) {
	t.Logf("Burning ZCN tokens that will be minted for WZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-zcn --amount %s --path %s --bridge_config %s",
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
