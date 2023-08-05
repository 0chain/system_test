package client

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
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

type ChimneyClient struct {
	BaseHttpClient
	model.HealthyServiceProviders
}

func NewChimneyClient(networkEntrypoint string) *ChimneyClient {
	chimneyClient := &ChimneyClient{}
	chimneyClient.HttpClient = resty.New()

	if err := chimneyClient.selectHealthyServiceProviders(networkEntrypoint); err != nil {
		log.Fatalln(err)
	}

	return chimneyClient
}

func (c *ChimneyClient) getHealthyNodes(nodes []string, serviceProviderType int) ([]string, error) {
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

func (c *ChimneyClient) getHealthyMiners(miners []string) ([]string, error) {
	return c.getHealthyNodes(miners, MinerServiceProvider)
}

func (c *ChimneyClient) getHealthyShaders(sharders []string) ([]string, error) {
	return c.getHealthyNodes(sharders, SharderServiceProvider)
}

func (c *ChimneyClient) getHealthyBlobbers(blobbers []string) ([]string, error) {
	return c.getHealthyNodes(blobbers, BlobberServiceProvider)
}

func (c *ChimneyClient) selectHealthyServiceProviders(networkEntrypoint string) error {
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

func (c *ChimneyClient) executeForGivenServiceProviders(
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
			t.Logf("Miner %s. Response: %s", serviceProvider, string(newResp.Body()))
			notExpectedExecutionResponseCounter++
		}
	}

	if notExpectedExecutionResponseCounter > expectedExecutionResponseCounter {
		return nil, ErrExecutionConsensus
	}

	return resp, selectMostFrequentError(respErrors)
}

func (c *ChimneyClient) executeForAllServiceProviders(
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

func (c *ChimneyClient) V1ClientPut(t *test.SystemTest, clientPutRequest model.Wallet, requiredStatusCode int) (*model.Wallet, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1TransactionPut(
	t *test.SystemTest,
	internalTransactionPutRequest model.InternalTransactionPutRequest,
	requiredStatusCode int,
) (*model.TransactionPutResponse, *resty.Response, error) { //nolint

	return c.V1TransactionPutWithNonceAndServiceProviders(t, internalTransactionPutRequest, requiredStatusCode, 0, nil)
}

func (c *ChimneyClient) V1TransactionPutWithNonceAndServiceProviders(
	t *test.SystemTest,
	internalTransactionPutRequest model.InternalTransactionPutRequest,
	requiredStatusCode, withNonce int, withProviders []string,
) (*model.TransactionPutResponse, *resty.Response, error) { //nolint

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
		TransactionType:  internalTransactionPutRequest.TxnType,
		TransactionFee:   TxFee,
		TransactionData:  string(data),
		CreationDate:     time.Now().Unix(),
		Version:          TxVersion,
	}

	if withNonce != 0 {
		transactionPutRequest.TransactionNonce = withNonce
	}

	if internalTransactionPutRequest.TransactionData.Name == "pour" {
		transactionPutRequest.TransactionFee = 0
	} else {
		fee := c.estimateTxnFee(t, &transactionPutRequest)
		transactionPutRequest.TransactionFee = fee
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

	resp, err := c.executeForGivenServiceProviders(
		t,
		NewURLBuilder().SetPath(TransactionPut),
		&model.ExecutionRequest{
			Body:               transactionPutRequest,
			Dst:                &transactionPutResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		serviceProviders)

	transactionPutResponse.Request = transactionPutRequest

	return transactionPutResponse, resp, err
}

func (c *ChimneyClient) estimateTxnFee(t *test.SystemTest, transactionPutRequest *model.TransactionPutRequest) int64 {
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
	require.Nil(t, err)

	err = json.Unmarshal(resp.Body(), &fee)
	require.NoError(t, err)
	return fee.Fee
}

func (c *ChimneyClient) V1TransactionGetConfirmation(
	t *test.SystemTest,
	transactionGetConfirmationRequest model.TransactionGetConfirmationRequest,
	requiredStatusCode int,
) (*model.TransactionGetConfirmationResponse, *resty.Response, error) { //nolint

	var transactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	urlBuilder := NewURLBuilder().
		SetPath(TransactionGetConfirmation).
		AddParams("hash", transactionGetConfirmationRequest.Hash)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &transactionGetConfirmationResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return transactionGetConfirmationResponse, resp, err
}

func (c *ChimneyClient) V1ClientGetBalance(t *test.SystemTest, clientGetBalanceRequest model.ClientGetBalanceRequest, requiredStatusCode int) (*model.ClientGetBalanceResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1ClientGetReadPoolBalance(t *test.SystemTest, clientGetReadBalanceRequest model.ClientGetReadPoolBalanceRequest, requiredStatusCode int) (*model.ClientGetReadPoolBalanceResponse, *resty.Response, error) { //nolint
	var clientGetReadPoolBalanceResponse *model.ClientGetReadPoolBalanceResponse

	urlBuilder := NewURLBuilder().SetPath(ClientReadPool).AddParams("client_id", clientGetReadBalanceRequest.ClientID).SetPathVariable("sc_address", StorageSmartContractAddress)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &clientGetReadPoolBalanceResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return clientGetReadPoolBalanceResponse, resp, err
}

func (c *ChimneyClient) V1QueryRewards(t *test.SystemTest, queryRewardsRequest model.QueryRewardsRequest, requiredStatusCode int) (*model.QueryRewardsResponse, *resty.Response, error) {
	var queryRewardsResponse *model.QueryRewardsResponse

	urlBuilder := NewURLBuilder().SetPath(QueryRewards).AddParams("query", queryRewardsRequest.Query)

	resp, err := c.executeForAllServiceProviders(
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Body:               queryRewardsRequest,
			Dst:                &queryRewardsResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		MinerServiceProvider)

	return queryRewardsResponse, resp, err
}

func (c *ChimneyClient) V1SCRestGetAllMiners(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetMinerSharderResponse, *resty.Response, error) {
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

func (c *ChimneyClient) V1SCRestGetAllSharders(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetMinerSharderResponse, *resty.Response, error) {
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

func (c *ChimneyClient) V1SCRestGetAllBlobbers(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetBlobberResponse, *resty.Response, error) {
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

func (c *ChimneyClient) V1SCRestGetAllValidators(t *test.SystemTest, requiredStatusCode int) ([]*model.SCRestGetValidatorResponse, *resty.Response, error) {
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

func (c *ChimneyClient) V1SCRestGetFirstBlobbers(t *test.SystemTest, blobbersCount, requiredStatusCode int) ([]*model.SCRestGetBlobberResponse, *resty.Response, error) {
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

func (c *ChimneyClient) V1SCRestGetBlobber(t *test.SystemTest, scRestGetBlobberRequest model.SCRestGetBlobberRequest, requiredStatusCode int) (*model.SCRestGetBlobberResponse, *resty.Response, error) {
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

func (c *ChimneyClient) V1BlobberGetHashNodeRoot(t *test.SystemTest, blobberGetHashnodeRequest *model.BlobberGetHashnodeRequest, requiredStatusCode int) (*model.BlobberGetHashnodeResponse, *resty.Response, error) {
	var hashnode *model.BlobberGetHashnodeResponse

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetHashnodeRequest.ClientId,
		"X-App-Client-Key":       blobberGetHashnodeRequest.ClientKey,
		"X-App-Client-Signature": blobberGetHashnodeRequest.ClientSignature,
		"allocation":             blobberGetHashnodeRequest.AllocationID,
		"ALLOCATION-ID":          blobberGetHashnodeRequest.AllocationID,
	}

	url := blobberGetHashnodeRequest.URL + "/" + strings.Replace(GetHashNodeRoot, ":allocation", blobberGetHashnodeRequest.AllocationID, 1)

	resp, err := c.executeForServiceProvider(t,
		url,
		model.ExecutionRequest{
			Headers:            headers,
			Dst:                &hashnode,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
	)
	return hashnode, resp, err
}

func (c *ChimneyClient) V1SCRestGetAllocation(t *test.SystemTest, scRestGetAllocationRequest model.SCRestGetAllocationRequest, requiredStatusCode int) (*model.SCRestGetAllocationResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1SCRestGetAllocationBlobbers(t *test.SystemTest, scRestGetAllocationBlobbersRequest *model.SCRestGetAllocationBlobbersRequest, requiredStatusCode int) (*model.SCRestGetAllocationBlobbersResponse, *resty.Response, error) { //nolint
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
		t,
		urlBuilder,
		&model.ExecutionRequest{
			Dst:                &blobbers,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	scRestGetAllocationBlobbersResponse.Blobbers = blobbers
	scRestGetAllocationBlobbersResponse.BlobberRequirements = scRestGetAllocationBlobbersRequest.BlobberRequirements

	return scRestGetAllocationBlobbersResponse, resp, err
}

func (c *ChimneyClient) V1SCRestGetFreeAllocationBlobbers(t *test.SystemTest, scRestGetFreeAllocationBlobbersRequest *model.FreeAllocationData, requiredStatusCode int) (*model.SCRestGetFreeAllocationBlobbersResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1SCRestOpenChallenge(t *test.SystemTest, scRestOpenChallengeRequest model.SCRestOpenChallengeRequest, requiredStatusCode int) (*model.SCRestOpenChallengeResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1MinerGetStats(t *test.SystemTest, requiredStatusCode int) (*model.GetMinerStatsResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1SharderGetStats(t *test.SystemTest, requiredStatusCode int) (*model.GetSharderStatsResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1SharderGetSCState(t *test.SystemTest, scStateGetRequest model.SCStateGetRequest, requiredStatusCode int) (*model.SCStateGetResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) CreateWalletWithMnemonicsInReturnValue(t *test.SystemTest) (wallet *model.Wallet, mnemonic string) {
	mnemonic = crypto.GenerateMnemonics(t)
	wallet = c.CreateWalletForMnemonic(t, mnemonic)
	return
}

func (c *ChimneyClient) CreateWallet(t *test.SystemTest) *model.Wallet {
	wallet, _ := c.CreateWalletWithMnemonicsInReturnValue(t)
	return wallet
}

func (c *ChimneyClient) CreateWalletForMnemonic(t *test.SystemTest, mnemonic string) *model.Wallet {
	createdWallet, err := c.CreateWalletForMnemonicWithoutAssertion(t, mnemonic)
	require.Nil(t, err)

	publicKeyBytes, _ := hex.DecodeString(createdWallet.Keys.PublicKey.SerializeToHexStr())
	clientId := crypto.Sha3256(publicKeyBytes)

	require.Equal(t, createdWallet.Id, clientId)
	require.Equal(t, createdWallet.PublicKey, createdWallet.Keys.PublicKey.SerializeToHexStr())

	return createdWallet
}

func (c *ChimneyClient) CreateWalletForMnemonicWithoutAssertion(t *test.SystemTest, mnemonic string) (*model.Wallet, error) {
	keyPair := crypto.GenerateKeys(t, mnemonic)
	publicKeyBytes, err := hex.DecodeString(keyPair.PublicKey.SerializeToHexStr())
	if err != nil {
		return nil, err
	}

	clientId := crypto.Sha3256(publicKeyBytes)
	createdWallet := model.Wallet{Id: clientId, PublicKey: keyPair.PublicKey.SerializeToHexStr(), Keys: keyPair}

	return &createdWallet, err
}

// ExecuteFaucet provides basic assertions
func (c *ChimneyClient) ExecuteFaucet(t *test.SystemTest, wallet *model.Wallet, requiredTransactionStatus int) {
	c.ExecuteFaucetWithTokens(t, wallet, 9.0, requiredTransactionStatus)
	c.ExecuteFaucetWithTokens(t, wallet, 9.0, requiredTransactionStatus)
}

// ExecuteFaucet provides basic assertions
func (c *ChimneyClient) ExecuteFaucetWithTokens(t *test.SystemTest, wallet *model.Wallet, tokens float64, requiredTransactionStatus int) {
	t.Log("Execute faucet...")

	pourZCN := tokenomics.IntToZCN(tokens)
	faucetTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      FaucetSmartContractAddress,
			TransactionData: model.NewFaucetTransactionData(),
			Value:           pourZCN,
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, faucetTransactionPutResponse)

	var faucetTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		faucetTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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
func (c *ChimneyClient) ExecuteFaucetWithAssertions(t *test.SystemTest, wallet *model.Wallet, requiredTransactionStatus int) {
	t.Log("Execute faucet with assertions...")

	faucetTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      FaucetSmartContractAddress,
			TransactionData: model.NewFaucetTransactionData(),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, faucetTransactionPutResponse)

	var faucetTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		faucetTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
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

func (c *ChimneyClient) CreateAllocation(t *test.SystemTest,
	wallet *model.Wallet,
	scRestGetAllocationBlobbersResponse *model.SCRestGetAllocationBlobbersResponse,
	requiredTransactionStatus int) string {
	return c.CreateAllocationWithLockValue(t, wallet, scRestGetAllocationBlobbersResponse, 5000.0, requiredTransactionStatus)
}

func (c *ChimneyClient) CreateAllocationWithLockValue(t *test.SystemTest,
	wallet *model.Wallet,
	scRestGetAllocationBlobbersResponse *model.SCRestGetAllocationBlobbersResponse,
	lockValue float64,
	requiredTransactionStatus int) string {
	t.Log("Create allocation...")

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

	var createAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

func (c *ChimneyClient) CreateFreeAllocation(t *test.SystemTest,
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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

func (c *ChimneyClient) UpdateAllocation(
	t *test.SystemTest,
	wallet *model.Wallet,
	allocationID string,
	uar *model.UpdateAllocationRequest,
	requiredTransactionStatus int) {
	t.Log("Update allocation...")
	uar.ID = allocationID
	updateAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewUpdateAllocationTransactionData(uar),
			Value:           tokenomics.IntToZCN(0.1),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updateAllocationTransactionPutResponse)
	txnHash := updateAllocationTransactionPutResponse.Request.Hash

	var updateAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

		return updateAllocationTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
}

func (c *ChimneyClient) AddFreeStorageAssigner(
	t *test.SystemTest,
	wallet *model.Wallet,
	requiredTransactionStatus int) {
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
	txnHash := freeAllocationTransactionPutResponse.Request.Hash

	var freeAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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
}

func (c *ChimneyClient) MakeAllocationFree(
	t *test.SystemTest,
	wallet *model.Wallet,
	allocationID, marker string,
	requiredTransactionStatus int) {
	t.Log("Update allocation...")
	freeAllocationTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: StorageSmartContractAddress,
			TransactionData: model.NewFreeAllocationTransactionData(&model.FreeAllocationRequest{

				AllocationID: allocationID,
				Marker:       marker,
			}),
			Value:   tokenomics.IntToZCN(0.1),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, freeAllocationTransactionPutResponse)
	txnHash := freeAllocationTransactionPutResponse.Request.Hash

	var freeAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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
}

func (c *ChimneyClient) UpdateAllocationBlobbers(t *test.SystemTest, wallet *model.Wallet, newBlobberID, oldBlobberID, allocationID string, requiredTransactionStatus int) {
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
			Value:   tokenomics.IntToZCN(0.1),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updateAllocationTransactionPutResponse)
	txnHash := updateAllocationTransactionPutResponse.Request.Hash

	var updateAllocationTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

		return updateAllocationTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()
}

func (c *ChimneyClient) CancelAllocation(
	t *test.SystemTest,
	wallet *model.Wallet,
	allocationID string,
	requiredTransactionStatus int,
) string {
	t.Logf("Cancel allocation %v...", allocationID)

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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

func (c *ChimneyClient) GetAllocationBlobbers(t *test.SystemTest, wallet *model.Wallet, blobberRequirements *model.BlobberRequirements, requiredStatusCode int) *model.SCRestGetAllocationBlobbersResponse {
	t.Log("Get allocation blobbers...")

	scRestGetAllocationBlobbersResponse := &model.SCRestGetAllocationBlobbersResponse{}
	err := error(nil)

	scRestGetAllocationBlobbersResponse, _, err = c.V1SCRestGetAllocationBlobbers(
		t,
		&model.SCRestGetAllocationBlobbersRequest{
			ClientID:            wallet.Id,
			ClientKey:           wallet.PublicKey,
			BlobberRequirements: *blobberRequirements,
		}, requiredStatusCode)

	if err != nil {
		// check if errors contains not enough blobbers
		if strings.Contains(err.Error(), "not enough blobbers") {
			t.Log("Not enough blobbers, waiting for 30 seconds...")

			str := err.Error()
			re := regexp.MustCompile(`\d+`)
			maxBlobberAvailableForAllocationString := re.FindString(str)

			// convert match to int
			maxBlobberAvailableForAllocation, err := strconv.Atoi(maxBlobberAvailableForAllocationString)
			require.Nil(t, err)

			blobberRequirements.DataShards = (int64(maxBlobberAvailableForAllocation) + 1) / 2
			blobberRequirements.ParityShards = int64(maxBlobberAvailableForAllocation) / 2

			return c.GetAllocationBlobbers(t, wallet, blobberRequirements, requiredStatusCode)
		} else {
			require.Nil(t, err)
		}
	}

	return scRestGetAllocationBlobbersResponse
}

func (c *ChimneyClient) GetFreeAllocationBlobbers(
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

func (c *ChimneyClient) GetAllocation(t *test.SystemTest, allocationID string, requiredStatusCode int) *model.SCRestGetAllocationResponse {
	t.Log("Get allocation...")

	var (
		scRestGetAllocation *model.SCRestGetAllocationResponse
		resp                *resty.Response //nolint
		err                 error
	)

	wait.PoolImmediately(t, time.Second*30, func() bool {
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

func (c *ChimneyClient) GetWalletBalance(t *test.SystemTest, wallet *model.Wallet, requiredStatusCode int) *model.ClientGetBalanceResponse {
	t.Log("Get wallet balance...")

	clientGetBalanceResponse, resp, err := c.V1ClientGetBalance(
		t,
		model.ClientGetBalanceRequest{
			ClientID: wallet.Id,
		},
		requiredStatusCode)

	if err != nil {
		t.Logf("Error getting wallet balance: %v", err)
		clientGetBalanceResponse.Balance = 0
		return clientGetBalanceResponse
	}
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, clientGetBalanceResponse)

	return clientGetBalanceResponse
}

func (c *ChimneyClient) GetReadPoolBalance(t *test.SystemTest, wallet *model.Wallet, requiredStatusCode int) *model.ClientGetReadPoolBalanceResponse {
	t.Log("Get read pool balance...")

	clientGetReadPoolBalanceResponse, resp, err := c.V1ClientGetReadPoolBalance(
		t,
		model.ClientGetReadPoolBalanceRequest{
			ClientID: wallet.Id,
		},
		requiredStatusCode)

	if err != nil {
		t.Logf("Error getting readpool balance: %v", err)
		return clientGetReadPoolBalanceResponse
	}
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, clientGetReadPoolBalanceResponse)

	return clientGetReadPoolBalanceResponse
}

func (c *ChimneyClient) GetRewardsByQuery(t *test.SystemTest, query string, requiredStatusCode int) *model.QueryRewardsResponse {
	t.Log("Get rewards by query...")

	queryRewardsResponse, resp, err := c.V1QueryRewards(
		t,
		model.QueryRewardsRequest{
			Query: query,
		},
		requiredStatusCode)

	if err != nil {
		t.Logf("Error getting rewards by query: %v", err)
		return queryRewardsResponse
	}

	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, queryRewardsResponse)

	return queryRewardsResponse
}

func (c *ChimneyClient) UpdateBlobber(t *test.SystemTest, wallet *model.Wallet, scRestGetBlobberResponse *model.SCRestGetBlobberResponse, requiredTransactionStatus int) {
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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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
func (c *ChimneyClient) CreateStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, requiredTransactionStatus int) string {
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
			Value:   tokenomics.IntToZCN(10.0),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createStakePoolTransactionPutResponse)

	var createStakePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

func (c *ChimneyClient) UnlockStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, requiredTransactionStatus int) string {
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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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
func (c *ChimneyClient) CreateMinerStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, tokens float64, requiredTransactionStatus int) string {
	t.Log("Create miner/sharder stake pool...")

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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

func (c *ChimneyClient) UnlockMinerStakePool(t *test.SystemTest, wallet *model.Wallet, providerType int, providerID string, requiredTransactionStatus int) string {
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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

// CreateWritePoolWrapper does not provide deep test of used components
func (c *ChimneyClient) CreateWritePool(t *test.SystemTest, wallet *model.Wallet, allocationId string, tokens float64, requiredTransactionStatus int) string {
	t.Log("Create write pool...")

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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

func (c *ChimneyClient) UnlockWritePool(t *test.SystemTest, wallet *model.Wallet, allocationId string, requiredTransactionStatus int) string {
	t.Log("Unlock Write pool...")

	unlockWritePoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet,
			ToClientID: StorageSmartContractAddress,
			TransactionData: model.NewUnlockWritePoolTransactionData(
				model.CreateWritePoolRequest{
					AllocationID: allocationId,
				}),
			Value:   tokenomics.IntToZCN(0.1),
			TxnType: SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, unlockWritePoolTransactionPutResponse)

	var unlockWritePoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		unlockWritePoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: unlockWritePoolTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if unlockWritePoolTransactionGetConfirmationResponse == nil {
			return false
		}

		return unlockWritePoolTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return unlockWritePoolTransactionGetConfirmationResponse.Hash
}

// CreateReadPoolWrapper does not provide deep test of used components
func (c *ChimneyClient) CreateReadPool(t *test.SystemTest, wallet *model.Wallet, tokens float64, requiredTransactionStatus int) string {
	t.Log("Create Read pool...")

	createReadPoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewCreateReadPoolTransactionData(),
			Value:           tokenomics.IntToZCN(tokens),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, createReadPoolTransactionPutResponse)

	var createReadPoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		createReadPoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: createReadPoolTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if createReadPoolTransactionGetConfirmationResponse == nil {
			return false
		}

		return createReadPoolTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return createReadPoolTransactionGetConfirmationResponse.Hash
}

func (c *ChimneyClient) UnlockReadPool(t *test.SystemTest, wallet *model.Wallet, requiredTransactionStatus int) string {
	t.Log("Unlock Read pool...")

	unlockReadPoolTransactionPutResponse, resp, err := c.V1TransactionPut(
		t,
		model.InternalTransactionPutRequest{
			Wallet:          wallet,
			ToClientID:      StorageSmartContractAddress,
			TransactionData: model.NewUnlockReadPoolTransactionData(),
			Value:           tokenomics.IntToZCN(0.1),
			TxnType:         SCTxType,
		},
		HttpOkStatus)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, unlockReadPoolTransactionPutResponse)

	var unlockReadPoolTransactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	wait.PoolImmediately(t, time.Minute*2, func() bool {
		unlockReadPoolTransactionGetConfirmationResponse, resp, err = c.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: unlockReadPoolTransactionPutResponse.Entity.Hash,
			},
			HttpOkStatus)
		if err != nil {
			return false
		}

		if resp == nil {
			return false
		}

		if unlockReadPoolTransactionGetConfirmationResponse == nil {
			return false
		}

		return unlockReadPoolTransactionGetConfirmationResponse.Status == requiredTransactionStatus
	})

	wallet.IncNonce()

	return unlockReadPoolTransactionGetConfirmationResponse.Hash
}

func (c *ChimneyClient) V1SCRestGetStakePoolStat(t *test.SystemTest, scRestGetStakePoolStatRequest model.SCRestGetStakePoolStatRequest, requiredStatusCode int) (*model.SCRestGetStakePoolStatResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) V1SCRestGetUserStakePoolStat(t *test.SystemTest, scRestGetUserStakePoolStatRequest model.SCRestGetUserStakePoolStatRequest, requiredStatusCode int) (*model.SCRestGetUserStakePoolStatResponse, *resty.Response, error) { //nolint
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

func (c *ChimneyClient) GetStakePoolStat(t *test.SystemTest, providerID, providerType string) *model.SCRestGetStakePoolStatResponse {
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

func (c *ChimneyClient) CollectRewards(t *test.SystemTest, wallet *model.Wallet, providerID string, providerType, requiredTransactionStatus int) (txnData *model.TransactionGetConfirmationResponse, fee int64) {
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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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

func (c *ChimneyClient) GetBlobber(t *test.SystemTest, blobberID string, requiredStatusCode int) *model.SCRestGetBlobberResponse {
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

func (c *ChimneyClient) V1BlobberGetFileRefs(t *test.SystemTest, blobberGetFileRefsRequest *model.BlobberGetFileRefsRequest, requiredStatusCode int) (*model.BlobberGetFileRefsResponse, *resty.Response, error) {
	var blobberGetFileResponse *model.BlobberGetFileRefsResponse

	url := blobberGetFileRefsRequest.URL + strings.Replace(GetFileRef, ":allocation_id", blobberGetFileRefsRequest.AllocationID, 1) + "?" + "path=" + blobberGetFileRefsRequest.RemotePath + "&" + "refType=" + blobberGetFileRefsRequest.RefType

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetFileRefsRequest.ClientID,
		"X-App-Client-Key":       blobberGetFileRefsRequest.ClientKey,
		"X-App-Client-Signature": blobberGetFileRefsRequest.ClientSignature,
		"ALLOCATION-ID":          blobberGetFileRefsRequest.AllocationID,
	}
	resp, err := c.executeForServiceProvider(
		t,
		url,
		model.ExecutionRequest{
			Dst:                &blobberGetFileResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberGetFileResponse, resp, err
}

func (c *ChimneyClient) V1BlobberGetFileRefPaths(t *test.SystemTest, blobberFileRefPathRequest *model.BlobberFileRefPathRequest, requiredStatusCode int) (*model.BlobberFileRefPathResponse, *resty.Response, error) {
	var blobberFileRefPathResponse *model.BlobberFileRefPathResponse

	url := blobberFileRefPathRequest.URL + strings.Replace(GetFileRefPath, ":allocation_id", blobberFileRefPathRequest.AllocationID, 1) + "?" + "path=" + blobberFileRefPathRequest.Path

	headers := map[string]string{
		"X-App-Client-Id":        blobberFileRefPathRequest.ClientID,
		"X-App-Client-Key":       blobberFileRefPathRequest.ClientKey,
		"X-App-Client-Signature": blobberFileRefPathRequest.ClientSignature,
		"ALLOCATION-ID":          blobberFileRefPathRequest.AllocationID,
	}
	resp, err := c.executeForServiceProvider(
		t,
		url,
		model.ExecutionRequest{
			Dst:                &blobberFileRefPathResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberFileRefPathResponse, resp, err
}

func (c *ChimneyClient) V1BlockGetLatestFinalizedMagicBlock(t *test.SystemTest, hash string, requiredStatusCode int) (*resty.Response, error) {
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

func (c *ChimneyClient) V1BlobberObjectTree(t *test.SystemTest, blobberObjectTreeRequest *model.BlobberObjectTreeRequest, requiredStatusCode int) (*model.BlobberObjectTreePathResponse, *resty.Response, error) {
	var blobberObjectTreePathResponse *model.BlobberObjectTreePathResponse

	url := blobberObjectTreeRequest.URL + strings.Replace(GetObjectTree, ":allocation_id", blobberObjectTreeRequest.AllocationID, 1) + "?" + "path=" + blobberObjectTreeRequest.Path

	headers := map[string]string{
		"X-App-Client-Id":        blobberObjectTreeRequest.ClientID,
		"X-App-Client-Key":       blobberObjectTreeRequest.ClientKey,
		"X-App-Client-Signature": blobberObjectTreeRequest.ClientSignature,
		"ALLOCATION-ID":          blobberObjectTreeRequest.AllocationID,
	}
	resp, err := c.executeForServiceProvider(
		t,
		url,
		model.ExecutionRequest{
			Dst:                &blobberObjectTreePathResponse,
			RequiredStatusCode: requiredStatusCode,
			Headers:            headers,
		},
		HttpGETMethod)
	return blobberObjectTreePathResponse, resp, err
}

//----------------------------------------------------------
// ZCN SC
//----------------------------------------------------------

func (c *ChimneyClient) BurnZcn(t *test.SystemTest, wallet *model.Wallet, address string, amount float64, requiredTransactionStatus int) string {
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

	wait.PoolImmediately(t, time.Minute*2, func() bool {
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
