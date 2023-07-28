package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeMint(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Mint WZCN tokens")

	t.Parallel()

	t.Run("Mint WZCN tokens", func(t *test.SystemTest) {
		output, err := executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnZcn(t, "1", false)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Transaction completed successfully:")

		output, err = mintWrappedZcnTokens(t, true)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})

	t.Run("Mint ZCN tokens", func(t *test.SystemTest) {
		_, err := createWalletForName(t, configPath, escapedTestName(t))
		require.NoError(t, err)

		output, err := burnEth(t, "10000000000", true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:")

		output, err = mintZcnTokens(t, true)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Done.")
	})
}

// nolint
func mintZcnTokens(t *test.SystemTest, retry bool) ([]string, error) {
	t.Logf("Mint ZCN tokens using WZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-mint-zcn --silent "+
			"--configDir ./config --config %s --path %s --wallet %s",
		configPath,
		configDir,
		escapedTestName(t)+"_wallet.json",
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

// nolint
func mintWrappedZcnTokens(t *test.SystemTest, retry bool) ([]string, error) {
	t.Logf("Mint WZCN tokens using ZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-mint-wzcn --silent "+
			"--configDir ./config --config %s --path %s --wallet %s",
		configPath,
		configDir,
		escapedTestName(t)+"_wallet.json",
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
