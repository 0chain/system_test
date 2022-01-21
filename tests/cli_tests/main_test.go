package cli_tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/core/logger"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/gosdk/zboxcore/blockchain"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/gosdk/zcncore"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const scOwnerWallet = "sc_owner"
const blobberOwnerWallet = "blobber_owner"
const minerNodeDelegateWalletName = "miner_node_delegate"
const sharderNodeDelegateWalletName = "sharder_node_delegate"

const (
	ZCNStatusSuccess int = 0
	ZCNStatusError   int = 1
)

var configPath string
var cfgFile string
var networkFile string
var walletFile string
var walletClientID string
var walletClientKey string
var cDir string
var bSilent bool
var allocUnderRepair bool
var walletJSON string
var clientWallet *zcncrypto.Wallet

type ZCNStatus struct {
	walletString string
	wg           *sync.WaitGroup
	success      bool
	errMsg       string
}

func (zcn *ZCNStatus) OnWalletCreateComplete(status int, wallet string, err string) {
	defer zcn.wg.Done()
	if status == ZCNStatusError {
		zcn.success = false
		zcn.errMsg = err
		zcn.walletString = ""
		return
	}
	zcn.success = true
	zcn.errMsg = ""
	zcn.walletString = wallet
	return
}

func TestMain(m *testing.M) {
	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {
		configPath = "./zbox_config.yaml"
		cliutils.Logger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

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
	configDir := "./config"
	cfg, err := conf.LoadConfigFile(filepath.Join(configDir, configPath))
	if err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
	var networkFile string
	if networkFile == "" {
		networkFile = "network.yaml"
	}
	network, _ := conf.LoadNetworkFile(filepath.Join(".", networkFile))

	// syncing loggers
	logger.SyncLoggers([]*logger.Logger{zcncore.GetLogger(), sdk.GetLogger()})

	// set the log file
	zcncore.SetLogFile("cmdlog.log", false)
	sdk.SetLogFile("cmdlog.log", false)

	if network.IsValid() {
		zcncore.SetNetwork(network.Miners, network.Sharders)
		conf.InitChainNetwork(&conf.Network{
			Miners:   network.Miners,
			Sharders: network.Sharders,
		})
	}

	err = zcncore.InitZCNSDK(cfg.BlockWorker, cfg.SignatureScheme,
		zcncore.WithChainID(cfg.ChainID),
		zcncore.WithMinSubmit(cfg.MinSubmit),
		zcncore.WithMinConfirmation(cfg.MinConfirmation),
		zcncore.WithConfirmationChainLength(cfg.ConfirmationChainLength))
	if err != nil {
		fmt.Println("Error initializing core SDK.", err)
		os.Exit(1)
	}

	// is freshly created wallet?
	var fresh bool

	wallet := &zcncrypto.Wallet{}
	if (&walletClientID != nil) && (len(walletClientID) > 0) && (&walletClientKey != nil) && (len(walletClientKey) > 0) {
		wallet.ClientID = walletClientID
		wallet.ClientKey = walletClientKey
		var clientBytes []byte

		clientBytes, err = json.Marshal(wallet)
		walletJSON = string(clientBytes)
		if err != nil {
			fmt.Println("Invalid wallet data passed:" + walletClientID + " " + walletClientKey)
			os.Exit(1)
		}
		clientWallet = wallet
		fresh = false
	} else {
		var walletFilePath string
		if &walletFile != nil && len(walletFile) > 0 {
			if filepath.IsAbs(walletFile) {
				walletFilePath = walletFile
			} else {
				walletFilePath = configDir + string(os.PathSeparator) + walletFile
			}
		} else {
			walletFilePath = configDir + string(os.PathSeparator) + "wallet.json"
		}

		if _, err = os.Stat(walletFilePath); os.IsNotExist(err) {
			wg := &sync.WaitGroup{}
			statusBar := &ZCNStatus{wg: wg}
			wg.Add(1)
			err = zcncore.CreateWallet(statusBar)
			if err == nil {
				wg.Wait()
			} else {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			if len(statusBar.walletString) == 0 || !statusBar.success {
				fmt.Println("Error creating the wallet." + statusBar.errMsg)
				os.Exit(1)
			}
			fmt.Println("ZCN wallet created")
			walletJSON = string(statusBar.walletString)
			file, err := os.Create(walletFilePath)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			defer file.Close()
			fmt.Fprintf(file, walletJSON)

			fresh = true
		} else {
			f, err := os.Open(walletFilePath)
			if err != nil {
				fmt.Println("Error opening the wallet", err)
				os.Exit(1)
			}
			clientBytes, err := ioutil.ReadAll(f)
			if err != nil {
				fmt.Println("Error reading the wallet", err)
				os.Exit(1)
			}
			walletJSON = string(clientBytes)
		}
		//minerjson, _ := json.Marshal(miners)
		//sharderjson, _ := json.Marshal(sharders)
		err = json.Unmarshal([]byte(walletJSON), wallet)
		clientWallet = wallet
		if err != nil {
			fmt.Println("Invalid wallet at path:" + walletFilePath)
			os.Exit(1)
		}
	}

	//init the storage sdk with the known miners, sharders and client wallet info
	err = sdk.InitStorageSDK(walletJSON, cfg.BlockWorker, cfg.ChainID, cfg.SignatureScheme, cfg.PreferredBlobbers)
	if err != nil {
		fmt.Println("Error in sdk init", err)
		os.Exit(1)
	}

	// additional settings depending network latency
	blockchain.SetMaxTxnQuery(cfg.MaxTxnQuery)
	blockchain.SetQuerySleepTime(cfg.QuerySleepTime)

	conf.InitClientConfig(&cfg)

	if network.IsValid() {
		sdk.SetNetwork(network.Miners, network.Sharders)
	}

	sdk.SetNumBlockDownloads(10)

	if fresh {
		fmt.Println("Creating related read pool for storage smart-contract...")
		if err = sdk.CreateReadPool(); err != nil {
			fmt.Printf("Failed to create read pool: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Read pool created successfully")
	}

	exitRun := m.Run()
	os.Exit(exitRun)
}
