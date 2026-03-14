package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	rcloneBinary   = "../rclone-zus"
	zboxBinary     = "../zbox"
	zwalletBin     = "../zwallet"
	zboxConfigFile = "zbox_config.yaml"
	cliConfigDir   = "../config"
)

// createRcloneConfig creates a rclone.conf and a dedicated config directory
// containing wallet.json and config.yaml as required by the rclone-zus backend.
// Returns the path to rclone.conf.
func createRcloneConfig(t *test.SystemTest, allocationID, walletFile string) string {
	// rclone-zus expects config_dir to contain wallet.json and config.yaml (hardcoded names)
	rcloneCfgDir := t.TempDir()

	// Copy wallet to config_dir as wallet.json
	srcWallet := filepath.Join(cliConfigDir, walletFile)
	walletBytes, err := os.ReadFile(srcWallet)
	require.Nil(t, err, "failed to read wallet %s", srcWallet)
	err = os.WriteFile(filepath.Join(rcloneCfgDir, "wallet.json"), walletBytes, 0600)
	require.Nil(t, err, "failed to write wallet.json")

	// Copy zbox_config.yaml to config_dir as config.yaml
	srcConfig := filepath.Join(cliConfigDir, zboxConfigFile)
	configBytes, err := os.ReadFile(srcConfig)
	require.Nil(t, err, "failed to read config %s", srcConfig)
	err = os.WriteFile(filepath.Join(rcloneCfgDir, "config.yaml"), configBytes, 0600)
	require.Nil(t, err, "failed to write config.yaml")

	// Resolve to absolute path to avoid CWD-relative issues
	absCfgDir, err := filepath.Abs(rcloneCfgDir)
	require.Nil(t, err)

	// Write rclone.conf with correct field names for the zus backend
	confPath := filepath.Join(t.TempDir(), "rclone.conf")
	content := fmt.Sprintf(`[zus-test]
type = zus
allocation_id = %s
config_dir = %s
`, allocationID, absCfgDir)
	err = os.WriteFile(confPath, []byte(content), 0600)
	require.Nil(t, err, "failed to write rclone.conf")
	return confPath
}

// setupRcloneAllocation creates a wallet, funds it, and creates an allocation.
// Returns the allocation ID.
func setupRcloneAllocation(t *test.SystemTest, testName string) string {
	walletFile := testName + "_wallet.json"

	// Create wallet
	output, err := cliutils.RunCommand(t,
		fmt.Sprintf("%s create-wallet --silent --wallet %s --configDir %s --config %s",
			zwalletBin, walletFile, cliConfigDir, zboxConfigFile),
		3, 30*time.Second)
	require.Nil(t, err, "wallet creation failed: %s", strings.Join(output, "\n"))

	// Verify wallet file was created on disk
	walletPath := filepath.Join(cliConfigDir, walletFile)
	require.FileExists(t, walletPath, "wallet file not created at %s", walletPath)

	// Fund wallet via faucet (also registers it on-chain)
	output, err = cliutils.RunCommand(t,
		fmt.Sprintf("%s faucet --methodName pour --input '{Pay}' --tokens 100 --silent --wallet %s --configDir %s --config %s",
			zwalletBin, walletFile, cliConfigDir, zboxConfigFile),
		3, 60*time.Second)
	require.Nil(t, err, "faucet failed: %s", strings.Join(output, "\n"))

	// Wait for balance to be confirmed before creating allocation
	for attempt := 0; attempt < 10; attempt++ {
		balOutput, _ := cliutils.RunCommand(t,
			fmt.Sprintf("%s getbalance --silent --wallet %s --configDir %s --config %s",
				zwalletBin, walletFile, cliConfigDir, zboxConfigFile),
			1, 15*time.Second)
		balStr := strings.Join(balOutput, "\n")
		if strings.Contains(balStr, "Balance:") || strings.Contains(balStr, "zcn") {
			t.Logf("Wallet funded: %s", balStr)
			break
		}
		t.Logf("Balance not yet visible (attempt %d/10), waiting 3s...", attempt+1)
		time.Sleep(3 * time.Second)
	}

	// Create allocation
	output, err = cliutils.RunCommand(t,
		fmt.Sprintf("%s newallocation --lock 5 --size 5368709120 --data 2 --parity 2 --silent --wallet %s --configDir %s --config %s",
			zboxBinary, walletFile, cliConfigDir, zboxConfigFile),
		3, 2*time.Minute)
	require.Nil(t, err, "failed to create allocation: %s", strings.Join(output, "\n"))

	// Extract allocation ID from "Allocation created: <hash>" line.
	// Must not match wallet client_id printed in SDK init logs (same 64-hex format).
	re := regexp.MustCompile(`[a-f0-9]{64}`)
	for _, line := range output {
		if strings.Contains(strings.ToLower(line), "allocation created") || strings.Contains(line, "Allocation created") {
			match := re.FindString(line)
			if match != "" {
				t.Logf("Created allocation: %s", match)
				return match
			}
		}
	}
	// Fallback: last 64-hex match (allocation hash is printed after wallet IDs)
	var lastMatch string
	for _, line := range output {
		if match := re.FindString(line); match != "" {
			lastMatch = match
		}
	}
	if lastMatch != "" {
		t.Logf("Created allocation (fallback): %s", lastMatch)
		return lastMatch
	}
	t.Fatalf("could not extract allocation ID from: %s", strings.Join(output, "\n"))
	return ""
}

func TestRcloneZusBasicOperations(testSetup *testing.T) {
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

	t.RunSequentiallyWithTimeout("Upload and download a file", 10*time.Minute, func(t *test.SystemTest) {
		allocID := setupRcloneAllocation(t, "rclone_upload_download")
		rcloneConf := createRcloneConfig(t, allocID, "rclone_upload_download_wallet.json")

		// Create test file
		tmpFile := filepath.Join(t.TempDir(), "test_upload.txt")
		err := os.WriteFile(tmpFile, []byte("hello from rclone-zus test"), 0644)
		require.Nil(t, err)

		// Upload
		output, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s copy %s zus-test:/ --config %s -v", rcloneBinary, tmpFile, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err, "upload failed: %s", strings.Join(output, "\n"))

		// List remote
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s ls zus-test:/ --config %s", rcloneBinary, rcloneConf),
			3, time.Minute)
		require.Nil(t, err, "ls failed: %s", strings.Join(output, "\n"))
		outputStr := strings.Join(output, "\n")
		require.Contains(t, outputStr, "test_upload.txt", "uploaded file not found in listing")

		// Download
		downloadDir := t.TempDir()
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s copy zus-test:/ %s --config %s -v", rcloneBinary, downloadDir, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err, "download failed: %s", strings.Join(output, "\n"))

		// Verify content
		downloaded, err := os.ReadFile(filepath.Join(downloadDir, "test_upload.txt"))
		require.Nil(t, err, "downloaded file not found")
		require.Equal(t, "hello from rclone-zus test", string(downloaded))

		t.Log("Upload -> Download round-trip verified")
	})

	t.RunSequentiallyWithTimeout("List directories and check size", 5*time.Minute, func(t *test.SystemTest) {
		allocID := setupRcloneAllocation(t, "rclone_list_size")
		rcloneConf := createRcloneConfig(t, allocID, "rclone_list_size_wallet.json")

		// Upload file to a subdirectory
		tmpFile := filepath.Join(t.TempDir(), "nested.txt")
		err := os.WriteFile(tmpFile, []byte("nested content"), 0644)
		require.Nil(t, err)

		_, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s copy %s zus-test:/subdir/ --config %s", rcloneBinary, tmpFile, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err)

		// Test lsd
		output, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s lsd zus-test:/ --config %s", rcloneBinary, rcloneConf),
			3, time.Minute)
		require.Nil(t, err, "lsd failed")
		require.Contains(t, strings.Join(output, "\n"), "subdir")

		// Test size
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s size zus-test:/ --config %s", rcloneBinary, rcloneConf),
			3, time.Minute)
		require.Nil(t, err, "size failed")
		outputStr := strings.Join(output, "\n")
		require.Contains(t, outputStr, "Total objects: 1")
		require.Contains(t, outputStr, "Total size:")

		t.Log("lsd and size commands verified")
	})

	t.RunSequentiallyWithTimeout("Tree shows directory structure", 5*time.Minute, func(t *test.SystemTest) {
		allocID := setupRcloneAllocation(t, "rclone_tree")
		rcloneConf := createRcloneConfig(t, allocID, "rclone_tree_wallet.json")

		// Create nested directory structure
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "level1")
		require.Nil(t, os.MkdirAll(subDir, 0755))
		require.Nil(t, os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0644))
		require.Nil(t, os.WriteFile(filepath.Join(subDir, "child.txt"), []byte("child"), 0644))

		// Upload tree
		_, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s copy %s zus-test:/tree/ --config %s", rcloneBinary, tmpDir, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err)

		// Verify tree
		output, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s tree zus-test:/ --config %s", rcloneBinary, rcloneConf),
			3, time.Minute)
		require.Nil(t, err, "tree failed")
		outputStr := strings.Join(output, "\n")
		require.Contains(t, outputStr, "root.txt")
		require.Contains(t, outputStr, "child.txt")

		t.Log("tree command verified")
	})
}
