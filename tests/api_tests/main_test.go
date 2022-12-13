package api_tests

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/test"
)

var (
	apiClient *client.APIClient
	sdkClient *client.SDKClient
	ethClient *client.ETHClient

	sdkWallet          *model.Wallet
	sdkWalletMnemonics string

	delegatedWallet = new(model.Wallet)
)

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig := config.Parse(configPath)

	sdkClient = client.NewSDKClient(parsedConfig.BlockWorker, parsedConfig.EthereumNodeURL)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)
	ethClient = client.NewETHClient(parsedConfig.EthereumNodeURL)

	defaultTestTimeout, err := time.ParseDuration(parsedConfig.DefaultTestCaseTimeout)
	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.DefaultTestTimeout = defaultTestTimeout
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}

	t := test.NewSystemTest(new(testing.T))

	sdkWalletMnemonics = crypto.GenerateMnemonics(t)
	sdkWallet = apiClient.RegisterWalletForMnemonic(t, sdkWalletMnemonics)
	sdkClient.SetWallet(t, sdkWallet, sdkWalletMnemonics)

	var delegatedSdkWallet model.SdkWallet
	delegatedSdkWallet.UnmarshalFile("config/blobber_owner_wallet.json")
	keys := crypto.GenerateKeys(t, delegatedSdkWallet.Mnemonics)
	delegatedWallet.FromSdkWallet(delegatedSdkWallet, keys)

	walletBalance := apiClient.GetWalletBalance(t, delegatedWallet, client.HttpOkStatus)
	delegatedWallet.Nonce = int(walletBalance.Nonce)

	os.Exit(m.Run())
}
