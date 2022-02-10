package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// todo: enable tests
func TestBurnTicket(t *testing.T) {
	t.Parallel()

	t.Run("Get ZCN burn ticket", func(t *testing.T) {
		t.Skipf("Skipping due to context deadline error when burning ZCN tokens")
		t.Parallel()

		output, err := getZcnBurnTicket(t, "0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95", false)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})

	t.Run("Get WZCN burn ticket", func(t *testing.T) {
		t.Skipf("Skipping due to Authorizer failing to register in the 0chain")
		t.Parallel()

		output, err := getWrappedZcnBurnTicket(t, "0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95", false)
		require.Nil(t, err, "error: '%s'", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})
}

//nolint
func getZcnBurnTicket(t *testing.T, hash string, retry bool) ([]string, error) {
	t.Logf("Get ZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-get-zcn-burn %s --silent "+
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
		"./zwallet bridge-get-zcn-burn %s --silent "+
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
