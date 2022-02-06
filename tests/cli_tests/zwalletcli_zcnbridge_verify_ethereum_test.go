package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	Address = "0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230"
)

func TestBridgeVerify(t *testing.T) {
	t.Parallel()

	const (
		Help = "Verify ethereum transaction"
	)

	var zwallet = func(cmd, hash string) ([]string, error) {
		t.Logf("%s for %s", Help, hash)
		run := fmt.Sprintf("./zwallet %s --hash %s", cmd, hash)
		run += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)
		return cliutils.RunCommand(t, run, 3, time.Second*5)
	}

	t.Run("Verify ethereum transaction", func(t *testing.T) {
		t.Parallel()

		output, err := zwallet(
			"bridge-verify",
			Address,
		)

		require.Nil(t, err, "error trying to verify transaction", strings.Join(output, "\n"))
		require.Equal(t, "Transaction verification success: "+Address, output[len(output)-1])
	})
}
