package client

import (
	"errors"
	"github.com/0chain/gosdk/zcncore"
)

func createReadPool() (err error) {
	var (
		txn    zcncore.TransactionScheme
		status = NewZCNStatus()
	)

	if txn, err = zcncore.NewTransaction(status, 0); err != nil {
		return
	}

	status.Begin()
	if err = txn.CreateReadPool(0); err != nil {
		return
	}
	status.Wait()

	if status.success {
		status.success = false

		status.Begin()
		if err = txn.Verify(); err != nil {
			return
		}
		status.Wait()

		if status.success {
			return // nil
		}
	}

	return errors.New(status.errMsg)
}
