package api_tests

import (
	"log"
	"os"
	"runtime"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
)

var (
	apiClient          *client.APIClient
	sdkClient          *client.SDKClient
	sdkWallet          *model.Wallet
	sdkWalletMnemonics string
)

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	goMaxProcs := runtime.GOMAXPROCS(24)
	log.Printf("GOMAXPROCS environment variable is set to [%v]", goMaxProcs)

	parsedConfig := config.Parse(configPath)

	sdkClient = client.NewSDKClient(parsedConfig.BlockWorker)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)

	t := new(testing.T)

	sdkWalletMnemonics = crypto.GenerateMnemonics(t)
	sdkWallet = apiClient.RegisterWalletForMnemonic(t, sdkWalletMnemonics)
	sdkClient.SetWallet(t, sdkWallet, sdkWalletMnemonics)

	os.Exit(m.Run())
}
