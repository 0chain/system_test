package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBurnTicket(t *testing.T) {
	t.Parallel()

	var zwallet = func(cmd, hash, help string) ([]string, error) {
		run := fmt.Sprintf("./zwallet %s --hash %s", cmd, hash)
		t.Logf("%s: hash: %s", help, hash)
		t.Log(run)
		return cliutils.RunCommand(t, run, 3, time.Second*15)
	}

	t.Run("Get ZCN burn ticket", func(t *testing.T) {
		t.Skipf("Skipping due to context deadline error when burning ZCN tokens")
		t.Parallel()

		output, err := zwallet(
			"bridge-get-zcn-burn",
			"0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95",
			"Get ZCN burn ticket",
		)

		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})

	t.Run("Get WZCN burn ticket", func(t *testing.T) {
		t.Skipf("Skipping due to Authorizer failing to register in the 0chain")
		t.Parallel()

		output, err := zwallet(
			"bridge-get-wzcn-burn",
			"0x607abfece03c42afb446c77ffc81783f2d8fb614774d3fe241eb54cb52943f95",
			"Get WZCN burn ticket",
		)

		require.Nil(t, err, "error: '%s'", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification [OK]")
	})
}
