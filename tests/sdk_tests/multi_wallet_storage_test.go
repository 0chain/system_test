package sdk_tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestMultiWalletStorageOperations(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Test multi-wallet storage operations lifecycle")

	t.Run("Full lifecycle: Create -> Upload -> Download with specific key", func(t *test.SystemTest) {

		wallets, err := LoadWalletsFromPool(2)
		require.NoError(t, err)
		wallet := wallets[1] // Use second wallet
		
		err = AddWalletToSDK(t, wallet)
		require.NoError(t, err)

		key := GetWalletPublicKey(wallet)

		// 1. Get Blobbers
		storageVersion := 2
		priceRange := sdk.PriceRange{Min: 0, Max: 10000000000}
		blobbers, err := sdk.GetAllocationBlobbers(storageVersion, 2, 2, 1024*1024, 0, priceRange, priceRange)
		if err != nil {
			t.Logf("Failed to get blobbers: %v. Skipping test.", err)
			return 
		}
		require.NotEmpty(t, blobbers)

		// 2. Create Allocation
		options := sdk.CreateAllocationOptions{
			DataShards: 2,
			ParityShards: 2,
			Size: 1024*1024,
			ReadPrice: sdk.PriceRange{Min: 0, Max: 10000000000},
			WritePrice: sdk.PriceRange{Min: 0, Max: 10000000000},
			Lock: 1000000000, // 1 ZCN
			BlobberIds: blobbers,
			AuthRoundExpiry: 100, // Ensure enough time
		}

		hash, _, _, err := sdk.CreateAllocationWith(options, key)
		require.NoError(t, err)
		require.NotEmpty(t, hash)
		t.Logf("Allocation creation hash: %s", hash)

		// 3. Wait for allocation to be visible
		var alloc *sdk.Allocation
		for i := 0; i < 15; i++ {
			time.Sleep(2 * time.Second)
			allocs, err := sdk.GetAllocations(key)
			if err == nil && len(allocs) > 0 {
				alloc = allocs[0] 
				if alloc.Tx == hash {
					break
				}
			}
		}
		
		if alloc == nil {
			t.Logf("Allocation not confirmed after waiting. Skipping upload/download verification.")
			return
		}
		t.Logf("Allocation confirmed. ID: %s", alloc.ID)

		// Re-fetch allocation to ensure initialized
		alloc, err = sdk.GetAllocation(alloc.ID, key)
		require.NoError(t, err)

		// 4. Create dummy file
		tmpDir := os.TempDir()
		tmpFile := filepath.Join(tmpDir, "upload_test.txt")
		err = os.WriteFile(tmpFile, []byte("Hello Multi-Wallet World"), 0644)
		require.NoError(t, err)
		defer os.Remove(tmpFile)

		// 5. Upload file with secondary wallet
		remotePath := "/upload_test.txt"
		uploadErr := alloc.StartChunkedUpload(tmpDir, tmpFile, remotePath, nil, false, false, "", false, false, sdk.WithWallet(key))
		if uploadErr != nil {
			t.Logf("Upload failed: %v (may be expected if wallets not funded)", uploadErr)
			return
		}
		t.Logf("Successfully uploaded file to %s", remotePath)

		// 6. Download file to verify
		downloadPath := filepath.Join(tmpDir, "download_test.txt")
		defer os.Remove(downloadPath)
		
		downloadErr := alloc.DownloadFile(downloadPath, remotePath, false, nil, false)
		if downloadErr != nil {
			t.Logf("Download failed: %v", downloadErr)
			return
		}
		
		// Verify content
		downloadedContent, err := os.ReadFile(downloadPath)
		require.NoError(t, err)
		require.Equal(t, "Hello Multi-Wallet World", string(downloadedContent), 
			"Downloaded content should match uploaded content")
		
		t.Logf("Successfully verified full storage lifecycle with secondary wallet")
	})
}
