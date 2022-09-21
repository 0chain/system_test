package client

import (
	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"log"
)

type SDKClient struct {
	blockWorker string
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
