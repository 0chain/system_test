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
	zboxClient *client.ZboxClient
	ethClient *client.ETHClient
	parsedConfig       *config.Config
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

	sdkClient = client.NewSDKClient(parsedConfig.BlockWorker, parsedConfig.EthereumNodeURL)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)
	ethClient = client.NewETHClient(parsedConfig.EthereumNodeURL)
	parsedConfig = config.Parse(configPath)
	zboxClient = client.NewZboxClient(parsedConfig.ZboxUrl, parsedConfig.ZboxPhoneNumber)

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

	delegatedSdkWallet := new(model.SdkWallet)
	delegatedSdkWallet.UnmarshalFile(parsedConfig.DelegatedWalletLocation)
	keys := crypto.GenerateKeys(t, delegatedSdkWallet.Mnemonics)
	delegatedWallet.FromSdkWallet(delegatedSdkWallet, keys)

	var clientGetBalanceResponse *model.ClientGetBalanceResponse
	clientGetBalanceResponse, _, err = apiClient.V1ClientGetBalance(
		t,
		model.ClientGetBalanceRequest{
			ClientID: delegatedWallet.Id,
		},
		client.HttpOkStatus)
	if err != nil {
		delegatedWallet = nil
	} else {
		delegatedWallet.Nonce = int(clientGetBalanceResponse.Nonce)
	}
	os.Exit(m.Run())
}
