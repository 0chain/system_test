package api_tests

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	coreClient "github.com/0chain/gosdk/core/client"
	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/core/zcncrypto"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

var (
	apiClient *client.APIClient
	zs3Client *client.ZS3Client
	// sdkClient        *client.SDKClient
	zboxClient       *client.ZboxClient
	zvaultClient     *client.ZvaultClient
	zauthClient      *client.ZauthClient
	chimneyClient    *client.APIClient
	chimneySdkClient *client.SDKClient
	sdkClient        *client.SDKClient
	// sdkWallet                   *model.Wallet
	ownerWallet                 *model.Wallet
	ownerWalletMnemonics        string
	blobberOwnerWallet          *model.Wallet
	blobberOwnerWalletMnemonics string
	parsedConfig                *config.Config
	sdkWallet                   *zcncrypto.Wallet

	initialisedWallets []*model.Wallet
	walletIdx          int64
	walletMutex        sync.Mutex

	zauthAvailable     bool
	zboxAvailable      bool
	zvaultAvailable    bool
	zs3Available       bool
	firebaseTokenValid bool
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
	zboxClient = client.NewZboxClient(parsedConfig.ZboxUrl)
	zvaultClient = client.NewZvaultClient(parsedConfig.ZvaultUrl)
	zauthClient = client.NewZauthClient(parsedConfig.ZauthUrl)

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

	err = coreClient.Init(context.Background(), conf.Config{
		BlockWorker:     parsedConfig.BlockWorker,
		SignatureScheme: "bls0chain",
		ChainID:         "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe",
		MaxTxnQuery:     5,
		QuerySleepTime:  5,
		MinSubmit:       10,
		MinConfirmation: 10,
	})
	require.NoError(t, err)

	blobberOwnerWalletMnemonics = parsedConfig.BlobberOwnerWalletMnemonics
	blobberOwnerWallet = apiClient.CreateWalletForMnemonic(t, blobberOwnerWalletMnemonics)

	ownerWalletMnemonics = parsedConfig.OwnerWalletMnemonics
	ownerWallet = apiClient.CreateWalletForMnemonic(t, ownerWalletMnemonics)

	// Read the content of the file
	fileContent, err := os.ReadFile("./config/wallets.json")
	if err != nil {
		log.Println("Error reading file:", err)
		return
	}

	fileWallets := []WalletFile{}

	// Parse the JSON data into a list of strings
	err = json.Unmarshal(fileContent, &fileWallets)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		return
	}

	for i := range fileWallets {
		wallet := fileWallets[i]
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
			log.Println("Error decoding JSON:", err)
		}
		err = initialisedWallet.Keys.PrivateKey.DeserializeHexStr(wallet.Keys[0].PrivateKey)
		if err != nil {
			log.Println("Error decoding JSON:", err)
		}

		initialisedWallets = append(initialisedWallets, initialisedWallet)
	}

	// Initialize sdkWallet from the first pre-funded wallet (used by transcoder and SDK-based tests)
	if len(initialisedWallets) > 0 {
		w := initialisedWallets[0]
		var dateCreated string
		if w.CreationDate != nil {
			dateCreated = time.Unix(int64(*w.CreationDate), 0).Format(time.RFC3339)
		}
		sdkWallet = &zcncrypto.Wallet{
			ClientID:  w.Id,
			ClientKey: w.PublicKey,
			Keys: []zcncrypto.KeyPair{{
				PublicKey:  w.PublicKey,
				PrivateKey: w.Keys.PrivateKey.Value,
			}},
			Mnemonic:    w.Mnemonics,
			Version:     "1.0",
			DateCreated: dateCreated,
		}
		log.Printf("sdkWallet initialized: %s", w.Id[:16])
	}

	// Check service availability
	zboxAvailable = isZboxResponding()
	zauthAvailable = isServiceResponding(parsedConfig.ZauthUrl)
	zvaultAvailable = isServiceResponding(parsedConfig.ZvaultUrl)
	zs3Available = isServiceResponding(parsedConfig.ZS3ServerUrl)
	firebaseTokenValid = parsedConfig.FirebaseAPIKey != "" && parsedConfig.FirebaseEmail != "" && parsedConfig.FirebasePassword != ""

	// Obtain real Firebase ID token for 0box API tests.
	// Without this, X_APP_ID_TOKEN stays as "test_firebase_token" and 0box
	// rejects all requests that require authenticated owner records.
	if firebaseTokenValid {
		if err := client.RefreshFirebaseToken(parsedConfig.FirebaseAPIKey, parsedConfig.FirebaseEmail, parsedConfig.FirebasePassword); err != nil {
			log.Printf("WARNING: Firebase token refresh failed: %v — 0box tests requiring auth will fail", err)
		}
		if parsedConfig.FirebaseEmailR != "" && parsedConfig.FirebasePasswordR != "" {
			if err := client.RefreshFirebaseToken_R(parsedConfig.FirebaseAPIKey, parsedConfig.FirebaseEmailR, parsedConfig.FirebasePasswordR); err != nil {
				log.Printf("WARNING: Firebase _R token refresh failed: %v", err)
			}
		}
	}

	os.Exit(m.Run())
}

func initialiseSCWallet() *model.Wallet {
	// read the file sc_owner_wallet.json
	fileContent, err := os.ReadFile("./config/sc_owner_wallet.json")
	if err != nil {
		log.Println("Error reading file:", err)
		return nil
	}

	fileWallet := WalletFile{}

	// Parse the JSON data into a list of strings
	err = json.Unmarshal(fileContent, &fileWallet)

	if err != nil {
		log.Println("Error decoding JSON:", err)
		return nil
	}

	wallet := &model.Wallet{
		Id:        fileWallet.ClientId,
		Version:   fileWallet.Version,
		PublicKey: fileWallet.Keys[0].PublicKey,
		Nonce:     0,
		Keys:      &model.KeyPair{},
		Mnemonics: fileWallet.Mnemonics,
	}

	err = wallet.Keys.PublicKey.DeserializeHexStr(fileWallet.Keys[0].PublicKey)
	if err != nil {
		log.Println("Error decoding JSON:", err)
	}
	err = wallet.Keys.PrivateKey.DeserializeHexStr(fileWallet.Keys[0].PrivateKey)
	if err != nil {
		log.Println("Error decoding JSON:", err)
	}

	return wallet
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

func createWallet(t *test.SystemTest) *model.Wallet {
	walletMutex.Lock()
	wallet := initialisedWallets[walletIdx]
	walletIdx++
	balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
	wallet.Nonce = int(balance.Nonce)
	walletMutex.Unlock()

	return wallet
}

func isZboxResponding() bool {
	for i := 0; i < 6; i++ {
		resp, err := http.Get(parsedConfig.ZboxUrl) //nolint
		if err == nil {
			resp.Body.Close()
			return true
		}
		if i < 5 {
			time.Sleep(5 * time.Second)
		}
	}
	return false
}

func isServiceResponding(url string) bool {
	if url == "" {
		return false
	}
	for i := 0; i < 6; i++ {
		resp, err := http.Get(url) //nolint
		if err == nil {
			resp.Body.Close()
			return true
		}
		if i < 5 {
			time.Sleep(5 * time.Second)
		}
	}
	return false
}
