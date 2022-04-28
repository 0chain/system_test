package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/go-resty/resty/v2"
	"testing"
)

func v1ClientPut(t *testing.T, walletRequest model.Wallet) (*model.Wallet, *resty.Response, error) {
	var walletResponse *model.Wallet

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/client/put", walletRequest, &walletResponse)
	return walletResponse, httpResponse, httpError
}
func v1TransactionPut(t *testing.T, walletRequest model.Transaction) (*model.Transaction, *resty.Response, error) {
	var transactionResponse *model.Transaction

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/transaction/put", walletRequest, &transactionResponse)
	return transactionResponse, httpResponse, httpError
}
