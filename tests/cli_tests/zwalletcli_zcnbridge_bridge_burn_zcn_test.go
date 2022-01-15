package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeBurnZCN(t *testing.T) {
	t.Parallel()

	const (
		Help = "Burn ZCN tokens that will be minted for WZCN tokens"
	)

	var zwallet = func(cmd, amount string) ([]string, error) {
		t.Logf("%s, amount: %s", Help, amount)
		run := fmt.Sprintf("./zwallet %s --amount %s", cmd, amount)
		return cliutils.RunCommandWithoutRetry(run)
	}

	t.Run("Burn ZCN tokens", func(t *testing.T) {
		t.Parallel()

		output, err := zwallet(
			"bridge-burn-zcn",
			"1",
		)

		require.Nil(t, err, "error trying to burn ZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification successful")
	})
}
