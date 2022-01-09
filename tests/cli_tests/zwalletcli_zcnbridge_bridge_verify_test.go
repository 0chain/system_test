package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeVerify(t *testing.T) {
	t.Parallel()

	t.Run("Verify ethereum transaction", func(t *testing.T) {
		t.Parallel()

		const (
			cmd = "bridge-verify"
		)

		output, err := bridgeCmd(
			t,
			cmd,
			"0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230",
		)

		require.Nil(t, err, "error trying to verify transaction", strings.Join(output, "\n"))
		require.Equal(t, "Transaction verification success: 0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230", output[len(output)-1])
	})
}

// bridge-verify
func bridgeCmd(t *testing.T, cmd, hash string) ([]string, error) {
	t.Logf("Verify ethereum transaction for " + hash)
	run := fmt.Sprintf("./zwallet %s --hash %s", cmd, hash)
	return cliutils.RunCommandWithoutRetry(run)
}
