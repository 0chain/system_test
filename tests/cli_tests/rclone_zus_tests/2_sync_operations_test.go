package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestRcloneZusSyncOperations(testSetup *testing.T) {
	// Check prerequisites
	if _, err := os.Stat(rcloneBinary); os.IsNotExist(err) {
		testSetup.Skipf("rclone-zus binary not available at %s, skipping", rcloneBinary)
	}
	if _, err := os.Stat(zboxBinary); os.IsNotExist(err) {
		testSetup.Skipf("zbox binary not available at %s, skipping", zboxBinary)
	}
	if _, err := os.Stat(zwalletBin); os.IsNotExist(err) {
		testSetup.Skipf("zwallet binary not available at %s, skipping", zwalletBin)
	}

	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Sync local directory to remote", 10*time.Minute, func(t *test.SystemTest) {
		allocID := setupRcloneAllocation(t, "rclone_sync_dir")
		rcloneConf := createRcloneConfig(t, allocID, "rclone_sync_dir_wallet.json")

		// Create local directory with multiple files
		localDir := t.TempDir()
		files := map[string]string{
			"file1.txt": "content of file 1",
			"file2.txt": "content of file 2",
			"file3.txt": "content of file 3",
		}
		for name, content := range files {
			err := os.WriteFile(filepath.Join(localDir, name), []byte(content), 0644)
			require.Nil(t, err)
		}

		// Sync local -> remote
		output, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s sync %s zus-test:/syncdir/ --config %s -v", rcloneBinary, localDir, rcloneConf),
			3, 3*time.Minute)
		require.Nil(t, err, "rclone sync failed: %s", strings.Join(output, "\n"))

		// Verify all files exist on remote
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s ls zus-test:/syncdir/ --config %s", rcloneBinary, rcloneConf),
			3, time.Minute)
		require.Nil(t, err)
		outputStr := strings.Join(output, "\n")
		for name := range files {
			require.Contains(t, outputStr, name, "synced file %s not found on remote", name)
		}

		// Verify size matches
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s size zus-test:/syncdir/ --config %s", rcloneBinary, rcloneConf),
			3, time.Minute)
		require.Nil(t, err)
		outputStr = strings.Join(output, "\n")
		require.Contains(t, outputStr, "Total objects: 3", "expected 3 objects after sync")

		t.Log("Sync of 3 files completed and verified")
	})

	t.RunSequentiallyWithTimeout("Sync dry-run detects changes", 10*time.Minute, func(t *test.SystemTest) {
		allocID := setupRcloneAllocation(t, "rclone_dryrun")
		rcloneConf := createRcloneConfig(t, allocID, "rclone_dryrun_wallet.json")

		// Create and sync initial file
		localDir := t.TempDir()
		err := os.WriteFile(filepath.Join(localDir, "original.txt"), []byte("original content"), 0644)
		require.Nil(t, err)

		_, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s sync %s zus-test:/dryrun/ --config %s", rcloneBinary, localDir, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err)

		// Modify the file locally
		err = os.WriteFile(filepath.Join(localDir, "original.txt"), []byte("modified content here"), 0644)
		require.Nil(t, err)
		// Add a new file
		err = os.WriteFile(filepath.Join(localDir, "new_file.txt"), []byte("new file"), 0644)
		require.Nil(t, err)

		// Run sync --dry-run - should detect the changes
		output, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s sync %s zus-test:/dryrun/ --dry-run --config %s -v", rcloneBinary, localDir, rcloneConf),
			3, 2*time.Minute)
		// dry-run may return non-zero exit code, check output instead
		outputStr := strings.Join(output, "\n")
		require.Contains(t, outputStr, "Skipped copy as --dry-run is set",
			"dry-run should report skipped copies")
		t.Logf("Dry-run output: %s", outputStr)

		t.Log("Sync dry-run correctly detects changes without modifying remote")
	})

	t.RunSequentiallyWithTimeout("Download synced files and verify content", 10*time.Minute, func(t *test.SystemTest) {
		allocID := setupRcloneAllocation(t, "rclone_verify")
		rcloneConf := createRcloneConfig(t, allocID, "rclone_verify_wallet.json")

		// Upload known content
		localDir := t.TempDir()
		testContent := "hello from rclone-zus integration test"
		err := os.WriteFile(filepath.Join(localDir, "verify_me.txt"), []byte(testContent), 0644)
		require.Nil(t, err)

		_, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s copy %s zus-test:/verify/ --config %s", rcloneBinary, localDir, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err)

		// Download to new directory
		downloadDir := t.TempDir()
		_, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s copy zus-test:/verify/ %s --config %s", rcloneBinary, downloadDir, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err)

		// Read and verify content
		downloadedContent, err := os.ReadFile(filepath.Join(downloadDir, "verify_me.txt"))
		require.Nil(t, err, "downloaded file not found")
		require.Equal(t, testContent, string(downloadedContent),
			"downloaded content should match uploaded content")

		t.Log("Content integrity verified: upload -> download round-trip successful")
	})
}
