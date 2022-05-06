package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/go-resty/resty/v2"
)

const FAUCET_SMART_CONTRACT_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
const STORAGE_SMART_CONTRACT_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"

func v1ClientPut(t *testing.T, walletRequest model.Wallet) (*model.Wallet, *resty.Response, error) {
	var wallet *model.Wallet

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/client/put", walletRequest, &wallet)

	return wallet, httpResponse, httpError
}

func v1TransactionPut(t *testing.T, walletRequest *model.Transaction) (*model.TransactionResponse, *resty.Response, error) {
	var transaction *model.TransactionResponse

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/transaction/put", walletRequest, &transaction)

	return transaction, httpResponse, httpError
}

func v1TransactionGetConfirmation(t *testing.T, hash string) (*model.Confirmation, *resty.Response, error) {
	var confirmation *model.Confirmation

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/transaction/get/confirmation?hash="+hash, &confirmation)

	return confirmation, httpResponse, httpError
}

func v1ClientGetBalance(t *testing.T, clientId string) (*model.Balance, *resty.Response, error) {
	var balance *model.Balance

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/client/get/balance?client_id="+clientId, &balance)

	return balance, httpResponse, httpError
}

func v1ScrestAllocation(t *testing.T, clientId string) (*model.Allocation, *resty.Response, error) {
	var allocation *model.Allocation

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/screst/"+STORAGE_SMART_CONTRACT_ADDRESS+"/allocation?allocation="+clientId, &allocation)

	return allocation, httpResponse, httpError
}
