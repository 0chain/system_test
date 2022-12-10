package client

import (
	"context"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"log"
)

type ETHClient struct {
	ethereumClient *ethclient.Client
}

func NewETHClient(ethereumNodeURL string) *ETHClient {
	ethereumClient, err := ethclient.Dial(ethereumNodeURL)
	if err != nil {
		log.Fatalln(err)
	}
	return &ETHClient{
		ethereumClient: ethereumClient}
}

func (e *ETHClient) IsTransactionPending(t *test.SystemTest, hash string) bool {
	hashHex := common.HexToHash(hash)
	trx, pending, err := e.ethereumClient.TransactionByHash(context.Background(), hashHex)
	require.NoError(t, err)
	return trx == nil || pending
}
