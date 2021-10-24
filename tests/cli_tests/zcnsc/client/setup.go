package client

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/gosdk/zcncore"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
)

const (
	configFileName = "0chain.yaml"
)

var (
	configFile     string
	configDir      string
	walletFile     string
	walletFilePath string
	silent         bool
	clientConfig   string
	minSubmit      int
	minCfm         int
	cfmChainLength int
	Wallet         *zcncrypto.Wallet
)

func init() {
	cfgFileFlag := flag.String("config", "", "config file (default is 0chain.yaml)")
	walletFileFlag := flag.String("wallet", "", "wallet file (default is wallet.json)")
	configDirFlag := flag.String("configDir", "", "configuration directory (default is $HOME/.zcn)")
	silentFlag := flag.Bool("silent", false, "Do not print sdk logs in stderr (prints by default)")

	flag.Parse()

	configDir = *configDirFlag
	configFile = *cfgFileFlag
	walletFile = *walletFileFlag
	silent = *silentFlag
}

func InitClient() {
	fmt.Println("---------------------")
	fmt.Println("Started InitClient...")
	initSdk()
	initWallet()

	err := registerWallet()
	if err != nil {
		ExitWithError(err.Error())
	}
}

func InitNewClientClient(path string) {
	fmt.Println("---------------------")
	fmt.Println("Started InitNewClientClient...")

	walletFilePath = path

	initWallet()

	err := registerWallet()
	if err != nil {
		ExitWithError(err.Error())
	}
}


func registerWallet() error {
	fmt.Println("---------------------------")
	fmt.Println("Started Register wallets...")

	statusBar := NewZCNStatus()

	statusBar.Begin()
	_ = zcncore.RegisterToMiners(Wallet, statusBar)
	statusBar.Wait()

	if statusBar.success {
		fmt.Println("Wallet registered at miners: ")
		fmt.Println("Wallet ClientID: " + Wallet.ClientID)
		fmt.Println("Wallet ClientKey: " + Wallet.ClientKey)
	} else {
		PrintError("Wallet registration failed. " + statusBar.errMsg)
		return errors.New(statusBar.errMsg)
	}
	return nil
}

func initWallet() {
	fmt.Println("---------------------")
	fmt.Println("Started InitWallet...")
	var fresh bool

	if _, err := os.Stat(walletFilePath); os.IsNotExist(err) {
		clientConfig = createWallet(walletFilePath)
		fresh = true
	} else {
		clientBytes := openWallet(walletFilePath)
		clientConfig = string(clientBytes)
	}

	wallet := &zcncrypto.Wallet{}
	err := json.Unmarshal([]byte(clientConfig), wallet)
	Wallet = wallet
	if err != nil {
		ExitWithError("Invalid wallet at path:" + walletFilePath)
	}

	err = zcncore.SetWalletInfo(clientConfig, false)
	if err != nil {
		ExitWithError(err.Error())
	}

	if fresh {
		log.Print("Creating related read pool for storage of smart-contract...")
		if err = createReadPool(); err != nil {
			log.Panicf("Failed to create read pool: %v", err)
		}
		log.Printf("Read pool created successfully")
	}
}

func initSdk() {
	fmt.Println("------------------")
	fmt.Println("Started InitSDK...")
	chainConfig := viper.New()
	configDir := readChainConfig(chainConfig)

	blockWorker := chainConfig.GetString("block_worker")
	signScheme := chainConfig.GetString("server_chain.signature_scheme")
	chainID := chainConfig.GetString("server_chain.id")
	minSubmit = chainConfig.GetInt("server_chain.min_submit")
	minCfm = chainConfig.GetInt("server_chain.min_confirmation")
	cfmChainLength = chainConfig.GetInt("server_chain.confirmation_chain_length")

	if len(walletFile) > 0 {
		walletFilePath = path.Join(configDir, walletFile)
	} else {
		walletFilePath = path.Join(configDir, "wallet.json")
	}

	zcncore.SetLogFile("cmdlog.log", !silent)

	err := zcncore.InitZCNSDK(
		blockWorker,
		signScheme,
		zcncore.WithChainID(chainID),
		zcncore.WithMinSubmit(minSubmit),
		zcncore.WithMinConfirmation(minCfm),
		zcncore.WithConfirmationChainLength(cfmChainLength),
	)
	if err != nil {
		ExitWithError(err.Error())
	}
}

func openWallet(walletFilePath string) []byte {
	f, err := os.Open(walletFilePath)
	if err != nil {
		ExitWithError("Error opening the wallet", err)
	}
	clientBytes, err := ioutil.ReadAll(f)
	if err != nil {
		ExitWithError("Error reading the wallet", err)
	}
	return clientBytes
}

func createWallet(walletFilePath string) string {
	fmt.Println("No wallet in path ", walletFilePath, "found. Creating wallet...")

	wg := &sync.WaitGroup{}
	statusBar := &ZCNStatus{wg: wg}
	wg.Add(1)

	err := zcncore.CreateWallet(statusBar)
	if err == nil {
		wg.Wait()
	} else {
		ExitWithError(err.Error())
	}

	if len(statusBar.walletString) == 0 || !statusBar.success {
		ExitWithError("Error creating the wallet." + statusBar.errMsg)
	}

	fmt.Println("ZCN wallet created!!")

	walletString := statusBar.walletString

	fmt.Println("Wallet string: " + walletString)
	file := createWalletPath(walletFilePath, walletString)
	_, _ = fmt.Fprint(file, walletString)

	return walletString
}

func createWalletPath(walletFilePath string, content string) *os.File {
	file, err := os.Create(walletFilePath)
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if err != nil {
		ExitWithError(err.Error())
	}

	_, err = file.Write([]byte(content))
	if err != nil {
		ExitWithError(err.Error())
	}

	return file
}

func readChainConfig(chainConfig *viper.Viper) string {
	var configDirLocal string

	if configDir != "" {
		configDirLocal = configDir
	} else {
		configDirLocal = getConfigDir()
	}

	chainConfig.AddConfigPath(configDirLocal)

	if len(configFile) > 0 {
		chainConfig.SetConfigFile(path.Join(configDirLocal, configFile))
	} else {
		chainConfig.SetConfigFile(path.Join(configDirLocal, configFileName))
	}

	if err := chainConfig.ReadInConfig(); err != nil {
		ExitWithError("Can't read config:", err)
	}

	return configDirLocal
}

func getConfigDir() string {
	if configDir != "" {
		return configDir
	}
	var configDir string
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	configDir = path.Join(home, ".zcn")
	return configDir
}
