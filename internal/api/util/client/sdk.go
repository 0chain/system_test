package client

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/0chain/gosdk/core/client"
	"github.com/0chain/gosdk/zcncore"

	"github.com/0chain/gosdk/constants"
	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/blockchain"
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
	wg       *sync.WaitGroup
	isRepair bool
	success  bool
	err      error
}

type MultiOperationOption func(alloc *sdk.Allocation)

func (cb *StatusCallback) Started(allocationId, filePath string, op, totalBytes int) {

}

func (cb *StatusCallback) InProgress(allocationId, filePath string, op, completedBytes int, data []byte) {
}

func (cb *StatusCallback) RepairCompleted(filesRepaired int) {
	if cb.err == nil {
		cb.success = true
	}
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
	cb.err = err
	if !cb.isRepair {
		cb.wg.Done()
	}
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

func (c *SDKClient) SetWallet(t *test.SystemTest, wallet *model.Wallet) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.wallet = &model.SdkWallet{
		ClientID:  wallet.Id,
		ClientKey: wallet.PublicKey,
		Keys: []*model.SdkKeyPair{{
			PrivateKey: wallet.Keys.PrivateKey.SerializeToHexStr(),
			PublicKey:  wallet.Keys.PublicKey.SerializeToHexStr(),
		}},
		Mnemonics: wallet.Mnemonics,
		Version:   wallet.Version,
	}

	serializedWallet, err := c.wallet.String()
	require.NoError(t, err, "failed to serialize wallet object", wallet)

	err = client.InitSDK(
		"{}",
		c.blockWorker,
		"",
		crypto.BLS0Chain,
		int64(wallet.Nonce), true,
	)
	require.NoError(t, err, ErrInitStorageSDK)

	err = zcncore.SetGeneralWalletInfo(serializedWallet, crypto.BLS0Chain)
	require.NoError(t, err, "Error in Setting general wallet info")
}

func (c *SDKClient) UploadFile(t *test.SystemTest, allocationID string, options ...int64) (tmpFilePath string, actualSizeUploaded int64) {
	fileSize := int64(65636)
	if len(options) > 0 {
		fileSize = int64(options[0])
	}
	return c.UploadFileWithParams(t, allocationID, fileSize, "")
}

func (c *SDKClient) UploadFileWithParams(t *test.SystemTest, allocationID string, fileSize int64, path string) (tmpFilePath string, actualSizeUploaded int64) {
	t.Log("Uploading file to allocation", allocationID)

	uploadOp := c.AddUploadOperation(t, path, "", fileSize)
	c.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

	return uploadOp.FileMeta.RemoteName, fileSize
}

func (c *SDKClient) UpdateFileWithParams(t *test.SystemTest, allocationID string, fileSize int64, path string) (tmpFilePath string, actualSizeUploaded int64) {
	t.Log("Updating file to allocation", allocationID)

	updateOp := c.AddUpdateOperation(t, "/"+filepath.Join("", path), path, fileSize)
	c.MultiOperation(t, allocationID, []sdk.OperationRequest{updateOp})

	return updateOp.FileMeta.RemoteName, fileSize
}

func (c *SDKClient) DeleteFile(t *test.SystemTest, allocationID, fpath string) {
	t.Logf("Deleting file %s from allocation %s", fpath, allocationID)
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	deleteOp := c.AddDeleteOperation(t, allocationID, "/"+filepath.Join("", filepath.Base(fpath)))
	c.MultiOperation(t, allocationID, []sdk.OperationRequest{deleteOp})
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
				if closer, ok := ops[i].FileReader.(io.Closer); ok {
					_ = closer.Close()
				}
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

func (c *SDKClient) AddUploadOperation(t *test.SystemTest, path, format string, opts ...int64) sdk.OperationRequest {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*"+format)
	if err != nil {
		require.NoError(t, err)
	}

	defer func(name string) {
		_ = os.RemoveAll(name)
	}(tmpFile.Name())

	var actualSize int64 = 1024
	if len(opts) > 0 {
		actualSize = opts[0]
	}

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive

	_, err = tmpFile.Write(rawBuf)
	require.NoError(t, err)
	_, err = tmpFile.Seek(0, 0)
	require.NoError(t, err)

	remoteName := filepath.Base(path)
	remotePath := "/" + filepath.Join(filepath.Dir(path), filepath.Base(path))
	if path == "" {
		remoteName = filepath.Base(tmpFile.Name())
		remotePath = "/" + filepath.Join("", filepath.Base(tmpFile.Name()))
	}

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: remoteName,
		RemotePath: remotePath,
	}

	t.Log("fileMeta", fileMeta)

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	return sdk.OperationRequest{
		OperationType: constants.FileOperationInsert,
		FileReader:    tmpFile,
		FileMeta:      fileMeta,
		Workdir:       homeDir,
		RemotePath:    fileMeta.RemotePath,
	}
}

// fileSize in GB number, eg for 1GB file, fileSize = 1
func (c *SDKClient) AddUploadOperationForBigFile(t *test.SystemTest, allocationID string, fileSize int) sdk.OperationRequest {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	for i := 0; i < fileSize; i++ {
		buf := make([]byte, 1024*1024*1024)
		_, err = rand.Read(buf)
		require.NoError(t, err)
		_, err = tmpFile.Write(buf)
		require.NoError(t, err)
	}
	_, err = tmpFile.Seek(0, 0)
	require.NoError(t, err)
	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: 1024 * 1024 * 1024 * int64(fileSize),
		RemoteName: filepath.Base(tmpFile.Name()),
		RemotePath: "/" + filepath.Join("", filepath.Base(tmpFile.Name())),
	}

	homeDir, err := config.GetHomeDir()
	require.NoError(t, err)

	return sdk.OperationRequest{
		OperationType: constants.FileOperationInsert,
		FileReader:    tmpFile,
		FileMeta:      fileMeta,
		Workdir:       homeDir,
		RemotePath:    fileMeta.RemotePath,
		Opts: []sdk.ChunkedUploadOption{
			sdk.WithChunkNumber(500),
		},
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
		DestName:      newName,
	}
}

func (c *SDKClient) AddUpdateOperation(t *test.SystemTest, remotePath, remoteName string, fileSize int64) sdk.OperationRequest {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	tmpFile, err := os.CreateTemp("", "*")
	if err != nil {
		require.NoError(t, err)
	}

	rawBuf := make([]byte, fileSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive
	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: fileSize,
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

	const fileSize int64 = 1024

	rawBuf := make([]byte, fileSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		require.NoError(t, err)
	} //nolint:gosec,revive

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: fileSize,
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
		// Set consensus threshold to DataShards + 1 (standard formula for repair operations)
		// This ensures we have enough consensus even if some blobbers fail
		consensusThreshold := alloc.DataShards + 1
		alloc.SetConsensusThreshold(consensusThreshold)
		alloc.Blobbers = blobbers
	}
}
