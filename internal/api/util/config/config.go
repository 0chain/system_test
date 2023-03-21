package config

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3" //nolint
)

// ConfigPathEnv contains name of env variable
const ConfigPathEnv = "CONFIG_PATH"

// DefaultConfigPath contains default value of ConfigPathEnv
const DefaultConfigPath = "./config/api_tests_config.yaml"

type Config struct {
	BlockWorker            		string `yaml:"block_worker"`
	ZboxUrl                		string `yaml:"0box_url"`
	ZboxPhoneNumber        		string `yaml:"0box_phone_number"`
	DefaultTestCaseTimeout 		string `yaml:"default_test_case_timeout"`
	BlobberOwnerWalletMnemonics string `yaml:"blobber_owner_wallet_mnemonics"`
	OwnerWalletMnemonics 		string `yaml:"owner_wallet_mnemonics"`
	ZS3ServerUrl           		string `yaml:"zs3_server_url"`
	EthereumAddress				string `yaml:"ethereum_address"`
}

func Parse(configPath string) *Config {
	var result *Config

	file, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalln("Failed to read config file! due to error: " + err.Error())
	}
	err = yaml.Unmarshal(file, &result) //nolint
	if err != nil {
		log.Fatalln("failed to deserialise config file due to error: " + err.Error())
	}

	return result
}

func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}
	return homeDir, nil
}

func CreateFreeStorageMarker(
	t *test.SystemTest,
	wallet *model.SdkWallet,
	assignerWallet *model.SdkWallet,
) string {
	marker := model.FreeStorageMarker{
		Recipient:  wallet.ClientID,
		FreeTokens: 5,
		Timestamp:  time.Now().Unix(),
	}

	forSignatureBytes, err := json.Marshal(&marker)
	require.Nil(t, err, "Could not marshal marker")

	data := hex.EncodeToString(forSignatureBytes)
	rawHash, err := hex.DecodeString(data)
	require.Nil(t, err, "failed to decode hex %s", data)
	require.NotNil(t, rawHash, "failed to decode hex %s", data)
	secretKey := crypto.ToSecretKey(t, assignerWallet.ToCliModelWalletFile())
	marker.Signature = crypto.Sign(t, string(rawHash), secretKey)
	marker.Assigner = assignerWallet.ClientID

	markerJson, err := json.Marshal(marker)
	require.Nil(t, err, "Could not marshal marker")

	return string(markerJson)
}