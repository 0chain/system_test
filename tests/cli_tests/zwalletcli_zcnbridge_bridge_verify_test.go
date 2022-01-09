package cli_tests

import (
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeVerify(t *testing.T) {
	t.Parallel()

	t.Run("Verify ethereum transaction", func(t *testing.T) {
		t.Parallel()

		output, err := bridgeVerify(t, "0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230")

		require.Nil(t, err, "error trying to verify transaction", strings.Join(output, "\n"))
		require.Equal(t, "Transaction verification success: 0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230", output[len(output)-1])
	})
}

// bridge-verify
func bridgeVerify(t *testing.T, hash string) ([]string, error) {
	t.Logf("Verify ethereum transaction for " + hash)

	cmd := "./zwallet bridge-verify" +
		" --hash " + hash

	return cliutils.RunCommandWithoutRetry(cmd)
}
