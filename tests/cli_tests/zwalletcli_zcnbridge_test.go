package cli_tests

import (
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
func listAccounts(t *testing.T) ([]string, error) {
	t.Logf("List Ethereum account registered in local key chain in HOME (~/.zcn) folder")

	cmd := "./zwallet " + "eth-list-accounts"

	return cliutils.RunCommandWithoutRetry(cmd)
}

func deleteDefaultAccountInStorage(t *testing.T, address string) {
	keyDir := path.Join(getConfigDir(t), "wallets")

	err := filepath.Walk(keyDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			require.NoError(t, err)
			if strings.Contains(path, address) {
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
