package api_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/0chain/gosdk/constants"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/errgo.v2/errors"
	// coreClient "github.com/0chain/gosdk/core/client"
)

func Test0BoxTranscoder(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Logf("Starting 0Box Transcoder Tests...")

	// Create a new wallet
	headers := map[string]string{
		"X-App-Client-ID":        sdkWallet.ClientID,
		"X-App-Client-Key":       sdkWallet.ClientKey,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE,
		"X-APP-TYPE":             client.X_APP_BLIMP,
	}
	Teardown(t, headers)
	err := Create0boxTestWallet(t, headers)
	require.NoError(t, err)
	t.Logf("0box test wallet created: %s", headers["X-App-Client-ID"])

	t.Logf("Generating split key...")
	jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	zVaultHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)
	zVaultHeaders["X-User-ID"] = client.X_APP_USER_ID

	response, err = zvaultClient.Store(t, sdkWallet.Keys[0].PrivateKey, sdkWallet.Mnemonic, zVaultHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "StoreHandler: Response status code does not match expected. Output: [%v]", response.String())
	t.Logf("Wallet stored with StoreHandler: %v", response.String())

	t.Logf("Generating split key for client id: %s...", sdkWallet.ClientID)
	response, err = zvaultClient.GenerateSplitKey(t, sdkWallet.ClientID, zVaultHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "GenerateSplitKey: Status code does not match expected. Output: [%v]", response.String())
	t.Logf("Generated split key for client id: %s", sdkWallet.ClientID)

	t.Logf("Getting keys for client id: %s...", sdkWallet.ClientID)
	keys, response, err := zvaultClient.GetKeys(t, sdkWallet.ClientID, zVaultHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "GetKeys after StoreHandler: Status code does not match expected. Output: [%v]", response.String())
	require.Len(t, keys.Keys, 1)
	require.Equal(t, sdkWallet.ClientID, keys.Keys[0].ClientID, "Stored key's clientID: %s does not match sdkWallet clientID: %s", keys.Keys[0].ClientID, sdkWallet.ClientID)
	t.Logf("Verified key's clientID: %s", keys.Keys[0].ClientID)

	t.Cleanup(func() {
		t.Logf("Deleting wallet for client id: %s...", sdkWallet.ClientID)
		response, err = zvaultClient.Delete(t, sdkWallet.ClientID, zVaultHeaders)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Delete: Response status code does not match expected. Output: [%v]", response.String())
		t.Logf("Wallet deleted for client id: %s", sdkWallet.ClientID)
	})

	t.Logf("sharing wallet to 0box-server...")
	response, err = zvaultClient.ShareWallet(t, "0box-server", keys.Keys[0].PublicKey, zVaultHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	t.Logf("retrieving split keys...")
	keys, response, err = zvaultClient.GetKeys(t, sdkWallet.ClientID, zVaultHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	require.Equal(t, keys.Keys[0].ClientID, sdkWallet.ClientID)

	var sharedKey *model.SplitKey
	for i, k := range keys.Keys {
		t.Logf("split key[%d]: %v", i, *k)
		t.Logf("SharedTo[%d]: %v", i, k.SharedTo)
		if k.SharedTo == "0box-server" {
			sharedKey = k
			break
		}
	}
	require.NotNil(t, sharedKey, "Shared key to 0box-server not found")
	t.Logf("sharedKey: %v, splitKeys len %d", *sharedKey, len(keys.Keys))

	// create allocation
	t.Logf("creating new allocation through SDK...")
	var (
		readPrice  = sdk.PriceRange{Min: 0, Max: 0}
		writePrice = sdk.PriceRange{Min: 0, Max: 250000000}
	)
	options := sdk.CreateAllocationOptions{
		DataShards:   1,
		ParityShards: 1,
		Size:         2147483648, // 2GB
		ReadPrice: sdk.PriceRange{
			Min: readPrice.Min,
			Max: readPrice.Max,
		},
		WritePrice: sdk.PriceRange{
			Min: writePrice.Min,
			Max: writePrice.Max,
		},
		FileOptionsParams: &sdk.FileOptionsParameters{},
		Lock:              1000000000,
	}
	t.Logf("creating new allocation with options: %+v", options)
	allocationID, _, _, err := sdk.CreateAllocationWith(options)
	require.NoError(t, err, "Failed to create allocation with SDK")
	t.Logf("newallocation created with ID: %v", allocationID)

	// t.Cleanup(func() {
	// 	t.Logf("deleting allocation: %s", allocationID)
	// 	_, _, err := sdk.CancelAllocation(allocationID)
	// 	require.NoError(t, err, "Failed to cancel allocation with SDK")
	// 	t.Logf("allocation cancelled: %s", allocationID)
	// })

	// Run transcode subtests (they will run in parallel because RunSequentially now delegates to Run)
	timeout := 3 * time.Minute
	sleepTime := 20 * time.Second
	t.RunWithTimeout("Transcode MP4 file with web mode", timeout, func(t *test.SystemTest) {
		// Test MP4 transcoding
		fileName := "sample.mp4"
		transcodeFile(t, headers, allocationID, sharedKey, fileName, "web")

		// wait before first verification (20 seconds) then verify up to 3 times
		time.Sleep(sleepTime)
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": fmt.Sprintf("/%s/%s", fileName, fileName),
			"mode":      "web",
		}

		var verr error
		for i := 0; i < 3; i++ {
			t.Logf("verify attempt %d for sample.mp4", i+1)

			// First verify metadata status
			entity, response, err := zboxClient.GetMetadata(t, headers, queryParams)
			if err != nil {
				t.Logf("metadata verification attempt %d failed: %v, sleeping %s before retry", i+1, err, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if response.StatusCode() != 200 {
				t.Logf("metadata verification attempt %d failed: status code %d, sleeping %s before retry", i+1, response.StatusCode(), sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if entity == nil {
				t.Logf("metadata verification attempt %d failed: entity is nil, sleeping %s before retry", i+1, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if entity.Status != 6 {
				t.Logf("metadata verification attempt %d: status is %d (expected 6), sleeping %s before retry", i+1, entity.Status, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			t.Logf("Transcoding entity status verified: %d", entity.Status)

			// Metadata status is 6, proceed to verify transcoded file
			verr = verifyTranscodedFile(t, allocationID, fmt.Sprintf("/%s/preview", fileName), sharedKey)
			if verr == nil {
				break
			}
			t.Logf("file verification attempt %d failed: %v, sleeping %s before retry", i+1, verr, sleepTime)
			time.Sleep(sleepTime)
		}
		require.NoError(t, verr, "verification failed after 3 attempts")
	})

	t.RunWithTimeout("Transcode AVI file with web mode", timeout, func(t *test.SystemTest) {
		// Test AVI transcoding
		fileName := "sample.avi"
		transcodeFile(t, headers, allocationID, sharedKey, fileName, "web")

		// wait before first verification (20 seconds) then verify up to 3 times
		time.Sleep(sleepTime)
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": fmt.Sprintf("/%s/%s", fileName, fileName),
			"mode":      "web",
		}

		var verr error
		for i := 0; i < 3; i++ {
			t.Logf("verify attempt %d for sample.avi", i+1)

			// First verify metadata status
			entity, response, err := zboxClient.GetMetadata(t, headers, queryParams)
			if err != nil {
				t.Logf("metadata verification attempt %d failed: %v, sleeping %s before retry", i+1, err, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if response.StatusCode() != 200 {
				t.Logf("metadata verification attempt %d failed: status code %d, sleeping %s before retry", i+1, response.StatusCode(), sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if entity == nil {
				t.Logf("metadata verification attempt %d failed: entity is nil, sleeping %s before retry", i+1, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if entity.Status != 6 {
				t.Logf("metadata verification attempt %d: status is %d (expected 6), sleeping %s before retry", i+1, entity.Status, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			t.Logf("Transcoding entity status verified: %d", entity.Status)

			// Metadata status is 6, proceed to verify transcoded file
			verr = verifyTranscodedFile(t, allocationID, fmt.Sprintf("/%s/preview", fileName), sharedKey)
			if verr == nil {
				break
			}
			t.Logf("file verification attempt %d failed: %v, sleeping %s before retry", i+1, verr, sleepTime)
			time.Sleep(sleepTime)
		}
		require.NoError(t, verr, "verification failed after 3 attempts")
	})

	t.RunWithTimeout("Transcode MOV file with web mode", timeout, func(t *test.SystemTest) {
		// Test MOV transcoding
		fileName := "sample.mov"
		transcodeFile(t, headers, allocationID, sharedKey, fileName, "web")

		// wait before first verification (20 seconds) then verify up to 3 times
		time.Sleep(sleepTime)
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": fmt.Sprintf("/%s/%s", fileName, fileName),
			"mode":      "web",
		}

		var verr error
		for i := 0; i < 3; i++ {
			t.Logf("verify attempt %d for sample.mov", i+1)

			// First verify metadata status
			entity, response, err := zboxClient.GetMetadata(t, headers, queryParams)
			if err != nil {
				t.Logf("metadata verification attempt %d failed: %v, sleeping %s before retry", i+1, err, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if response.StatusCode() != 200 {
				t.Logf("metadata verification attempt %d failed: status code %d, sleeping %s before retry", i+1, response.StatusCode(), sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if entity == nil {
				t.Logf("metadata verification attempt %d failed: entity is nil, sleeping %s before retry", i+1, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			if entity.Status != 6 {
				t.Logf("metadata verification attempt %d: status is %d (expected 6), sleeping %s before retry", i+1, entity.Status, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
			t.Logf("Transcoding entity status verified: %d", entity.Status)

			// Metadata status is 6, proceed to verify transcoded file
			verr = verifyTranscodedFile(t, allocationID, fmt.Sprintf("/%s/preview", fileName), sharedKey)
			if verr == nil {
				break
			}
			t.Logf("file verification attempt %d failed: %v, sleeping %s before retry", i+1, verr, sleepTime)
			time.Sleep(sleepTime)
		}
		require.NoError(t, verr, "verification failed after 3 attempts")
	})

}

// transcodeFile makes the actual API call to the transcoder endpoint
func transcodeFile(t *test.SystemTest, headers map[string]string, allocationID string, splitKey *model.SplitKey, fileName, mode string) {
	t.Logf("transcoding file: %s with mode: %s", fileName, mode)

	t.Logf("splitKey fields: ClientID=%s, PublicKey=%s, PrivateKey=%s, UserID=%s, PeerPublicKey=%s",
		splitKey.ClientID, splitKey.PublicKey, splitKey.PrivateKey, splitKey.UserID, splitKey.PeerPublicKey)

	allocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err, "Failed to get allocation from SDK")

	// Create directory at remotepath "/{fileName}"
	remotepath := fmt.Sprintf("/%s", fileName)
	createDirOp := sdk.OperationRequest{
		OperationType: constants.FileOperationCreateDir,
		RemotePath:    remotepath,
	}
	err = allocation.DoMultiOperation([]sdk.OperationRequest{createDirOp})
	require.NoError(t, err, "Failed to create directory")

	// Use the test_files_small folder as the workDir for uploads
	curDir, err := os.Getwd()
	require.NoError(t, err, "Unable to get working directory")
	workDir := filepath.Join(curDir, "test_files_small")

	// Upload file to "/{fileName}"
	err = UploadFileBlobber(t, *sdkWallet, allocation, workDir, remotepath, fileName)
	require.NoError(t, err, "upload failed: %v", err)

	// Determine remote name and file size for metadata
	fi, err := os.Stat(filepath.Join(workDir, fileName))
	require.NoError(t, err)
	actualSize := fi.Size()

	// Metadata request with new structure
	metaBody := map[string]interface{}{
		"remotepath":    remotepath,
		"mode":          mode,
		"allocation_id": allocationID,
		"file_name":     fileName,
		"file_size":     actualSize,
		"file_path":     fmt.Sprintf("/%s/%s", fileName, fileName),
		"do_thumbnail":  false,
	}

	transcodingEntity, response, metaErr := zboxClient.CreateMetadata(t, headers, metaBody)
	require.NoError(t, metaErr, "createMetadata API call failed: %v", metaErr)
	require.Equal(t, 201, response.StatusCode(), "createMetadata API call failed: %v", response.String())
	require.NotNil(t, transcodingEntity, "transcodingEntity is nil")
	t.Logf("transcodingEntity: %v", *transcodingEntity)

	// Update upload status
	updateBody := map[string]interface{}{
		"id":     transcodingEntity.ID,
		"status": 1,
	}
	transcodingEntity, response, updateErr := zboxClient.UpdateUploadStatus(t, headers, updateBody)
	require.NoError(t, updateErr, "updateUploadStatus API call failed: %v", updateErr)
	require.Equal(t, 201, response.StatusCode(), "updateUploadStatus API call failed: %v", response.String())
	require.NotNil(t, transcodingEntity, "transcodingEntity is nil")
	t.Logf("transcodingEntity: %v", *transcodingEntity)

	t.Logf("Metadata post completed; fileName : %v, size_in_bytes; %v", fileName, actualSize)

}

func verifyTranscodedFile(t *test.SystemTest, allocationID, remotpath string, splitKey *model.SplitKey) error {
	t.Logf("verifying transcoded file at remote path: %s", remotpath)

	// List directory contents
	allocationObj, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return fmt.Errorf("failed to get allocation from SDK: %w", err)
	}

	listResult, err := allocationObj.ListDir(remotpath)
	if err != nil {
		return fmt.Errorf("failed to list directory: %w", err)
	}

	if listResult == nil {
		return fmt.Errorf("list result is nil")
	}

	// Log directory information
	t.Logf("Transcoded Directory Info: path: %s", remotpath)
	t.Logf("Directory Type: %s, Name: %s, Path: %s, Size: %d bytes, Hash: %s, FileMetaHash: %v, ActualSize: %d, NumFiles: %d, Directory Children: %d", 
	listResult.Type, listResult.Name, listResult.Path, listResult.Size, listResult.Hash, listResult.FileMetaHash, listResult.ActualSize, listResult.NumFiles, len(listResult.Children))

	// Calculate total size and print all fields for each file
	var totalSize int64
	if len(listResult.Children) > 0 {
		t.Logf("Files in directory (%d):", len(listResult.Children))
		for i, child := range listResult.Children {
			// Only count files (type "f"), not directories
			if child.Type == "f" {
				totalSize += child.Size
			}

			// Print all fields for each file using JSON marshaling to show all available fields
			childJSON, err := json.MarshalIndent(child, "    ", "  ")
			if err == nil {
				t.Logf("  File %d (all fields):", i+1)
				t.Logf("    %s", string(childJSON))
			} else {
				// Fallback: print basic fields if JSON marshaling fails
				t.Logf("  File %d:", i+1)
				t.Logf("    Type: %s", child.Type)
				t.Logf("    Name: %s", child.Name)
				t.Logf("    Path: %s", child.Path)
				t.Logf("    Size: %d bytes", child.Size)
				t.Logf("    Hash: %s", child.Hash)
				t.Logf("    LookupHash: %s", child.LookupHash)
				t.Logf("    MimeType: %s", child.MimeType)
				t.Logf("    NumBlocks: %d", child.NumBlocks)
			}
		}
	} else {
		t.Logf("No files found in directory")
	}

	t.Logf("Total directory size: %d bytes", totalSize)

	return nil
}

func UploadFileBlobber(t *test.SystemTest, wallet zcncrypto.Wallet, allocationObj *sdk.Allocation, workDir, remotePath, fileName string) error {

	wg := &sync.WaitGroup{}
	wg.Add(1)
	cb := &StatusBar{wg: wg, t: t}

	localSlice := []string{filepath.Join(workDir, fileName)}
	fileNameSlice := []string{fileName}
	thumbnailSlice := []string{""}
	encrypts := []bool{false}
	chunkNumbers := []int{0}
	remoteSlice := []string{remotePath}
	isUpdate := []bool{false}
	isWebstreaming := []bool{false}

	err := allocationObj.StartMultiUpload(workDir, localSlice, fileNameSlice, thumbnailSlice, encrypts, chunkNumbers, remoteSlice, isUpdate, isWebstreaming, cb)
	if err != nil {
		return errors.New("upload failed: " + err.Error())
	}

	wg.Wait()
	return nil
}

// StatusBar is to check status of any operation
type StatusBar struct {
	wg      *sync.WaitGroup
	success bool
	err     error
	t       *test.SystemTest

	totalBytes     int
	completedBytes int
	callback       func(totalBytes int, completedBytes int, err string)
}

var jsCallbackMutex sync.Mutex

// Started for statusBar
func (s *StatusBar) Started(allocationID, filePath string, op int, totalBytes int) {
	s.totalBytes = totalBytes
	if s.callback != nil {
		jsCallbackMutex.Lock()
		defer jsCallbackMutex.Unlock()
		s.callback(s.totalBytes, s.completedBytes, "")
	}
}

// InProgress for statusBar
func (s *StatusBar) InProgress(allocationID, filePath string, op int, completedBytes int, todo_name_var []byte) {
	s.completedBytes = completedBytes
	if s.callback != nil {
		jsCallbackMutex.Lock()
		defer jsCallbackMutex.Unlock()
		s.callback(s.totalBytes, s.completedBytes, "")
	}
}

// Completed for statusBar
func (s *StatusBar) Completed(allocationID, filePath string, filename string, mimetype string, size int, op int) {
	s.success = true

	s.completedBytes = s.totalBytes
	if s.callback != nil {
		jsCallbackMutex.Lock()
		defer jsCallbackMutex.Unlock()
		s.callback(s.totalBytes, s.completedBytes, "")
	}

	defer s.wg.Done()
}

// Error for statusBar
func (s *StatusBar) Error(allocationID string, filePath string, op int, err error) {
	s.success = false
	s.err = err
	defer func() {
		if r := recover(); r != nil {
			s.t.Errorf("Recovered in statusBar Error")
		}
	}()
	s.t.Errorf("Error in file operation." + err.Error())
	if s.callback != nil {
		jsCallbackMutex.Lock()
		defer jsCallbackMutex.Unlock()
		s.callback(s.totalBytes, s.completedBytes, err.Error())
	}
	s.wg.Done()
}

// RepairCompleted when repair is completed
func (s *StatusBar) RepairCompleted(filesRepaired int) {
	s.wg.Done()
}
