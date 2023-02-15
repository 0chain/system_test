package tenderly

import (
	"context"
	"fmt"
	"github.com/ybbus/jsonrpc/v3"
)

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

func (c *Client) CreateSnapshot() (string, error) {
	resp, err := c.client.Call(context.Background(), "evm_snapshot")
	if err != nil {
		return "", err
	}
	result, ok := resp.Result.(string)
	if !ok {
		return "", ErrConversion
	}
	return result, nil
}

func (c *Client) Revert(snapshotHash string) error {
	resp, err := c.client.Call(context.Background(), "evm_revert", snapshotHash)
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return nil
}
