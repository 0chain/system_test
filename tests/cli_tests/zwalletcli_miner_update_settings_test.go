package cli_tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestMinerUpdateSettings(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + minerNodeDelegateWallet + "_wallet.json"); err != nil {
		t.Skipf("blobber owner wallet located at %s is missing", "./config/"+minerNodeDelegateWallet+"_wallet.json")
	}
}

func listMiners(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-miners %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func minerUpdateSettings(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return minerUpdateSettingsForWallet(t, cliConfigFilename, params, minerNodeDelegateWallet)
}

func minerUpdateSettingsForWallet(t *testing.T, cliConfigFilename, params, wallet string) ([]string, error) {
	t.Log("Updating miner settings...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox mn-update-settings %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}
