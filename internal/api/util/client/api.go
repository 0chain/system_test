package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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
	GetHashNodeRoot            = "/v1/hashnode/root/:allocation"
	GetBlobbers                = "/v1/screst/:sc_address/getblobbers"
	GetStakePoolStat           = "/v1/screst/:sc_address/getStakePoolStat"
	GetAllocationBlobbers      = "/v1/screst/:sc_address/alloc_blobbers"
	SCRestGetOpenChallenges    = "/v1/screst/:sc_address/openchallenges"
	MinerGetStatus             = "/v1/miner/get/stats"
	SharderGetStatus           = "/v1/sharder/get/stats"
	SCStateGet                 = "/v1/scstate/get"
	SCRestGetAllocation        = "/v1/screst/:sc_address/allocation"
	SCRestGetBlobbers          = "/v1/screst/:sc_address/getBlobber"
	ChainGetStats              = "/v1/chain/get/stats"
	BlobberGetStats            = "/_stats"
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

		var formattedURL string
		switch serviceProviderType {
		case MinerServiceProvider:
			formattedURL = urlBuilder.SetPath(ChainGetStats).String()
		case SharderServiceProvider:
			formattedURL = urlBuilder.SetPath(ChainGetStats).String()
		case BlobberServiceProvider:
			formattedURL = urlBuilder.SetPath(BlobberGetStats).String()
		}

		healthResponse, err := c.httpClient.R().Get(formattedURL)
		if err == nil && healthResponse.IsSuccess() {
			log.Printf("%s is UP!", node)
			result = append(result, node)
			continue
		}

		log.Printf("%s is DOWN!", node)
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

	for _, error := range errors {
		frequencyCounters[error]++
		if frequencyCounters[error] > maxMatch {
			maxMatch = frequencyCounters[error]
			result = error
		}
	}

	return result
}

func (c *APIClient) V1ClientPut(t *testing.T, clientPutRequest model.Wallet, requiredStatusCode int) (*model.Wallet, *resty.Response, error) { //nolint
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

	transactionPutRequest.Hash = crypto.Sha3256([]byte(fmt.Sprintf("%d:%d:%s:%s:%d:%s",
		transactionPutRequest.CreationDate,
		transactionPutRequest.TransactionNonce,
		transactionPutRequest.ClientId,
		transactionPutRequest.ToClientId,
		transactionPutRequest.TransactionValue,
		crypto.Sha3256([]byte(transactionPutRequest.TransactionData)))))

	crypto.SignTransaction(t, &transactionPutRequest, &internalTransactionPutRequest.Wallet.Keys)

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

func (c *APIClient) V1BlobberGetHashNodeRoot(t *testing.T, blobberGetHashnodeRequest model.BlobberGetHashnodeRequest, requiredStatusCode int) (*model.BlobberGetHashnodeResponse, *resty.Response, error) {
	var hashnode *model.BlobberGetHashnodeResponse

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetHashnodeRequest.ClientId,
		"X-App-Client-Key":       blobberGetHashnodeRequest.ClientKey,
		"X-App-Client-Signature": blobberGetHashnodeRequest.ClientSignature,
		"allocation":             blobberGetHashnodeRequest.AllocationID,
	}

	url := blobberGetHashnodeRequest.URL + "/" + strings.Replace(GetHashNodeRoot, ":allocation", blobberGetHashnodeRequest.AllocationID, 1)

	resp, err := c.executeForServiceProvider(url,
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

	registeredWallet, httpResponse, err := c.V1ClientPut(t, walletRequest, expectedHttpStatus)
	registeredWallet.Keys = *keyPair

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
			TransactionData: model.NewUpdateAllocationTransactionData(model.UpdateAllocationRequest{
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

	var updateAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		updateAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			model.TransactionGetConfirmationRequest{
				Hash: allocationID,
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

func (c *APIClient) GetAllocationBlobbers(t *testing.T, wallet *model.Wallet, customBlobberRequirements *model.BlobberRequirements, requiredStatusCode int) *model.SCRestGetAllocationBlobbersResponse {
	t.Log("Get allocation blobbers...")

	var blobberRequirements model.BlobberRequirements

	if customBlobberRequirements != nil {
		blobberRequirements = *customBlobberRequirements
	} else {
		blobberRequirements = model.BlobberRequirements{
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
			OwnerId:        wallet.Id,
			OwnerPublicKey: wallet.PublicKey,
		}
	}

	scRestGetAllocationBlobbersResponse, resp, err := c.V1SCRestGetAllocationBlobbers(
		&model.SCRestGetAllocationBlobbersRequest{
			ClientID:            wallet.Id,
			ClientKey:           wallet.PublicKey,
			BlobberRequirements: blobberRequirements,
		},
		requiredStatusCode)
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
		AddParams("blobber_id", scRestGetStakePoolStatRequest.BlobberID)

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

func (c *APIClient) GetStakePoolStat(t *testing.T, blobberID string) *model.SCRestGetStakePoolStatResponse {
	t.Log("Get stake pool stat...")

	scRestGetStakePoolStat, resp, err := c.V1SCRestGetStakePoolStat(
		model.SCRestGetStakePoolStatRequest{
			BlobberID: blobberID,
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
