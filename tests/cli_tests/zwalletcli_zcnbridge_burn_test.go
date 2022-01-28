package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeBurn(t *testing.T) {
	t.Parallel()

	var zwallet = func(cmd, amount, help string) ([]string, error) {
		run := fmt.Sprintf(
			"./zwallet %s --amount %s --path %s --bridge_config %s",
			cmd,
			amount,
			configDir,
			bridgeClientConfigFile,
		)
		t.Logf("%s: amount %s", help, amount)
		t.Log(run)
		return cliutils.RunCommand(t, run, 3, time.Second*15)
	}

	t.Run("Burning WZCN tokens", func(t *testing.T) {
		t.Parallel()

		err := PrepareBridgeClient()
		require.NoError(t, err)

		output, err := zwallet(
			"bridge-burn-eth",
			"10",
			"Burning WZCN tokens that will be minted for ZCN tokens",
		)

		require.Nil(t, err, "error trying to burn WZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification: WZCN burn [OK]")
	})

	t.Run("Burning ZCN tokens", func(t *testing.T) {
		t.Skipf("Skipping due to transaction execution errr (context deadline error)")
		t.Parallel()

		output, err := zwallet(
			"bridge-burn-zcn",
			"1",
			"Burning ZCN tokens that will be minted for WZCN tokens",
		)

		require.Nil(t, err, "error trying to burn ZCN tokens: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Verification successful")
	})
}
