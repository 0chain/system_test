package api_tests

import (
	"strconv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"strings"
	"go.uber.org/zap"

	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/errgo.v2/errors"

	coreClient "github.com/0chain/gosdk/core/client"
)


func Test0BoxTranscoder(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	
	headers := zboxClient.NewZboxHeaders(client.X_APP_VULT)
	Teardown(t, headers)
	walletInput := NewTestWallet()
	_, response, err := zboxClient.CreateWallet(t, headers, walletInput)
	require.NoError(t, err)
	require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	testSetup.Logf("Wallet created: %v", walletInput)

	allocationInput := NewTestAllocation()
	allocation, response, err := zboxClient.CreateAllocation(t, headers, allocationInput)
	require.NoError(t, err)
	require.Equal(t, 201, response.StatusCode(), "Failed to create allocation. Output: [%v]", response.String())
	require.NotEmpty(t, allocation.ID)
	testSetup.Logf("Allocation Created ID: %v", allocation.ID)

	wallet, response, err := zboxClient.GetWalletKeys(t, headers)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	require.Equal(t, walletInput["name"], wallet.Name)
	require.Equal(t, walletInput["mnemonic"], wallet.Mnemonic)
	require.Equal(t, headers["X-App-Client-Key"], wallet.PublicKey)
	require.Equal(t, walletInput["description"], wallet.Description)

	// Generate split wallet/key and share to 0box-server before transcoding
	jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	oldHeaders := zvaultClient.NewZvaultHeaders(jwtToken.JwtToken)

	var generateWalletResponse *model.GenerateWalletResponse
	generateWalletResponse, response, err = zvaultClient.GenerateSplitWallet(t, oldHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	response, err = zvaultClient.GenerateSplitKey(t, generateWalletResponse.ClientID, oldHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	keys, response, err := zvaultClient.GetKeys(t, generateWalletResponse.ClientID, oldHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	require.Len(t, keys.Keys, 1)

	var sharedKey *model.SplitKey
	for _, k := range keys.Keys {
		if k.SharedTo == "65b32a635cffb6b6f3c73f09da617c29569a5f690662b5be57ed0d994f234335" {
			sharedKey = k
			break
		}
	}
	require.NotNil(t, sharedKey, "Shared key to 0box-server not found")


	// Share the generated split key to user "0box-server"
	response, err = zvaultClient.ShareWallet(t, "0box-server", sharedKey.PublicKey, oldHeaders)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

	// Run transcode subtests (they will run in parallel because RunSequentially now delegates to Run)
	t.RunSequentially("Transcode MP4 file with web mode", func(t *test.SystemTest) {
		// Test MP4 transcoding
		transcodeFile(t, headers, allocation.ID, sharedKey, "sample.mp4", "web")
		verifyTranscodedFile(t, allocation.ID, "/.transcoded/sample.mp4", "sample.mp4", sharedKey)
	})

	t.RunSequentially("Transcode AVI file with web mode", func(t *test.SystemTest) {
		// Test AVI transcoding
		transcodeFile(t, headers, allocation.ID, sharedKey, "sample.avi", "web")
		verifyTranscodedFile(t, allocation.ID, "/.transcoded/sample.mp4", "sample.mp4", sharedKey)

	})

	t.RunSequentially("Transcode MOV file with web mode", func(t *test.SystemTest) {
		// Test MOV transcoding
		transcodeFile(t, headers, allocation.ID, sharedKey, "sample.mov", "web")
		verifyTranscodedFile(t, allocation.ID, "/.transcoded/sample.mp4", "sample.mp4", sharedKey)
	})

}


// transcodeFile makes the actual API call to the transcoder endpoint
func transcodeFile(t *test.SystemTest, headers map[string]string, allocationID string, splitKey *model.SplitKey, fileName, mode string) {
	

	// TODO: compute a real lookup hash from the uploaded file. Use a
	// placeholder random string for now.
	lookupHash := fmt.Sprintf("TODO-random-%d", time.Now().UnixNano())

	w := zcncrypto.Wallet{
		ClientID:   splitKey.ClientID,
		ClientKey:  splitKey.PublicKey,
		Keys: []zcncrypto.KeyPair{ {
			PublicKey: splitKey.PublicKey,
			PrivateKey: splitKey.PrivateKey,
		}},
		PeerPublicKey: splitKey.PeerPublicKey,
		IsSplit: true,
	}
	coreClient.AddWallet(w)
	defer coreClient.RemoveWallet(w.ClientID)

	allocation, err := sdk.GetAllocation(allocationID, w.Keys[0].PublicKey)
	require.NoError(t, err, "Failed to get allocation from SDK")
	
	// Use the test_files_small folder as the workDir for uploads
	workDir := filepath.Join(".", "tests", "api_tests", "test_files_small")
	UploadFileBlobber(t, w, allocation, workDir, "/", fileName)

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

	_, _, metaErr := zboxClient.CreateMetadata(t, headers, metaBody)
	require.NoError(t, metaErr, "createMetadata API call failed: %v", metaErr)

	// Update upload status
	updateBody := map[string]string{
		"remotepath":  fmt.Sprintf("/%s", fileName),
		"status":      "1",
		"lookup_hash": lookupHash,
	}
	_, _, updateErr := zboxClient.UpdateUploadStatus(t, headers, updateBody)
	require.NoError(t, updateErr, "updateUploadStatus API call failed: %v", updateErr)

	t.Logf("Metadata post completed; fileName : %v, size_in_bytes; %v", fileName, actualSize)

}

func verifyTranscodedFile(t *test.SystemTest, allocationID, remotpath, fileName string, splitKey *model.SplitKey) error {
	w := zcncrypto.Wallet{
		ClientID:   splitKey.ClientID,
		ClientKey:  splitKey.PublicKey,
		Keys: []zcncrypto.KeyPair{ {
			PublicKey: splitKey.PublicKey,
			PrivateKey: splitKey.PrivateKey,
		}},
		PeerPublicKey: splitKey.PeerPublicKey,
		IsSplit: true,
	}
	coreClient.AddWallet(w)
	defer coreClient.RemoveWallet(w.ClientID)

	// download the file
	allocationObj, err := sdk.GetAllocation(allocationID, w.Keys[0].PublicKey)
	require.NoError(t, err, "Failed to get allocation from SDK")

	wg := &sync.WaitGroup{}
	wg.Add(1)
	cb := StatusBar{wg: wg, t: t}

	prefix := strings.Join([]string{allocationID, fileName}, "_") + "_"
	downloadPath, err := os.MkdirTemp("", prefix)
	require.NoError(t, err, "failed to create temporary download directory")

	err = allocationObj.DownloadFile(downloadPath, remotpath, true, &cb, true)
	require.NoError(t, err, "download file failed: %v", err)
	wg.Wait()
	require.True(t, cb.success, "download file reported unsuccessful")

	fi, statErr := os.Stat(filepath.Join(downloadPath, fileName))
	require.NoError(t, statErr, "failed to stat downloaded file: %v", statErr)
	f, openErr := os.Open(filepath.Join(downloadPath, fileName))
	require.NoError(t, openErr, "failed to open downloaded file: %v", openErr)
	t.Log("Transcoded File Info: ", "file info", zap.String("path", filepath.Join(downloadPath, fileName)), zap.Int64("size", fi.Size()), zap.String("mode", fi.Mode().String()))
	_ = f.Close()

	// defer cleanup of downloaded file
	defer func() {
		err := os.RemoveAll(downloadPath)
		if err != nil {
			t.Error("error removing downloaded path", zap.String("path", downloadPath), zap.Error(err))
		}
	}()
	return nil
}

// Benchmark transcoding performance
func BenchmarkTranscoder(b *testing.B) {
	t := test.NewSystemTest(&testing.T{})
	
	headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
	
	// Create wallet and allocation for benchmark
	err := Create0boxTestWallet(t, headers)
	if err != nil {
		b.Fatalf("Failed to create test wallet: %v", err)
	}

	allocationInput := NewTestAllocation()
	allocation, response, err := zboxClient.CreateAllocation(t, headers, allocationInput)
	if err != nil || response.StatusCode() != 201 {
		b.Fatalf("Failed to create allocation: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transcodeFile(t, headers, allocation.ID, nil, "sample.mp4", "web")
	}
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
	t 	 	*test.SystemTest

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

