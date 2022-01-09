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

func TestEthRegisterAccount(t *testing.T) {
	t.Parallel()

	t.Run("Register ethereum account in local key storage", func(t *testing.T) {
		t.Parallel()

		keyDir := path.Join(GetConfigDir(t), "wallets")

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

		output, err := ethRegisterAccount(t,
			"tag volcano eight thank tide danger coast health above argue embrace heavy",
			"password",
		)

		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Imported account 0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947")
	})
}

// eth-register-account
func ethRegisterAccount(t *testing.T, mnemonic, password string) ([]string, error) {
	t.Logf("register ethereum account using mnemonic and protected with password in HOME (~/.zcn) folder")

	cmd := "./zwallet " + "eth-register-account" +
		" --password " + password +
		" --mnemonic " + fmt.Sprintf("\"%s\"", mnemonic)

	return cliutils.RunCommandWithoutRetry(cmd)
}

func GetConfigDir(t *testing.T) string {
	var configDir string
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	configDir = home + "/.zcn"
	return configDir
}
