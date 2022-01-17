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

	const (
		Help = "Verify ethereum transaction"
	)

	var zwallet = func(cmd, hash string) ([]string, error) {
		t.Logf("%s for %s", Help, hash)
		run := fmt.Sprintf("./zwallet %s --hash %s", cmd, hash)
		return cliutils.RunCommand(run, 3, ,time.Second*5)
	}

	t.Run("Verify ethereum transaction", func(t *testing.T) {
		t.Parallel()

		output, err := zwallet(
			"bridge-verify",
			"0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230",
		)

		require.Nil(t, err, "error trying to verify transaction", strings.Join(output, "\n"))
		require.Equal(t, "Transaction verification success: 0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230", output[len(output)-1])
	})
}
