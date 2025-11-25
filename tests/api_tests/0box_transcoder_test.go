package api_tests

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

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
	const maxPrice = math.MaxUint64 / 100
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
		Lock: 1000000000,
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
		lookupHash := fmt.Sprintf("TODO-random-%d", time.Now().UnixNano())
		remotepath := "/sample.mp4"
		transcodeFile(t, headers, allocationID, sharedKey, filepath.Base(remotepath), "web", lookupHash)

		// wait before first verification (20 seconds) then verify up to 3 times
		time.Sleep(sleepTime)
		var verr error
		for i := 0; i < 3; i++ {
			t.Logf("verify attempt %d for sample.mp4", i+1)
			verr = verifyTranscodedFile(t, allocationID, fmt.Sprintf("/.transcoded/%s.mp4", lookupHash), fmt.Sprintf("%s.mp4", lookupHash), sharedKey)
			if verr == nil {
				break
			}
			t.Logf("verification attempt %d failed: %v, sleeping %s before retry", i+1, verr, sleepTime)
			time.Sleep(sleepTime)
		}
		require.NoError(t, verr, "verification failed after 3 attempts")
	})

	t.RunWithTimeout("Transcode AVI file with web mode", timeout, func(t *test.SystemTest) {
		// Test AVI transcoding
		lookupHash := fmt.Sprintf("TODO-random-%d", time.Now().UnixNano())
		remotepath := "/sample.avi"
		transcodeFile(t, headers, allocationID, sharedKey, filepath.Base(remotepath), "web", lookupHash)

		// wait before first verification (20 seconds) then verify up to 3 times
		time.Sleep(sleepTime)
		var verr error
		for i := 0; i < 3; i++ {
			t.Logf("verify attempt %d for sample.avi", i+1)
			verr = verifyTranscodedFile(t, allocationID, fmt.Sprintf("/.transcoded/%s.mp4", lookupHash), fmt.Sprintf("%s.mp4", lookupHash), sharedKey)
			if verr == nil {
				break
			}
			t.Logf("verification attempt %d failed: %v, sleeping %s before retry", i+1, verr, sleepTime)
			time.Sleep(sleepTime)
		}
		require.NoError(t, verr, "verification failed after 3 attempts")
	})

	t.RunWithTimeout("Transcode MOV file with web mode", timeout, func(t *test.SystemTest) {
		// Test MOV transcoding
		lookupHash := fmt.Sprintf("TODO-random-%d", time.Now().UnixNano())
		remotepath := "/sample.mov"
		transcodeFile(t, headers, allocationID, sharedKey, filepath.Base(remotepath), "web", lookupHash)

		// wait before first verification (20 seconds) then verify up to 3 times
		time.Sleep(sleepTime)
		var verr error
		for i := 0; i < 3; i++ {
			t.Logf("verify attempt %d for sample.mov", i+1)
			verr = verifyTranscodedFile(t, allocationID, fmt.Sprintf("/.transcoded/%s.mp4", lookupHash), fmt.Sprintf("%s.mp4", lookupHash), sharedKey)
			if verr == nil {
				break
			}
			t.Logf("verification attempt %d failed: %v, sleeping %s before retry", i+1, verr, sleepTime)
				time.Sleep(sleepTime)
		}
		require.NoError(t, verr, "verification failed after 3 attempts")
	})

}

// transcodeFile makes the actual API call to the transcoder endpoint
func transcodeFile(t *test.SystemTest, headers map[string]string, allocationID string, splitKey *model.SplitKey, fileName, mode, lookupHash string) {
	t.Logf("transcoding file: %s with mode: %s", fileName, mode)

	t.Logf("splitKey fields: ClientID=%s, PublicKey=%s, PrivateKey=%s, UserID=%s, PeerPublicKey=%s", 
		splitKey.ClientID, splitKey.PublicKey, splitKey.PrivateKey, splitKey.UserID, splitKey.PeerPublicKey)

	// w := zcncrypto.Wallet{
	// 	ClientID:  splitKey.ClientID,
	// 	ClientKey: splitKey.PublicKey,
	// 	Keys: []zcncrypto.KeyPair{{
	// 		PublicKey:  splitKey.PublicKey,
	// 		PrivateKey: splitKey.PrivateKey,
	// 	}},
	// 	PeerPublicKey: splitKey.PeerPublicKey,
	// 	IsSplit:       true,
	// }
	// coreClient.AddWallet(w)
	// defer coreClient.RemoveWallet(w.Keys[0].PublicKey)

	allocation, err := sdk.GetAllocation(allocationID)
	require.NoError(t, err, "Failed to get allocation from SDK")

	// Use the test_files_small folder as the workDir for uploads
	curDir, err := os.Getwd()
	require.NoError(t, err, "Unable to get working directory")
	workDir := filepath.Join(curDir, "test_files_small")
	err = UploadFileBlobber(t, *sdkWallet, allocation, workDir, "/", fileName)
	require.NoError(t, err, "upload failed: %v", err)

	// Determine remote name and file size for metadata
	fi, err := os.Stat(filepath.Join(workDir, fileName))
	require.NoError(t, err)
	actualSize := fi.Size()

	// Metadata request
	metaBody := map[string]string{
		"remotepath":    fmt.Sprintf("/%s", fileName),
		"user_id":       splitKey.UserID,
		"mode":          mode,
		"allocation_id": allocationID,
		"file_name":     fileName,
		"file_size":     strconv.FormatInt(actualSize, 10),
		"lookup_hash":   lookupHash,
	}

	transcodingData, response, metaErr := zboxClient.CreateMetadata(t, headers, metaBody)
	require.NoError(t, metaErr, "createMetadata API call failed: %v", metaErr)
	require.Equal(t, 201, response.StatusCode(), "createMetadata API call failed: %v", response.String())
	require.NotNil(t, transcodingData, "transcodingData is nil")
	t.Logf("transcodingData: %v", *transcodingData)

	// Update upload status
	updateBody := map[string]string{
		"remotepath":  fmt.Sprintf("/%s", fileName),
		"status":      "1",
		"lookup_hash": lookupHash,
	}
	transcodingData, response, updateErr := zboxClient.UpdateUploadStatus(t, headers, updateBody)
	require.NoError(t, updateErr, "updateUploadStatus API call failed: %v", updateErr)
	require.Equal(t, 201, response.StatusCode(), "updateUploadStatus API call failed: %v", response.String())
	require.NotNil(t, transcodingData, "transcodingData is nil")
	t.Logf("transcodingData: %v", *transcodingData)

	t.Logf("Metadata post completed; fileName : %v, size_in_bytes; %v", fileName, actualSize)

}

func verifyTranscodedFile(t *test.SystemTest, allocationID, remotpath, fileName string, splitKey *model.SplitKey) error {
	t.Logf("verifying transcoded file: %s at remote path: %s", fileName, remotpath)
	// w := zcncrypto.Wallet{
	// 	ClientID:  splitKey.ClientID,
	// 	ClientKey: splitKey.PublicKey,
	// 	Keys: []zcncrypto.KeyPair{{
	// 		PublicKey:  splitKey.PublicKey,
	// 		PrivateKey: splitKey.PrivateKey,
	// 	}},
	// 	PeerPublicKey: splitKey.PeerPublicKey,
	// 	IsSplit:       true,
	// }
	// coreClient.AddWallet(w)
	// defer coreClient.RemoveWallet(w.Keys[0].PublicKey)

	// download the file
	allocationObj, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return fmt.Errorf("failed to get allocation from SDK: %w", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	cb := StatusBar{wg: wg, t: t}

	downloadPath := t.TempDir()

	if err := allocationObj.DownloadFile(downloadPath, remotpath, true, &cb, true); err != nil {
		return fmt.Errorf("download file failed: %w", err)
	}
	wg.Wait()
	if !cb.success {
		if cb.err != nil {
			return fmt.Errorf("download reported unsuccessful: %w", cb.err)
		}
		return fmt.Errorf("download reported unsuccessful")
	}

	fi, statErr := os.Stat(filepath.Join(downloadPath, fileName))
	if statErr != nil {
		return fmt.Errorf("failed to stat downloaded file: %w", statErr)
	}
	f, openErr := os.Open(filepath.Join(downloadPath, fileName))
	if openErr != nil {
		return fmt.Errorf("failed to open downloaded file: %w", openErr)
	}
	_ = f.Close()

	t.Logf("Transcoded File Info: file info: %s, size: %d, mode: %s", filepath.Join(downloadPath, fileName), fi.Size(), fi.Mode().String())
	_ = os.RemoveAll(downloadPath)

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
