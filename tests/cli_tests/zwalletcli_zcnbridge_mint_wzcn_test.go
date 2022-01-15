package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeMintWZCN(t *testing.T) {
	t.Parallel()

	const (
		Help = "Mint WZCN tokens using ZCN burn ticket"
	)

	var zwallet = func(cmd, hash string) ([]string, error) {
		t.Logf("%s, hash: %s", Help, hash)
		run := fmt.Sprintf("./zwallet %s --hash %s", cmd, hash)
		return cliutils.RunCommandWithoutRetry(run)
	}

	t.Run("Mint WZCN tokens", func(t *testing.T) {
		t.Parallel()

		output, err := zwallet(
			"bridge-mint-wzcn",
			"0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95",
		)

		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})
}
