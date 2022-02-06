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

		output, err := burnEth(t, "10", configDir, bridgeClientConfigFile, false)
		require.Nil(t, err, "error trying to burn WZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification: WZCN burn [OK]")
	})

	t.Run("Burning ZCN tokens", func(t *testing.T) {
		t.Skipf("Skipping due to transaction execution errr (context deadline error)")
		t.Parallel()

		output, err := burnZcn(t, "1", configDir, bridgeClientConfigFile, false)
		require.Nil(t, err, "error trying to burn ZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification successful")
	})
}

func burnZcn(t *testing.T, amount, path, bridgeClientConfigFile string, retry bool) ([]string, error) {
	t.Logf("Burning ZCN tokens that will be minted for WZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-zcn %s -- path %s --bridge_config %s --silent "+
			"--configDir ./config --config %s --wallet %s",
		amount,
		path,
		bridgeClientConfigFile,
		configPath,
		escapedTestName(t)+"_wallet.json",
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func burnEth(t *testing.T, amount, path, bridgeClientConfigFile string, retry bool) ([]string, error) {
	t.Logf("Burning WZCN tokens that will be minted for ZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-eth %s -- path %s --bridge_config %s --silent "+
			"--configDir ./config --config %s --wallet %s",
		amount,
		path,
		bridgeClientConfigFile,
		configPath,
		escapedTestName(t)+"_wallet.json",
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
