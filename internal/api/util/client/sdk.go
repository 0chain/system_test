package client

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/0chain/gosdk/constants"
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

func (c *SDKClient) MultiOperation(t *test.SystemTest, allocationID string, ops []sdk.OperationRequest) {

	defer func() {
		for _, op := range ops {
			if op.OperationType == constants.FileOperationInsert || op.OperationType == constants.FileOperationUpdate {
				_ = os.RemoveAll(op.FileMeta.Path)
			}
		}
	}()

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err)

	err = sdkAllocation.DoMultiOperation(ops)
	require.NoError(t, err)
}

func (c *SDKClient) AddUploadOperation(t *test.SystemTest, allocationID string) sdk.OperationRequest {
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

func (c *SDKClient) AddDeleteOperation(t *test.SystemTest, allocationID string, remotePath string) sdk.OperationRequest {

	return sdk.OperationRequest{
		OperationType: constants.FileOperationDelete,
		RemotePath:    remotePath,
	}
}

func (c *SDKClient) AddRenameOperation(t *test.SystemTest, allocationID string, remotePath string, newName string) sdk.OperationRequest {

	return sdk.OperationRequest{
		OperationType: constants.FileOperationRename,
		RemotePath:    remotePath,
		DestName:      "/" + filepath.Join("", newName),
	}
}

func (c *SDKClient) AddUpdateOperation(t *test.SystemTest, allocationID string, remotePath, remoteName string) sdk.OperationRequest {

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
