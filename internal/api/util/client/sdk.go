package client

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/0chain/gosdk/constants"
	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/blockchain"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/gosdk/zboxcore/zboxutil"
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
	wg       *sync.WaitGroup
	isRepair bool
	success  bool
}

type MultiOperationOption func(alloc *sdk.Allocation)

func (cb *StatusCallback) Started(allocationId, filePath string, op, totalBytes int) {

}

func (cb *StatusCallback) InProgress(allocationId, filePath string, op, completedBytes int, data []byte) {
}

func (cb *StatusCallback) RepairCompleted(filesRepaired int) {
	cb.success = true
	cb.wg.Done()
}

func (cb *StatusCallback) Completed(allocationId, filePath, filename, mimetype string, size, op int) {
	if !cb.isRepair {
		cb.success = true
		cb.wg.Done()
	}
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
		fileMeta, buf, false, false, false, zboxutil.NewConnectionId())
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
		fileMeta, buf, true, false, false, zboxutil.NewConnectionId())
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
		fileMeta, buf, true, false, false, zboxutil.NewConnectionId())
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
	wg := &sync.WaitGroup{}
	wg.Add(1)
	err = sdkAllocation.DownloadFile(localpath, "/"+remotepath, false, &StatusCallback{
		wg: wg,
	}, true)
	require.NoError(t, err)
	wg.Wait()
}

func (c *SDKClient) DownloadFileWithParam(t *test.SystemTest, alloc *sdk.Allocation, remotepath, localpath string, wg *sync.WaitGroup, isFinal bool) {
	wg.Add(1)
	err := alloc.DownloadFile(localpath, remotepath, false, &StatusCallback{
		wg: wg,
	}, isFinal)
	require.NoError(t, err)
}

func (c *SDKClient) GetFileList(t *test.SystemTest, allocationID, path string) *sdk.ListResult {
	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	fileList, err := sdkAllocation.ListDir(path)
	require.NoError(t, err)

	return fileList
}

func (c *SDKClient) Rollback(t *test.SystemTest, allocationID string) {
	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	status, err := sdkAllocation.GetCurrentVersion()
	require.NoError(t, err)
	require.True(t, status)
}

func (c *SDKClient) MultiOperation(t *test.SystemTest, allocationID string, ops []sdk.OperationRequest, multiOps ...MultiOperationOption) {
	defer func() {
		for i := 0; i < len(ops); i++ {
			if ops[i].OperationType == constants.FileOperationInsert || ops[i].OperationType == constants.FileOperationUpdate {
				_ = os.RemoveAll(ops[i].FileMeta.Path)
			}
		}
	}()

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	for _, opt := range multiOps {
		opt(sdkAllocation)
	}

	err = sdkAllocation.DoMultiOperation(ops)
	require.NoError(t, err)
}

func (c *SDKClient) AddUploadOperation(t *test.SystemTest, allocationID string, opts ...int64) sdk.OperationRequest {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	var actualSize int64 = 1024
	if len(opts) > 0 {
		actualSize = opts[0]
	}

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(tmpFile.Name()),
		RemotePath: "/" + filepath.Join("", filepath.Base(tmpFile.Name())),
	}

	buf := bytes.NewBuffer(rawBuf)

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	return sdk.OperationRequest{
		OperationType: constants.FileOperationInsert,
		FileReader:    buf,
		FileMeta:      fileMeta,
		Workdir:       homeDir,
		RemotePath:    fileMeta.RemotePath,
	}
}

func (c *SDKClient) AddDeleteOperation(t *test.SystemTest, allocationID, remotePath string) sdk.OperationRequest {
	return sdk.OperationRequest{
		OperationType: constants.FileOperationDelete,
		RemotePath:    remotePath,
	}
}

func (c *SDKClient) AddRenameOperation(t *test.SystemTest, allocationID, remotePath, newName string) sdk.OperationRequest {
	return sdk.OperationRequest{
		OperationType: constants.FileOperationRename,
		RemotePath:    remotePath,
		DestName:      "/" + filepath.Join("", newName),
	}
}

func (c *SDKClient) AddUpdateOperation(t *test.SystemTest, allocationID, remotePath, remoteName string) sdk.OperationRequest {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	const actualSize int64 = 1024

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive
	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemotePath: remotePath,
		RemoteName: remoteName,
	}

	buf := bytes.NewBuffer(rawBuf)

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	return sdk.OperationRequest{
		OperationType: constants.FileOperationUpdate,
		FileReader:    buf,
		FileMeta:      fileMeta,
		Workdir:       homeDir,
	}
}

func (c *SDKClient) AddMoveOperation(t *test.SystemTest, allocationID, remotePath, destPath string) sdk.OperationRequest {
	return sdk.OperationRequest{
		OperationType: constants.FileOperationMove,
		RemotePath:    remotePath,
		DestPath:      destPath,
	}
}

func (c *SDKClient) AddCopyOperation(t *test.SystemTest, allocationID, remotePath, destPath string) sdk.OperationRequest {
	return sdk.OperationRequest{
		OperationType: constants.FileOperationCopy,
		RemotePath:    remotePath,
		DestPath:      destPath,
	}
}

func (c *SDKClient) AddUploadOperationWithPath(t *test.SystemTest, allocationID, remotePath string) sdk.OperationRequest {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	const actualSize int64 = 1024

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(tmpFile.Name()),
		RemotePath: remotePath + filepath.Join("", filepath.Base(tmpFile.Name())),
	}

	buf := bytes.NewBuffer(rawBuf)

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	return sdk.OperationRequest{
		OperationType: constants.FileOperationInsert,
		FileReader:    buf,
		FileMeta:      fileMeta,
		Workdir:       homeDir,
	}
}

func (c *SDKClient) AddCreateDirOperation(t *test.SystemTest, allocationID, remotePath string) sdk.OperationRequest {
	return sdk.OperationRequest{
		OperationType: constants.FileOperationCreateDir,
		RemotePath:    remotePath,
	}
}

func (c *SDKClient) RepairAllocation(t *test.SystemTest, allocationID string) {
	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	statusBar := &StatusCallback{
		wg:       wg,
		isRepair: true,
	}
	err = sdkAllocation.RepairAlloc(statusBar)
	require.NoError(t, err)
	wg.Wait()
	require.True(t, statusBar.success)
}

func WithRepair(blobbers []*blockchain.StorageNode) MultiOperationOption {
	return func(alloc *sdk.Allocation) {
		alloc.SetConsensusThreshold()
		alloc.Blobbers = blobbers
	}
}
