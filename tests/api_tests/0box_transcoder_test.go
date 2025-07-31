package api_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
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

	t.RunSequentially("Transcode MP4 file with web mode", func(t *test.SystemTest) {
		// Test MP4 transcoding
		testTranscodeFile(t, headers, allocation.ID, "sample.mp4", "mp4", "web", true, wallet)
	})

	t.RunSequentially("Transcode HLS file with web mode", func(t *test.SystemTest) {
		// Test HLS transcoding
		testTranscodeFile(t, headers, allocation.ID, "sample.m3u8", "hls", "web", false, wallet)
	})

	t.RunSequentially("Transcode AVI file with web mode", func(t *test.SystemTest) {
		// Test AVI transcoding
		testTranscodeFile(t, headers, allocation.ID, "sample.avi", "avi", "web", true, wallet)
	})

}

// testTranscodeFile is a helper function to test transcoding of a specific file
func testTranscodeFile(t *test.SystemTest, headers map[string]string, allocationID, fileName, fileFormat, mode string, createThumbnail bool, wallet *model.ZboxWallet) {
	startTime := time.Now()
	response, err := transcodeFile(t, headers, allocationID, fileName, fileFormat, mode, createThumbnail, wallet)
	transcodeTime := time.Since(startTime)
	t.Logf("Transcode time for %s file (%s mode): %v", fileFormat, mode, transcodeTime)

	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode(), "Transcode failed for %s file. Output: [%v]", fileFormat, response.String())
}

// transcodeFile makes the actual API call to the transcoder endpoint
func transcodeFile(t *test.SystemTest, headers map[string]string, allocationID, fileName, fileFormat, mode string, createThumbnail bool, wallet *model.ZboxWallet) (*resty.Response, error) {
	// Read the test file
	filePath := filepath.Join("test_files", fileName)
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read test file %s: %w", filePath, err)
	}

	// Prepare headers for transcoder request
	transcodeHeaders := make(map[string]string)
	for k, v := range headers {
		transcodeHeaders[k] = v
	}
	
	// Add transcoder-specific headers
	transcodeHeaders["Content-Type"] = "application/octet-stream"
	transcodeHeaders["X-File-Name"] = fileName
	transcodeHeaders["X-Allocation-Id"] = allocationID
	transcodeHeaders["X-Mode"] = mode
	transcodeHeaders["remote_path"] = fmt.Sprintf("/%s", fileName)
	transcodeHeaders["mnemonic"] = wallet.Mnemonic // Use the wallet's mnemonic
	
	if createThumbnail {
		transcodeHeaders["create_thumbnail"] = "true"
	} else {
		transcodeHeaders["create_thumbnail"] = "false"
	}

	// Make the API call using the zbox client's HTTP client
	// We'll construct the URL manually since we can't access the private field
	url := fmt.Sprintf("%s/v2/transcode", parsedConfig.ZboxUrl)
	
	resp, err := zboxClient.HttpClient.R().
		SetHeaders(transcodeHeaders).
		SetBody(fileData).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("transcode API call failed: %w", err)
	}

	return resp, nil
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
	
	// Get wallet for the transcoding calls
	wallet, _, err := zboxClient.GetWalletKeys(t, headers)
	if err != nil {
		b.Fatalf("Failed to get wallet: %v", err)
	}
	
	for i := 0; i < b.N; i++ {
		_, err := transcodeFile(t, headers, allocation.ID, "sample.mp4", "mp4", "web", true, wallet)
		if err != nil {
			b.Fatalf("Transcode failed: %v", err)
		}
	}
} 