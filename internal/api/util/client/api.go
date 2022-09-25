package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
	"log"
	"strconv"
	"testing"
	"time"

	resty "github.com/go-resty/resty/v2"
)

// Statuses of http based responses
const (
	HttpOkStatus       = 200
	HttpNotFoundStatus = 400
)

// Contains all methods used for http based requests
const (
	HttpPOSTMethod = iota + 1
	HttpGETMethod
	HttpPUTMethod
)

// Contains all used url paths in the client
const (
	GetAllocationBlobbers      = "/v1/screst/:sc_address/alloc_blobbers"
	SCRestGetOpenChallenges    = "/v1/screst/:sc_address/openchallenges"
	MinerGetStatus             = "/v1/miner/get/stats"
	SharderGetStatus           = "/v1/sharder/get/stats"
	SCStateGet                 = "/v1/scstate/get"
	SCRestGetAllocation        = "/v1/screst/:sc_address/allocation"
	SCRestGetBlobbers          = "/v1/screst/:sc_address/getBlobber"
	ChainGetStats              = "/v1/chain/get/stats"
	ClientPut                  = "/v1/client/put"
	TransactionPut             = "/v1/transaction/put"
	TransactionGetConfirmation = "/v1/transaction/get/confirmation"
	ClientGetBalance           = "/v1/client/get/balance"
	GetNetworkDetails          = "/network"
)

// Contains all used service providers
const (
	MinerServiceProvider = iota
	SharderServiceProvider
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

//
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
	model.NetworkHealthResources

	httpClient *resty.Client //nolint
}

func NewAPIClient(networkEntrypoint string) *APIClient {
	apiClient := &APIClient{
		httpClient: resty.New(), //nolint
	}

	if err := apiClient.selectHealthServiceProviders(networkEntrypoint); err != nil {
		log.Fatalln(err)
	}

	return apiClient
}

func (c *APIClient) getHealthyNodes(nodes []string) []string {
	var result []string
	for _, node := range nodes {
		formattedURL := NewURLBuilder().MustShiftParse(node).SetPath(ChainGetStats).String()

		healthResponse, err := c.httpClient.R().Get(formattedURL)
		if err == nil && healthResponse.IsSuccess() {
			log.Printf("%s is UP!", node)
			result = append(result, node)
			continue
		}

		log.Printf("%s is DOWN!", node)
	}
	return result
}

func (c *APIClient) getHealthyMiners(miners []string) []string {
	return c.getHealthyNodes(miners)
}

func (c *APIClient) getHealthyShaders(shaders []string) []string {
	return c.getHealthyNodes(shaders)
}

func (c *APIClient) selectHealthServiceProviders(networkEntrypoint string) error {
	url := NewURLBuilder().MustShiftParse(networkEntrypoint).SetPath(GetNetworkDetails).String()

	resp, err := c.httpClient.R().Get(url)
	if err != nil {
		return ErrNetworkHealth
	}

	var networkDNSResponse *model.NetworkDNSResponse

	err = json.Unmarshal(resp.Body(), &networkDNSResponse)
	if err != nil {
		return ErrNetworkHealth
	}

	healthyMiners := c.getHealthyMiners(networkDNSResponse.Miners)
	if len(healthyMiners) == 0 {
		return ErrNoMinersHealth
	}

	c.NetworkHealthResources.Miners = healthyMiners

	healthySharders := c.getHealthyShaders(networkDNSResponse.Sharders)
	if len(healthySharders) == 0 {
		return ErrNoShadersHealth
	}

	c.NetworkHealthResources.Sharders = healthySharders
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
		err = json.Unmarshal(resp.Body(), executionRequest.Dst)
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
		serviceProviders = c.NetworkHealthResources.Miners
	case SharderServiceProvider:
		serviceProviders = c.NetworkHealthResources.Sharders
	}

	for _, serviceProvider := range serviceProviders {
		formattedURL := urlBuilder.MustShiftParse(serviceProvider).String()

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

	for _, error := range errors {
		frequencyCounters[error]++
		if frequencyCounters[error] > maxMatch {
			maxMatch = frequencyCounters[error]
			result = error
		}
	}

	return result
}

func (c *APIClient) V1ClientPut(clientPutRequest model.ClientPutRequest, requiredStatusCode int) (*model.Wallet, *resty.Response, error) { //nolint
	var clientPutResponse *model.ClientPutResponse

	urlBuilder := NewURLBuilder().SetPath(ClientPut)

	mnemonics := crypto.GenerateMnemonics()
	keyPair := crypto.GenerateKeys(mnemonics)
	publicKeyBytes, err := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	if err != nil {
		log.Fatalln(err)
	}

	if clientPutRequest.ClientID == "" {
		clientPutRequest.ClientID = crypto.Sha3256(publicKeyBytes)
	}

	if clientPutRequest.ClientKey == "" {
		clientPutRequest.ClientKey = keyPair.PublicKey.SerializeToHexStr()
	}

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

	wallet := &model.Wallet{
		ClientID:  clientPutResponse.Id,
		ClientKey: clientPutResponse.PublicKey,
		Keys: []*sys.KeyPair{{
			PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
			PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
		}},
		DateCreated: strconv.Itoa(*clientPutResponse.CreationDate),
		Mnemonics:   mnemonics,
		Version:     clientPutResponse.Version,
		Nonce:       clientPutResponse.Nonce,
		RawKeys:     keyPair,
	}

	return wallet, resp, err
}

func (c *APIClient) V1TransactionPut(internalTransactionPutRequest model.InternalTransactionPutRequest, requiredStatusCode int) (*model.TransactionPutResponse, *resty.Response, error) { //nolint
	var transactionPutResponse *model.TransactionPutResponse

	data, err := json.Marshal(internalTransactionPutRequest.TransactionData)
	if err != nil {
		log.Fatalln(err)
	}

	transactionPutRequest := model.TransactionPutRequest{
		ClientId:         internalTransactionPutRequest.Wallet.ClientID,
		PublicKey:        internalTransactionPutRequest.Wallet.ClientKey,
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

	transactionPutRequest.Hash = crypto.Sha3256([]byte(fmt.Sprintf("%d:%d:%s:%s:%d:%s",
		transactionPutRequest.CreationDate,
		transactionPutRequest.TransactionNonce,
		transactionPutRequest.ClientId,
		transactionPutRequest.ToClientId,
		transactionPutRequest.TransactionValue,
		crypto.Sha3256([]byte(transactionPutRequest.TransactionData)))))

	hashToSign, err := hex.DecodeString(transactionPutRequest.Hash)
	if err != nil {
		log.Fatalln(err)
	}

	transactionPutRequest.Signature = internalTransactionPutRequest.Wallet.RawKeys.PrivateKey.Sign(string(hashToSign)).
		SerializeToHexStr()

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

func (c *APIClient) V1TransactionGetConfirmation(transactionGetConfirmationRequest model.TransactionGetConfirmationRequest, requiredStatusCode, requiredTransactionStatus int) (*model.TransactionGetConfirmationResponse, *resty.Response, error) { //nolint
	var transactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	urlBuilder := NewURLBuilder().
		SetPath(TransactionGetConfirmation).
		AddParams("hash", transactionGetConfirmationRequest.Hash)

	var (
		resp *resty.Response //nolint
		err  error
	)

	wait.PoolImmediately(time.Minute*2, func() bool {
		resp, err = c.executeForAllServiceProviders(
			urlBuilder,
			model.ExecutionRequest{
				Dst:                &transactionGetConfirmationResponse,
				RequiredStatusCode: requiredStatusCode,
			},
			HttpGETMethod,
			SharderServiceProvider)
		if err != nil {
			return false
		}

		if resp.StatusCode() != requiredStatusCode {
			return false
		}

		if transactionGetConfirmationResponse.Status != requiredTransactionStatus {
			return false
		}

		return true
	})

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

func (c *APIClient) V1SCRestGetAllocationBlobbers(scRestGetAllocationBlobbersRequest *model.SCRestGetAllocationBlobbersRequest, requiredStatusCode int) (model.SCRestGetAllocationBlobbersResponse, *resty.Response, error) { //nolint
	var scRestGetAllocationBlobbersResponse model.SCRestGetAllocationBlobbersResponse

	blobberRequirements := model.BlobberRequirements{
		Size:           10000,
		DataShards:     1,
		ParityShards:   1,
		ExpirationDate: time.Now().Add(time.Minute * 20).Unix(),
		ReadPriceRange: model.PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		WritePriceRange: model.PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		OwnerId:        scRestGetAllocationBlobbersRequest.ClientID,
		OwnerPublicKey: scRestGetAllocationBlobbersRequest.ClientKey,
	}

	data, err := json.Marshal(blobberRequirements)
	if err != nil {
		log.Fatalln(err)
	}

	urlBuilder := NewURLBuilder().
		SetPath(GetAllocationBlobbers).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("allocation_data", string(data))

	var blobbers *[]string

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                &blobbers,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	scRestGetAllocationBlobbersResponse.Blobbers = blobbers
	scRestGetAllocationBlobbersResponse.BlobberRequirements = blobberRequirements

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

// RegisterWalletWrapper does not provide deep test of used components
func (c *APIClient) RegisterWalletWrapper(t *testing.T) *model.Wallet {
	t.Log("Register wallet...")

	wallet, resp, err := c.V1ClientPut(model.ClientPutRequest{}, HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)

	return wallet
}

// ExecuteFaucetWrapper does not provide deep test of used components
func (c *APIClient) ExecuteFaucetWrapper(t *testing.T, wallet *model.Wallet) {
	t.Log("Execute faucet...")

	faucetTransactionPutResponse, resp, err := c.V1TransactionPut(
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      FaucetSmartContractAddress,
			TransactionData: model.NewFaucetTransactionData()},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, faucetTransactionPutResponse)

	faucetTransactionGetConfirmationResponse, resp, err := c.V1TransactionGetConfirmation(
		model.TransactionGetConfirmationRequest{
			Hash: faucetTransactionPutResponse.Entity.Hash,
		},
		HttpOkStatus,
		TxSuccessfulStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, faucetTransactionGetConfirmationResponse)

	wallet.IncNonce()
}

// CreateAllocationWrapper does not provide deep test of used components
func (c *APIClient) CreateAllocationWrapper(t *testing.T, wallet *model.Wallet, scRestGetAllocationBlobbersResponse model.SCRestGetAllocationBlobbersResponse) string {
	t.Log("Create allocation...")

	createAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
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

	createAllocationTransactionGetConfirmationResponse, resp, err := c.V1TransactionGetConfirmation(
		model.TransactionGetConfirmationRequest{
			Hash: createAllocationTransactionPutResponse.Entity.Hash,
		},
		HttpOkStatus,
		TxSuccessfulStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createAllocationTransactionGetConfirmationResponse)

	wallet.IncNonce()

	return createAllocationTransactionPutResponse.Entity.Hash
}

// CreateStakePoolWrapper does not provide deep test of used components
func (c *APIClient) CreateStakePoolWrapper(t *testing.T) {
	t.Log("Create stake pool...")
}

// UpdateAllocationWrapper does not provide deep test of used components
func (c *APIClient) UpdateAllocationWrapper(t *testing.T) {
	t.Log("Update allocation...")
}

func (c *APIClient) GetAllocationBlobbersWrapper(t *testing.T, wallet *model.Wallet) model.SCRestGetAllocationBlobbersResponse {
	t.Log("Get allocation blobbers...")

	scRestGetAllocationBlobbersResponse, resp, err := c.V1SCRestGetAllocationBlobbers(
		&model.SCRestGetAllocationBlobbersRequest{
			ClientID:  wallet.ClientID,
			ClientKey: wallet.ClientKey,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)

	return scRestGetAllocationBlobbersResponse
}

// GetAllocationWrapper does not provide deep test of used components
func (c *APIClient) GetAllocationWrapper(t *testing.T, allocationID string) *model.SCRestGetAllocationResponse {
	t.Log("Get allocation...")

	scRestGetAllocation, resp, err := c.V1SCRestGetAllocation(
		model.SCRestGetAllocationRequest{
			AllocationID: allocationID,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, scRestGetAllocation)

	return scRestGetAllocation
}

// GetWalletBalanceWrapper does not provide deep test of used components
func (c *APIClient) GetWalletBalanceWrapper(t *testing.T, wallet *model.Wallet) *model.ClientGetBalanceResponse {
	t.Log("Get wallet balance...")

	clientGetBalanceResponse, resp, err := c.V1ClientGetBalance(
		model.ClientGetBalanceRequest{
			ClientID: wallet.ClientID,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)

	return clientGetBalanceResponse
}
