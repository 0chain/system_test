package zsdk

import (
	"context"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/gosdk/zmagmacore/errors"
	"github.com/0chain/gosdk/zmagmacore/transaction"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
)

type (
	// getBalanceCB implements zcncore.GetBalanceCallback.
	getBalanceCB struct {
		balance chan int64
		err     error
	}
)

func newGetBalanceCB() *getBalanceCB {
	return &getBalanceCB{
		balance: make(chan int64),
	}
}

// OnBalanceAvailable implements zcncore.GetBalanceCallback.
func (b *getBalanceCB) OnBalanceAvailable(status int, value int64, _ string) {
	if status != zcncore.StatusSuccess {
		b.err = errors.New("get_balance", "responding balance failed")
		b.balance <- value
		return
	}
	b.balance <- value
}

// GetConsumerBalance returns balance of the configured consumer.
func GetConsumerBalance(cfg *config.Config) (int64, error) {
	err := Init(cfg.Consumer.KeysFile, cfg.Consumer.NodeDir, cfg.Consumer.ExtID, cfg)
	if err != nil {
		return 0, err
	}
	balCB := newGetBalanceCB()
	err = zcncore.GetBalance(balCB)
	if err != nil {
		return 0, err
	}
	if balCB.err != nil {
		return 0, err
	}

	return <-balCB.balance, nil
}

// GetProviderBalance returns balance of the configured provider.
func GetProviderBalance(cfg *config.Config) (int64, error) {
	err := Init(cfg.Provider.KeysFile, cfg.Provider.NodeDir, cfg.Provider.ExtID, cfg)
	if err != nil {
		return 0, err
	}
	balCB := newGetBalanceCB()
	err = zcncore.GetBalance(balCB)
	if err != nil {
		return 0, err
	}
	if balCB.err != nil {
		return 0, err
	}

	return <-balCB.balance, nil
}

const (
	faucetScAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	pourFuncName    = "pour"
)

// Pour executes pours function of fauces sc.
//
// NOTE: before using Pour you need to configure wallet by Init.
func Pour(val int64) error {
	txn, err := transaction.NewTransactionEntity()
	if err != nil {
		return err
	}

	txnHash, err := txn.ExecuteSmartContract(context.Background(), faucetScAddress, pourFuncName, "", val)
	if err != nil {
		return err
	}

	_, err = transaction.VerifyTransaction(context.Background(), txnHash)
	if err != nil {
		return err
	}

	return nil
}
