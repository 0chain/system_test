package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	TransactionHash = "0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95"
)

// todo: enable tests
func TestBridgeMint(testSetup *testing.T) {
	t := &test.SystemTest{T: testSetup}

	t.Parallel()

	t.Run("Mint WZCN tokens", func(t *test.SystemTest) {
		t.Skip("Skipping due to deployment issue")
		t.Parallel()

		output, err := mintWrappedZcnTokens(t, TransactionHash, false)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})

	t.Run("Mint ZCN tokens", func(t *test.SystemTest) {
		t.Skip("Skipping due to deployment issue")
		t.Parallel()

		output, err := mintZcnTokens(t, TransactionHash, false)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})
}

// nolint
func mintZcnTokens(t *test.SystemTest, transactionHash string, retry bool) ([]string, error) {
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

// nolint
func mintWrappedZcnTokens(t *test.SystemTest, transactionHash string, retry bool) ([]string, error) {
	t.Logf("Mint WZCN tokens using ZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-mint-wzcn %s --silent "+
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
