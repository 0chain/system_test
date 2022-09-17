package api_tests

import (
	"github.com/0chain/system_test/internal/api/util"
	resty "github.com/go-resty/resty/v2"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
)

const FAUCET_SMART_CONTRACT_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
const STORAGE_SMART_CONTRACT_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"

func v1ClientPut(t *testing.T, walletRequest model.Wallet, consensusCategoriser util.ConsensusMetFunction) (*model.Wallet, *resty.Response, error) { //nolint
	var wallet *model.Wallet

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/client/put", consensusCategoriser, walletRequest, &wallet)

	return wallet, httpResponse, httpError
}

func v1TransactionPut(t *testing.T, walletRequest *model.Transaction, consensusCategoriser util.ConsensusMetFunction) (*model.TransactionResponse, *resty.Response, error) { //nolint
	var transaction *model.TransactionResponse

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/transaction/put", consensusCategoriser, walletRequest, &transaction)

	return transaction, httpResponse, httpError
}

func v1TransactionGetConfirmation(t *testing.T, hash string, consensusCategoriser util.ConsensusMetFunction) (*model.Confirmation, *resty.Response, error) { //nolint
	var confirmation *model.Confirmation

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/transaction/get/confirmation?hash="+hash, consensusCategoriser, &confirmation)

	return confirmation, httpResponse, httpError
}

func v1ClientGetBalance(t *testing.T, clientId string, consensusCategoriser util.ConsensusMetFunction) (*model.Balance, *resty.Response, error) { //nolint
	var balance *model.Balance

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/client/get/balance?client_id="+clientId, consensusCategoriser, &balance)

	return balance, httpResponse, httpError
}

func v1ScrestAllocation(t *testing.T, clientId string, consensusCategoriser util.ConsensusMetFunction) (*model.Allocation, *resty.Response, error) { //nolint
	var allocation *model.Allocation

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/screst/"+STORAGE_SMART_CONTRACT_ADDRESS+"/allocation?allocation="+clientId, consensusCategoriser, &allocation)

	return allocation, httpResponse, httpError
}

func v1ScrestAllocBlobbers(t *testing.T, allocationData string, consensusCategoriser util.ConsensusMetFunction) (*[]string, *resty.Response, error) { //nolint
	var blobbers *[]string

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/screst/"+STORAGE_SMART_CONTRACT_ADDRESS+"/alloc_blobbers?allocation_data="+allocationData, consensusCategoriser, &blobbers)

	return blobbers, httpResponse, httpError
}

func v1ScrestOpenChallenges(t *testing.T, storageSmartContractAddress string, blobberId string, consensusCategoriser util.ConsensusMetFunction) (*resty.Response, error) { //nolint
	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/screst/"+storageSmartContractAddress+"/openchallenges?blobber="+blobberId, consensusCategoriser, nil)
	return httpResponse, httpError
}

func v1MinerGetStats(t *testing.T, consensusCategoriser util.ConsensusMetFunction) (*model.MinerStats, *resty.Response, error) { //nolint
	var stats *model.MinerStats

	httpResponse, httpError := zeroChain.GetFromMiners(t, "/v1/miner/get/stats", consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}

func v1SharderGetStats(t *testing.T, consensusCategoriser util.ConsensusMetFunction) (*model.SharderStats, *resty.Response, error) { //nolint
	var stats *model.SharderStats

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/sharder/get/stats", consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}

func v1SharderGetSCState(t *testing.T, sharderSCStateRequest model.SharderSCStateRequest, consensusCategoriser util.ConsensusMetFunction) (*model.SharderSCStateResponse, *resty.Response, error) { //nolint
	var stats *model.SharderSCStateResponse

	formData := map[string]string{
		"sc_address": sharderSCStateRequest.SCAddress,
		"key":        sharderSCStateRequest.Key,
	}

	httpResponse, httpError := zeroChain.PostToShardersWithFormData(t, "/v1/scstate/get", consensusCategoriser, formData, nil, &stats)

	return stats, httpResponse, httpError
}
