package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	// TransactionHash = "0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95"
)

// todo: enable tests
func TestBridgeMint(t *testing.T) {
	t.Parallel()

	t.Run("Mint WZCN tokens", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		// burn some tokens on 0chain
		output, err = burnZcn(t, "2", bridgeClientConfigFile, true)
		require.Nil(t, err, "error burning tokens")
		require.Greater(t, len(output), 0)
		
		// get hash from output
		hashregex := regexp.MustCompile("[a-f0-9]{64}")
		transactionHash := hashregex.FindStringSubmatch(output[0])[1]

		output, err = mintWrappedZcnTokens(t, transactionHash, false)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})

	// t.Run("Mint ZCN tokens", func(t *testing.T) {
	// 	t.Skip("Skipping due to deployment issue")
	// 	t.Parallel()

	// 	output, err := mintZcnTokens(t, TransactionHash, false)
	// 	require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
	// 	require.Greater(t, len(output), 0)
	// 	require.Contains(t, output[len(output)-1], "Verification [OK]")
	// })
}

//nolint
func mintZcnTokens(t *testing.T, transactionHash string, retry bool) ([]string, error) {
	t.Logf("Mint ZCN tokens using WZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-mint-zcn %s --silent "+
			"--configDir ./config --config %s --path %s",
		transactionHash,
		configPath,
		configDir,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func mintWrappedZcnTokens(t *testing.T, transactionHash string, retry bool) ([]string, error) {
	t.Logf("Mint WZCN tokens using ZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-mint-wzcn --hash %s --silent "+
			"--configDir ./config --config %s --path %s",
		transactionHash,
		configPath,
		configDir,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
