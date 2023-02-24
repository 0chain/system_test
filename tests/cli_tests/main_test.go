package cli_tests

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/spf13/viper"
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
	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.DefaultTestTimeout = defaultTestTimeout
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}
}

const (
	zcnscOwner                      = "wallets/zcnsc_owner"
	scOwnerWallet                   = "wallets/sc_owner"
	blobberOwnerWallet              = "wallets/blobber_owner"
	validatorOwnerWallet            = "wallets/validator_owner"
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
)

var (
	configPath             string
	configDir              string
	bridgeClientConfigFile string
	bridgeOwnerConfigFile  string
)

func TestMain(m *testing.M) {
	configPath = os.Getenv("CONFIG_PATH")
	configDir = os.Getenv("CONFIG_DIR")
	bridgeClientConfigFile = os.Getenv("BRIDGE_CONFIG_FILE")
	bridgeOwnerConfigFile = os.Getenv("BRIDGE_OWNER_CONFIG_FILE")

	if bridgeClientConfigFile == "" {
		bridgeClientConfigFile = DefaultConfigBridgeFileName
	}

	if bridgeOwnerConfigFile == "" {
		bridgeOwnerConfigFile = DefaultConfigOwnerFileName
	}

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

	exitRun := m.Run()
	os.Exit(exitRun)
}
