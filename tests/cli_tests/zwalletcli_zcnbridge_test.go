package cli_tests

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// eth-register-account
func registerAccountInStorage(t *testing.T, mnemonic, password string) ([]string, error) {
	t.Logf("Register ethereum account using mnemonic and protected with password in HOME (~/.zcn) folder")

	cmd := "./zwallet " + "eth-register-account" +
		" --password " + password +
		" --mnemonic " + fmt.Sprintf("\"%s\"", mnemonic)

	return cliutils.RunCommandWithoutRetry(cmd)
}

// eth-register-account
func listAccounts(t *testing.T) ([]string, error) {
	t.Logf("List Ethereum account registered in local key chain in HOME (~/.zcn) folder")

	cmd := "./zwallet " + "eth-list-accounts"

	return cliutils.RunCommandWithoutRetry(cmd)
}

func deleteDefaultAccountInStorage(t *testing.T) {
	keyDir := path.Join(getConfigDir(t), "wallets")

	err := filepath.Walk(keyDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			require.NoError(t, err)
			if strings.Contains(path, "c49926c4124cee1cba0ea94ea31a6c12318df947") {
				err = os.Remove(path)
				require.NoError(t, err)
			}
		}
		return nil
	})

	require.NoError(t, err)
}

func getConfigDir(t *testing.T) string {
	var configDir string
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	configDir = home + "/.zcn"
	return configDir
}
