package client

import (
	"bytes"
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/0chain/gosdk/core/common"
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
		fileMeta, buf, false, false)
	require.NoError(t, err)
	require.Nil(t, chunkedUpload.Start())

	return filepath.Join("", filepath.Base(tmpFile.Name()))
}

// getBlobberNotPartOfAllocation returns a blobber not part of current allocation
func (c *SDKClient) GetBlobberNotPartOfAllocation(t *test.SystemTest, allocationID string) (string, error) {
	a, err := sdk.GetAllocation(allocationID)
	require.Nil(t, err)

	blobbers, err := sdk.GetBlobbers(true)
	require.Nil(t, err)

	for _, blobber := range blobbers {
		for _, b := range a.BlobberDetails {
			if blobber.ID != common.Key(b.BlobberID) {
				return b.BlobberID, nil
			}
		}
	}

	return "", fmt.Errorf("failed to get blobber not part of allocation")
}

func (c *SDKClient) GetRandomBlobber(t *test.SystemTest, except_blobber string) (string, error) {
	mrand.Seed(time.Now().Unix()) //nolint:gosec,revive
	blobbers, err := sdk.GetBlobbers(true)
	require.Nil(t, err)

	mrand.Shuffle(len(blobbers), func(i, j int) { blobbers[i], blobbers[j] = blobbers[j], blobbers[i] })
	for _, blobber := range blobbers {
		if blobber.ID != common.Key(except_blobber) {
			return string(blobber.ID), nil
		}
	}
	return "", fmt.Errorf("failed to get blobbers")
}

func (c *SDKClient) VerifyFileRefFromBlobber(t *test.SystemTest, allocationID, blobberID, remoteFile string) {
	fref, err := sdk.GetFileRefFromBlobber(allocationID, blobberID, remoteFile)
	require.Nil(t, err)
	require.NotNil(t, fref) // not nil when the file exists
}
