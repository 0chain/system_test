package api_tests

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

var (
	apiClient *client.APIClient
	zs3Client *client.ZS3Client
	//sdkClient        *client.SDKClient
	zboxClient       *client.ZboxClient
	chimneyClient    *client.APIClient
	chimneySdkClient *client.SDKClient
	sdkClient        *client.SDKClient
	//sdkWallet                   *model.Wallet
	ownerWallet                 *model.Wallet
	ownerWalletMnemonics        string
	blobberOwnerWallet          *model.Wallet
	blobberOwnerWalletMnemonics string
	parsedConfig                *config.Config

	initialisedWallets []*model.Wallet
	walletIdx          int64
)

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig = config.Parse(configPath)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)
	zs3Client = client.NewZS3Client(parsedConfig.ZS3ServerUrl)
	zboxClient = client.NewZboxClient(parsedConfig.ZboxUrl, parsedConfig.ZboxPhoneNumber)
	chimneyClient = client.NewAPIClient(parsedConfig.ChimneyTestNetwork)
	chimneySdkClient = client.NewSDKClient(parsedConfig.ChimneyTestNetwork)
	sdkClient = client.NewSDKClient(parsedConfig.BlockWorker)

	defaultTestTimeout, err := time.ParseDuration(parsedConfig.DefaultTestCaseTimeout)
	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.DefaultTestTimeout = defaultTestTimeout
		test.SmokeTestMode, _ = strconv.ParseBool(os.Getenv("SMOKE_TEST_MODE"))
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}

	t := test.NewSystemTest(new(testing.T))

	err = zcncore.Init(getConfigForZcnCoreInit(parsedConfig.BlockWorker))
	require.NoError(t, err)

	blobberOwnerWalletMnemonics = parsedConfig.BlobberOwnerWalletMnemonics
	blobberOwnerWallet = apiClient.CreateWalletForMnemonic(t, blobberOwnerWalletMnemonics)

	ownerWalletMnemonics = parsedConfig.OwnerWalletMnemonics
	ownerWallet = apiClient.CreateWalletForMnemonic(t, ownerWalletMnemonics)

	// Read the content of the file
	fileContent, err := os.ReadFile("./config/wallets.json")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	fileWallets := []WalletFile{}

	// Parse the JSON data into a list of strings
	err = json.Unmarshal(fileContent, &fileWallets)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	for _, wallet := range fileWallets {
		initialisedWallet := &model.Wallet{
			Id:        wallet.ClientId,
			Version:   wallet.Version,
			PublicKey: wallet.Keys[0].PublicKey,
			Nonce:     0,
			Keys:      &model.KeyPair{},
			Mnemonics: wallet.Mnemonics,
		}

		err := initialisedWallet.Keys.PublicKey.DeserializeHexStr(wallet.Keys[0].PublicKey)
		if err != nil {
			fmt.Println("Error decoding JSON:", err)
		}
		err = initialisedWallet.Keys.PrivateKey.DeserializeHexStr(wallet.Keys[0].PrivateKey)
		if err != nil {
			fmt.Println("Error decoding JSON:", err)
		}

		initialisedWallets = append(initialisedWallets, initialisedWallet)
	}

	fmt.Println("initialisedWallets", initialisedWallets[0].Id)

	os.Exit(m.Run())
}

func getConfigForZcnCoreInit(blockWorker string) string {
	configMap := map[string]interface{}{
		"block_worker":              blockWorker,
		"signature_scheme":          "bls0chain",
		"min_submit":                50,
		"min_confirmation":          50,
		"confirmation_chain_length": 3,
		"max_txn_query":             5,
		"query_sleep_time":          5,
	}

	b, _ := json.Marshal(configMap)
	return string(b)
}

type WalletFile struct {
	ClientId  string `json:"client_id"`
	ClientKey string `json:"client_key"`
	Keys      []struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
	} `json:"keys"`
	Mnemonics       string      `json:"mnemonics"`
	Version         string      `json:"version"`
	DateCreated     time.Time   `json:"date_created"`
	Nonce           int         `json:"nonce"`
	ChainID         string      `json:"ChainID"`
	SignatureScheme interface{} `json:"SignatureScheme"`
}
