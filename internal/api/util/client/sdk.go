package client

import (
	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"log"
)

type SDKClient struct {
	networkEntrypoint string
}

func NewSDKClient(networkEntrypoint string) *SDKClient {
	sdkClient := &SDKClient{
		networkEntrypoint: networkEntrypoint}

	conf.InitClientConfig(&conf.Config{
		BlockWorker:             networkEntrypoint,
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
		c.networkEntrypoint,
		"",
		crypto.BLS0Chain,
		nil,
		int64(wallet.Nonce))
	if err != nil {
		log.Fatalln(ErrInitStorageSDK)
	}
}
