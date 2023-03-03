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
	apiClient          *client.APIClient
	zs3Client          *client.ZS3Client
	sdkClient          *client.SDKClient
	zboxClient         *client.ZboxClient
	sdkWallet          *model.Wallet
	sdkWalletMnemonics string
	parsedConfig       *config.Config
)

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig = config.Parse(configPath)
	sdkClient = client.NewSDKClient(parsedConfig.BlockWorker)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)
	zs3Client = client.NewZS3Client(parsedConfig.ZS3ServerUrl)
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

	os.Exit(m.Run())
}
