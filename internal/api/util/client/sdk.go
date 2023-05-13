package client

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/0chain/gosdk/constants"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/gosdk/core/conf"
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

func (c *SDKClient) UploadFile(t *test.SystemTest, allocationID string) string {
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

	return filepath.Join("", filepath.Base(tmpFile.Name()))
}

// getBlobberNotPartOfAllocation returns a blobber not part of current allocation
func (c *SDKClient) InitSDK(wallet string) error {
	f, err := os.Open(wallet)
	if err != nil {
		return nil
	}
	clientBytes, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	walletJSON := string(clientBytes)

	err = sdk.InitStorageSDK(
		walletJSON,
		c.blockWorker,
		"",
		crypto.BLS0Chain,
		nil,
		0,
	)
	return err
}

func (c *SDKClient) GetBlobberNotPartOfAllocation(t *test.SystemTest, walletname, allocationID string) (string, error) {
	err := c.InitSDK(walletname)
	if err != nil {
		return "", err
	}

	a, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", nil
	}

	blobbers, err := sdk.GetBlobbers(true)
	if err != nil {
		return "", err
	}

	for _, blobber := range blobbers {
		for _, b := range a.BlobberDetails {
			if blobber.ID != common.Key(b.BlobberID) {
				return b.BlobberID, nil
			}
		}
	}

	return "", fmt.Errorf("failed to get blobber not part of allocation")
}

func generateRandomIndex(sliceLen int64) (*big.Int, error) {
	// Generate a random index within the range of the slice
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(sliceLen))
	if err != nil {
		return nil, err
	}
	return randomIndex, nil
}

func (c *SDKClient) GetRandomBlobber(t *test.SystemTest, walletname, except_blobber string) (string, error) {
	err := c.InitSDK(walletname)
	if err != nil {
		return "", err
	}
	blobbers, err := sdk.GetBlobbers(true)
	require.Nil(t, err)

	var randomBlobber string
	for range blobbers {
		randomIndex, err := generateRandomIndex(int64(len(blobbers)))
		require.Nil(t, err)

		blobber := blobbers[randomIndex.Int64()].ID
		if blobber != common.Key(except_blobber) {
			randomBlobber = string(blobber)
			break
		}
	}

	return randomBlobber, fmt.Errorf("failed to get blobbers")
}

func (c *SDKClient) VerifyFileRefFromBlobber(t *test.SystemTest, walletname, allocationID, blobberID, remoteFile string) {
	err := c.InitSDK(walletname)
	require.Nil(t, err)

	fref, err := sdk.GetFileRefFromBlobber(allocationID, blobberID, remoteFile)
	require.Nil(t, err)
	require.NotNil(t, fref) // not nil when the file exists
}

func (c *SDKClient) GetFileList(t *test.SystemTest, allocationID, path string) *sdk.ListResult {
	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	fileList, err := sdkAllocation.ListDir(path)
	require.NoError(t, err)

	return fileList
}

func (c *SDKClient) MultiOperation(t *test.SystemTest, allocationID string, ops []sdk.OperationRequest) {
	defer func() {
		for i := 0; i < len(ops); i++ {
			if ops[i].OperationType == constants.FileOperationInsert || ops[i].OperationType == constants.FileOperationUpdate {
				_ = os.RemoveAll(ops[i].FileMeta.Path)
			}
		}
	}()

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	err = sdkAllocation.DoMultiOperation(ops)
	require.NoError(t, err)
}

func (c *SDKClient) AddUploadOperation(t *test.SystemTest, allocationID string) sdk.OperationRequest {
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
