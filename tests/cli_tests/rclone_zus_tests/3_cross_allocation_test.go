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

func TestRcloneZusCrossAllocationTransfer(testSetup *testing.T) {
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

	t.RunSequentiallyWithTimeout("Cross-wallet transfer via local staging", 15*time.Minute, func(t *test.SystemTest) {
		// Setup two separate wallets with allocations
		allocA := setupRcloneAllocation(t, "rclone_cross_a")
		allocB := setupRcloneAllocation(t, "rclone_cross_b")
		rcloneConfA := createRcloneConfig(t, allocA, "rclone_cross_a_wallet.json")
		rcloneConfB := createRcloneConfig(t, allocB, "rclone_cross_b_wallet.json")

		// Create test files
		srcDir := t.TempDir()
		require.Nil(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("cross-wallet test file 1"), 0644))
		require.Nil(t, os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("cross-wallet test file 2 with more content"), 0644))

		// Step 1: Upload to wallet A
		output, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s copy %s zus-test:cross_test --config %s -v", rcloneBinary, srcDir, rcloneConfA),
			3, 2*time.Minute)
		require.Nil(t, err, "upload to wallet A failed: %s", strings.Join(output, "\n"))

		// Verify files on wallet A
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s lsf zus-test:cross_test --config %s", rcloneBinary, rcloneConfA),
			3, 30*time.Second)
		require.Nil(t, err, "lsf wallet A failed: %s", strings.Join(output, "\n"))
		listing := strings.Join(output, "\n")
		require.Contains(t, listing, "file1.txt", "file1.txt not found on wallet A")
		require.Contains(t, listing, "file2.txt", "file2.txt not found on wallet A")

		// Step 2: Download from wallet A to local staging
		stagingDir := t.TempDir()
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s copy zus-test:cross_test %s --config %s", rcloneBinary, stagingDir, rcloneConfA),
			3, 2*time.Minute)
		require.Nil(t, err, "download from wallet A failed: %s", strings.Join(output, "\n"))

		// Verify staging files exist
		require.FileExists(t, filepath.Join(stagingDir, "file1.txt"))
		require.FileExists(t, filepath.Join(stagingDir, "file2.txt"))

		// Step 3: Upload from staging to wallet B
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s copy %s zus-test:from_wallet_a --config %s -v", rcloneBinary, stagingDir, rcloneConfB),
			3, 2*time.Minute)
		require.Nil(t, err, "upload to wallet B failed: %s", strings.Join(output, "\n"))

		// Step 4: Verify files on wallet B
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s lsf zus-test:from_wallet_a --config %s", rcloneBinary, rcloneConfB),
			3, 30*time.Second)
		require.Nil(t, err, "lsf wallet B failed: %s", strings.Join(output, "\n"))
		listingB := strings.Join(output, "\n")
		require.Contains(t, listingB, "file1.txt", "file1.txt not found on wallet B")
		require.Contains(t, listingB, "file2.txt", "file2.txt not found on wallet B")

		// Step 5: Download from wallet B and verify content integrity
		verifyDir := t.TempDir()
		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s copy zus-test:from_wallet_a %s --config %s", rcloneBinary, verifyDir, rcloneConfB),
			3, 2*time.Minute)
		require.Nil(t, err, "download from wallet B failed: %s", strings.Join(output, "\n"))

		content1, err := os.ReadFile(filepath.Join(verifyDir, "file1.txt"))
		require.Nil(t, err)
		require.Equal(t, "cross-wallet test file 1", string(content1), "file1.txt content mismatch after cross-wallet transfer")

		content2, err := os.ReadFile(filepath.Join(verifyDir, "file2.txt"))
		require.Nil(t, err)
		require.Equal(t, "cross-wallet test file 2 with more content", string(content2), "file2.txt content mismatch after cross-wallet transfer")

		t.Log("Cross-wallet transfer verified: files transferred from wallet A to wallet B with content integrity")
	})
}
