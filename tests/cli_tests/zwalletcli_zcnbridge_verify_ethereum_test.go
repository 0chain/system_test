package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	Address = "0x31925839586949a96e72cacf25fed7f47de5faff78adc20946183daf3c4cf230"
)

func TestBridgeVerify(testSetup *testing.T) {
	t := &test.SystemTest{T: testSetup}

	t.Parallel()

	t.Run("Verify ethereum transaction", func(t *test.SystemTest) {
		t.Skip("Skip till fixed")
		t.Parallel()

		output, err := verifyBridgeTransaction(t, Address, false)
		require.Nil(t, err, "error trying to verify transaction", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Equal(t, "Transaction verification success: "+Address, output[len(output)-1])
	})
}

func verifyBridgeTransaction(t *test.SystemTest, address string, retry bool) ([]string, error) { // nolint
	t.Logf("verifying ethereum transaction...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-verify --hash %s --silent --configDir ./config --config %s --path %s",
		address,
		configPath,
		configDir,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
