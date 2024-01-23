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

	t.RunSequentiallyWithTimeout("Mint WZCN tokens", time.Minute*10, func(t *test.SystemTest) {
		err := tenderlyClient.InitBalance(ethereumAddress)
		require.NoError(t, err)

		err = tenderlyClient.InitErc20Balance(tokenAddress, ethereumAddress)
		require.NoError(t, err)

		output, err := resetUserNonce(t, true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)

		output, err = burnZcn(t, "1", false)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Transaction completed successfully:")

		output, err = mintWrappedZcnTokens(t, true)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Done.")
	})

	t.RunSequentiallyWithTimeout("Mint ZCN tokens", time.Minute*10, func(t *test.SystemTest) {
		err := tenderlyClient.InitBalance(ethereumAddress)
		require.NoError(t, err)

		err = tenderlyClient.InitErc20Balance(tokenAddress, ethereumAddress)
		require.NoError(t, err)

		createWalletForName(escapedTestName(t))

		output, err := burnEth(t, "1000000000000", true)
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

// nolint
func resetUserNonce(t *test.SystemTest, retry bool) ([]string, error) {
	t.Logf("Reset user nonce...")
	cmd := fmt.Sprintf(
		"./zwallet reset-user-nonce --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		configPath,
		escapedTestName(t)+"_wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 6, time.Second*10)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
