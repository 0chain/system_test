package cli_tests

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/tenderly"

	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/spf13/viper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func setupDefaultConfig() {
	viper.SetDefault("nodes.miner01ID", "73ad5727612116c025bb4405bf3adb4a4a04867ae508c51cf885395bffc8a949")
	viper.SetDefault("nodes.miner02ID", "3ec9a42db3355f33c35750ce589ed717c08787997b7f34a7f1f9fb0a03f2b17c")
	viper.SetDefault("nodes.miner03ID", "c6f4b8ce5da386b278ba8c4e6cf98b24b32d15bc675b4d12c95e082079c91937")
	viper.SetDefault("nodes.sharder01ID", "ea26431f8adb7061766f1d6bbcc3b292d70dd59960d857f04b8a75e6a5bbe04f")
	viper.SetDefault("nodes.sharder02ID", "30001a01a888584772b7fee13934021ab8557e0ed471c0a3a454e9164180aef1")
}

// SetupConfig setups the main configuration system.
func setupConfig() {
	setupDefaultConfig()
	path := filepath.Join(".", "config")

	viper.SetConfigName("nodes")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(fmt.Errorf("fatal error config file: %s", err))
	}

	miner01ID = viper.GetString("nodes.miner01ID")
	miner02ID = viper.GetString("nodes.miner02ID")
	miner03ID = viper.GetString("nodes.miner03ID")
	sharder01ID = viper.GetString("nodes.sharder01ID")
	sharder02ID = viper.GetString("nodes.sharder02ID")

	parsedConfig := config.Parse(filepath.Join(".", path, "cli_tests_config.yaml"))
	defaultTestTimeout, err := time.ParseDuration(parsedConfig.DefaultTestCaseTimeout)
	s3AccessKey = parsedConfig.S3AccessKey
	s3SecretKey = parsedConfig.S3SecretKey
	s3bucketName = parsedConfig.S3BucketName
	s3BucketNameAlternate = parsedConfig.S3BucketNameAlternate

	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.SmokeTestMode, _ = strconv.ParseBool(os.Getenv("SMOKE_TEST_MODE"))
		test.DefaultTestTimeout = defaultTestTimeout
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(fmt.Errorf("fatal error config file: %s", err))
	}

	ethereumNodeURL = viper.GetString("ethereum_node_url")
	tokenAddress = viper.GetString("bridge.token_address")
	ethereumAddress = viper.GetString("bridge.ethereum_address")
}

const (
	zcnscOwner                      = "wallets/zcnsc_owner"
	scOwnerWallet                   = "wallets/sc_owner"
	blobberOwnerWallet              = "wallets/blobber_owner"
	miner01NodeDelegateWalletName   = "wallets/miner01_node_delegate"
	miner02NodeDelegateWalletName   = "wallets/miner02_node_delegate"
	miner03NodeDelegateWalletName   = "wallets/miner03_node_delegate"
	sharder01NodeDelegateWalletName = "wallets/sharder01_node_delegate"
	sharder02NodeDelegateWalletName = "wallets/sharder02_node_delegate"
)

var (
	miner01ID   string
	miner02ID   string
	miner03ID   string
	sharder01ID string
	sharder02ID string

	ethereumNodeURL       string
	tokenAddress          string
	ethereumAddress       string
	s3SecretKey           string
	s3AccessKey           string
	s3bucketName          string
	s3BucketNameAlternate string
	S3Client              *s3.S3
)

var (
	configPath string
	configDir  string
)

var tenderlyClient *tenderly.Client

func TestMain(m *testing.M) {
	configPath = os.Getenv("CONFIG_PATH")
	configDir = os.Getenv("CONFIG_DIR")

	if configDir == "" {
		configDir = getConfigDir()
	}

	if configPath == "" {
		configPath = "./zbox_config.yaml"
		cliutils.Logger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	configDir, _ = filepath.Abs(configDir)

	if !strings.EqualFold(strings.TrimSpace(os.Getenv("SKIP_CONFIG_CLEANUP")), "true") {
		if files, err := filepath.Glob("./config/*.json"); err == nil {
			for _, f := range files {
				// skip deleting the SC owner wallet and blobber owner wallet
				if strings.HasSuffix(f, zcnscOwner+"_wallet.json") ||
					strings.HasSuffix(f, scOwnerWallet+"_wallet.json") ||
					strings.HasSuffix(f, blobberOwnerWallet+"_wallet.json") ||
					strings.HasSuffix(f, miner01NodeDelegateWalletName+"_wallet.json") ||
					strings.HasSuffix(f, miner02NodeDelegateWalletName+"_wallet.json") ||
					strings.HasSuffix(f, miner03NodeDelegateWalletName+"_wallet.json") ||
					strings.HasSuffix(f, sharder01NodeDelegateWalletName+"_wallet.json") ||
					strings.HasSuffix(f, sharder02NodeDelegateWalletName+"_wallet.json") {
					continue
				}
				_ = os.Remove(f)
			}
		}

		if files, err := filepath.Glob("./config/*.txt"); err == nil {
			for _, f := range files {
				_ = os.Remove(f)
			}
		}

		if files, err := filepath.Glob("./tmp/*.txt"); err == nil {
			for _, f := range files {
				_ = os.Remove(f)
			}
		}
	}

	setupConfig()

	tenderlyClient = tenderly.NewClient(ethereumNodeURL)

	// Create a session with AWS
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-2"), // Replace with your desired AWS region
		Credentials: credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, ""),
	})
	if err != nil {
		log.Fatalln("Failed to create AWS session:", err)
		return
	}

	// Create an S3 client
	S3Client = s3.New(sess)

	err = tenderlyClient.InitBalance(ethereumAddress)
	if err != nil {
		log.Fatalln(err)
	}

	err = tenderlyClient.InitErc20Balance(tokenAddress, ethereumAddress)
	if err != nil {
		log.Fatalln(err)
	}

	exitRun := m.Run()

	os.Exit(exitRun)
}
