package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const scOwnerWallet = "sc_owner"
const blobberOwnerWallet = "blobber_owner"
const minerNodeDelegateWalletName = "miner_node_delegate"
const sharderNodeDelegateWalletName = "sharder_node_delegate"

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
				if strings.HasSuffix(f, scOwnerWallet+"_wallet.json") || strings.HasSuffix(f, blobberOwnerWallet+"_wallet.json") ||
					strings.HasSuffix(f, minerNodeDelegateWalletName+"_wallet.json") || strings.HasSuffix(f, sharderNodeDelegateWalletName+"_wallet.json") {
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

	// Blobber-stake
	walletName := "blobber-stake-wallet_wallet.json"
	_, err := cliutils.RunCommandWithoutRetry("./zbox register --silent " + "--wallet " + walletName + " --configDir ./config --config " + configPath)
	ExitWithError(err)

	blobbers := []climodel.BlobberInfo{}
	output, err := cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zbox ls-blobbers --json --silent --wallet %s_wallet.json --configDir ./config --config %s", walletName, configPath))
	ExitWithError(err)

	err = json.Unmarshal([]byte(output[0]), &blobbers)
	ExitWithError(err)

	if len(blobbers) <= 0 {
		cliutils.Logger.Error("No blobbers found in blobber list")
		os.Exit(1)
	}

	if len(blobbers) < 10 {
		_, err = cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet faucet --methodName pour --tokens %d --input {} --silent --wallet %s --configDir ./config --config %s",
			len(blobbers),
			walletName,
			configPath))
		ExitWithError(err)
	} else {
		for i := 0; i < len(blobbers); i++ {
			_, err = cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet faucet --methodName pour --tokens %d --input {} --silent --wallet %s --configDir ./config --config %s",
				1,
				walletName,
				configPath))
			ExitWithError(err)
		}
	}

	cliutils.Logger.Info("Enough balance. Moving to staking now...")

	done := 0
	for done < len(blobbers) {
		blobber := blobbers[done]
		cliutils.Logger.Info("Getting stake pool info for :", blobber.Id)
		output, err = cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zbox sp-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", "--blobber_id "+blobber.Id+" --json", walletName, configPath))
		ExitWithError(err)

		stakePool := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		ExitWithError(err)

		if stakePool.Balance < 1 {
			output, err = cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", " --blobber_id "+blobber.Id+" --json ", walletName, configPath))
			if err != nil {
				ExitWithError(err)
			}
			cliutils.Logger.Info("Successfully staked for blobber ", blobber.Id, " with output ", output)
			done++
		} else {
			fmt.Println("Blobber already have stake pool, URL : ", blobber.Id, " poolID : ", stakePool.ID, " amount : ", stakePool.Balance)
			done++
		}
	}

	exitRun := m.Run()
	os.Exit(exitRun)
}

func ExitWithError(err error) {
	if err != nil {
		cliutils.Logger.Error(err)
		os.Exit(1)
	}
}
