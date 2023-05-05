package client

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

type SDKClient struct {
	Mutex sync.Mutex

	blockWorker string
	wallet      *model.SdkWallet
}

type StatusCallback struct {
	wg      *sync.WaitGroup
	success bool
}

func (cb *StatusCallback) Started(allocationId, filePath string, op, totalBytes int) {
	cb.wg.Add(1)
}

func (cb *StatusCallback) InProgress(allocationId, filePath string, op, completedBytes int, data []byte) {
}

func (cb *StatusCallback) RepairCompleted(filesRepaired int) {}

func (cb *StatusCallback) Completed(allocationId, filePath, filename, mimetype string, size, op int) {
	cb.success = true
	cb.wg.Done()
}

func (cb *StatusCallback) Error(allocationID, filePath string, op int, err error) {
	cb.success = false
	cb.wg.Done()
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

func (c *SDKClient) SetWallet(t *test.SystemTest, wallet *model.Wallet, mnemonics string) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
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

func (c *SDKClient) UploadFile(t *test.SystemTest, allocationID string) (tmpFilePath string, actualSizeUploaded int64) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	defer func(name string) {
		_ = os.RemoveAll(name)
	}(tmpFile.Name())

	const actualSize int64 = 1024

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive

	buf := bytes.NewBuffer(rawBuf)

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(tmpFile.Name()),
		RemotePath: "/" + filepath.Join("", filepath.Base(tmpFile.Name())),
	}

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	chunkedUpload, err := sdk.CreateChunkedUpload(homeDir, sdkAllocation,
		fileMeta, buf, false, false, false)
	require.NoError(t, err)
	require.Nil(t, chunkedUpload.Start())

	return filepath.Join("", filepath.Base(tmpFile.Name())), actualSize
}

func (c *SDKClient) DeleteFile(t *test.SystemTest, allocationID, fpath string) {
	t.Logf("Deleting file %s from allocation %s", fpath, allocationID)
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	err = sdkAllocation.DeleteFile("/" + filepath.Join("", filepath.Base(fpath)))
	require.NoError(t, err)
}

func (c *SDKClient) UpdateFileBigger(t *test.SystemTest, allocationID, fpath string, fsize int64) (tmpFilePath string, actualSizeUploaded int64) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	defer func(name string) {
		_ = os.RemoveAll(name)
	}(tmpFile.Name())

	actualSize := fsize * 2

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive

	buf := bytes.NewBuffer(rawBuf)

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(fpath),
		RemotePath: "/" + filepath.Join("", filepath.Base(fpath)),
	}

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	chunkedUpload, err := sdk.CreateChunkedUpload(homeDir, sdkAllocation,
		fileMeta, buf, true, false)
	require.NoError(t, err)
	require.Nil(t, chunkedUpload.Start())

	return fpath, actualSize
}

func (c *SDKClient) UpdateFileSmaller(t *test.SystemTest, allocationID, fpath string, fsize int64) (tmpFilePath string, actualSizeUploaded int64) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	require.Greater(t, fsize, int64(0), "Cannot create a file with size less than 0")

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	defer func(name string) {
		_ = os.RemoveAll(name)
	}(tmpFile.Name())

	actualSize := fsize / 2

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive

	buf := bytes.NewBuffer(rawBuf)

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(fpath),
		RemotePath: "/" + filepath.Join("", filepath.Base(fpath)),
	}

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	chunkedUpload, err := sdk.CreateChunkedUpload(homeDir, sdkAllocation,
		fileMeta, buf, true, false)
	require.NoError(t, err)
	require.Nil(t, chunkedUpload.Start())

	return fpath, actualSize
}

func (c *SDKClient) DownloadFile(t *test.SystemTest, allocationID, remotepath, localpath string) {
	t.Logf("Downloading file %s to %s from allocation %s", remotepath, localpath, allocationID)
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	err = sdkAllocation.DownloadFile(localpath, "/"+remotepath, false, &StatusCallback{
		wg: &sync.WaitGroup{},
	})
	require.NoError(t, err)
}
