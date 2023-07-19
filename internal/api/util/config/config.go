package config

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	BlockWorker                 string `yaml:"block_worker"`
	ZboxUrl                     string `yaml:"0box_url"`
	ZboxPhoneNumber             string `yaml:"0box_phone_number"`
	DefaultTestCaseTimeout      string `yaml:"default_test_case_timeout"`
	ZS3ServerUrl                string `yaml:"zs3_server_url"`
	S3SecretKey                 string `yaml:"s3_secret_key"`
	S3AccessKey                 string `yaml:"s3_access_key"`
	EthereumAddress             string `yaml:"ethereum_address"`
	S3BucketName                string `yaml:"s3_bucket_name"`
	BlobberOwnerWalletMnemonics string `yaml:"blobber_owner_wallet_mnemonics"`
	OwnerWalletMnemonics        string `yaml:"owner_wallet_mnemonics"`
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
	freeToken := 5.0
	nonce := time.Now().Unix()
	marker := model.FreeStorageMarker{
		Recipient:  wallet.ClientID,
		FreeTokens: freeToken,
		Nonce:      nonce,
	}

	reqmarker := fmt.Sprintf("%s:%f:%d", wallet.ClientID, freeToken, nonce)
	data := hex.EncodeToString([]byte(reqmarker))

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
