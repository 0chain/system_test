package cli_tests

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func getConfigDir() string {
	var configDir string
	curr, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	configDir = filepath.Join(curr, "config")
	return configDir
}

func updateValidatorInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating validator info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox validator-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, blobberOwnerWallet, cliConfigFilename), 3, time.Second*30)
}
