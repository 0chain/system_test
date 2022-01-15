package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeBurnWZCN(t *testing.T) {
	t.Parallel()

	const (
		Help = "Burn WZCN tokens that will be minted on ZCN chain"
	)

	var zwallet = func(cmd, amount string) ([]string, error) {
		t.Logf("%s, amount: %s", Help, amount)
		run := fmt.Sprintf("./zwallet %s --amount %s", cmd, amount)
		return cliutils.RunCommandWithoutRetry(run)
	}

	t.Run("Burn WZCN tokens", func(t *testing.T) {
		t.Parallel()

		output, err := zwallet(
			"bridge-burn-eth",
			"10",
		)

		require.Nil(t, err, "error trying to burn WZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification: WZCN burn [OK]")
	})
}
