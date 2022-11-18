package cli_tests

import (
	"fmt"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestBurnTicket(t *testing.T) {
	t.Parallel()

}

//nolint
func getZcnBurnTicket(t *testing.T, hash string, retry bool) ([]string, error) {
	t.Logf("Get ZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-get-zcn-burn --hash %s --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		hash,
		configPath,
		//escapedTestName(t)+"_wallet.json",
		"wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func getWrappedZcnBurnTicket(t *testing.T, hash string, retry bool) ([]string, error) {
	t.Logf("Get WZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-get-zcn-burn %s --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		hash,
		configPath,
		escapedTestName(t)+"_wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
