// nolint:typecheck
package tenderly

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

// CreateSnapshot creates network snapshot with a help of Ethereum JSON-RPC method call.
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

// ShadowRevert guarantees to revert a state of Ethereum network using snapshot hash with a help of Ethereum JSON-RPC method call.
func (c *Client) ShadowRevert(snapshotHash string) {
	notificationStream := make(chan os.Signal, 1)
	load := time.NewTicker(time.Millisecond * 500)

	signal.Notify(notificationStream, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-notificationStream:
				if err := c.revert(snapshotHash); err != nil {
					log.Fatalln(err)
				}
			case <-load.C:
			}
		}
	}()
}

// revert reverts a state of Ethereum network using snapshot hash with a help of Ethereum JSON-RPC method call.
func (c *Client) revert(snapshotHash string) error {
	resp, err := c.client.Call(context.Background(), "evm_revert", snapshotHash)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return errors.New(resp.Error.Error())
	}
	return nil
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
