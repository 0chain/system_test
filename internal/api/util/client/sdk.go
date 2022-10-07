package client

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/stretchr/testify/require"
)

type SDKClient struct {
	blockWorker string
	wallet      *model.Wallet
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

func (c *SDKClient) SetWallet(wallet *model.Wallet) {
	c.wallet = wallet

	err := sdk.InitStorageSDK(
		wallet.String(),
		c.blockWorker,
		"",
		crypto.BLS0Chain,
		nil,
		int64(wallet.Nonce))
	if err != nil {
		log.Fatalln(ErrInitStorageSDK, err)
	}
}

func (c *SDKClient) UploadFileWithSpecificName(t *testing.T, allocationID string) string {
	tmpFile, err := ioutil.TempFile("", "*")
	if err != nil {
		log.Fatalln(err)
	}

	defer func(name string) {
		err := os.Remove(name)
		if err != nil {

		}
	}(tmpFile.Name())

	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
	}(tmpFile)

	const actualSize int64 = 1024

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		log.Fatalln(err)
	} //nolint:gosec,revive

	buf := bytes.NewBuffer(rawBuf)

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(tmpFile.Name()),
		RemotePath: filepath.Join("/", filepath.Base(tmpFile.Name())),
	}

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.Nil(t, err)

	homeDir, err := config.GetHomeDir()
	require.Nil(t, err)

	chunkedUpload, err := sdk.CreateChunkedUpload(homeDir, sdkAllocation,
		fileMeta, buf, false, false)
	require.Nil(t, err)
	require.Nil(t, chunkedUpload.Start())

	err = sdkAllocation.CommitMetaTransaction(filepath.Join("/", filepath.Base(tmpFile.Name())), "Upload", "", "", nil, new(model.StubStatusBar))
	require.Nil(t, err)

	c.wallet.IncNonce()

	return filepath.Join("/", filepath.Base(tmpFile.Name()))
}

func (c *SDKClient) UploadSomeFile(t *testing.T, allocationID string) {
	tmpFile, err := ioutil.TempFile("", "*")
	if err != nil {
		log.Fatalln(err)
	}

	defer func(name string) {
		err := os.Remove(name)
		if err != nil {

		}
	}(tmpFile.Name())

	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
	}(tmpFile)

	const actualSize int64 = 1024

	rawBuf := make([]byte, actualSize)
	_, err = rand.Read(rawBuf)
	if err != nil {
		log.Fatalln(err)
	} //nolint:gosec,revive

	buf := bytes.NewBuffer(rawBuf)

	fileMeta := sdk.FileMeta{
		Path:       tmpFile.Name(),
		ActualSize: actualSize,
		RemoteName: filepath.Base(tmpFile.Name()),
		RemotePath: filepath.Join("/", filepath.Base(tmpFile.Name())),
	}

	sdkAllocation, err := sdk.GetAllocation(allocationID)
	require.Nil(t, err)

	homeDir, err := config.GetHomeDir()
	require.Nil(t, err)

	chunkedUpload, err := sdk.CreateChunkedUpload(homeDir, sdkAllocation,
		fileMeta, buf, false, false)
	require.Nil(t, err)
	require.Nil(t, chunkedUpload.Start())

	err = sdkAllocation.CommitMetaTransaction(filepath.Join("/", filepath.Base(tmpFile.Name())), "Upload", "", "", nil, new(model.StubStatusBar))
	require.Nil(t, err)

	c.wallet.IncNonce()
}
