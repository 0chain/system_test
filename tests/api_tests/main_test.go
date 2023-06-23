package api_tests

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/test"
)

var (
	apiClient                   *client.APIClient
	zs3Client                   *client.ZS3Client
	sdkClient                   *client.SDKClient
	zboxClient                  *client.ZboxClient
	sdkWallet                   *model.Wallet
	sdkWalletMnemonics          string
	ownerWallet                 *model.Wallet
	ownerWalletMnemonics        string
	blobberOwnerWallet          *model.Wallet
	blobberOwnerWalletMnemonics string
	parsedConfig                *config.Config
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
	configMap := map[string]interface{}{
		"block_worker":              parsedConfig.BlockWorker,
		"signature_scheme":          "bls0chain",
		"min_submit":                50,
		"min_confirmation":          50,
		"confirmation_chain_length": 3,
		"max_txn_query":             5,
		"query_sleep_time":          5,
	}

	b, _ := json.Marshal(configMap)
	zcncore.Init(string(b))
	defaultTestTimeout, err := time.ParseDuration(parsedConfig.DefaultTestCaseTimeout)
	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.DefaultTestTimeout = defaultTestTimeout
		test.SmokeTestMode, _ = strconv.ParseBool(os.Getenv("SMOKE_TEST_MODE"))
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}

	t := test.NewSystemTest(new(testing.T))

	sdkWalletMnemonics = crypto.GenerateMnemonics(t)
	sdkWallet = apiClient.CreateWalletForMnemonic(t, sdkWalletMnemonics)
	sdkClient.SetWallet(t, sdkWallet, sdkWalletMnemonics)

	blobberOwnerWalletMnemonics = parsedConfig.BlobberOwnerWalletMnemonics
	blobberOwnerWallet = apiClient.CreateWalletForMnemonic(t, blobberOwnerWalletMnemonics)

	ownerWalletMnemonics = parsedConfig.OwnerWalletMnemonics
	ownerWallet = apiClient.CreateWalletForMnemonic(t, ownerWalletMnemonics)

	os.Exit(m.Run())
}
