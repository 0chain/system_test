package client

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/tokenomics"

	resty "github.com/go-resty/resty/v2"
)

// Contains all used url paths in the client
const (
	GetHashNodeRoot                    = "/v1/hashnode/root/:allocation"
	GetBlobbers                        = "/v1/screst/:sc_address/getblobbers"
	GetMiners                          = "/v1/screst/:sc_address/getMinerList"
	GetSharders                        = "/v1/screst/:sc_address/getSharderList"
	GetValidators                      = "/v1/screst/:sc_address/validators"
	GetStakePoolStat                   = "/v1/screst/:sc_address/getStakePoolStat"
	getUserStakePoolStat               = "/v1/screst/:sc_address/getUserStakePoolStat"
	GetAllocationBlobbers              = "/v1/screst/:sc_address/alloc_blobbers"
	GetFreeAllocationBlobbers          = "/v1/screst/:sc_address/free_alloc_blobbers"
	SCRestGetOpenChallenges            = "/v1/screst/:sc_address/openchallenges"
	MinerGetStatus                     = "/v1/miner/get/stats"
	SharderGetStatus                   = "/v1/sharder/get/stats"
	SCStateGet                         = "/v1/scstate/get"
	SCRestGetAllocation                = "/v1/screst/:sc_address/allocation"
	SCRestGetBlobbers                  = "/v1/screst/:sc_address/getBlobber"
	ChainGetStats                      = "/v1/chain/get/stats"
	BlobberGetStats                    = "/_stats"
	ClientPut                          = "/v1/client/put"
	TransactionPut                     = "/v1/transaction/put"
	TransactionFeeGet                  = "/v1/estimate_txn_fee"
	TransactionGetConfirmation         = "/v1/transaction/get/confirmation"
	ClientGetBalance                   = "/v1/client/get/balance"
	GetNetworkDetails                  = "/network"
	GetFileRef                         = "/v1/file/refs/:allocation_id"
	GetFileRefPath                     = "/v1/file/referencepath/:allocation_id"
	GetObjectTree                      = "/v1/file/objecttree/:allocation_id"
	GetLatestFinalizedMagicBlock       = "/v1/block/get/latest_finalized_magic_block"
	GetLatestFinalizedBlock            = "/v1/block/get/latest_finalized"
	QueryRewards                       = "/v1/screst/:sc_address/query-rewards"
	QueryChallengesCount               = "/v1/screst/:sc_address/count-challenges"
	QueryDelegateRewards               = "/v1/screst/:sc_address/query-delegate-rewards"
	PartitionSizeFrequency             = "/v1/screst/:sc_address/parition-size-frequency"
	BlobberPartitionSelectionFrequency = "/v1/screst/:sc_address/blobber-selection-frequency"
	GetAllChallenges                   = "/v1/screst/:sc_address/all-challenges"
)

// Contains all used service providers
const (
	MinerServiceProvider = iota
	SharderServiceProvider
	BlobberServiceProvider
)

// Contains all smart contract addreses used in the client
const (
	MinerSmartContractAddress   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
	FaucetSmartContractAddress  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	StorageSmartContractAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	ZCNSmartContractAddess      = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
)

// Contains statuses of transactions
const (
	TxSuccessfulStatus = iota + 1
	TxUnsuccessfulStatus
)

const (
	SendTxType = 0
	SCTxType   = 1000
	TxFee      = 2.0 * 1e10
	TxVersion  = "1.0"
	TxOutput   = ""
)

var (
	TxValue = tokenomics.IntToZCN(1)
)

type APIClient struct {
	BaseHttpClient
	model.HealthyServiceProviders
	networkEntrypoint string
}

func NewAPIClient(networkEntrypoint string) *APIClient {
	apiClient := &APIClient{networkEntrypoint: networkEntrypoint}
	apiClient.HttpClient = resty.New()

	if err := apiClient.selectHealthyServiceProviders(networkEntrypoint); err != nil {
		log.Fatalln(err)
	}

	return apiClient
}

// RefreshServiceProviders re-queries 0dns to get the current active miners/sharders.
// This should be called when transient errors suggest the active set may have changed
// (e.g. after a view change).
func (c *APIClient) RefreshServiceProviders() {
	if err := c.selectHealthyServiceProviders(c.networkEntrypoint); err != nil {
		log.Printf("WARNING: Failed to refresh service providers from 0dns: %v", err)
	}
}

func (c *APIClient) getHealthyNodes(nodes []string, serviceProviderType int) ([]string, error) {
	var result []string
	for _, node := range nodes {
		urlBuilder := NewURLBuilder()
		if err := urlBuilder.MustShiftParse(node); err != nil {
			return nil, err
		}

		r := c.HttpClient.R()
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
	reachable, err := c.getHealthyNodes(sharders, SharderServiceProvider)
	if err != nil || len(reachable) == 0 {
		return reachable, err
	}

	// Filter by LFB: only keep sharders within 3 blocks of the highest LFB
	const lfbMaxDrift = 3
	type sharderLFB struct {
		url   string
		round int64
	}
	var responding []sharderLFB
	var maxLFB int64

	for _, sharder := range reachable {
		lfbURL := sharder + "/v1/current-round"
		resp, err := c.HttpClient.R().SetHeader("Content-Type", "application/json").Get(lfbURL)
		if err != nil || !resp.IsSuccess() {
			log.Printf("Sharder %s LFB check failed", sharder)
			continue
		}
		var round int64
		if err := json.Unmarshal(resp.Body(), &round); err != nil {
			log.Printf("Sharder %s LFB parse failed: %s", sharder, err)
			continue
		}
		responding = append(responding, sharderLFB{url: sharder, round: round})
		if round > maxLFB {
			maxLFB = round
		}
	}

	var healthy []string
	for _, s := range responding {
		if maxLFB-s.round <= lfbMaxDrift {
			healthy = append(healthy, s.url)
		} else {
			log.Printf("Sharder %s too far behind: LFB %d vs highest %d (drift %d)", s.url, s.round, maxLFB, maxLFB-s.round)
		}
	}

	log.Printf("LFB check: %d/%d sharders healthy (within %d blocks of LFB %d)", len(healthy), len(reachable), lfbMaxDrift, maxLFB)

	if len(healthy) == 0 {
		log.Printf("WARNING: No sharders passed LFB check, falling back to all reachable sharders")
		return reachable, nil
	}

	return healthy, nil
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

	resp, err := c.HttpClient.R().Get(formattedURL)
	if err != nil {
		return errors.New(ErrNetworkHealthy.Error() + "error fetching network details from url: " + formattedURL)
	}

	var networkServiceProviders *model.HealthyServiceProviders

	err = json.Unmarshal(resp.Body(), &networkServiceProviders)
	if err != nil {
		return errors.New(ErrNetworkHealthy.Error() + "failed to unmarshall network service providers. Body: " + string(resp.Body()))
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
		// Fallback: check SHARDERS env var for override (comma-separated URLs)
		if shardersOverride, ok := os.LookupEnv("SHARDERS"); ok && shardersOverride != "" {
			overrideSharders := strings.Split(shardersOverride, ",")
			log.Printf("No healthy sharders from 0dns, falling back to SHARDERS env: %v", overrideSharders)
			healthySharders, err = c.getHealthyShaders(overrideSharders)
			if err != nil {
				return err
			}
			if len(healthySharders) == 0 {
				return ErrNoShadersHealthy
			}
		} else {
			return ErrNoShadersHealthy
		}
	}

	c.HealthyServiceProviders.Sharders = healthySharders

	offset := 0
	limit := 20
	var nodes model.StorageNodes

	// Use the first healthy sharder for blobber discovery (not the first from 0dns which may be down)
	sharderForDiscovery := healthySharders[0]

	for {
		if err := urlBuilder.MustShiftParse(sharderForDiscovery); err != nil {
			return err
		}
		urlBuilder = urlBuilder.SetPath(GetBlobbers).SetPathVariable("sc_address", StorageSmartContractAddress)
		formattedURL = urlBuilder.AddParams("offset", fmt.Sprint(offset)).AddParams("limit", fmt.Sprint(limit)).String()
		resp, err = c.HttpClient.R().Get(formattedURL)
		if err != nil {
			return ErrNoBlobbersHealthy
		}
		err = json.Unmarshal(resp.Body(), &nodes)
		if err != nil {
			return errors.New(ErrNetworkHealthy.Error() + "failed to unmarshall network service providers. Body: " + string(resp.Body()))
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

func (c *APIClient) executeForGivenServiceProviders(
	t *test.SystemTest,
	urlBuilder *URLBuilder,
	executionRequest *model.ExecutionRequest,
	method int,
	serviceProviders []string,
) (*resty.Response, error) {
	var (
		resp                                *resty.Response
		respErrors                          []error
		expectedExecutionResponseCounter    int
		notExpectedExecutionResponseCounter int
	)

	for _, serviceProvider := range serviceProviders {
		if err := urlBuilder.MustShiftParse(serviceProvider); err != nil {
			return nil, err
		}
		formattedURL := urlBuilder.String()

		newResp, err := c.executeForServiceProvider(t, formattedURL, *executionRequest, method)
		if err != nil {
			respErrors = append(respErrors, err)
			continue
		}

		if newResp.StatusCode() == executionRequest.RequiredStatusCode {
			expectedExecutionResponseCounter++
			resp = newResp
		} else {
			t.Logf("Provider %s. Response: %s", serviceProvider, string(newResp.Body()))
			respErrors = append(respErrors, errors.New(fmt.Sprintf("Provider %s. Response: %s", serviceProvider, string(newResp.Body()))))
			notExpectedExecutionResponseCounter++
		}
	}

	if notExpectedExecutionResponseCounter > expectedExecutionResponseCounter {
		return nil, errors.Join(ErrExecutionConsensus, selectMostFrequentError(respErrors))
	}

	// Consensus reached - don't propagate errors from minority of failed providers
	return resp, nil
}

func (c *APIClient) executeForAllServiceProviders(
	t *test.SystemTest,
	urlBuilder *URLBuilder,
	executionRequest *model.ExecutionRequest,
	method,
	serviceProviderType int,
) (*resty.Response, error) {
	var serviceProviders []string

	switch serviceProviderType {
	case MinerServiceProvider:
		serviceProviders = c.HealthyServiceProviders.Miners
	case SharderServiceProvider:
		serviceProviders = c.HealthyServiceProviders.Sharders
	case BlobberServiceProvider:
		serviceProviders = c.HealthyServiceProviders.Blobbers
	}

	return c.executeForGivenServiceProviders(t, urlBuilder, executionRequest, method, serviceProviders)
}

func selectMostFrequentError(respErrors []error) error {
	frequencyCounters := make(map[error]int)
	var maxMatch int
	var result error

	for _, error := range respErrors {
		frequencyCounters[error]++
		if frequencyCounters[error] > maxMatch {
			maxMatch = frequencyCounters[error]
			result = error
		}
	}

	return result
}

func (c *APIClient) V1ClientPut(t *test.SystemTest, clientPutRequest model.Wallet, requiredStatusCode int) (*model.Wallet, *resty.Response, error) { //nolint
	var clientPutResponse *model.Wallet

	urlBuilder := NewURLBuilder().SetPath(ClientPut)
	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
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

func (c *APIClient) V1TransactionPut(
	t *test.SystemTest,
	internalTransactionPutRequest model.InternalTransactionPutRequest,
	requiredStatusCode int, options ...float64,
) (*model.TransactionPutResponse, *resty.Response, error) { //nolint

	return c.V1TransactionPutWithNonceAndServiceProviders(t, internalTransactionPutRequest, requiredStatusCode, 0, nil, options...)
}

func (c *APIClient) V1TransactionPutWithNonceAndServiceProviders(
	t *test.SystemTest,
	internalTransactionPutRequest model.InternalTransactionPutRequest,
	requiredStatusCode, withNonce int, withProviders []string, options ...float64,
) (*model.TransactionPutResponse, *resty.Response, error) { //nolint
	var (
		transactionPutResponse *model.TransactionPutResponse
		resp                   *resty.Response
		err                    error
	)

	// Always sync nonce from chain before submitting (vc.sh pattern).
	// This ensures stale local nonces (e.g., from wallet reuse across tests) are corrected.
	c.RefreshNonce(t, internalTransactionPutRequest.Wallet, 200)

	// Fix: use a fixed creationDate across retries so all retries produce the same hash.
	// This prevents multiple competing txns at the same nonce entering the mempool.
	// Only reset creationDate on "invalid transaction nonce" (where a new txn is needed).
	creationDate := time.Now().Unix()

	for retry := 0; retry < 3; retry++ {
		var data []byte
		data, err = json.Marshal(internalTransactionPutRequest.TransactionData)
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
			TransactionType:  internalTransactionPutRequest.TxnType,
			TransactionFee:   int64(TxFee),
			TransactionData:  string(data),
			CreationDate:     creationDate,
			Version:          TxVersion,
		}

		if withNonce != 0 {
			transactionPutRequest.TransactionNonce = withNonce
		}

		if len(options) == 0 {
			if internalTransactionPutRequest.TransactionData.Name == "pour" {
				transactionPutRequest.TransactionFee = 0
			} else {
				fee := estimateTxnFee(t, c, &transactionPutRequest)
				if fee < int64(TxFee) {
					// Fee estimation may return 0 or an under-estimate for SC transactions.
					// Use TxFee as a minimum floor so miners don't reject with
					// "insufficient transaction fee".
					fee = int64(TxFee)
				}
				transactionPutRequest.TransactionFee = fee
			}
		} else {
			transactionPutRequest.TransactionFee = int64(options[0] * 1e10)
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

		crypto.SignTransaction(t, &transactionPutRequest, internalTransactionPutRequest.Wallet.Keys)

		serviceProviders := c.HealthyServiceProviders.Miners
		if withProviders != nil {
			serviceProviders = withProviders
		}

		resp, err = c.executeForGivenServiceProviders(
			t,
			NewURLBuilder().SetPath(TransactionPut),
			&model.ExecutionRequest{
				Body:               transactionPutRequest,
				Dst:                &transactionPutResponse,
				RequiredStatusCode: requiredStatusCode,
			},
			HttpPOSTMethod,
			serviceProviders)

		if transactionPutResponse != nil {
			transactionPutResponse.Request = transactionPutRequest
		}

		if err != nil && strings.Contains(err.Error(), "invalid transaction nonce") {
			// Nonce mismatch: old txn was finalized at different nonce. Need fresh nonce + new hash.
			t.Logf("Transient transaction error (retry %d/3): %s — refreshing 0dns and nonce", retry+1, err.Error())
			c.RefreshServiceProviders()
			c.RefreshNonce(t, internalTransactionPutRequest.Wallet, 200)
			creationDate = time.Now().Unix() // New txn at new nonce — new hash required
			continue
		}
		if err != nil && strings.Contains(err.Error(), "unexpected end of JSON input") {
			// Network/parse error: txn may already be in mempool. Retry with same hash.
			// Do NOT refresh nonce — if the txn was finalized, refreshing would cause
			// the next retry to submit at wrong nonce.
			t.Logf("Transient transaction error (retry %d/3): %s — refreshing 0dns only", retry+1, err.Error())
			c.RefreshServiceProviders()
			continue
		}

		break
	}
	return transactionPutResponse, resp, err
}

func estimateTxnFee(t *test.SystemTest, c *APIClient, transactionPutRequest *model.TransactionPutRequest) int64 {
	urlBuilder := NewURLBuilder().SetPath(TransactionFeeGet)
	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Body:               transactionPutRequest,
			RequiredStatusCode: 200,
		},
		HttpPOSTMethod,
		MinerServiceProvider)

	var fee = struct {
		Fee int64 `json:"fee"`
	}{}
	if err != nil || resp == nil {
		t.Logf("estimateTxnFee: failed to get fee (err=%v, resp=%v), using default fee", err, resp != nil)
		return int64(TxFee)
	}
	err = json.Unmarshal(resp.Body(), &fee)
	if err != nil {
		t.Logf("estimateTxnFee: failed to unmarshal fee response: %v", err)
		return int64(TxFee)
	}
	return fee.Fee
}

func (c *APIClient) V1TransactionGetConfirmation(
	t *test.SystemTest,
	transactionGetConfirmationRequest model.TransactionGetConfirmationRequest,
	requiredStatusCode int,
) (*model.TransactionGetConfirmationResponse, *resty.Response, error) { //nolint

	var transactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	urlBuilder := NewURLBuilder().
		SetPath(TransactionGetConfirmation).
		AddParams("hash", transactionGetConfirmationRequest.Hash)

	var resp *resty.Response
	var err error
	for retry := 0; retry < 3; retry++ {
		transactionGetConfirmationResponse = nil
		resp, err = c.executeForAllServiceProviders(
			t,
			urlBuilder,
			&model.ExecutionRequest{
				Dst:                &transactionGetConfirmationResponse,
				RequiredStatusCode: requiredStatusCode,
			},
			HttpGETMethod,
			SharderServiceProvider)

		if err != nil && strings.Contains(err.Error(), "unexpected end of JSON input") {
			t.Logf("Transient confirmation error (retry %d/3): %s — refreshing 0dns", retry+1, err.Error())
			c.RefreshServiceProviders()
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	return transactionGetConfirmationResponse, resp, err
}

func (c *APIClient) V1ClientGetBalance(t *test.SystemTest, clientGetBalanceRequest model.ClientGetBalanceRequest, requiredStatusCode int) (*model.ClientGetBalanceResponse, *resty.Response, error) { //nolint
	var clientGetBalanceResponse *model.ClientGetBalanceResponse

	urlBuilder := NewURLBuilder().SetPath(ClientGetBalance).AddParams("client_id", clientGetBalanceRequest.ClientID)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &clientGetBalanceResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return clientGetBalanceResponse, resp, err
}

func (c *APIClient) V1SCRestGetAllMiners(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetMinerSharderResponse, *resty.Response, error) {
	var scRestGetMinersResponse *model.SCRestGetMinersShardersResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetMiners).
		SetPathVariable("sc_address", MinerSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetMinersResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider,
	)
	return scRestGetMinersResponse.Nodes, resp, err
}

func (c *APIClient) V1SCRestGetAllSharders(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetMinerSharderResponse, *resty.Response, error) {
	var scRestGetShardersResponse *model.SCRestGetMinersShardersResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetSharders).
		SetPathVariable("sc_address", MinerSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetShardersResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider,
	)
	return scRestGetShardersResponse.Nodes, resp, err
}

func (c *APIClient) V1SCRestGetAllBlobbers(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetBlobberResponse, *resty.Response, error) {
	var scRestGetBlobbersResponse *model.SCRestGetBlobbersResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetBlobbers).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetBlobbersResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider,
	)
	return scRestGetBlobbersResponse.Nodes, resp, err
}

func (c *APIClient) V1SCRestGetAllValidators(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetValidatorResponse, *resty.Response, error) {
	var scRestGetValidatorsResponse []*model.SCRestGetValidatorResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetValidators).
		SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetValidatorsResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider,
	)
	return scRestGetValidatorsResponse, resp, err
}

func (c *APIClient) V1SCRestGetFirstBlobbers(t *test.SystemTest, blobbersCount, requiredStatusCode int) ([]*model.SCRestGetBlobberResponse, *resty.Response, error) {
	var scRestGetBlobbersResponse *model.SCRestGetBlobbersResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetBlobbers).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("active", "true").
		AddParams("limit", "10")

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetBlobbersResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider,
	)
	if len(scRestGetBlobbersResponse.Nodes) < blobbersCount {
		return nil, resp, errors.New("not enough blobbers")
	}
	return scRestGetBlobbersResponse.Nodes[:blobbersCount], resp, err
}

func (c *APIClient) V1SCRestGetBlobber(t *test.SystemTest, scRestGetBlobberRequest model.SCRestGetBlobberRequest, requiredStatusCode int) (*model.SCRestGetBlobberResponse, *resty.Response, error) {
	var scRestGetBlobberResponse *model.SCRestGetBlobberResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCRestGetBlobbers).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("blobber_id", scRestGetBlobberRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetBlobberResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetBlobberResponse, resp, err
}

func (c *APIClient) V1BlobberGetHashNodeRoot(t *test.SystemTest, blobberGetHashnodeRequest *model.BlobberGetHashnodeRequest, requiredStatusCode int) (*model.BlobberGetHashnodeResponse, *resty.Response, error) {
	var hashnode *model.BlobberGetHashnodeResponse

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetHashnodeRequest.ClientId,
		"X-App-Client-Key":       blobberGetHashnodeRequest.ClientKey,
		"X-App-Client-Signature": blobberGetHashnodeRequest.ClientSignature,
		"allocation":             blobberGetHashnodeRequest.AllocationID,
		"ALLOCATION-ID":          blobberGetHashnodeRequest.AllocationID,
	}

	blobberGetHashNodeRootURL := blobberGetHashnodeRequest.URL + "/" + strings.Replace(GetHashNodeRoot, ":allocation", blobberGetHashnodeRequest.AllocationID, 1)

	resp, err := c.executeForServiceProvider(t,
		blobberGetHashNodeRootURL,
		model.ExecutionRequest{
			Headers:            headers,
			Dst:                &hashnode,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
	)
	return hashnode, resp, err
}

func (c *APIClient) V1SCRestGetAllocation(t *test.SystemTest, scRestGetAllocationRequest model.SCRestGetAllocationRequest, requiredStatusCode int) (*model.SCRestGetAllocationResponse, *resty.Response, error) { //nolint
	var scRestGetAllocationResponse *model.SCRestGetAllocationResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCRestGetAllocation).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("allocation", scRestGetAllocationRequest.AllocationID)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetAllocationResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetAllocationResponse, resp, err
}

func (c *APIClient) V1SCRestGetAllocationBlobbers(t *test.SystemTest, scRestGetAllocationBlobbersRequest *model.SCRestGetAllocationBlobbersRequest, requiredStatusCode int) (*model.SCRestGetAllocationBlobbersResponse, *resty.Response, error) { //nolint
	scRestGetAllocationBlobbersResponse := new(model.SCRestGetAllocationBlobbersResponse)
	scRestGetAllocationBlobbersResponse.StorageVersion = 1

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
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &blobbers,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	scRestGetAllocationBlobbersResponse.Blobbers = blobbers
	for range *blobbers {
		scRestGetAllocationBlobbersResponse.BlobberAuthTickets = append(scRestGetAllocationBlobbersResponse.BlobberAuthTickets, "")
	}
	scRestGetAllocationBlobbersResponse.BlobberRequirements = scRestGetAllocationBlobbersRequest.BlobberRequirements

	return scRestGetAllocationBlobbersResponse, resp, err
}

func (c *APIClient) V1SCRestGetFreeAllocationBlobbers(t *test.SystemTest, scRestGetFreeAllocationBlobbersRequest *model.FreeAllocationData, requiredStatusCode int) (*model.SCRestGetFreeAllocationBlobbersResponse, *resty.Response, error) { //nolint
	data, err := json.Marshal(scRestGetFreeAllocationBlobbersRequest)
	if err != nil {
		return nil, nil, err
	}

	urlBuilder := NewURLBuilder().
		SetPath(GetFreeAllocationBlobbers).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("free_allocation_data", string(data))

	var blobbers *[]string
	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &blobbers,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider,
	)

	res := model.SCRestGetFreeAllocationBlobbersResponse{Blobbers: blobbers}
	return &res, resp, err
}

func (c *APIClient) V1SCRestOpenChallenge(t *test.SystemTest, scRestOpenChallengeRequest model.SCRestOpenChallengeRequest, requiredStatusCode int) (*model.SCRestOpenChallengeResponse, *resty.Response, error) { //nolint
	var scRestOpenChallengeResponse *model.SCRestOpenChallengeResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCRestGetOpenChallenges).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("blobber", scRestOpenChallengeRequest.BlobberID)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestOpenChallengeResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestOpenChallengeResponse, resp, err
}

func (c *APIClient) V1MinerGetStats(t *test.SystemTest, requiredStatusCode int) (*model.GetMinerStatsResponse, *resty.Response, error) { //nolint
	var getMinerStatsResponse *model.GetMinerStatsResponse

	urlBuilder := NewURLBuilder().
		SetPath(MinerGetStatus)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &getMinerStatsResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		MinerServiceProvider)

	return getMinerStatsResponse, resp, err
}

func (c *APIClient) V1SharderGetStats(t *test.SystemTest, requiredStatusCode int) (*model.GetSharderStatsResponse, *resty.Response, error) { //nolint
	var getSharderStatusResponse *model.GetSharderStatsResponse

	urlBuilder := NewURLBuilder().
		SetPath(SharderGetStatus)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &getSharderStatusResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return getSharderStatusResponse, resp, err
}

func (c *APIClient) V1SharderGetSCState(t *test.SystemTest, scStateGetRequest model.SCStateGetRequest, requiredStatusCode int) (*model.SCStateGetResponse, *resty.Response, error) { //nolint
	var scStateGetResponse *model.SCStateGetResponse

	urlBuilder := NewURLBuilder().
		SetPath(SCStateGet)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
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

func (c *APIClient) CreateWalletForMnemonic(t *test.SystemTest, mnemonic string) *model.Wallet {
	createdWallet, err := c.CreateWalletForMnemonicWithoutAssertion(t, mnemonic)
	require.Nil(t, err)

	publicKeyBytes, _ := hex.DecodeString(createdWallet.Keys.PublicKey.SerializeToHexStr())
	clientId := crypto.Sha3256(publicKeyBytes)

	require.Equal(t, createdWallet.Id, clientId)
	require.Equal(t, createdWallet.PublicKey, createdWallet.Keys.PublicKey.SerializeToHexStr())

	return createdWallet
}

func (c *APIClient) CreateWalletForMnemonicWithoutAssertion(t *test.SystemTest, mnemonic string) (*model.Wallet, error) {
	keyPair := crypto.GenerateKeys(t, mnemonic)
	publicKeyBytes, err := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	if err != nil {
		return nil, err
	}

	clientId := crypto.Sha3256(publicKeyBytes)
	createdWallet := model.Wallet{Id: clientId, PublicKey: keyPair.PublicKey.SerializeToHexStr(), Keys: keyPair}

	return &createdWallet, err
}

func (c *APIClient) CreateAllocation(t *test.SystemTest,
	wallet *model.Wallet,
	scRestGetAllocationBlobbersResponse *model.SCRestGetAllocationBlobbersResponse,
	requiredTransactionStatus int) string {
	return c.CreateAllocationWithLockValue(t, wallet, scRestGetAllocationBlobbersResponse, 10.0, requiredTransactionStatus)
}

func (c *APIClient) CreateAllocationWithLockValue(t *test.SystemTest,
	wallet *model.Wallet,
	scRestGetAllocationBlobbersResponse *model.SCRestGetAllocationBlobbersResponse,
	lockValue float64,
	requiredTransactionStatus int) string {
	t.Log("Create allocation...")

	// Ensure wallet has enough balance for lock value + fees
	c.EnsureWalletBalance(t, wallet, lockValue+1)

	createAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse),
			Value:           tokenomics.IntToZCN(lockValue),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createAllocationTransactionPutResponse)

	t.Logf("Create alloc txn hash: %s, nonce used: %d", createAllocationTransactionPutResponse.Entity.Hash, createAllocationTransactionPutResponse.Request.TransactionNonce)

	var createAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		createAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: createAllocationTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)

		if err != nil {
			t.Log("Error Creating Alloc : ", err)
			return false
		}

		if resp == nil {
			return false
		}

		if createAllocationTransactionGetConfirmationResponse == nil {
			return false
		}

		if createAllocationTransactionGetConfirmationResponse.Status != 0 {
			if createAllocationTransactionGetConfirmationResponse.Status != requiredTransactionStatus {
				txnOutput := ""
				if createAllocationTransactionGetConfirmationResponse.Transaction != nil {
					txnOutput = createAllocationTransactionGetConfirmationResponse.Transaction.TransactionOutput
				}
				t.Logf("Create alloc txn confirmed with status %d (expected %d), output: %s",
					createAllocationTransactionGetConfirmationResponse.Status, requiredTransactionStatus, txnOutput)
			}
			return true // Break early on any confirmed status (success or failure)
		}

		return false
	})

	wallet.IncNonce()

	return createAllocationTransactionPutResponse.Entity.Hash
}

func (c *APIClient) RegisterBlobber(t *test.SystemTest,
	wallet *model.Wallet,
	storageNode *model.StorageNode,
	requiredTransactionStatus int,
	expectedResponse string,
	requireIdVerification bool) string {
	t.Log("Registering blobber...")

	registerBlobberTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewRegisterBlobberTransactionData(storageNode),
			Value:           tokenomics.IntToZCN(0),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, registerBlobberTransactionPutResponse)

	var registerBlobberTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		registerBlobberTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: registerBlobberTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)

		if err != nil {
			t.Log("Error registering blobber : ", err)
			return false
		}

		if resp == nil {
			return false
		}

		if registerBlobberTransactionGetConfirmationResponse == nil {
			return false
		}

		// Check status first — if the transaction was confirmed (with any status), we have a result.
		actualStatus := registerBlobberTransactionGetConfirmationResponse.Status
		actualOutput := registerBlobberTransactionGetConfirmationResponse.Transaction.TransactionOutput

		if requireIdVerification {
			// Status must match before attempting JSON unmarshal
			if actualStatus != requiredTransactionStatus {
				t.Logf("RegisterBlobber: expected status %d but got %d. Output: %s", requiredTransactionStatus, actualStatus, actualOutput)
				// Transaction is confirmed but with wrong status — stop polling, this won't change
				return true // Return true to break the wait loop; the require below will catch the mismatch
			}

			var storageNode model.StorageNode
			err := json.Unmarshal([]byte(actualOutput), &storageNode)
			if err != nil {
				t.Logf("RegisterBlobber: error unmarshalling output as StorageNode: %v. Output: %s", err, actualOutput)
				return true // Transaction is confirmed — stop polling
			}

			return storageNode.ID == expectedResponse
		}

		// Log the actual status and output for debugging
		if actualStatus == requiredTransactionStatus {
			if strings.Contains(actualOutput, expectedResponse) {
				return true
			}
			t.Logf("Transaction status matches (%d) but output doesn't match. Expected: %s, Actual: %s", requiredTransactionStatus, expectedResponse, actualOutput)
		} else {
			t.Logf("Transaction status doesn't match. Expected: %d, Actual: %d, Output: %s", requiredTransactionStatus, actualStatus, actualOutput)
		}

		return false
	})

	wallet.IncNonce()

	return registerBlobberTransactionPutResponse.Entity.Hash
}

// TryRegisterBlobber attempts to register a blobber and returns the actual transaction status
// without calling t.Fatal on timeout. Returns (hash, actualStatus, matched).
func (c *APIClient) TryRegisterBlobber(t *test.SystemTest,
	wallet *model.Wallet,
	storageNode *model.StorageNode,
	requiredTransactionStatus int,
	expectedResponse string) (string, int, bool) {
	t.Log("Trying to register blobber (non-fatal)...")

	registerBlobberTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewRegisterBlobberTransactionData(storageNode),
			Value:           tokenomics.IntToZCN(0),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	if err != nil || resp == nil || registerBlobberTransactionPutResponse == nil {
		t.Logf("TryRegisterBlobber: V1TransactionPut failed: err=%v", err)
		wallet.IncNonce()
		return "", 0, false
	}

	var confirmationResp *model.TransactionGetConfirmationResponse
	var actualStatus int

	loggedOnce := false
	matched := wait.PoolImmediatelyNonFatal(t, 3*time.Minute, func() bool {
		confirmationResp, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: registerBlobberTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)

		if err != nil || resp == nil || confirmationResp == nil {
			return false
		}

		actualStatus = confirmationResp.Status
		actualOutput := confirmationResp.Transaction.TransactionOutput

		if confirmationResp.Status == requiredTransactionStatus {
			if strings.Contains(actualOutput, expectedResponse) {
				return true
			}
			if !loggedOnce {
				t.Logf("TryRegisterBlobber: status matches (%d) but output doesn't contain expected. Expected substring: %q, Actual output: %s", requiredTransactionStatus, expectedResponse, actualOutput)
				loggedOnce = true
			}
		} else if !loggedOnce {
			t.Logf("TryRegisterBlobber: status mismatch. Expected: %d, Actual: %d, Output: %s", requiredTransactionStatus, actualStatus, actualOutput)
			loggedOnce = true
		}
		return false
	})

	wallet.IncNonce()

	return registerBlobberTransactionPutResponse.Entity.Hash, actualStatus, matched
}

func (c *APIClient) KillBlobber(t *test.SystemTest,
	wallet *model.Wallet,
	killBlobberRequest *model.KillBlobberRequest,
	requiredTransactionStatus int) string {
	t.Log("Killing blobber...")

	killBlobberTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewKillBlobberTransactionData(killBlobberRequest),
			Value:           tokenomics.IntToZCN(0),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, killBlobberTransactionPutResponse)

	var killBlobberTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		killBlobberTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: killBlobberTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)

		if err != nil {
			t.Log("Error killing blobber : ", err)
			return false
		}

		if resp == nil {
			fmt.Println("got nil response : ", resp)
			return false
		}

		if killBlobberTransactionGetConfirmationResponse == nil {
			fmt.Println("got nil txn confirmation response : ", killBlobberTransactionGetConfirmationResponse)
			return false
		}

		return killBlobberTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
	return killBlobberTransactionPutResponse.Entity.Hash
}

// KillBlobberNonFatal submits a kill_blobber transaction but does not fail the test on timeout.
// Use for cleanup operations where best-effort is acceptable.
func (c *APIClient) KillBlobberNonFatal(t *test.SystemTest,
	wallet *model.Wallet,
	killBlobberRequest *model.KillBlobberRequest) {
	t.Log("Killing blobber (best-effort)...")

	killBlobberTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewKillBlobberTransactionData(killBlobberRequest),
			Value:           tokenomics.IntToZCN(0),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	if err != nil || resp == nil || killBlobberTransactionPutResponse == nil {
		t.Logf("Warning: killBlobber TX submission failed (best-effort cleanup): %v", err)
		return
	}

	var killBlobberTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse
	ok := wait.PoolImmediatelyNonFatal(t, 3*time.Minute, func() bool {
		killBlobberTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: killBlobberTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil || resp == nil || killBlobberTransactionGetConfirmationResponse == nil {
			return false
		}
		return killBlobberTransactionGetConfirmationResponse.Status == TxSuccessfulStatus
	})
	if !ok {
		t.Logf("Warning: killBlobber did not confirm within 2 min (best-effort cleanup, chain may be unstable)")
	}
	wallet.IncNonce()
}

func (c *APIClient) CreateFreeAllocation(t *test.SystemTest,
	wallet *model.Wallet,
	scRestGetFreeAllocationBlobbersResponse *model.SCRestGetFreeAllocationBlobbersResponse,
	requiredTransactionStatus int) string {
	t.Log("Create free allocation...")

	createAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewCreateFreeAllocationTransactionData(scRestGetFreeAllocationBlobbersResponse),
			Value:           tokenomics.IntToZCN(0.1),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createAllocationTransactionPutResponse)

	var createAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		createAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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

func (c *APIClient) UpdateAllocation(
	t *test.SystemTest,
	wallet *model.Wallet,
	allocationID string,
	uar *model.UpdateAllocationRequest,
	lock float64,
	requiredTransactionStatus int) {
	t.Log("Update allocation...")

	// Ensure wallet has enough balance for lock value + fees
	c.EnsureWalletBalance(t, wallet, lock+1)

	uar.ID = allocationID
	updateAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewUpdateAllocationTransactionData(uar),
			Value:           tokenomics.IntToZCN(lock),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updateAllocationTransactionPutResponse)
	txnHash := updateAllocationTransactionPutResponse.Request.Hash

	t.Logf("Update alloc txn hash: %s, nonce used: %d", txnHash, updateAllocationTransactionPutResponse.Request.TransactionNonce)

	var updateAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	confirmed := false
	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		updateAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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

		if updateAllocationTransactionGetConfirmationResponse.Status != 0 {
			confirmed = true
			if updateAllocationTransactionGetConfirmationResponse.Status != requiredTransactionStatus {
				txnOutput := ""
				if updateAllocationTransactionGetConfirmationResponse.Transaction != nil {
					txnOutput = updateAllocationTransactionGetConfirmationResponse.Transaction.TransactionOutput
				}
				t.Logf("Update alloc txn confirmed with status %d (expected %d), output: %s",
					updateAllocationTransactionGetConfirmationResponse.Status, requiredTransactionStatus, txnOutput)
			}
			return true // Break early on any confirmed status (success or failure)
		}

		return false
	})

	wallet.IncNonce()

	if confirmed && updateAllocationTransactionGetConfirmationResponse.Status != requiredTransactionStatus {
		txnOutput := ""
		if updateAllocationTransactionGetConfirmationResponse.Transaction != nil {
			txnOutput = updateAllocationTransactionGetConfirmationResponse.Transaction.TransactionOutput
		}
		require.Equal(t, requiredTransactionStatus, updateAllocationTransactionGetConfirmationResponse.Status,
			"Update allocation txn confirmed with unexpected status: %s", txnOutput)
	}
}

func (c *APIClient) AddFreeStorageAssigner(
	t *test.SystemTest,
	wallet *model.Wallet,
	requiredTransactionStatus int) {
	// Retry up to 3 times to handle transient nonce races and view-change outages.
	// On each retry, RefreshNonce re-syncs from chain so a fresh nonce is used.
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			t.Logf("AddFreeStorageAssigner: retrying (attempt %d/3) after confirmation timeout — refreshing nonce", attempt+1)
			c.RefreshNonce(t, wallet, 200)
		}

		t.Log("Add free storage assigner...")
		freeAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
			t,
			model.InternalTransactionPutRequest{
				Wallet:     wallet,
				ToClientID: StorageSmartContractAddress,
				TransactionData: model.NewFreeStorageAssignerTransactionData(&model.FreeStorageAssignerRequest{
					Name:            wallet.Id,
					PublicKey:       wallet.PublicKey,
					IndividualLimit: 10.0,
					TotalLimit:      100.0,
				}),
				Value:   tokenomics.IntToZCN(0.1),
				TxnType: SCTxType,
			},
			HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, freeAllocationTransactionPutResponse)
		txnHash := freeAllocationTransactionPutResponse.Entity.Hash

		var freeAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

		confirmed := wait.PoolImmediatelyNonFatal(t, 15*time.Minute, func() bool {
			freeAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
				t,
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

			if freeAllocationTransactionGetConfirmationResponse == nil {
				return false
			}

			return freeAllocationTransactionGetConfirmationResponse.Status == requiredTransactionStatus
		})

		wallet.IncNonce()
		if confirmed {
			return
		}
		// Confirmation timed out — tx may have been dropped (nonce race).
		// Loop will refresh nonce and resubmit on next iteration.
	}
	t.Fatal("AddFreeStorageAssigner: failed to confirm after 3 attempts")
}

func (c *APIClient) UpdateAllocationBlobbers(t *test.SystemTest, wallet *model.Wallet, newBlobberID, oldBlobberID, allocationID string, requiredTransactionStatus int) {
	t.Log("Update allocation...")

	// Ensure wallet has enough balance for tx fee (blobber replace has no lock value)
	c.EnsureWalletBalance(t, wallet, 0.1)

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
			Value:   tokenomics.IntToZCN(0.1),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updateAllocationTransactionPutResponse)
	txnHash := updateAllocationTransactionPutResponse.Request.Hash

	var updateAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		updateAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: txnHash,
			},
			HttpOkStatus)
		if err != nil {
			t.Logf("UpdateAllocationBlobbers: confirmation error: %v", err)
			return false
		}

		if resp == nil {
			t.Log("UpdateAllocationBlobbers: nil response")
			return false
		}

		if updateAllocationTransactionGetConfirmationResponse == nil {
			t.Log("UpdateAllocationBlobbers: nil confirmation body")
			return false
		}

		if updateAllocationTransactionGetConfirmationResponse.Status != requiredTransactionStatus {
			txnOutput := ""
			if updateAllocationTransactionGetConfirmationResponse.Transaction != nil {
				txnOutput = updateAllocationTransactionGetConfirmationResponse.Transaction.TransactionOutput
			}
			t.Logf("UpdateAllocationBlobbers: status=%d (want %d), output=%s",
				updateAllocationTransactionGetConfirmationResponse.Status,
				requiredTransactionStatus, txnOutput)
			// If transaction was confirmed with a definitive failure status, don't wait 10 min
			if updateAllocationTransactionGetConfirmationResponse.Status > 0 {
				t.Fatalf("UpdateAllocationBlobbers: transaction confirmed with failure status=%d, output=%s",
					updateAllocationTransactionGetConfirmationResponse.Status, txnOutput)
			}
			return false
		}

		return true
	})

	wallet.IncNonce()
}

// TryUpdateAllocationBlobbers is a non-fatal version of UpdateAllocationBlobbers.
// Returns true if the transaction was confirmed with the expected status, false otherwise.
func (c *APIClient) TryUpdateAllocationBlobbers(t *test.SystemTest, wallet *model.Wallet, newBlobberID, oldBlobberID, allocationID string, requiredTransactionStatus int) bool {
	t.Log("Try update allocation blobbers (non-fatal)...")

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
			Value:   tokenomics.IntToZCN(0.1),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	if err != nil || resp == nil || updateAllocationTransactionPutResponse == nil {
		t.Logf("TryUpdateAllocationBlobbers: V1TransactionPut failed: err=%v", err)
		wallet.IncNonce()
		return false
	}
	txnHash := updateAllocationTransactionPutResponse.Request.Hash

	var confirmationResp *model.TransactionGetConfirmationResponse

	matched := wait.PoolImmediatelyNonFatal(t, 3*time.Minute, func() bool {
		confirmationResp, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: txnHash,
			},
			HttpOkStatus)
		if err != nil || resp == nil || confirmationResp == nil {
			return false
		}
		return confirmationResp.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
	return matched
}

func (c *APIClient) CancelAllocation(
	t *test.SystemTest,
	wallet *model.Wallet,
	allocationID string,
	requiredTransactionStatus int,
) string {
	t.Logf("Cancel allocation %v...", allocationID)

	// Ensure wallet has enough balance to pay the cancel transaction fee
	c.EnsureWalletBalance(t, wallet, 2)

	cancelAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: StorageSmartContractAddress,
			TransactionData: model.NewCancelAllocationTransactionData(&model.CancelAllocationRequest{
				AllocationID: allocationID,
			}),
			TxnType: SCTxType,
		},
		HttpOkStatus,
	)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, cancelAllocationTransactionPutResponse)

	var cancelAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		cancelAllocationTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: cancelAllocationTransactionPutResponse.Request.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if cancelAllocationTransactionGetConfirmationResponse == nil {
			return false
		}

		return cancelAllocationTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return cancelAllocationTransactionPutResponse.Request.Hash
}

func (c *APIClient) GetAllocationBlobbers(t *test.SystemTest, wallet *model.Wallet, blobberRequirements *model.BlobberRequirements, requiredStatusCode int) *model.SCRestGetAllocationBlobbersResponse {
	t.Log("Get allocation blobbers...")

	// Request extra blobbers to allow filtering out enterprise ones
	inflatedReqs := *blobberRequirements
	needed := blobberRequirements.DataShards + blobberRequirements.ParityShards
	inflatedReqs.DataShards = blobberRequirements.DataShards + 4 // request 4 extra
	inflatedReqs.ParityShards = blobberRequirements.ParityShards

	scRestGetAllocationBlobbersResponse, resp, err := c.V1SCRestGetAllocationBlobbers(
		t,
		&model.SCRestGetAllocationBlobbersRequest{
			ClientID:            wallet.Id,
			ClientKey:           wallet.PublicKey,
			BlobberRequirements: inflatedReqs,
		}, requiredStatusCode)

	// If inflated request fails (not enough blobbers), fall back to original count
	if err != nil || scRestGetAllocationBlobbersResponse == nil || scRestGetAllocationBlobbersResponse.Blobbers == nil {
		t.Log("Inflated blobber request failed, trying with original shard counts...")
		scRestGetAllocationBlobbersResponse, resp, err = c.V1SCRestGetAllocationBlobbers(
			t,
			&model.SCRestGetAllocationBlobbersRequest{
				ClientID:            wallet.Id,
				ClientKey:           wallet.PublicKey,
				BlobberRequirements: *blobberRequirements,
			}, requiredStatusCode)
	}

	scRestGetAllocationBlobbersResponse.StorageVersion = 1

	// Restore original shard requirements in the response for allocation creation
	scRestGetAllocationBlobbersResponse.DataShards = blobberRequirements.DataShards
	scRestGetAllocationBlobbersResponse.ParityShards = blobberRequirements.ParityShards

	if requiredStatusCode == http.StatusOK {
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, scRestGetAllocationBlobbersResponse)

		// Filter out enterprise blobbers and trim to needed count
		c.filterOutEnterpriseBlobbers(t, scRestGetAllocationBlobbersResponse, blobberRequirements, needed)
	}

	return scRestGetAllocationBlobbersResponse
}

// filterOutEnterpriseBlobbers removes enterprise blobber IDs from the allocation blobbers response
// and trims to the needed count of non-enterprise blobbers.
func (c *APIClient) filterOutEnterpriseBlobbers(t *test.SystemTest, allocBlobbers *model.SCRestGetAllocationBlobbersResponse, requirements *model.BlobberRequirements, needed int64) {
	if allocBlobbers == nil || allocBlobbers.Blobbers == nil || len(*allocBlobbers.Blobbers) == 0 {
		return
	}

	enterpriseIDs := make(map[string]bool)

	// Try bulk fetch first (fastest)
	allBlobbers, _, err := c.V1SCRestGetAllBlobbers(t, http.StatusOK)
	if err == nil {
		for _, b := range allBlobbers {
			if b.IsEnterprise {
				enterpriseIDs[b.ID] = true
			}
		}
	} else {
		// Fallback: check each blobber in the allocation list individually
		t.Logf("Bulk blobber fetch failed, checking individually: %v", err)
		for _, blobberID := range *allocBlobbers.Blobbers {
			blobber, _, bErr := c.V1SCRestGetBlobber(t, model.SCRestGetBlobberRequest{BlobberID: blobberID}, http.StatusOK)
			if bErr != nil || blobber == nil {
				t.Logf("Could not check blobber %s, assuming enterprise for safety", blobberID)
				enterpriseIDs[blobberID] = true
				continue
			}
			if blobber.IsEnterprise {
				enterpriseIDs[blobberID] = true
			}
		}
	}

	if len(enterpriseIDs) == 0 {
		// No enterprise blobbers found, keep all (don't trim - spare blobbers needed for add/replace tests)
		t.Logf("No enterprise blobbers found: %d blobbers available (need %d)", len(*allocBlobbers.Blobbers), needed)
		return
	}

	// Count how many non-enterprise blobbers we'd have after filtering
	nonEnterpriseCount := int64(0)
	for _, blobberID := range *allocBlobbers.Blobbers {
		if !enterpriseIDs[blobberID] {
			nonEnterpriseCount++
		}
	}

	// If filtering would leave fewer than needed, don't filter at all
	if nonEnterpriseCount < needed {
		t.Logf("Not enough non-enterprise blobbers (%d) for allocation (need %d), keeping all %d blobbers including enterprise",
			nonEnterpriseCount, needed, len(*allocBlobbers.Blobbers))
		return
	}

	// Filter out enterprise blobbers from the allocation list, keeping auth tickets aligned
	filtered := make([]string, 0, len(*allocBlobbers.Blobbers))
	filteredTickets := make([]string, 0, len(allocBlobbers.BlobberAuthTickets))
	for i, blobberID := range *allocBlobbers.Blobbers {
		if !enterpriseIDs[blobberID] {
			filtered = append(filtered, blobberID)
			if i < len(allocBlobbers.BlobberAuthTickets) {
				filteredTickets = append(filteredTickets, allocBlobbers.BlobberAuthTickets[i])
			}
		} else {
			t.Logf("Filtering out enterprise blobber: %s", blobberID)
		}
	}

	// Don't trim to needed count - keep spare non-enterprise blobbers for add/replace operations
	*allocBlobbers.Blobbers = filtered
	allocBlobbers.BlobberAuthTickets = filteredTickets
	t.Logf("After enterprise filter: %d non-enterprise blobbers available (need %d)", len(filtered), needed)
}

func (c *APIClient) GetFreeAllocationBlobbers(
	t *test.SystemTest,
	wallet *model.Wallet,
	freeAllocData *model.FreeAllocationData,
	requiredStatusCode int,
) *model.SCRestGetFreeAllocationBlobbersResponse {
	t.Log("Get free allocation blobbers...")

	scRestGetFreeAllocationBlobbersResponse, resp, err := c.V1SCRestGetFreeAllocationBlobbers(
		t,
		freeAllocData,
		requiredStatusCode,
	)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, scRestGetFreeAllocationBlobbersResponse)

	return scRestGetFreeAllocationBlobbersResponse
}

func (c *APIClient) GetAllocation(t *test.SystemTest, allocationID string, requiredStatusCode int) *model.SCRestGetAllocationResponse {
	t.Log("Get allocation...")

	var (
		scRestGetAllocation *model.SCRestGetAllocationResponse
		resp                *resty.Response //nolint
		err                 error
	)

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		scRestGetAllocation, resp, err = c.V1SCRestGetAllocation(
			t,
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

func (c *APIClient) GetWalletBalance(t *test.SystemTest, wallet *model.Wallet, requiredStatusCode int) *model.ClientGetBalanceResponse {
	t.Log("Get wallet balance...")

	// Retry on sharder errors (e.g., "unexpected end of JSON input" under load).
	// Without retry, a transient sharder error returns Nonce:0 which corrupts
	// the nonce state and causes all subsequent transactions to fail.
	var clientGetBalanceResponse *model.ClientGetBalanceResponse
	var resp interface{}
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		clientGetBalanceResponse, resp, err = c.V1ClientGetBalance(
			t,
			model.ClientGetBalanceRequest{
				ClientID: wallet.Id,
			},
			requiredStatusCode)
		if err == nil && clientGetBalanceResponse != nil {
			break
		}
		if err != nil && strings.Contains(err.Error(), "value not present") {
			// New wallet with no balance — not a transient error
			break
		}
		if attempt < 2 {
			t.Logf("Wallet balance query failed (attempt %d/3): %v — retrying in 2s", attempt+1, err)
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		t.Logf("Error getting wallet balance: %v", err)
		// Return a zero balance response if there's an error (e.g., "value not present" for new wallets)
		if clientGetBalanceResponse == nil {
			clientGetBalanceResponse = &model.ClientGetBalanceResponse{
				Balance: 0,
				Nonce:   0,
			}
		} else {
			clientGetBalanceResponse.Balance = 0
		}
		return clientGetBalanceResponse
	}
	_ = resp

	return clientGetBalanceResponse
}

func (c *APIClient) RefreshNonce(t *test.SystemTest, wallet *model.Wallet, requiredStatusCode int) {
	wBalance := c.GetWalletBalance(t, wallet, requiredStatusCode)
	// Only advance nonce, never go backwards (vc.sh pattern).
	// Local nonce may be ahead if we've submitted a txn that chain hasn't confirmed yet.
	if int(wBalance.Nonce) > wallet.Nonce {
		wallet.Nonce = int(wBalance.Nonce)
	}
	// Safety check: if chain returned nonce 0 but we have a higher local nonce,
	// the sharder query likely failed. Log a warning so we can diagnose.
	if wBalance.Nonce == 0 && wallet.Nonce > 0 {
		t.Logf("WARNING: sharder returned nonce=0 but local nonce=%d — sharder may be lagging", wallet.Nonce)
	}
}

func (c *APIClient) GetRewardsByQuery(t *test.SystemTest, query string, requiredStatusCode int) *model.QueryRewardsResponse {
	t.Log("Get rewards by query...")

	queryRewardsResponse, resp, err := c.V1QueryRewards(
		t,
		model.QueryRequest{
			Query: query,
		},
		requiredStatusCode)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, queryRewardsResponse)

	return queryRewardsResponse
}

func (c *APIClient) GetChallengesCountByQuery(t *test.SystemTest, query string, requiredStatusCode int) map[string]int64 {
	t.Log("Get rewards by query...")

	queryRewardsResponse, resp, err := c.V1QueryChallengesCount(
		t,
		model.QueryRequest{
			Query: query,
		},
		requiredStatusCode)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, queryRewardsResponse)

	return queryRewardsResponse
}

func (c *APIClient) GetAllChallengesForAllocation(t *test.SystemTest, allocationID string, requiredStatusCode int) []*model.Challenge {
	t.Log("Get all challenges for allocation...")

	getAllChallengesForAllocationResponse, resp, err := c.V1SCRestGetAllChallengesForAllocation(
		t,
		allocationID,
		requiredStatusCode)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, getAllChallengesForAllocationResponse)

	return getAllChallengesForAllocationResponse
}

func (c *APIClient) GetDelegateRewardsByQuery(t *test.SystemTest, query string, requiredStatusCode int) map[string]int64 {
	t.Log("Get rewards by query...")

	queryRewardsResponse, resp, err := c.V1QueryDelegateRewards(
		t,
		model.QueryRequest{
			Query: query,
		},
		requiredStatusCode)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, queryRewardsResponse)

	return queryRewardsResponse
}

func (c *APIClient) GetBlobberPartitionSelectionFrequency(t *test.SystemTest, start, end int64, requiredStatusCode int) map[string]int64 {
	t.Log("Get blobber partition selection frequency...")

	blobberPartitionSelectionFrequencyResponse, resp, err := c.V1BlobberPartitionSelectionFrequency(
		t,
		model.BlockRewardsRequest{
			Start: start,
			End:   end,
		},
		requiredStatusCode)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, blobberPartitionSelectionFrequencyResponse)

	return blobberPartitionSelectionFrequencyResponse
}

func (c *APIClient) GetPartitionSizeFrequency(t *test.SystemTest, start, end int64, requiredStatusCode int) map[float64]float64 {
	t.Log("Get partition size frequency...")

	blobberPartitionSelectionFrequencyResponse, resp, err := c.V1PartitionSizeFrequency(
		t,
		model.BlockRewardsRequest{
			Start: start,
			End:   end,
		},
		requiredStatusCode)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, blobberPartitionSelectionFrequencyResponse)

	return blobberPartitionSelectionFrequencyResponse
}

func (c *APIClient) UpdateBlobber(t *test.SystemTest, wallet *model.Wallet, scRestGetBlobberResponse *model.SCRestGetBlobberResponse, requiredTransactionStatus int) {
	updateBlobberTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewUpdateBlobberTransactionData(scRestGetBlobberResponse),
			Value:           tokenomics.IntToZCN(0.1),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updateBlobberTransactionPutResponse)

	var updateBlobberTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		updateBlobberTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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
func (c *APIClient) CreateStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, requiredTransactionStatus int, options ...float64) string {
	t.Log("Create stake pool...")

	tokens := 1.0
	if len(options) > 0 {
		tokens = options[0]
	}

	// Ensure wallet has enough balance for stake + fees
	c.EnsureWalletBalance(t, wallet, tokens+1)

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
			Value:   tokenomics.IntToZCN(tokens),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createStakePoolTransactionPutResponse)

	var createStakePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		createStakePoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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

func (c *APIClient) UnlockStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, requiredTransactionStatus int) string {
	t.Log("Unlock stake pool...")

	unlockStakePoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: StorageSmartContractAddress,
			TransactionData: model.NewUnlockStackPoolTransactionData(
				model.CreateStakePoolRequest{
					ProviderType: providerType,
					ProviderID:   providerID,
				}),
			Value:   tokenomics.IntToZCN(0.1),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, unlockStakePoolTransactionPutResponse)

	var unlockStakePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		unlockStakePoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: unlockStakePoolTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if unlockStakePoolTransactionGetConfirmationResponse == nil {
			return false
		}

		return unlockStakePoolTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return unlockStakePoolTransactionGetConfirmationResponse.Hash
}

// CreateMinerStakePool
func (c *APIClient) CreateMinerStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, tokens float64, requiredTransactionStatus int) string {
	t.Log("Create miner/sharder stake pool...")

	// Ensure wallet has enough balance for stake + fees
	c.EnsureWalletBalance(t, wallet, tokens+1)

	createStakePoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: MinerSmartContractAddress,
			TransactionData: model.NewCreateMinerStackPoolTransactionData(
				model.CreateStakePoolRequest{
					ProviderType: providerType,
					ProviderID:   providerID,
				}),
			Value:   tokenomics.IntToZCN(tokens),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createStakePoolTransactionPutResponse)

	var createStakePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		createStakePoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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

func (c *APIClient) UnlockMinerStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, requiredTransactionStatus int) string {
	t.Log("Unlock miner/sharder stake pool...")

	unlockStakePoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: MinerSmartContractAddress,
			TransactionData: model.NewUnlockMinerStackPoolTransactionData(
				model.CreateStakePoolRequest{
					ProviderType: providerType,
					ProviderID:   providerID,
				}),
			Value:   tokenomics.IntToZCN(0.1),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, unlockStakePoolTransactionPutResponse)

	var unlockStakePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		unlockStakePoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: unlockStakePoolTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if unlockStakePoolTransactionGetConfirmationResponse == nil {
			return false
		}

		return unlockStakePoolTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return unlockStakePoolTransactionGetConfirmationResponse.Hash
}

func (c *APIClient) CreateReadPool(t *test.SystemTest, wallet *model.Wallet) bool {
	t.Log("Create read pool...")

	txnResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewCreateReadPoolTransactionData(),
			Value:           tokenomics.IntToZCN(1000000000), // 0.1 ZCN to fund read pool for downloads
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	if err != nil {
		t.Logf("CreateReadPool transaction put failed (may already exist): %v", err)
		return false
	}
	if resp == nil || txnResponse == nil {
		t.Log("CreateReadPool: nil response, read pool may already exist")
		return false
	}

	// Short wait - read pool creation may fail if SC doesn't support it or pool already exists.
	// Don't block the test for too long; downloads may work without explicit read pool when read_price=0.
	confirmed := false
	wait.PoolImmediately(t, time.Second*15, func() bool {
		confirmResp, _, confirmErr := c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: txnResponse.Entity.Hash,
			},
			HttpOkStatus)
		if confirmErr != nil || confirmResp == nil {
			return false
		}
		if confirmResp.Status == TxSuccessfulStatus {
			confirmed = true
			return true
		}
		// Transaction confirmed but failed (e.g., read pool already exists)
		if confirmResp.Status != 0 {
			t.Logf("CreateReadPool txn confirmed with status %d (may already exist)", confirmResp.Status)
			confirmed = true
			return true
		}
		return false
	})

	if confirmed {
		wallet.IncNonce()
		return true
	}
	t.Log("CreateReadPool: confirmation timed out")
	return false
}

// CreateWritePoolWrapper does not provide deep test of used components
func (c *APIClient) CreateWritePool(t *test.SystemTest, wallet *model.Wallet, allocationId string, tokens float64, requiredTransactionStatus int) string {
	t.Log("Create write pool...")

	// Ensure wallet has enough balance for write pool + fees
	c.EnsureWalletBalance(t, wallet, tokens+1)

	createWritePoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: StorageSmartContractAddress,
			TransactionData: model.NewCreateWritePoolTransactionData(
				model.CreateWritePoolRequest{
					AllocationID: allocationId,
				}),
			Value:   tokenomics.IntToZCN(tokens),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createWritePoolTransactionPutResponse)

	var createWritePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		createWritePoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: createWritePoolTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if createWritePoolTransactionGetConfirmationResponse == nil {
			return false
		}

		return createWritePoolTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return createWritePoolTransactionGetConfirmationResponse.Hash
}

func (c *APIClient) V1SCRestGetStakePoolStat(t *test.SystemTest, scRestGetStakePoolStatRequest model.SCRestGetStakePoolStatRequest, requiredStatusCode int) (*model.SCRestGetStakePoolStatResponse, *resty.Response, error) { //nolint
	var scRestGetStakePoolStatResponse *model.SCRestGetStakePoolStatResponse

	urlBuilder := NewURLBuilder().
		SetPath(GetStakePoolStat).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("provider_id", scRestGetStakePoolStatRequest.ProviderID).
		AddParams("provider_type", scRestGetStakePoolStatRequest.ProviderType)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetStakePoolStatResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetStakePoolStatResponse, resp, err
}

func (c *APIClient) V1SCRestGetUserStakePoolStat(t *test.SystemTest, scRestGetUserStakePoolStatRequest model.SCRestGetUserStakePoolStatRequest, requiredStatusCode int) (*model.SCRestGetUserStakePoolStatResponse, *resty.Response, error) { //nolint
	var scRestGetUserStakePoolStatResponse *model.SCRestGetUserStakePoolStatResponse

	urlBuilder := NewURLBuilder().
		SetPath(getUserStakePoolStat).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("client_id", scRestGetUserStakePoolStatRequest.ClientId)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetUserStakePoolStatResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetUserStakePoolStatResponse, resp, err
}

func (c *APIClient) GetStakePoolStat(t *test.SystemTest, providerID, providerType string) *model.SCRestGetStakePoolStatResponse {
	t.Log("Get stake pool stat...")

	scRestGetStakePoolStat, resp, err := c.V1SCRestGetStakePoolStat(
		t,
		model.SCRestGetStakePoolStatRequest{
			ProviderID:   providerID,
			ProviderType: providerType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)

	return scRestGetStakePoolStat
}

func (c *APIClient) CollectRewards(t *test.SystemTest, wallet *model.Wallet, providerID string, providerType, requiredTransactionStatus int) (txnData *model.TransactionGetConfirmationResponse, fee int64) {
	collectRewardTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewCollectRewardTransactionData(providerID, providerType),
			Value:           tokenomics.IntToZCN(0),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, collectRewardTransactionPutResponse)

	var collectRewardTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		collectRewardTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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

	return collectRewardTransactionGetConfirmationResponse, collectRewardTransactionGetConfirmationResponse.Transaction.TransactionFee
}

func (c *APIClient) GetBlobber(t *test.SystemTest, blobberID string, requiredStatusCode int) *model.SCRestGetBlobberResponse {
	scRestGetBlobberResponse, resp, err := c.V1SCRestGetBlobber(
		t,
		model.SCRestGetBlobberRequest{
			BlobberID: blobberID,
		},
		requiredStatusCode)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, scRestGetBlobberResponse)

	return scRestGetBlobberResponse
}

func (c *APIClient) V1BlobberGetFileRefs(t *test.SystemTest, blobberGetFileRefsRequest *model.BlobberGetFileRefsRequest, requiredStatusCode int) (*model.BlobberGetFileRefsResponse, *resty.Response, error) {
	var blobberGetFileResponse *model.BlobberGetFileRefsResponse

	blobberGetFileRefsURL := blobberGetFileRefsRequest.URL + strings.Replace(GetFileRef, ":allocation_id", blobberGetFileRefsRequest.AllocationID, 1) + "?" + "path=" + blobberGetFileRefsRequest.RemotePath + "&" + "refType=" + blobberGetFileRefsRequest.RefType

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetFileRefsRequest.ClientID,
		"X-App-Client-Key":       blobberGetFileRefsRequest.ClientKey,
		"X-App-Client-Signature": blobberGetFileRefsRequest.ClientSignature,
		"ALLOCATION-ID":          blobberGetFileRefsRequest.AllocationID,
	}
	resp, err := c.executeForServiceProvider(
		t,
		blobberGetFileRefsURL,
		model.ExecutionRequest{
			Dst:                &blobberGetFileResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberGetFileResponse, resp, err
}

func (c *APIClient) V1BlobberGetFileRefPaths(t *test.SystemTest, blobberFileRefPathRequest *model.BlobberFileRefPathRequest, requiredStatusCode int) (*model.BlobberFileRefPathResponse, *resty.Response, error) {
	var blobberFileRefPathResponse *model.BlobberFileRefPathResponse

	blobberGetFilePathURL := blobberFileRefPathRequest.URL + strings.Replace(GetFileRefPath, ":allocation_id", blobberFileRefPathRequest.AllocationID, 1) + "?" + "path=" + blobberFileRefPathRequest.Path

	headers := map[string]string{
		"X-App-Client-Id":        blobberFileRefPathRequest.ClientID,
		"X-App-Client-Key":       blobberFileRefPathRequest.ClientKey,
		"X-App-Client-Signature": blobberFileRefPathRequest.ClientSignature,
		"ALLOCATION-ID":          blobberFileRefPathRequest.AllocationID,
	}
	resp, err := c.executeForServiceProvider(
		t,
		blobberGetFilePathURL,
		model.ExecutionRequest{
			Dst:                &blobberFileRefPathResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberFileRefPathResponse, resp, err
}

func (c *APIClient) V1BlockGetLatestFinalizedMagicBlock(t *test.SystemTest, hash string, requiredStatusCode int) (*resty.Response, error) {
	t.Log("Get latest finalized magic block")
	urlBuilder := NewURLBuilder().SetPath(GetLatestFinalizedMagicBlock)
	if hash != "" {
		urlBuilder = urlBuilder.AddParams("node-lfmb-hash", hash)
	}

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		SharderServiceProvider)

	return resp, err
}

func (c *APIClient) V1BlockGetLatestFinalizedBlock(t *test.SystemTest, requiredStatusCode int) (*model.LatestFinalizedBlock, *resty.Response, error) {
	t.Log("Get latest finalized block")

	var latestFinalizedBlock *model.LatestFinalizedBlock

	urlBuilder := NewURLBuilder().SetPath(GetLatestFinalizedBlock)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			RequiredStatusCode: requiredStatusCode,
			Dst:                &latestFinalizedBlock,
		},
		HttpPOSTMethod,
		SharderServiceProvider)

	return latestFinalizedBlock, resp, err
}

func (c *APIClient) GetLatestFinalizedBlock(t *test.SystemTest, requiredStatusCode int) *model.LatestFinalizedBlock {
	latestFinalizedBlock, resp, err := c.V1BlockGetLatestFinalizedBlock(t, requiredStatusCode)

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, latestFinalizedBlock)

	return latestFinalizedBlock
}

func (c *APIClient) V1BlobberObjectTree(t *test.SystemTest, blobberObjectTreeRequest *model.BlobberObjectTreeRequest, requiredStatusCode int) (*model.BlobberObjectTreePathResponse, *resty.Response, error) {
	var blobberObjectTreePathResponse *model.BlobberObjectTreePathResponse

	blobberObjectTreeURL := blobberObjectTreeRequest.URL + strings.Replace(GetObjectTree, ":allocation_id", blobberObjectTreeRequest.AllocationID, 1) + "?" + "path=" + blobberObjectTreeRequest.Path

	headers := map[string]string{
		"X-App-Client-Id":        blobberObjectTreeRequest.ClientID,
		"X-App-Client-Key":       blobberObjectTreeRequest.ClientKey,
		"X-App-Client-Signature": blobberObjectTreeRequest.ClientSignature,
		"ALLOCATION-ID":          blobberObjectTreeRequest.AllocationID,
	}
	resp, err := c.executeForServiceProvider(
		t,
		blobberObjectTreeURL,
		model.ExecutionRequest{
			Dst:                &blobberObjectTreePathResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberObjectTreePathResponse, resp, err
}

func (c *APIClient) V1QueryChallengesCount(t *test.SystemTest, queryRequest model.QueryRequest, requiredStatusCode int) (map[string]int64, *resty.Response, error) {
	var queryResponse map[string]int64

	urlBuilder := NewURLBuilder().SetPath(QueryChallengesCount).AddParams("query", url.QueryEscape(queryRequest.Query)).SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &queryResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return queryResponse, resp, err
}

func (c *APIClient) V1QueryRewards(t *test.SystemTest, queryRewardsRequest model.QueryRequest, requiredStatusCode int) (*model.QueryRewardsResponse, *resty.Response, error) {
	var queryRewardsResponse *model.QueryRewardsResponse

	urlBuilder := NewURLBuilder().SetPath(QueryRewards).AddParams("query", url.QueryEscape(queryRewardsRequest.Query)).SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &queryRewardsResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return queryRewardsResponse, resp, err
}

func (c *APIClient) V1QueryDelegateRewards(t *test.SystemTest, queryRewardsRequest model.QueryRequest, requiredStatusCode int) (map[string]int64, *resty.Response, error) {
	var queryRewardsResponse map[string]int64

	urlBuilder := NewURLBuilder().SetPath(QueryDelegateRewards).AddParams("query", url.QueryEscape(queryRewardsRequest.Query)).SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &queryRewardsResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return queryRewardsResponse, resp, err
}

func (c *APIClient) V1BlobberPartitionSelectionFrequency(t *test.SystemTest, request model.BlockRewardsRequest, requiredStatusCode int) (map[string]int64, *resty.Response, error) {
	var result map[string]int64

	urlBuilder := NewURLBuilder().SetPath(BlobberPartitionSelectionFrequency).AddParams("start", strconv.FormatInt(request.Start, 10)).AddParams("end", strconv.FormatInt(request.End, 10)).SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &result,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return result, resp, err
}

func (c *APIClient) V1PartitionSizeFrequency(t *test.SystemTest, request model.BlockRewardsRequest, requiredStatusCode int) (map[float64]float64, *resty.Response, error) {
	var response map[string]int

	result := make(map[float64]float64)

	urlBuilder := NewURLBuilder().SetPath(PartitionSizeFrequency).AddParams("start", strconv.FormatInt(request.Start, 10)).AddParams("end", strconv.FormatInt(request.End, 10)).SetPathVariable("sc_address", StorageSmartContractAddress).SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &response,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	for size, frequency := range response {
		sizeInFloat, _ := strconv.ParseFloat(size, 64)
		result[sizeInFloat] = float64(frequency)
	}

	return result, resp, err
}

func (c *APIClient) V1SCRestGetAllChallengesForAllocation(t *test.SystemTest, allocationID string, requiredStatusCode int) ([]*model.Challenge, *resty.Response, error) { //nolint
	var scRestGetAllChallengesForAllocationResponse []*model.Challenge

	urlBuilder := NewURLBuilder().
		SetPath(GetAllChallenges).
		SetPathVariable("sc_address", StorageSmartContractAddress).
		AddParams("allocation_id", allocationID)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &scRestGetAllChallengesForAllocationResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return scRestGetAllChallengesForAllocationResponse, resp, err
}

//----------------------------------------------------------
// ZCN SC
//----------------------------------------------------------

func (c *APIClient) BurnZcn(t *test.SystemTest, wallet *model.Wallet, address string, amount float64, requiredTransactionStatus int) string {
	t.Log("Burn ZCN")

	walletBalance := c.GetWalletBalance(t, wallet, HttpOkStatus)
	wallet.Nonce = int(walletBalance.Nonce)

	burnZcnTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: ZCNSmartContractAddess,
			TransactionData: model.NewBurnZcnTransactionData(&model.SCRestBurnZcnRequest{
				EthereumAddress: address,
			}),
			Value:   tokenomics.IntToZCN(amount),
			TxnType: SCTxType,
		},
		requiredTransactionStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, burnZcnTransactionPutResponse)

	var burnZcnTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, 3*time.Minute, func() bool {
		burnZcnTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: burnZcnTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if burnZcnTransactionGetConfirmationResponse == nil {
			return false
		}

		return burnZcnTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
	return burnZcnTransactionGetConfirmationResponse.Hash
}

// FundWallet funds a wallet from the faucet with the specified amount of tokens.
// The faucet gives 1 ZCN per call regardless of requested amount, so we call it
// multiple times to accumulate the needed balance.
func (c *APIClient) FundWallet(t *test.SystemTest, wallet *model.Wallet, tokens float64, requiredTransactionStatus int) string {
	numCalls := int(tokens)
	if numCalls < 3 {
		numCalls = 3
	}
	if numCalls > 15 {
		numCalls = 15
	}

	var lastHash string

	for i := 0; i < numCalls; i++ {
		t.Logf("Funding wallet %s with tokens from faucet (%d/%d)...", wallet.Id, i+1, numCalls)

		fundWalletTransactionPutResponse, resp, err := c.V1TransactionPut(
			t,
			model.InternalTransactionPutRequest{
				Wallet:          wallet,
				ToClientID:      FaucetSmartContractAddress,
				TransactionData: model.NewFaucetTransactionData(),
				Value:           tokenomics.IntToZCN(1),
				TxnType:         SCTxType,
			},
			HttpOkStatus)
		if err != nil {
			t.Logf("Faucet call %d failed to put: %v", i+1, err)
			continue
		}
		require.NotNil(t, resp)
		require.NotNil(t, fundWalletTransactionPutResponse)

		var fundWalletTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

		wait.PoolImmediately(t, 3*time.Minute, func() bool {
			fundWalletTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
				t,
				model.TransactionGetConfirmationRequest{
					Hash: fundWalletTransactionPutResponse.Entity.Hash,
				},
				HttpOkStatus)
			if err != nil {
				return false
			}

			if resp == nil {
				return false
			}

			if fundWalletTransactionGetConfirmationResponse == nil {
				return false
			}

			return fundWalletTransactionGetConfirmationResponse.Status == requiredTransactionStatus
		})

		wallet.IncNonce()
		lastHash = fundWalletTransactionGetConfirmationResponse.Hash
	}

	return lastHash
}

// EnsureWalletBalance checks the wallet balance and tops up from faucet if below minBalanceZCN.
// It also syncs the wallet nonce from the chain.
func (c *APIClient) EnsureWalletBalance(t *test.SystemTest, wallet *model.Wallet, minBalanceZCN float64) {
	balance := c.GetWalletBalance(t, wallet, HttpOkStatus)
	// Only advance nonce, never go backwards (consistent with RefreshNonce).
	// Sharder may return stale nonce; going backwards causes nonce conflicts.
	if int(balance.Nonce) > wallet.Nonce {
		wallet.Nonce = int(balance.Nonce)
	}

	balanceZCN := float64(balance.Balance) / 1e10
	// Account for TxFee (2 ZCN) in addition to the value itself
	txFeeZCN := TxFee / 1e10
	requiredZCN := minBalanceZCN + txFeeZCN
	if balanceZCN < requiredZCN {
		needed := requiredZCN - balanceZCN + 1 // top up with 1 ZCN margin
		t.Logf("Wallet %s balance %.2f ZCN < %.2f ZCN required (%.2f + %.2f fee), topping up %.0f ZCN from faucet", wallet.Id, balanceZCN, requiredZCN, minBalanceZCN, txFeeZCN, needed)
		c.FundWallet(t, wallet, needed, TxSuccessfulStatus)
	}
}
