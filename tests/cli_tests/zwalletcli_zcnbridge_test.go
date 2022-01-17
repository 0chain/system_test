package cli_tests

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
