package client

import (
	"bytes"
	"crypto/rand"
	"errors"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

type SDKClient struct {
	mu sync.Mutex

	blockWorker string
	wallet      *model.SdkWallet
}

func NewSDKClient(blockWorker string) *SDKClient {
	sdkClient := &SDKClient{
		blockWorker: blockWorker}

	conf.InitClientConfig(&conf.Config{
		BlockWorker:             blockWorker,
		SignatureScheme:         crypto.BLS0Chain,
		MinSubmit:               50,
		MinConfirmation:         50,
		ConfirmationChainLength: 3,
	})

	return sdkClient
}

// StartSession executes all actions in one sdk client session
func (c *SDKClient) StartSession(callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	callback()
}

func (c *SDKClient) SetWallet(t *test.SystemTest, wallet *model.Wallet, mnemonics string) {
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
