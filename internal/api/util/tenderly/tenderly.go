package tenderly

import (
	"context"
	"errors"

	jsonrpc "github.com/ybbus/jsonrpc/v3" // nolint
)

// Client represents Ethereum client, which
// uses Tenderly fork node to perform snapshots
// and revert changes using requests to EVM
type Client struct {
	client jsonrpc.RPCClient // nolint
}

func NewClient(tenderlyNodeURL string) *Client {
	client := jsonrpc.NewClient(tenderlyNodeURL) // nolint
	return &Client{
		client: client,
	}
}

// Creates network snapshot with a help of Ethereum JSON-RPC method call.
// Returns snapshot hash, which is available to be used to revert a state of the network
func (c *Client) CreateSnapshot() (string, error) {
	resp, err := c.client.Call(context.Background(), "evm_snapshot")
	if err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", errors.New(resp.Error.Error())
	}
	result, ok := resp.Result.(string)
	if !ok {
		return "", ErrConversion
	}
	return result, nil
}

// Reverts a state of Ethereum network using snapshot hash with a help of Ethereum JSON-RPC method call.
func (c *Client) Revert(snapshotHash string) error {
	resp, err := c.client.Call(context.Background(), "evm_revert", snapshotHash)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return errors.New(resp.Error.Error())
	}
	return nil
}
