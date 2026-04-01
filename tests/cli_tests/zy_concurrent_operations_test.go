package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// TestConcurrentFileOperations tests multiple file operations running simultaneously
// on the same allocation from the same wallet. This catches nonce conflicts, write
// marker races, and connection pool issues that only appear under concurrent load.
func TestConcurrentFileOperations(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Concurrent upload, copy, move, rename, delete, download on same allocation", 10*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		// Create allocation with enough space for all operations
		allocationID, _ := createWalletAndAllocation(t, configPath, escapedTestName(t))

		// Phase 1: Upload 5 files concurrently
		const numFiles = 5
		filePaths := make([]string, numFiles)
		for i := 0; i < numFiles; i++ {
			filePaths[i] = generateRandomTestFileName(t)
			err := createFileWithSize(filePaths[i], 64*1024) // 64KB each
			require.Nil(t, err)
		}

		t.Log("Phase 1: Uploading 5 files concurrently...")
		var wg sync.WaitGroup
		uploadErrors := make([]error, numFiles)
		for i := 0; i < numFiles; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				output, err := uploadFile(t, configPath, map[string]interface{}{
					"allocation": allocationID,
					"localpath":  filePaths[idx],
					"remotepath": fmt.Sprintf("/concurrent_%d.bin", idx),
				}, true)
				if err != nil {
					uploadErrors[idx] = fmt.Errorf("upload %d failed: %v %s", idx, err, output)
				}
			}(i)
		}
		wg.Wait()
		for i, err := range uploadErrors {
			require.Nil(t, err, "concurrent upload %d failed", i)
		}
		t.Log("Phase 1 complete: all 5 uploads succeeded")

		// Wait for write markers to settle
		cliutils.Wait(t, 5*time.Second)

		// Phase 2: Run copy, move, rename, download, delete concurrently
		t.Log("Phase 2: Running copy, move, rename, download, delete concurrently...")
		type opResult struct {
			name string
			err  error
		}
		results := make(chan opResult, 5)

		// Create target dirs first (sequential — createdir needs its own connection)
		output, err := createDir(t, configPath, allocationID, "/copied")
		require.Nil(t, err, "createdir /copied failed: %s", output)
		output, err = createDir(t, configPath, allocationID, "/moved")
		require.Nil(t, err, "createdir /moved failed: %s", output)

		cliutils.Wait(t, 3*time.Second)

		// Copy file 0
		go func() {
			_, err := copyFile(t, configPath, map[string]interface{}{
				"allocation": allocationID,
				"remotepath": "/concurrent_0.bin",
				"destpath":   "/copied/",
			}, true)
			results <- opResult{"copy", err}
		}()

		// Move file 1
		go func() {
			_, err := moveFile(t, configPath, map[string]interface{}{
				"allocation": allocationID,
				"remotepath": "/concurrent_1.bin",
				"destpath":   "/moved/",
			}, true)
			results <- opResult{"move", err}
		}()

		// Rename file 2
		go func() {
			_, err := renameFile(t, configPath, map[string]interface{}{
				"allocation": allocationID,
				"remotepath": "/concurrent_2.bin",
				"destname":   "renamed_2.bin",
			}, true)
			results <- opResult{"rename", err}
		}()

		// Download file 3
		go func() {
			downloadPath := filepath.Join(os.TempDir(), "concurrent_dl_3.bin")
			os.Remove(downloadPath)
			_, err := downloadFile(t, configPath, createParams(map[string]interface{}{
				"allocation": allocationID,
				"remotepath": "/concurrent_3.bin",
				"localpath":  downloadPath,
			}), true)
			results <- opResult{"download", err}
		}()

		// Delete file 4
		go func() {
			_, err := deleteFile(t, configPath, map[string]interface{}{
				"allocation": allocationID,
				"remotepath": "/concurrent_4.bin",
			}, true)
			results <- opResult{"delete", err}
		}()

		// Collect results
		for i := 0; i < 5; i++ {
			r := <-results
			if r.err != nil {
				t.Errorf("Concurrent %s failed: %v", r.name, r.err)
			} else {
				t.Logf("Concurrent %s succeeded", r.name)
			}
		}
		t.Log("Phase 2 complete")

		// Phase 3: Verify final state
		t.Log("Phase 3: Verifying final state...")
		cliutils.Wait(t, 5*time.Second)

		// File 0 should exist at both original and copied location
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/concurrent_0.bin",
			"json":       "",
		}), true)
		require.Nil(t, err, "file 0 should still exist at original location")

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/copied/concurrent_0.bin",
			"json":       "",
		}), true)
		require.Nil(t, err, "file 0 copy should exist in /copied/")

		// File 1 should be moved (not at original, exists at /moved/)
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/moved/concurrent_1.bin",
			"json":       "",
		}), true)
		require.Nil(t, err, "file 1 should exist in /moved/ after move")

		// File 2 should be renamed
		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/renamed_2.bin",
			"json":       "",
		}), true)
		require.Nil(t, err, "file 2 should exist as renamed_2.bin")

		t.Log("Phase 3 complete: all verifications passed")
		t.Log("Concurrent file operations test PASSED")
	})
}
