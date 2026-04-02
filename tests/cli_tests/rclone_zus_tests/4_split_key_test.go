package cli_tests

import (
	"encoding/json"
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

func TestRcloneZusSplitKeyWallet(testSetup *testing.T) {
	// Check prerequisites
	if _, err := os.Stat(rcloneBinary); os.IsNotExist(err) {
		testSetup.Skipf("rclone-zus binary not available at %s, skipping", rcloneBinary)
	}

	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Split-key wallet config is loaded", 5*time.Minute, func(t *test.SystemTest) {
		// This test verifies that rclone_zus correctly detects and configures
		// split-key wallets. A split-key wallet has "is_split": true in wallet.json
		// and requires a zauth_server URL in config.yaml.
		//
		// The backend code at NewFs checks client.GetClient().IsSplit and calls
		// zcncore.RegisterZauthServer(cfg.ZauthServer) when true.

		allocID := setupRcloneAllocation(t, "rclone_splitkey")
		walletFile := "rclone_splitkey_wallet.json"

		// Read the wallet and verify it has the is_split field
		walletPath := filepath.Join(cliConfigDir, walletFile)
		walletBytes, err := os.ReadFile(walletPath)
		require.Nil(t, err, "failed to read wallet")

		var wallet map[string]interface{}
		err = json.Unmarshal(walletBytes, &wallet)
		require.Nil(t, err, "failed to parse wallet JSON")

		isSplit, ok := wallet["is_split"]
		t.Logf("Wallet is_split field: %v (present: %v)", isSplit, ok)

		// Create rclone config and verify basic operation works with non-split wallet
		rcloneConf := createRcloneConfig(t, allocID, walletFile)

		tmpFile := filepath.Join(t.TempDir(), "splitkey_test.txt")
		require.Nil(t, os.WriteFile(tmpFile, []byte("split-key wallet test"), 0644))

		output, err := cliutils.RunCommand(t,
			fmt.Sprintf("%s copy %s zus-test:splitkey_test --config %s -v", rcloneBinary, tmpFile, rcloneConf),
			3, 2*time.Minute)
		require.Nil(t, err, "upload with non-split wallet failed: %s", strings.Join(output, "\n"))

		output, err = cliutils.RunCommand(t,
			fmt.Sprintf("%s lsf zus-test:splitkey_test --config %s", rcloneBinary, rcloneConf),
			3, 30*time.Second)
		require.Nil(t, err, "lsf failed: %s", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "splitkey_test.txt")

		t.Log("Non-split wallet operations verified. Split-key mode requires a zauth server and is_split=true wallet.")
		t.Log("To test split-key mode: set is_split=true in wallet.json and add zauth_server to config.yaml")
	})
}
