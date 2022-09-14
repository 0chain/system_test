package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/sc_address"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
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

// GetChainStats all used url paths in the client
const (
	ChainGetStats              = "/v1/chain/get/stats"
	ClientPut                  = "/v1/client/put"
	TransactionPut             = "/v1/transaction/put"
	TransactionGetConfirmation = "/v1/transaction/get/confirmation"
	ClientGetBalance           = "/v1/client/get/balance"
)

// Contains all used service providers
const (
	MinerServiceProvider = iota
	SharderServiceProvider
)

// Statuses of transactions
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

type Client struct {
	model.NetworkHealthResources

	httpClient *resty.Client //nolint
}

func (c *Client) getHealthyNodes(nodes []string) []string {
	var result []string
	for _, node := range nodes {
		formattedURL := NewURLBuilder().SetHost(node).SetPath(ChainGetStats).String()

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

func (c *Client) getHealthyMiners(miners []string) []string {
	return c.getHealthyNodes(miners)
}

func (c *Client) getHealthyShaders(shaders []string) []string {
	return c.getHealthyNodes(shaders)
}

func (c *Client) selectHealthServiceProviders(networkEntrypoint string) error {
	resp, err := c.httpClient.R().Get(networkEntrypoint)
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

func NewClient(networkEntrypoint string) *Client {
	apiClient := &Client{
		httpClient: resty.New(), //nolint
	}

	if err := apiClient.selectHealthServiceProviders(networkEntrypoint); err != nil {
		log.Fatalln(err)
	}

	return apiClient
}

func (c *Client) matchConsensus() bool {
	return false
}

func (c *Client) executeForServiceProvider(url string, executionRequest model.ExecutionRequest, method int) (*resty.Response, error) { //nolint
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

	//	t.Logf("GET on miner [" + miner + "] endpoint [" + endpoint + "] was unsuccessful, resulting in HTTP [" + resp.Status() + "] and body [" + resp.String() + "]")
	err = json.Unmarshal(resp.Body(), executionRequest.Dst)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) executeForAllServiceProviders(urlBuilder *URLBuilder, executionRequest model.ExecutionRequest, method, serviceProviderType int) (*resty.Response, error) {
	var (
		resp      *resty.Response
		lastError error
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
		formattedURL := urlBuilder.SetHost(serviceProvider).String()

		newResp, err := c.executeForServiceProvider(formattedURL, executionRequest, method)
		if err != nil {
			lastError = err
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

	return resp, nil
}

func (c *Client) V1ClientPut(clientPutWalletRequest model.ClientPutWalletRequest, requiredStatusCode int) (*model.ClientPutWalletResponse, *resty.Response, error) { //nolint
	var clientPutWalletResponse *model.ClientPutWalletResponse

	urlBuilder := NewURLBuilder().SetPath(ClientPut)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Body:               clientPutWalletRequest,
			Dst:                clientPutWalletResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		MinerServiceProvider)

	return clientPutWalletResponse, resp, err
}

func (c *Client) V1TransactionPut(wallet *model.Wallet, transactionData model.TransactionData, requiredStatusCode int) (*model.TransactionPutResponse, *resty.Response, error) { //nolint
	var transactionPutResponse *model.TransactionPutResponse

	data, err := json.Marshal(transactionData)
	if err != nil {
		log.Fatalln(err)
	}

	transactionPutRequest := model.TransactionPutRequest{
		PublicKey:        wallet.ClientKey,
		ClientId:         wallet.ClientID,
		ToClientId:       sc_address.StorageSmartContractAddress,
		TransactionNonce: wallet.Nonce + 1,
		TxnOutputHash:    TxOutput,
		TransactionValue: TxValue,
		TransactionType:  TxType,
		TransactionFee:   TxFee,
		TransactionData:  string(data),
		CreationDate:     time.Now().Unix(),
		Version:          TxVersion,
	}

	urlBuilder := NewURLBuilder().SetPath(TransactionPut)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Body:               transactionPutRequest,
			Dst:                transactionPutResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpPOSTMethod,
		MinerServiceProvider)

	return transactionPutResponse, resp, err
}

func (c *Client) V1TransactionGetConfirmation(transactionGetConfirmationRequest *model.TransactionGetConfirmationRequest, requiredStatusCode int) (*model.TransactionGetConfirmationResponse, *resty.Response, error) { //nolint
	var transactionGetConfirmationResponse *model.TransactionGetConfirmationResponse

	urlBuilder := NewURLBuilder().
		SetPath(TransactionGetConfirmation).
		SetPathVariable("hash", transactionGetConfirmationRequest.Hash)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Dst:                transactionGetConfirmationResponse,
			RequiredStatusCode: requiredStatusCode,
		},
		HttpGETMethod,
		SharderServiceProvider)

	return transactionGetConfirmationResponse, resp, err
}

func (c *Client) V1ClientGetBalance(clientId string, requiredStatusCode int) (*model.Balance, *resty.Response, error) { //nolint
	var clientPutWalletResponse *model.ClientPutWalletResponse

	urlBuilder := NewURLBuilder().SetPath(ClientPut)

	resp, err := c.executeForAllServiceProviders(
		urlBuilder,
		model.ExecutionRequest{
			Body: clientPutWalletRequest,
			Dst:  clientPutWalletResponse,
		},
		HttpPOSTMethod,
		MinerServiceProvider)

	return clientPutWalletResponse, resp, err

	//var balance *model.Balance

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("?client_id="+clientId), consensusCategoriser, &balance)

	return balance, httpResponse, httpError
}

func mostDominantError(errors []error) error {
	var mostFrequent error
	topFrequencyCount := 0

	for _, currentError := range errors {
		currentFrequencyCount := 0
		for _, compareToError := range errors {
			if currentError == compareToError {
				currentFrequencyCount++
			}
		}

		if currentFrequencyCount > topFrequencyCount {
			topFrequencyCount = currentFrequencyCount
			mostFrequent = currentError
		}
	}

	return mostFrequent
}

func v1ScrestAllocation(t *testing.T, clientId string, consensusCategoriser ConsensusMetFunction) (*model.Allocation, *resty.Response, error) { //nolint
	var allocation *model.Allocation

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", StorageSmartContractAddress, "/allocation?allocation="+clientId), consensusCategoriser, &allocation)

	return allocation, httpResponse, httpError
}

func v1ScrestAllocBlobbers(t *testing.T, allocationData string, consensusCategoriser ConsensusMetFunction) (*[]string, *resty.Response, error) { //nolint
	var blobbers *[]string

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", StorageSmartContractAddress, "/alloc_blobbers?allocation_data=")+allocationData, consensusCategoriser, &blobbers)

	return blobbers, httpResponse, httpError
}

func v1ScrestOpenChallenges(t *testing.T, storageSmartContractAddress string, blobberId string, consensusCategoriser ConsensusMetFunction) (*resty.Response, error) { //nolint
	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", storageSmartContractAddress, "/openchallenges?blobber="+blobberId), consensusCategoriser, nil)
	return httpResponse, httpError
}

func v1MinerGetStats(t *testing.T, consensusCategoriser ConsensusMetFunction) (*model.MinerStats, *resty.Response, error) { //nolint
	var stats *model.MinerStats

	httpResponse, httpError := zeroChain.GetFromMiners(t, "/v1/miner/get/stats", consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}

func v1SharderGetStats(t *testing.T, consensusCategoriser ConsensusMetFunction) (*model.SharderStats, *resty.Response, error) { //nolint
	var stats *model.SharderStats

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/sharder/get/stats", consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}

func v1SharderGetSCState(t *testing.T, SCAddress, key string, consensusCategoriser ConsensusMetFunction) (*model.SharderSCStateResponse, *resty.Response, error) { //nolint
	var stats *model.SharderSCStateResponse

	formData := map[string]string{
		"sc_address": SCAddress,
		"key":        key,
	}

	httpResponse, httpError := zeroChain.PostToShardersWithFormData(t, "/v1/scstate/get", consensusCategoriser, formData, nil, &stats)

	return stats, httpResponse, httpError
}

//Uploads a new file to blobber
func v1BlobberFileUpload(t *testing.T, blobberUploadFileRequest model.BlobberUploadFileRequest) (*model.BlobberUploadFileResponse, *resty.Response, error) {
	var stats *model.BlobberUploadFileResponse

	payload := new(bytes.Buffer)
	writer := multipart.NewWriter(payload)
	uploadFile, err := writer.CreateFormFile("uploadFile", filepath.Base(blobberUploadFileRequest.Meta.FilePath))
	if err != nil {
		return nil, nil, err
	}

	_, err = io.Copy(uploadFile, blobberUploadFileRequest.File)
	if err != nil {
		return nil, nil, err
	}

	err = writer.WriteField("connection_id", blobberUploadFileRequest.Meta.ConnectionID)
	if err != nil {
		return nil, nil, err
	}

	metaData, err := json.Marshal(blobberUploadFileRequest.Meta)
	if err != nil {
		return nil, nil, err
	}

	err = writer.WriteField("uploadMeta", string(metaData))
	if err != nil {
		return nil, nil, err
	}

	headers := map[string]string{
		"X-App-Client-Id":        blobberUploadFileRequest.ClientID,
		"X-App-Client-Key":       blobberUploadFileRequest.ClientKey,
		"X-App-Client-Signature": blobberUploadFileRequest.ClientSignature,
		"Content-Type":           writer.FormDataContentType(),
	}

	err = writer.Close()
	if err != nil {
		return nil, nil, err
	}

	httpResponse, httpError := zeroChain.PostToBlobber(t,
		blobberUploadFileRequest.URL,
		filepath.Join("/v1/file/upload", blobberUploadFileRequest.AllocationID),
		headers,
		nil,
		payload.Bytes(),
		&stats)

	return stats, httpResponse, httpError
}

//Queries all the files in certain allocation
func v1BlobberListFiles(t *testing.T, blobberListFilesRequest model.BlobberListFilesRequest) (*model.BlobberListFilesResponse, *resty.Response, error) {
	var stats *model.BlobberListFilesResponse

	params := map[string]string{
		"path_hash":  blobberListFilesRequest.PathHash,
		"path":       "/",
		"auth_token": "",
	}

	headers := map[string]string{
		"X-App-Client-Id":        blobberListFilesRequest.ClientID,
		"X-App-Client-Key":       blobberListFilesRequest.ClientKey,
		"X-App-Client-Signature": blobberListFilesRequest.ClientSignature,
	}

	httpResponse, httpError := zeroChain.GetFromBlobber(t,
		blobberListFilesRequest.URL,
		filepath.Join("/v1/file/list", blobberListFilesRequest.AllocationID),
		headers,
		params,
		&stats)

	return stats, httpResponse, httpError
}

//Queries files in certain allocation
func v1BlobberGetFileReferencePath(t *testing.T, blobberGetFileReferencePathRequest model.BlobberGetFileReferencePathRequest) (*model.BlobberGetFileReferencePathResponse, *resty.Response, error) {
	var stats *model.BlobberGetFileReferencePathResponse

	params := map[string]string{
		"paths": fmt.Sprintf("[\"%s\"]", "/"),
	}

	headers := map[string]string{
		"X-App-Client-Id":        blobberGetFileReferencePathRequest.ClientID,
		"X-App-Client-Key":       blobberGetFileReferencePathRequest.ClientKey,
		"X-App-Client-Signature": blobberGetFileReferencePathRequest.ClientSignature,
	}

	httpResponse, httpError := zeroChain.GetFromBlobber(t,
		blobberGetFileReferencePathRequest.URL,
		filepath.Join("/v1/file/referencepath", blobberGetFileReferencePathRequest.AllocationID),
		headers,
		params,
		&stats)

	return stats, httpResponse, httpError
}

//Commits all the actions in a certain opened connection
func v1BlobberCommitConnection(t *testing.T, blobberCommitConnectionRequest model.BlobberCommitConnectionRequest) (*model.BlobberCommitConnectionResponse, *resty.Response, error) {
	var stats *model.BlobberCommitConnectionResponse

	writeMarker, err := json.Marshal(blobberCommitConnectionRequest.WriteMarker)
	if err != nil {
		return nil, nil, err
	}

	formData := map[string]string{
		"connection_id": blobberCommitConnectionRequest.ConnectionID,
		"write_marker":  string(writeMarker),
	}

	headers := map[string]string{
		"X-App-Client-Id":   blobberCommitConnectionRequest.WriteMarker.ClientID,
		"X-App-Client-Key":  blobberCommitConnectionRequest.ClientKey,
		"Connection":        "Keep-Alive",
		"Cache-Control":     "no-cache",
		"Transfer-Encoding": "chunked",
	}

	httpResponse, httpError := zeroChain.PostToBlobber(t,
		blobberCommitConnectionRequest.URL,
		filepath.Join("/v1/connection/commit", blobberCommitConnectionRequest.WriteMarker.AllocationID),
		headers,
		formData,
		nil,
		&stats)

	return stats, httpResponse, httpError
}

//Commits all the actions in a certain opened connection
func v1BlobberDownloadFile(t *testing.T, blobberDownloadFileRequest model.BlobberDownloadFileRequest) (*model.BlobberDownloadFileResponse, *resty.Response, error) {
	var stats *model.BlobberDownloadFileResponse

	readMarker, err := json.Marshal(blobberDownloadFileRequest.ReadMarker)
	if err != nil {
		return nil, nil, err
	}

	formData := map[string]string{
		"path_hash":   blobberDownloadFileRequest.PathHash,
		"block_num":   blobberDownloadFileRequest.BlockNum,
		"num_blocks":  blobberDownloadFileRequest.NumBlocks,
		"read_marker": string(readMarker),
	}

	headers := map[string]string{
		"X-App-Client-Id":  blobberDownloadFileRequest.ReadMarker.ClientID,
		"X-App-Client-Key": blobberDownloadFileRequest.ReadMarker.ClientKey,
	}

	httpResponse, httpError := zeroChain.PostToBlobber(t,
		blobberDownloadFileRequest.URL,
		filepath.Join("/v1/file/download", blobberDownloadFileRequest.ReadMarker.AllocationID),
		headers,
		formData,
		nil,
		&stats)

	return stats, httpResponse, httpError
}

func v1SCRestGetBlobber(t *testing.T, blobberId string, consensusCategoriser ConsensusMetFunction) (*model.GetBlobberResponse, *resty.Response, error) {
	var stats *model.GetBlobberResponse

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", StorageSmartContractAddress, "/getBlobber?blobber_id="+blobberId), consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}
