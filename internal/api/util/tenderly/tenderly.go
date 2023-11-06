// nolint:typecheck
package tenderly

import (
	"context"
	"errors"

	"github.com/ybbus/jsonrpc/v3"
)

const InitialBalance = "0x56BC75E2D63100000" // 100 ethers in hex

// Client represents Ethereum client, which
// uses Tenderly fork node to perform snapshots
// and revert changes using requests to EVM
type Client struct {
	client jsonrpc.RPCClient
}

func NewClient(tenderlyNodeURL string) *Client {
	client := jsonrpc.NewClient(tenderlyNodeURL)
	return &Client{
		client: client,
	}
}

// InitBalance sets pre-defined initial balance for the given ethereum address
func (c *Client) InitBalance(ethereumAddress string) error {
	resp, err := c.client.Call(context.Background(), "tenderly_setBalance", []string{ethereumAddress}, InitialBalance)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return errors.New(resp.Error.Error())
	}
	return nil
}

// InitErc20Balance sets pre-defined initial balance for the given erc20 token address
func (c *Client) InitErc20Balance(tokenAddress, ethereumAddress string) error {
	resp, err := c.client.Call(context.Background(), "tenderly_setErc20Balance", tokenAddress, ethereumAddress, InitialBalance)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return errors.New(resp.Error.Error())
	}
	return nil
}
