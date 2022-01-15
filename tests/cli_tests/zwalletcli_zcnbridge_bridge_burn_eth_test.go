package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeBurnEth(t *testing.T) {
	t.Parallel()

	t.Run("Burn WZCN tokens", func(t *testing.T) {
		t.Parallel()

		output, err := zwalletCLIAmount(
			t,
			"bridge-burn-eth",
			"10",
		)

		require.Nil(t, err, "error trying to burn WZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification: WZCN burn [OK]")
	})
}

func zwalletCLIAmount(t *testing.T, cmd, amount string) ([]string, error) {
	t.Logf("burn WZCN tokens that will be minted on ZCN chain, amount: %s", amount)
	run := fmt.Sprintf("./zwallet %s --amount %s", cmd, amount)
	return cliutils.RunCommandWithoutRetry(run)
}
