package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/gosdk/zcnbridge"
	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

var (
	walletFileName       = "wallet.json"
	configDir            = "./config"
	configBridgeFileName = "bridge.yaml"
	configChainFileName  = "api_tests_config.yaml"
	logPath              = "logs"
	loglevel             = "info"
	development          = false
)

type SDKClient struct {
	mu sync.Mutex

	blockWorker string
	wallet      *model.SdkWallet

	bridge *zcnbridge.BridgeClient
}

func NewSDKClient(blockWorker, ethereumNodeURL string) *SDKClient {
	sdkClient := &SDKClient{
		blockWorker: blockWorker}

	conf.InitClientConfig(&conf.Config{
		BlockWorker:             blockWorker,
		SignatureScheme:         crypto.BLS0Chain,
		MinSubmit:               50,
		MinConfirmation:         50,
		ConfirmationChainLength: 3,
	})

	clientBytes, err := os.ReadFile(filepath.Join(configDir, walletFileName))
	if err != nil {
		log.Fatalln(err)
	}
	clientConfig := string(clientBytes)

	var wallet zcncrypto.Wallet
	err = json.Unmarshal(clientBytes, &wallet)
	if err != nil {
		log.Fatalln(err)
	}

	err = zcncore.SetWalletInfo(clientConfig, false)
	if err != nil {
		log.Fatalln(err)
	}

	err = zcncore.InitZCNSDK(blockWorker, "bls0chain",
		zcncore.WithChainID(""),
		zcncore.WithMinConfirmation(50),
		zcncore.WithMinSubmit(50),
		zcncore.WithConfirmationChainLength(3),
		zcncore.WithEthereumNode(ethereumNodeURL))
	if err != nil {
		log.Fatalln(err)
	}

	var configDirAbs string
	configDirAbs, err = filepath.Abs(configDir)
	if err != nil {
		log.Fatalln(err)
	}

	cfg := &zcnbridge.BridgeSDKConfig{
		ConfigDir:        &configDirAbs,
		ConfigBridgeFile: &configBridgeFileName,
		ConfigChainFile:  &configChainFileName,
		LogPath:          &logPath,
		LogLevel:         &loglevel,
		Development:      &development,
	}

	sdkClient.bridge = zcnbridge.SetupBridgeClientSDK(cfg, walletFileName)

	return sdkClient
}

func (c *SDKClient) SetWallet(t *test.SystemTest, wallet *model.Wallet, mnemonics string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.wallet = &model.SdkWallet{
		ClientID:  wallet.Id,
		ClientKey: wallet.PublicKey,
		Keys: []*model.SdkKeyPair{{
			PrivateKey: wallet.Keys.PrivateKey.SerializeToHexStr(),
			PublicKey:  wallet.Keys.PublicKey.SerializeToHexStr(),
		}},
		Mnemonics: mnemonics,
		Version:   wallet.Version,
	}

	serializedWallet, err := c.wallet.String()
	require.NoError(t, err, "failed to serialize wallet object", wallet)

	err = sdk.InitStorageSDK(
		serializedWallet,
		c.blockWorker,
		"",
		crypto.BLS0Chain,
		nil,
		int64(wallet.Nonce),
	)
	require.NoError(t, err, ErrInitStorageSDK)
}

func (c *SDKClient) UploadFile(t *test.SystemTest, allocationID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	defer func(name string) {
		require.Nil(t, os.RemoveAll(name))
	}(tmpFile.Name())

	const actualSize int64 = 1024

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	require.Nil(t, err)

	buf := bytes.NewBuffer(rawBuf)

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(tmpFile.Name()),
		RemotePath: filepath.Join(string(filepath.Separator), filepath.Base(tmpFile.Name())),
	}

	var sdkAllocation *sdk.Allocation
	sdkAllocation, err = sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	var homeDir string
	homeDir, err = config.GetHomeDir()
	require.NoError(t, err)

	var chunkedUpload *sdk.ChunkedUpload
	chunkedUpload, err = sdk.CreateChunkedUpload(homeDir, sdkAllocation,
		fileMeta, buf, false, false)
	require.NoError(t, err)
	require.Nil(t, chunkedUpload.Start())

	return filepath.Join(string(filepath.Separator), filepath.Base(tmpFile.Name()))
}

func (c *SDKClient) DownloadFile(t *test.SystemTest, allocationID, remotePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	allocation, err := sdk.GetAllocation(allocationID)
	require.Nil(t, err)

	var workingDir string
	workingDir, err = os.Getwd()
	require.Nil(t, err)

	localPath := filepath.Join(workingDir, path.Base(remotePath))
	err = allocation.DownloadFile(localPath, remotePath, new(model.StatusCallback))
	require.Nil(t, err)

	wait.PoolImmediately(t, time.Second*30, func() bool {
		if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
			return false
		}
		err = os.Remove(localPath)
		return err == nil
	})
}

func (c *SDKClient) IncreaseAllowance(t *test.SystemTest, amount uint64) {
	transaction, err := c.bridge.IncreaseBurnerAllowance(context.Background(), zcnbridge.Wei(amount))
	require.NoError(t, err)

	hash := transaction.Hash().Hex()
	_, err = zcnbridge.ConfirmEthereumTransaction(hash, 200, time.Second*2)
	require.NoError(t, err)
}

func (c *SDKClient) BurnWZCN(t *test.SystemTest, amount uint64) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	transaction, err := c.bridge.BurnWZCN(context.Background(), amount)
	require.NoError(t, err)
	return transaction.Hash().String()
}

func (c *SDKClient) BurnZCN(t *test.SystemTest, amount uint64) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	transaction, err := c.bridge.BurnZCN(context.Background(), amount)
	require.NoError(t, err)
	return transaction.Hash
}

func (c *SDKClient) MintZCN(t *test.SystemTest, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	payload, err := c.bridge.QueryZChainMintPayload(hash)
	require.NoError(t, err)
	_, err = c.bridge.MintZCN(context.Background(), payload)
	require.NoError(t, err)
}
