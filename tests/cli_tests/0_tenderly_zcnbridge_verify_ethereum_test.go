package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0TenderlyBridgeVerify(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if !tenderlyInitialized {
		t.Skip("Tenderly has not been initialized properly!")
	}

	t.SetSmokeTests("Verify ethereum transaction")

	t.RunSequentiallyWithTimeout("Verify ethereum transaction", time.Minute*10, func(t *test.SystemTest) {
		time.Sleep(time.Minute)

		output, err := burnEth(t, "1000000000000", true)
		require.Nil(t, err, output)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:")

		ethTxHash := getTransactionHash(output, true)

		output, err = verifyBridgeTransaction(t, ethTxHash, false)
		require.Nil(t, err, "error trying to verify transaction", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Equal(t, "Transaction verification success: "+ethTxHash, output[len(output)-1])
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
		return cliutils.RunCommand(t, cmd, 2, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
