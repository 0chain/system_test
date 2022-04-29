package api_tests

import (
	"github.com/0chain/system_test/internal/api/model"
	"github.com/go-resty/resty/v2"
	"testing"
)

const FAUCET_SMART_CONTRACT_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"

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
