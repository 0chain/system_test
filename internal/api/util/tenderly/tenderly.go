package tenderly

import (
	"context"
	"fmt"
<<<<<<< HEAD

=======
>>>>>>> 6c6282de (feature: added basic tenderly client)
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
<<<<<<< HEAD

//curl "https://api.tenderly.co/api/v1/account/YarikRevich/project/jgkdfjgkdf/fork/ca3894fd-9ecc-4f86-a10d-af7b91659a17" -H "X-Access-Key: Sc8gOOcwiVV6EoDZPFpK11V20bz-8UKJ" -H "Content-Type: application/json"
=======
>>>>>>> 6c6282de (feature: added basic tenderly client)
