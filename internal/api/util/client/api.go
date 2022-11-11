package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"

	resty "github.com/go-resty/resty/v2"
)

// Statuses of http based responses
const (
	HttpOkStatus         = 200
	HttpBadRequestStatus = 400
)

// Contains all methods used for http based requests
const (
	HttpPOSTMethod = iota + 1
	HttpGETMethod
	HttpPUTMethod
)

// Contains all used url paths in the client
const (
	GetGraphBlobberAllocated           = "/v1/screst/:sc_address/graph-blobber-allocated"
	GetGraphBlobberCapacity            = "/v1/screst/:sc_address/graph-blobber-capacity"
	GetGraphBlobberWritePrice          = "/v1/screst/:sc_address/graph-blobber-write-price"
	GetGraphBlobberServiceCharge       = "/v1/screst/:sc_address/graph-blobber-service-charge"
	GetGraphBlobberChallengesCompleted = "/v1/screst/:sc_address/graph-blobber-challenges-completed"
	GetGraphBlobberInactiveRounds      = "/v1/screst/:sc_address/graph-blobber-inactive-rounds"
	GetTotalTotalChallenges            = "/v1/screst/:sc_address/total-total-challenges"
	GetTotalMinted                     = "/v1/screst/:sc_address/total-minted"
	GetAverageWritePrice               = "/v1/screst/:sc_address/average-write-price"
	GetTotalBlobberCapacity            = "/v1/screst/:sc_address/total-blobber-capacity"
	GetTotalStaked                     = "/v1/screst/:sc_address/total-staked"
	GetTotalStoredData                 = "/v1/screst/:sc_address/total-stored-data"
	GetTotalAllocatedStorage           = "/v1/screst/:sc_address/total-allocation-storage"
	GetBlobbers                        = "/v1/screst/:sc_address/getblobbers"
	GetHashNodeRoot                    = "/v1/hashnode/root/:allocation"
	GetStakePoolStat                   = "/v1/screst/:sc_address/getStakePoolStat"
	GetAllocationBlobbers              = "/v1/screst/:sc_address/alloc_blobbers"
	SCRestGetOpenChallenges            = "/v1/screst/:sc_address/openchallenges"
	MinerGetStatus                     = "/v1/miner/get/stats"
	SharderGetStatus                   = "/v1/sharder/get/stats"
	SCStateGet                         = "/v1/scstate/get"
	SCRestGetAllocation                = "/v1/screst/:sc_address/allocation"
	SCRestGetBlobbers                  = "/v1/screst/:sc_address/getBlobber"
	ChainGetStats                      = "/v1/chain/get/stats"
	ClientPut                          = "/v1/client/put"
	TransactionPut                     = "/v1/transaction/put"
	TransactionGetConfirmation         = "/v1/transaction/get/confirmation"
	ClientGetBalance                   = "/v1/client/get/balance"
	BlobberGetStats                    = "/_stats"
	GetNetworkDetails                  = "/network"
	GetFileRef                         = "/v1/file/refs/:allocation_id"
	GetFileRefPath                     = "/v1/file/referencepath/:allocation_id"
	GetObjectTree                      = "/v1/file/objecttree/:allocation_id"
)

// Contains all used service providers
const (
	MinerServiceProvider = iota
	SharderServiceProvider
	BlobberServiceProvider
)

// Contains all smart contract addreses used in the client
const (
	FaucetSmartContractAddress  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	StorageSmartContractAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

// Contains statuses of transactions
const (
	TxSuccessfulStatus = iota + 1
	TxUnsuccessfulStatus
)

const (
	TxType    = 1000
	TxFee     = 0
	TxVersion = "1.0"
	TxOutput  = ""
)

var (
	TxValue = tokenomics.IntToZCN(1)
)

type APIClient struct {
	model.HealthyServiceProviders

	httpClient *resty.Client //nolint
}

func NewAPIClient(networkEntrypoint string) *APIClient {
	apiClient := &APIClient{
		httpClient: resty.New(), //nolint
	}

	if err := apiClient.selectHealthyServiceProviders(networkEntrypoint); err != nil {
		log.Fatalln(err)
	}

	return apiClient
}

func (c *APIClient) getHealthyNodes(nodes []string, serviceProviderType int) ([]string, error) {
	var result []string
	for _, node := range nodes {
		urlBuilder := NewURLBuilder()
		if err := urlBuilder.MustShiftParse(node); err != nil {
			return nil, err
		}

		r := c.httpClient.R()
		var formattedURL string
		switch serviceProviderType {
		case MinerServiceProvider:
			formattedURL = urlBuilder.SetPath(ChainGetStats).String()
		case SharderServiceProvider:
			formattedURL = urlBuilder.SetPath(ChainGetStats).String()
		case BlobberServiceProvider:
			formattedURL = urlBuilder.SetPath(BlobberGetStats).String()
			// /_stats requires username-password as it is an admin API.
			r.SetBasicAuth("admin", "password")
		}

		healthResponse, err := r.Get(formattedURL)
		if err == nil && healthResponse.IsSuccess() {
			log.Printf("%s is UP!", node)
			result = append(result, node)
			continue
		}

		status := healthResponse.StatusCode()
		response := healthResponse.Body()
		if err != nil {
			log.Printf("Read error %s for blobber %s.", err.Error(), node)
			continue
		}

		log.Printf("%s is DOWN! Status: %d, Message: %s", node, status, string(response))
	}
	return result, nil
}

func (c *APIClient) getHealthyMiners(miners []string) ([]string, error) {
	return c.getHealthyNodes(miners, MinerServiceProvider)
}

func (c *APIClient) getHealthyShaders(sharders []string) ([]string, error) {
	return c.getHealthyNodes(sharders, SharderServiceProvider)
}

func (c *APIClient) getHealthyBlobbers(blobbers []string) ([]string, error) {
	return c.getHealthyNodes(blobbers, BlobberServiceProvider)
}

func (c *APIClient) selectHealthyServiceProviders(networkEntrypoint string) error {
	urlBuilder := NewURLBuilder()
	if err := urlBuilder.MustShiftParse(networkEntrypoint); err != nil {
		return err
	}
	formattedURL := urlBuilder.SetPath(GetNetworkDetails).String()

	resp, err := c.httpClient.R().Get(formattedURL)
	if err != nil {
		return ErrNetworkHealthy
	}

	var networkServiceProviders *model.HealthyServiceProviders

	err = json.Unmarshal(resp.Body(), &networkServiceProviders)
	if err != nil {
		return ErrNetworkHealthy
	}

	healthyMiners, err := c.getHealthyMiners(networkServiceProviders.Miners)
	if err != nil {
		return err
	}
	if len(healthyMiners) == 0 {
		return ErrNoMinersHealthy
	}

	c.HealthyServiceProviders.Miners = healthyMiners

	healthySharders, err := c.getHealthyShaders(networkServiceProviders.Sharders)
	if err != nil {
		return err
	}
	if len(healthySharders) == 0 {
		return ErrNoShadersHealthy
	}

	c.HealthyServiceProviders.Sharders = healthySharders

	offset := 0
	limit := 20
	var nodes model.StorageNodes

	for {
		if err := urlBuilder.MustShiftParse(networkServiceProviders.Sharders[0]); err != nil {
			return err
		}
		urlBuilder = urlBuilder.SetPath(GetBlobbers).SetPathVariable("sc_address", StorageSmartContractAddress)
		formattedURL = urlBuilder.AddParams("offset", fmt.Sprint(offset)).AddParams("limit", fmt.Sprint(limit)).String()
		resp, err = c.httpClient.R().Get(formattedURL)
		if err != nil {
			return ErrNoBlobbersHealthy
		}
		err = json.Unmarshal(resp.Body(), &nodes)
		if err != nil {
			return ErrNetworkHealthy
		}

		if len(nodes.Nodes) == 0 {
			break
		}

		for _, node := range nodes.Nodes {
			networkServiceProviders.Blobbers = append(networkServiceProviders.Blobbers, node.BaseURL)
		}
		offset += limit
	}

	healthyBlobbers, err := c.getHealthyBlobbers(networkServiceProviders.Blobbers)
	if err != nil {
		return err
	}
	if len(healthyBlobbers) == 0 {
		return ErrNoBlobbersHealthy
	}

	c.HealthyServiceProviders.Blobbers = healthyBlobbers

	return nil
}

func (c *APIClient) executeForServiceProvider(url string, executionRequest model.ExecutionRequest, method int) (*resty.Response, error) { //nolint
	var (
		resp *resty.Response
		err  error
	)

	switch method {
	case HttpPUTMethod:
		resp, err = c.httpClient.R().SetHeaders(executionRequest.Headers).SetFormData(executionRequest.FormData).SetBody(executionRequest.Body).Put(url)
	case HttpPOSTMethod:
		resp, err = c.httpClient.R().SetHeaders(executionRequest.Headers).SetFormData(executionRequest.FormData).SetBody(executionRequest.Body).Post(url)
	case HttpGETMethod:
		resp, err = c.httpClient.R().SetHeaders(executionRequest.Headers).SetQueryParams(executionRequest.QueryParams).Get(url)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", url, ErrGetFromResource)
	}

	log.Printf("%s returned %s with status %s", url, resp.String(), resp.Status())
	if executionRequest.Dst != nil {
		src := resp.Body()
		if len(src) == 0 {
			return nil, ErrEmptyResponse
		}

		err = json.Unmarshal(src, executionRequest.Dst)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (c *APIClient) executeForAllServiceProviders(urlBuilder *URLBuilder, executionRequest model.ExecutionRequest, method, serviceProviderType int) (*resty.Response, error) {
	var (
		resp   *resty.Response
		errors []error
	)

	var expectedExecutionResponseCounter, notExpectedExecutionResponseCounter int

	var serviceProviders []string

	switch serviceProviderType {
	case MinerServiceProvider:
		serviceProviders = c.HealthyServiceProviders.Miners
	case SharderServiceProvider:
		serviceProviders = c.HealthyServiceProviders.Sharders
	case BlobberServiceProvider:
		serviceProviders = c.HealthyServiceProviders.Blobbers
	}

	for _, serviceProvider := range serviceProviders {
		if err := urlBuilder.MustShiftParse(serviceProvider); err != nil {
			return nil, err
		}
		formattedURL := urlBuilder.String()

		newResp, err := c.executeForServiceProvider(formattedURL, executionRequest, method)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if newResp.StatusCode() == executionRequest.RequiredStatusCode {
			expectedExecutionResponseCounter++
			resp = newResp
		} else {
			notExpectedExecutionResponseCounter++
		}
	}

	if notExpectedExecutionResponseCounter > expectedExecutionResponseCounter {
		return nil, ErrExecutionConsensus
	}

	return resp, selectMostFrequentError(errors)
}

func selectMostFrequentError(errors []error) error {
	frequencyCounters := make(map[error]int)
	var maxMatch int
	var result error

	for _, err := range errors {
		frequencyCounters[err]++
		if frequencyCounters[err] > maxMatch {
			maxMatch = frequencyCounters[err]
			result = err
		}
	}

	return result
}

func (c *APIClient) V1ClientPut(clientPutRequest model.Wallet, requiredStatusCode int) (*model.Wallet, *resty.Response, error) { //nolint
	var clientPutResponse *model.Wallet

	urlBuilder := NewURLBuilder().SetPath(ClientPut)
	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Body:               clientPutRequest,
			Dst:                &clientPutResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		MinerServiceProvider)

	if err != nil {
		return nil, resp, err
	}

	return clientPutResponse, resp, err
}

func (c *APIClient) V1TransactionPut(t *testing.T, internalTransactionPutRequest model.InternalTransactionPutRequest, requiredStatusCode int) (*model.TransactionPutResponse, *resty.Response, error) { //nolint
	var transactionPutResponse *model.TransactionPutResponse

	data, err := json.Marshal(internalTransactionPutRequest.TransactionData)
	if err != nil {
		return nil, nil, err
	}

	transactionPutRequest := model.TransactionPutRequest{
		ClientId:         internalTransactionPutRequest.Wallet.Id,
		PublicKey:        internalTransactionPutRequest.Wallet.PublicKey,
		ToClientId:       internalTransactionPutRequest.ToClientID,
		TransactionNonce: internalTransactionPutRequest.Wallet.Nonce + 1,
		TxnOutputHash:    TxOutput,
		TransactionValue: *TxValue,
		TransactionType:  TxType,
		TransactionFee:   TxFee,
		TransactionData:  string(data),
		CreationDate:     time.Now().Unix(),
		Version:          TxVersion,
	}

	if internalTransactionPutRequest.Value != nil {
		transactionPutRequest.TransactionValue = *internalTransactionPutRequest.Value
	}

	crypto.HashTransaction(&transactionPutRequest)
	crypto.SignTransaction(t, &transactionPutRequest, internalTransactionPutRequest.Wallet.Keys)

	urlBuilder := NewURLBuilder().SetPath(TransactionPut)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Body:               transactionPutRequest,
			Dst:                &transactionPutResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		MinerServiceProvider)

	transactionPutResponse.Request = transactionPutRequest

	return transactionPutResponse, resp, err
}

func (c *APIClient) V1TransactionGetConfirmation(transactionGetConfirmationRequest model.TransactionGetConfirmationRequest, requiredStatusCode int) (*model.TransactionGetConfirmationResponse, *resty.Response, error) { //nolint
	var transactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	urlBuilder := NewURLBuilder().
		SetPath(TransactionGetConfirmation).
		AddParams("hash", transactionGetConfirmationRequest.Hash)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &transactionGetConfirmationResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return transactionGetConfirmationResponse, resp, err
}

func (c *APIClient) V1ClientGetBalance(clientGetBalanceRequest model.ClientGetBalanceRequest, requiredStatusCode int) (*model.ClientGetBalanceResponse, *resty.Response, error) { //nolint
	var clientGetBalanceResponse *model.ClientGetBalanceResponse

	urlBuilder := NewURLBuilder().SetPath(ClientGetBalance).AddParams("client_id", clientGetBalanceRequest.ClientID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &clientGetBalanceResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return clientGetBalanceResponse, resp, err
}

func (c *APIClient) V1SCRestGetBlobber(scRestGetBlobberRequest model.SCRestGetBlobberRequest, requiredStatusCode int) (*model.SCRestGetBlobberResponse, *resty.Response, error) {
	var scRestGetBlobberResponse *model.SCRestGetBlobberResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCRestGetBlobbers).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("blobber_id", scRestGetBlobberRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &scRestGetBlobberResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetBlobberResponse, resp, err
}

func (c *APIClient) V1BlobberGetHashNodeRoot(blobberGetHashnodeRequest *model.BlobberGetHashnodeRequest, requiredStatusCode int) (*model.BlobberGetHashnodeResponse, *resty.Response, error) {
	var hashnode *model.BlobberGetHashnodeResponse

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetHashnodeRequest.ClientId,
		"X-App-Client-Key":       blobberGetHashnodeRequest.ClientKey,
		"X-App-Client-Signature": blobberGetHashnodeRequest.ClientSignature,
		"allocation":             blobberGetHashnodeRequest.AllocationID,
	}

	urlBuilder := NewURLBuilder()
	if err := urlBuilder.MustShiftParse(blobberGetHashnodeRequest.URL); err != nil {
		return nil, nil, err
	}
	formattedURL := urlBuilder.
		SetPath(GetHashNodeRoot).
		SetPathVariable("allocation", blobberGetHashnodeRequest.AllocationID).
		String()

	resp, err := c.executeForServiceProvider(formattedURL,
		model.ExecutionRequest{
			Headers:            headers,
			Dst:                &hashnode,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
	)
	return hashnode, resp, err
}

func (c *APIClient) V1SCRestGetAllocation(scRestGetAllocationRequest model.SCRestGetAllocationRequest, requiredStatusCode int) (*model.SCRestGetAllocationResponse, *resty.Response, error) { //nolint
	var scRestGetAllocationResponse *model.SCRestGetAllocationResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCRestGetAllocation).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("allocation", scRestGetAllocationRequest.AllocationID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &scRestGetAllocationResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetAllocationResponse, resp, err
}

func (c *APIClient) V1SCRestGetAllocationBlobbers(scRestGetAllocationBlobbersRequest *model.SCRestGetAllocationBlobbersRequest, requiredStatusCode int) (*model.SCRestGetAllocationBlobbersResponse, *resty.Response, error) { //nolint
	scRestGetAllocationBlobbersResponse := new(model.SCRestGetAllocationBlobbersResponse)

	data, err := json.Marshal(scRestGetAllocationBlobbersRequest.BlobberRequirements)
	if err != nil {
		return nil, nil, err
	}

	urlBuilder := NewURLBuilder().
		SetPath(GetAllocationBlobbers).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("allocation_data", string(data))

	var blobbers model.AllocationBlobbers

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &blobbers,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	scRestGetAllocationBlobbersResponse.Blobbers = model.ConvertInterfaceStringArray(blobbers)
	scRestGetAllocationBlobbersResponse.BlobberRequirements = scRestGetAllocationBlobbersRequest.BlobberRequirements

	return scRestGetAllocationBlobbersResponse, resp, err
}

func (c *APIClient) V1SCRestOpenChallenge(scRestOpenChallengeRequest model.SCRestOpenChallengeRequest, requiredStatusCode int) (*model.SCRestOpenChallengeResponse, *resty.Response, error) { //nolint
	var scRestOpenChallengeResponse *model.SCRestOpenChallengeResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCRestGetOpenChallenges).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("blobber", scRestOpenChallengeRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &scRestOpenChallengeResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestOpenChallengeResponse, resp, err
}

func (c *APIClient) V1MinerGetStats(requiredStatusCode int) (*model.GetMinerStatsResponse, *resty.Response, error) { //nolint
	var getMinerStatsResponse *model.GetMinerStatsResponse

	urlBuilder := NewURLBuilder().
		SetPath(MinerGetStatus)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getMinerStatsResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		MinerServiceProvider)

	return getMinerStatsResponse, resp, err
}

func (c *APIClient) V1SharderGetStats(requiredStatusCode int) (*model.GetSharderStatsResponse, *resty.Response, error) { //nolint
	var getSharderStatusResponse *model.GetSharderStatsResponse

	urlBuilder := NewURLBuilder().
		SetPath(SharderGetStatus)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getSharderStatusResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getSharderStatusResponse, resp, err
}

func (c *APIClient) V1SharderGetSCState(scStateGetRequest model.SCStateGetRequest, requiredStatusCode int) (*model.SCStateGetResponse, *resty.Response, error) { //nolint
	var scStateGetResponse *model.SCStateGetResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCStateGet)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			FormData: map[string]string{
				"sc_address": scStateGetRequest.SCAddress,
				"key":        scStateGetRequest.Key,
			},
			Dst:                &scStateGetResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		SharderServiceProvider)

	return scStateGetResponse, resp, err
}

func (c *APIClient) RegisterWallet(t *testing.T) *model.Wallet {
	mnemonic := crypto.GenerateMnemonics(t)

	return c.RegisterWalletForMnemonic(t, mnemonic)
}

func (c *APIClient) RegisterWalletForMnemonic(t *testing.T, mnemonic string) *model.Wallet {
	registeredWallet, httpResponse, err := c.RegisterWalletForMnemonicWithoutAssertion(t, mnemonic, HttpOkStatus)

	publicKeyBytes, _ := hex.DecodeString(registeredWallet.Keys.PublicKey.SerializeToHexStr())
	clientId := crypto.Sha3256(publicKeyBytes)

	require.Nil(t, err, "Unexpected error [%s] occurred registering wallet with http response [%s]", err, httpResponse)
	require.NotNil(t, registeredWallet, "Registered wallet was unexpectedly nil! with http response [%s]", httpResponse)
	require.Equal(t, "200 OK", httpResponse.Status())
	require.Equal(t, registeredWallet.Id, clientId)
	require.Equal(t, registeredWallet.PublicKey, registeredWallet.Keys.PublicKey.SerializeToHexStr())
	require.NotNil(t, registeredWallet.CreationDate, "Creation date is nil!")
	require.NotNil(t, registeredWallet.Version)

	return registeredWallet
}

func (c *APIClient) RegisterWalletForMnemonicWithoutAssertion(t *testing.T, mnemonic string, expectedHttpStatus int) (*model.Wallet, *resty.Response, error) {
	keyPair := crypto.GenerateKeys(t, mnemonic)
	publicKeyBytes, _ := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	clientId := crypto.Sha3256(publicKeyBytes)
	walletRequest := model.Wallet{Id: clientId, PublicKey: keyPair.PublicKey.SerializeToHexStr()}

	registeredWallet, httpResponse, err := c.V1ClientPut(walletRequest, expectedHttpStatus)
	registeredWallet.Keys = keyPair

	return registeredWallet, httpResponse, err
}

// ExecuteFaucet provides basic assertions
func (c *APIClient) ExecuteFaucet(t *testing.T, wallet *model.Wallet, requiredTransactionStatus int) {
	t.Log("Execute faucet...")

	faucetTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      FaucetSmartContractAddress,
			TransactionData: model.NewFaucetTransactionData()},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, faucetTransactionPutResponse)

	var faucetTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		faucetTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: faucetTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if faucetTransactionGetConfirmationResponse == nil {
			return false
		}

		return faucetTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
}

// ExecuteFaucetWithAssertions provides deep assertions
func (c *APIClient) ExecuteFaucetWithAssertions(t *testing.T, wallet *model.Wallet, requiredTransactionStatus int) {
	t.Log("Execute faucet with assertions...")

	faucetTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      FaucetSmartContractAddress,
			TransactionData: model.NewFaucetTransactionData()},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, faucetTransactionPutResponse)

	var faucetTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		faucetTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: faucetTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if faucetTransactionGetConfirmationResponse == nil {
			return false
		}

		return faucetTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	require.True(t, faucetTransactionPutResponse.Async)
	require.NotNil(t, faucetTransactionPutResponse.Entity)
	require.NotNil(t, faucetTransactionPutResponse.Entity.ChainId)
	require.Zero(t, faucetTransactionPutResponse.Entity.TransactionOutput)
	require.Zero(t, faucetTransactionPutResponse.Entity.TransactionStatus)

	require.Equal(t, faucetTransactionPutResponse.Request.Hash, faucetTransactionPutResponse.Entity.Hash)
	require.Equal(t, faucetTransactionPutResponse.Request.Version, faucetTransactionPutResponse.Entity.Version)
	require.Equal(t, faucetTransactionPutResponse.Request.ClientId, faucetTransactionPutResponse.Entity.ClientId)
	require.Equal(t, faucetTransactionPutResponse.Request.ToClientId, faucetTransactionPutResponse.Entity.ToClientId)
	require.Equal(t, faucetTransactionPutResponse.Request.PublicKey, faucetTransactionPutResponse.Entity.PublicKey)
	require.Equal(t, faucetTransactionPutResponse.Request.TransactionData, faucetTransactionPutResponse.Entity.TransactionData)
	require.Equal(t, faucetTransactionPutResponse.Request.TransactionValue, faucetTransactionPutResponse.Entity.TransactionValue)
	require.Equal(t, faucetTransactionPutResponse.Request.Signature, faucetTransactionPutResponse.Entity.Signature)
	require.Equal(t, faucetTransactionPutResponse.Request.CreationDate, faucetTransactionPutResponse.Entity.CreationDate)
	require.Equal(t, faucetTransactionPutResponse.Request.TransactionFee, faucetTransactionPutResponse.Entity.TransactionFee)
	require.Equal(t, faucetTransactionPutResponse.Request.TransactionType, faucetTransactionPutResponse.Entity.TransactionType)

	require.Equal(t, TxVersion, faucetTransactionGetConfirmationResponse.Version)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.BlockHash)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.PreviousBlockHash)
	require.Greater(t, faucetTransactionGetConfirmationResponse.CreationDate, int64(0))
	require.NotNil(t, faucetTransactionGetConfirmationResponse.MinerID)
	require.Greater(t, faucetTransactionGetConfirmationResponse.Round, int64(0))
	require.NotNil(t, faucetTransactionGetConfirmationResponse.Status)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.RoundRandomSeed)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.StateChangesCount)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.MerkleTreeRoot)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.MerkleTreePath)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.ReceiptMerkleTreeRoot)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.ReceiptMerkleTreePath)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.Transaction.TransactionOutput)
	require.NotNil(t, faucetTransactionGetConfirmationResponse.Transaction.TxnOutputHash)

	wallet.IncNonce()
}

func (c *APIClient) CreateAllocation(t *testing.T,
	wallet *model.Wallet,
	scRestGetAllocationBlobbersResponse *model.SCRestGetAllocationBlobbersResponse,
	requiredTransactionStatus int) string {
	t.Log("Create allocation...")

	createAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse),
			Value:           tokenomics.IntToZCN(0.1),
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createAllocationTransactionPutResponse)

	var createAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		createAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)

		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if createAllocationTransactionGetConfirmationResponse == nil {
			return false
		}

		return createAllocationTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return createAllocationTransactionPutResponse.Entity.Hash
}

func (c *APIClient) UpdateAllocationBlobbers(t *testing.T, wallet *model.Wallet, newBlobberID, oldBlobberID, allocationID string, requiredTransactionStatus int) {
	t.Log("Update allocation...")

	updateAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: StorageSmartContractAddress,
			TransactionData: model.NewUpdateAllocationTransactionData(&model.UpdateAllocationRequest{
				ID:              allocationID,
				AddBlobberId:    newBlobberID,
				RemoveBlobberId: oldBlobberID,
			}),
			Value: tokenomics.IntToZCN(0.1),
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updateAllocationTransactionPutResponse)
	txnHash := updateAllocationTransactionPutResponse.Request.Hash

	var updateAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		updateAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: txnHash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if updateAllocationTransactionGetConfirmationResponse == nil {
			return false
		}

		return updateAllocationTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
}

func (c *APIClient) GetAllocationBlobbers(t *testing.T, wallet *model.Wallet, blobberRequirements *model.BlobberRequirements, requiredStatusCode int) *model.SCRestGetAllocationBlobbersResponse {
	t.Log("Get allocation blobbers...")

	scRestGetAllocationBlobbersResponse, resp, err := c.V1SCRestGetAllocationBlobbers(
		&model.SCRestGetAllocationBlobbersRequest{
			ClientID:            wallet.Id,
			ClientKey:           wallet.PublicKey,
			BlobberRequirements: blobberRequirements,
		}, requiredStatusCode)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, scRestGetAllocationBlobbersResponse)

	return scRestGetAllocationBlobbersResponse
}

func (c *APIClient) GetAllocation(t *testing.T, allocationID string, requiredStatusCode int) *model.SCRestGetAllocationResponse {
	t.Log("Get allocation...")

	var (
		scRestGetAllocation *model.SCRestGetAllocationResponse
		resp                *resty.Response //nolint
		err                 error
	)

	wait.PoolImmediately(t, time.Second*30, func() bool {
		scRestGetAllocation, resp, err = c.V1SCRestGetAllocation(
			model.SCRestGetAllocationRequest{
				AllocationID: allocationID,
			},
			requiredStatusCode)
		return err == nil
	})

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, scRestGetAllocation)

	return scRestGetAllocation
}

func (c *APIClient) GetWalletBalance(t *testing.T, wallet *model.Wallet, requiredStatusCode int) *model.ClientGetBalanceResponse {
	t.Log("Get wallet balance...")

	clientGetBalanceResponse, resp, err := c.V1ClientGetBalance(
		model.ClientGetBalanceRequest{
			ClientID: wallet.Id,
		},
		requiredStatusCode)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, clientGetBalanceResponse)

	return clientGetBalanceResponse
}

func (c *APIClient) UpdateBlobber(t *testing.T, wallet *model.Wallet, scRestGetBlobberResponse *model.SCRestGetBlobberResponse, requiredTransactionStatus int) {
	updateBlobberTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewUpdateBlobberTransactionData(scRestGetBlobberResponse),
			Value:           tokenomics.IntToZCN(0.1),
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updateBlobberTransactionPutResponse)

	var updateBlobberTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		updateBlobberTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: updateBlobberTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if updateBlobberTransactionGetConfirmationResponse == nil {
			return false
		}

		return updateBlobberTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
}

// CreateStakePoolWrapper does not provide deep test of used components
func (c *APIClient) CreateStakePool(t *testing.T, wallet *model.Wallet, providerType int, providerID string, requiredTransactionStatus int) string {
	t.Log("Create stake pool...")

	createStakePoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: StorageSmartContractAddress,
			TransactionData: model.NewCreateStackPoolTransactionData(
				model.CreateStakePoolRequest{
					ProviderType: providerType,
					ProviderID:   providerID,
				}),
			Value: tokenomics.IntToZCN(0.5)},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createStakePoolTransactionPutResponse)

	var createStakePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		createStakePoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: createStakePoolTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if createStakePoolTransactionGetConfirmationResponse == nil {
			return false
		}

		return createStakePoolTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return createStakePoolTransactionGetConfirmationResponse.Hash
}

func (c *APIClient) V1SCRestGetStakePoolStat(scRestGetStakePoolStatRequest model.SCRestGetStakePoolStatRequest, requiredStatusCode int) (*model.SCRestGetStakePoolStatResponse, *resty.Response, error) { //nolint
	var scRestGetStakePoolStatResponse *model.SCRestGetStakePoolStatResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetStakePoolStat).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("provider_id", scRestGetStakePoolStatRequest.ProviderID).
		AddParams("provider_type", scRestGetStakePoolStatRequest.ProviderType)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &scRestGetStakePoolStatResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetStakePoolStatResponse, resp, err
}

func (c *APIClient) GetStakePoolStat(t *testing.T, providerID, providerType string) *model.SCRestGetStakePoolStatResponse {
	t.Log("Get stake pool stat...")

	scRestGetStakePoolStat, resp, err := c.V1SCRestGetStakePoolStat(
		model.SCRestGetStakePoolStatRequest{
			ProviderID:   providerID,
			ProviderType: providerType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)

	return scRestGetStakePoolStat
}

func (c *APIClient) CollectRewards(t *testing.T, wallet *model.Wallet, providerID string, providerType, requiredTransactionStatus int) {
	collectRewardTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewCollectRewardTransactionData(providerID, providerType),
			Value:           tokenomics.IntToZCN(0),
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, collectRewardTransactionPutResponse)

	var collectRewardTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		collectRewardTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: collectRewardTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if collectRewardTransactionGetConfirmationResponse == nil {
			return false
		}

		return collectRewardTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
}

func (c *APIClient) GetBlobber(t *testing.T, blobberID string, requiredStatusCode int) *model.SCRestGetBlobberResponse {
	scRestGetBlobberResponse, resp, err := c.V1SCRestGetBlobber(
		model.SCRestGetBlobberRequest{
			BlobberID: blobberID,
		},
		requiredStatusCode)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, scRestGetBlobberResponse)

	return scRestGetBlobberResponse
}

func (c *APIClient) V1SharderGetTotalStoredData(requiredStatusCode int) (*model.GetTotalStoredDataResponse, *resty.Response, error) { //nolint
	var getTotalStoredDataResponse *model.GetTotalStoredDataResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetTotalStoredData).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getTotalStoredDataResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getTotalStoredDataResponse, resp, err
}

func (c *APIClient) V1SharderGetTotalStaked(requiredStatusCode int) (*model.GetTotalStakedResponse, *resty.Response, error) { //nolint
	var getTotalStakedResponse *model.GetTotalStakedResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetTotalStaked).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getTotalStakedResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getTotalStakedResponse, resp, err
}

func (c *APIClient) V1SharderGetAverageWritePrice(requiredStatusCode int) (*model.GetAverageWritePriceResponse, *resty.Response, error) { //nolint
	var getAverageWritePriceResponse *model.GetAverageWritePriceResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetAverageWritePrice).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getAverageWritePriceResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getAverageWritePriceResponse, resp, err
}

func (c *APIClient) V1SharderGetTotalBlobberCapacity(requiredStatusCode int) (*model.GetTotalBlobberCapacityResponse, *resty.Response, error) { //nolint
	var getTotalBlobberCapacityResponse *model.GetTotalBlobberCapacityResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetTotalBlobberCapacity).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getTotalBlobberCapacityResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getTotalBlobberCapacityResponse, resp, err
}

func (c *APIClient) V1SharderGetTotalMinted(requiredStatusCode int) (*model.GetTotalMintedResponse, *resty.Response, error) { //nolint
	var getTotalMintedResponse *model.GetTotalMintedResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetTotalMinted).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getTotalMintedResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getTotalMintedResponse, resp, err
}

func (c *APIClient) V1SharderGetTotalAllocatedStorage(requiredStatusCode int) (*model.GetTotalAllocatedStorage, *resty.Response, error) { //nolint
	var getTotalAllocatedStorage *model.GetTotalAllocatedStorage

	urlBuilder := NewURLBuilder().
		SetPath(GetTotalAllocatedStorage).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getTotalAllocatedStorage,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getTotalAllocatedStorage, resp, err
}

func (c *APIClient) V1SharderGetTotalTotalChallenges(requiredStatusCode int) (*model.GetTotalTotalChallengesResponse, *resty.Response, error) { //nolint
	var getTotalTotalChallengesResponse *model.GetTotalTotalChallengesResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetTotalTotalChallenges).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getTotalTotalChallengesResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getTotalTotalChallengesResponse, resp, err
}

func (c *APIClient) V1SharderGetTotalSuccessfulChallenges(requiredStatusCode int) (*model.GetTotalSuccessfulChallengesResponse, *resty.Response, error) { //nolint
	var getTotalSuccessfulChallengesResponse *model.GetTotalSuccessfulChallengesResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetTotalTotalChallenges).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getTotalSuccessfulChallengesResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getTotalSuccessfulChallengesResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberInactiveRounds(getGraphBlobberInactiveRoundsRequest model.GetGraphBlobberInactiveRoundsRequest, requiredStatusCode int) (*model.GetGraphBlobberInactiveRoundsResponse, *resty.Response, error) { //nolint
	var getGraphBlobberInactiveRoundsResponse *model.GetGraphBlobberInactiveRoundsResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberInactiveRounds).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberInactiveRoundsRequest.DataPoints)).
		AddParams("id", getGraphBlobberInactiveRoundsRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberInactiveRoundsResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberInactiveRoundsResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberChallengesCompleted(getGraphBlobberChallengesPassedRequest model.GetGraphBlobberChallengesCompletedRequest, requiredStatusCode int) (*model.GetGraphBlobberChallengesCompletedResponse, *resty.Response, error) { //nolint
	var getGraphBlobberChallengesCompletedResponse *model.GetGraphBlobberChallengesCompletedResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberChallengesCompleted).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberChallengesPassedRequest.DataPoints)).
		AddParams("id", getGraphBlobberChallengesPassedRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberChallengesCompletedResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberChallengesCompletedResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberChallengesPassed(getGraphBlobberChallengesPassedRequest model.GetGraphBlobberChallengesPassedRequest, requiredStatusCode int) (*model.GetGraphBlobberChallengesPassedResponse, *resty.Response, error) { //nolint
	var getGraphBlobberChallengesPassedResponse *model.GetGraphBlobberChallengesPassedResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberChallengesCompleted).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberChallengesPassedRequest.DataPoints)).
		AddParams("id", getGraphBlobberChallengesPassedRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberChallengesPassedResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberChallengesPassedResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberServiceCharge(getGraphBlobberServiceChargeRequest model.GetGraphBlobberServiceChargeRequest, requiredStatusCode int) (*model.GetGraphBlobberServiceChargeResponse, *resty.Response, error) { //nolint
	var getGraphBlobberServiceChargeResponse *model.GetGraphBlobberServiceChargeResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberServiceCharge).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberServiceChargeRequest.DataPoints)).
		AddParams("id", getGraphBlobberServiceChargeRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberServiceChargeResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberServiceChargeResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberWritePrice(getGraphBlobberWritePriceRequest model.GetGraphBlobberWritePriceRequest, requiredStatusCode int) (*model.GetGraphBlobberWritePriceResponse, *resty.Response, error) {
	var getGraphBlobberWritePriceResponse *model.GetGraphBlobberWritePriceResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberWritePrice).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberWritePriceRequest.DataPoints)).
		AddParams("id", getGraphBlobberWritePriceRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberWritePriceResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberWritePriceResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberCapacity(getGraphBlobberCapacityRequest model.GetGraphBlobberCapacityRequest, requiredStatusCode int) (*model.GetGraphBlobberCapacityResponse, *resty.Response, error) {
	var getGraphBlobberCapacityResponse *model.GetGraphBlobberCapacityResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberCapacity).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberCapacityRequest.DataPoints)).
		AddParams("id", getGraphBlobberCapacityRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberCapacityResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberCapacityResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberAllocated(getGraphBlobberAllocatedRequest model.GetGraphBlobberAllocatedRequest, requiredStatusCode int) (*model.GetGraphBlobberAllocatedResponse, *resty.Response, error) {
	var getGraphBlobberAllocatedResponse *model.GetGraphBlobberAllocatedResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberAllocated).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberAllocatedRequest.DataPoints)).
		AddParams("id", getGraphBlobberAllocatedRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberAllocatedResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberAllocatedResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberSavedData(getGraphBlobberSavedDataRequest model.GetGraphBlobberSavedDataRequest, requiredStatusCode int) (*model.GetGraphBlobberSavedDataResponse, *resty.Response, error) {
	var getGraphBlobberSavedDataResponse *model.GetGraphBlobberSavedDataResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberAllocated).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberSavedDataRequest.DataPoints)).
		AddParams("id", getGraphBlobberSavedDataRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberSavedDataResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberSavedDataResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberReadData(getGraphBlobberReadDataRequest model.GetGraphBlobberReadDataRequest, requiredStatusCode int) (*model.GetGraphBlobberReadDataResponse, *resty.Response, error) {
	var getGraphBlobberReadDataResponse *model.GetGraphBlobberReadDataResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberAllocated).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberReadDataRequest.DataPoints)).
		AddParams("id", getGraphBlobberReadDataRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberReadDataResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberReadDataResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberOffersTotal(getGraphBlobberOffersTotalRequest model.GetGraphBlobberOffersTotalRequest, requiredStatusCode int) (*model.GetGraphBlobberOffersTotalResponse, *resty.Response, error) {
	var getGraphBlobberOffersTotalResponse *model.GetGraphBlobberOffersTotalResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberAllocated).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberOffersTotalRequest.DataPoints)).
		AddParams("id", getGraphBlobberOffersTotalRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberOffersTotalResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberOffersTotalResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberUnstakeTotal(getGraphBlobberUnstakeTotalRequest model.GetGraphBlobberUnstakeTotalRequest, requiredStatusCode int) (*model.GetGraphBlobberUnstakeTotalResponse, *resty.Response, error) {
	var getGraphBlobberUnstakeTotalResponse *model.GetGraphBlobberUnstakeTotalResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberAllocated).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberUnstakeTotalRequest.DataPoints)).
		AddParams("id", getGraphBlobberUnstakeTotalRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberUnstakeTotalResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberUnstakeTotalResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberTotalStake(getGraphBlobberTotalStakeRequest model.GetGraphBlobberTotalStakeRequest, requiredStatusCode int) (*model.GetGraphBlobberTotalStakeResponse, *resty.Response, error) {
	var getGraphBlobberTotalStakeResponse *model.GetGraphBlobberTotalStakeResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberAllocated).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberTotalStakeRequest.DataPoints)).
		AddParams("id", getGraphBlobberTotalStakeRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberTotalStakeResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberTotalStakeResponse, resp, err
}

func (c *APIClient) V1SharderGetGraphBlobberChallangesOpen(getGraphBlobberChallangesOpenRequest model.GetGraphBlobberChallengesOpenRequest, requiredStatusCode int) (*model.GetGraphBlobberChallengesOpenResponse, *resty.Response, error) {
	var getGraphBlobberChallangesOpenResponse *model.GetGraphBlobberChallengesOpenResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetGraphBlobberAllocated).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("data-points", strconv.Itoa(getGraphBlobberChallangesOpenRequest.DataPoints)).
		AddParams("id", getGraphBlobberChallangesOpenRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &getGraphBlobberChallangesOpenResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getGraphBlobberChallangesOpenResponse, resp, err
}

func (c *APIClient) V1BlobberGetFileRefs(t *testing.T, blobberGetFileRefsRequest *model.BlobberGetFileRefsRequest, requiredStatusCode int) (*model.BlobberGetFileRefsResponse, *resty.Response, error) {
	var blobberGetFileResponse *model.BlobberGetFileRefsResponse

	url := blobberGetFileRefsRequest.URL + strings.Replace(GetFileRef, ":allocation_id", blobberGetFileRefsRequest.AllocationID, 1) + "?" + "path=" + blobberGetFileRefsRequest.RemotePath + "&" + "refType=" + blobberGetFileRefsRequest.RefType

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetFileRefsRequest.ClientID,
		"X-App-Client-Key":       blobberGetFileRefsRequest.ClientKey,
		"X-App-Client-Signature": blobberGetFileRefsRequest.ClientSignature,
	}
	resp, err := c.executeForServiceProvider(
		url,
		model.ExecutionRequest{
			Dst:                &blobberGetFileResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberGetFileResponse, resp, err
}

func (c *APIClient) V1BlobberGetFileRefPaths(t *testing.T, blobberFileRefPathRequest *model.BlobberFileRefPathRequest, requiredStatusCode int) (*model.BlobberFileRefPathResponse, *resty.Response, error) {
	var blobberFileRefPathResponse *model.BlobberFileRefPathResponse

	url := blobberFileRefPathRequest.URL + strings.Replace(GetFileRefPath, ":allocation_id", blobberFileRefPathRequest.AllocationID, 1) + "?" + "path=" + blobberFileRefPathRequest.Path

	headers := map[string]string{
		"X-App-Client-Id":        blobberFileRefPathRequest.ClientID,
		"X-App-Client-Key":       blobberFileRefPathRequest.ClientKey,
		"X-App-Client-Signature": blobberFileRefPathRequest.ClientSignature,
	}
	resp, err := c.executeForServiceProvider(
		url,
		model.ExecutionRequest{
			Dst:                &blobberFileRefPathResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberFileRefPathResponse, resp, err
}

func (c *APIClient) V1BlobberObjectTree(t *testing.T, blobberObjectTreeRequest *model.BlobberObjectTreeRequest, requiredStatusCode int) (*model.BlobberObjectTreePathResponse, *resty.Response, error) {
	var blobberObjectTreePathResponse *model.BlobberObjectTreePathResponse

	url := blobberObjectTreeRequest.URL + strings.Replace(GetObjectTree, ":allocation_id", blobberObjectTreeRequest.AllocationID, 1) + "?" + "path=" + blobberObjectTreeRequest.Path

	headers := map[string]string{
		"X-App-Client-Id":        blobberObjectTreeRequest.ClientID,
		"X-App-Client-Key":       blobberObjectTreeRequest.ClientKey,
		"X-App-Client-Signature": blobberObjectTreeRequest.ClientSignature,
	}
	resp, err := c.executeForServiceProvider(
		url,
		model.ExecutionRequest{
			Dst:                &blobberObjectTreePathResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberObjectTreePathResponse, resp, err
}
