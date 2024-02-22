package cli_tests

import (
	"fmt"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func readPoolInfo(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return readPoolInfoWithWallet(t, escapedTestName(t), cliConfigFilename)
}

func readPoolInfoWithWallet(t *test.SystemTest, wallet, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 30*time.Second) // TODO replace with poller
	t.Logf("Getting read pool info...")
	return cliutils.RunCommand(t, "./zbox rp-info"+" --json --silent --wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func readPoolLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return readPoolLockWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func readPoolLockWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Locking read tokens...")
	cmd := fmt.Sprintf("./zbox rp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getDownloadCost(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return getDownloadCostWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func getDownloadCostWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Getting download cost...")
	cmd := fmt.Sprintf("./zbox get-download-cost %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func unitToZCN(unitCost float64, unit string) float64 {
	switch unit {
	case "SAS", "sas":
		unitCost /= 1e10
		return unitCost
	case "uZCN", "uzcn":
		unitCost /= 1e6
		return unitCost
	case "mZCN", "mzcn":
		unitCost /= 1e3
		return unitCost
	}
	return unitCost
}
