package cli_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const (
	zcnscOwner                      = "wallets/zcnsc_owner"
	scOwnerWallet                   = "wallets/sc_owner"
	blobberOwnerWallet              = "wallets/blobber_owner"
	miner01NodeDelegateWalletName   = "wallets/miner01_node_delegate"
	miner02NodeDelegateWalletName   = "wallets/miner02_node_delegate"
	miner03NodeDelegateWalletName   = "wallets/miner03_node_delegate"
	sharder01NodeDelegateWalletName = "wallets/sharder01_node_delegate"
	sharder02NodeDelegateWalletName = "wallets/sharder02_node_delegate"
	minerNodeWalletName             = "wallets/miner_node"
	sharderNodeWalletName           = "wallets/sharder_node"
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
					strings.HasSuffix(f, sharder02NodeDelegateWalletName+"_wallet.json") ||
					strings.HasSuffix(f, minerNodeWalletName+"_wallet.json") ||
					strings.HasSuffix(f, sharderNodeWalletName+"_wallet.json") {
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

	exitRun := m.Run()
	os.Exit(exitRun)
}
