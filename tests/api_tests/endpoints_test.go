package api_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	resty "github.com/go-resty/resty/v2"
	"io"
	"mime/multipart"
	"path/filepath"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
)

func v1ClientPut(t *testing.T, walletRequest model.ClientPutWalletRequest, consensusCategoriser endpoint.ConsensusMetFunction) (*model.ClientPutWalletResponse, *resty.Response, error) { //nolint
	var wallet *model.ClientPutWalletResponse

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/client/put", consensusCategoriser, walletRequest, &wallet)

	return wallet, httpResponse, httpError
}

func v1TransactionPut(t *testing.T, walletRequest *model.Transaction, consensusCategoriser endpoint.ConsensusMetFunction) (*model.TransactionResponse, *resty.Response, error) { //nolint
	var transaction *model.TransactionResponse

	httpResponse, httpError := zeroChain.PostToMiners(t, "/v1/transaction/put", consensusCategoriser, walletRequest, &transaction)

	return transaction, httpResponse, httpError
}

func v1TransactionGetConfirmation(t *testing.T, hash string, consensusCategoriser endpoint.ConsensusMetFunction) (*model.Confirmation, *resty.Response, error) { //nolint
	var confirmation *model.Confirmation

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/transaction/get/confirmation?hash="+hash), consensusCategoriser, &confirmation)

	return confirmation, httpResponse, httpError
}

func v1ClientGetBalance(t *testing.T, clientId string, consensusCategoriser endpoint.ConsensusMetFunction) (*model.Balance, *resty.Response, error) { //nolint
	var balance *model.Balance

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/client/get/balance?client_id="+clientId), consensusCategoriser, &balance)

	return balance, httpResponse, httpError
}

func v1ScrestAllocation(t *testing.T, clientId string, consensusCategoriser endpoint.ConsensusMetFunction) (*model.Allocation, *resty.Response, error) { //nolint
	var allocation *model.Allocation

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", endpoint.StorageSmartContractAddress, "/allocation?allocation="+clientId), consensusCategoriser, &allocation)

	return allocation, httpResponse, httpError
}

func v1ScrestAllocBlobbers(t *testing.T, allocationData string, consensusCategoriser endpoint.ConsensusMetFunction) (*[]string, *resty.Response, error) { //nolint
	var blobbers *[]string

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", endpoint.StorageSmartContractAddress, "/alloc_blobbers?allocation_data=")+allocationData, consensusCategoriser, &blobbers)

	return blobbers, httpResponse, httpError
}

func v1ScrestOpenChallenges(t *testing.T, storageSmartContractAddress string, blobberId string, consensusCategoriser endpoint.ConsensusMetFunction) (*resty.Response, error) { //nolint
	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", storageSmartContractAddress, "/openchallenges?blobber="+blobberId), consensusCategoriser, nil)
	return httpResponse, httpError
}

func v1MinerGetStats(t *testing.T, consensusCategoriser endpoint.ConsensusMetFunction) (*model.MinerStats, *resty.Response, error) { //nolint
	var stats *model.MinerStats

	httpResponse, httpError := zeroChain.GetFromMiners(t, "/v1/miner/get/stats", consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}

func v1SharderGetStats(t *testing.T, consensusCategoriser endpoint.ConsensusMetFunction) (*model.SharderStats, *resty.Response, error) { //nolint
	var stats *model.SharderStats

	httpResponse, httpError := zeroChain.GetFromSharders(t, "/v1/sharder/get/stats", consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}

func v1SharderGetSCState(t *testing.T, SCAddress, key string, consensusCategoriser endpoint.ConsensusMetFunction) (*model.SharderSCStateResponse, *resty.Response, error) { //nolint
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

func v1SCRestGetBlobber(t *testing.T, blobberId string, consensusCategoriser endpoint.ConsensusMetFunction) (*model.GetBlobberResponse, *resty.Response, error) {
	var stats *model.GetBlobberResponse

	httpResponse, httpError := zeroChain.GetFromSharders(t, filepath.Join("/v1/screst/", endpoint.StorageSmartContractAddress, "/getBlobber?blobber_id="+blobberId), consensusCategoriser, &stats)

	return stats, httpResponse, httpError
}
