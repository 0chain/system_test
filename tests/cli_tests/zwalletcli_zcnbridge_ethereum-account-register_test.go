package cli_tests

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

const (
	address  = "c49926c4124cee1cba0ea94ea31a6c12318df947"
	mnemonic = "tag volcano eight thank tide danger coast health above argue embrace heavy"
	password = "password"
)

func TestEthRegisterAccount(t *testing.T) {
	t.Parallel()

	zwallet := func(cmd, mnemonic, password string) ([]string, error) {
		t.Logf("Register ethereum account using mnemonic and protected with password in HOME (~/.zcn) folder")

		run := fmt.Sprintf(
			"./zwallet %s --password %s --mnemonic \"%s\"",
			cmd,
			password,
			mnemonic,
		)

		return cliutils.RunCommand(t, run, 3, time.Second*15)
	}

	zwalletList := func(cmd string) ([]string, error) {
		t.Logf("List ethereum account registered in local key chain in HOME (~/.zcn) folder")

		run := fmt.Sprintf("./zwallet %s", cmd)

		return cliutils.RunCommandWithoutRetry(run)
	}

	t.Run("Register ethereum account in local key storage", func(t *testing.T) {
		t.Parallel()

		output, err := deleteAndCreateAccount(t, zwallet)

		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Imported account 0x"+address)
	})

	t.Run("List ethereum account registered in local key storage", func(t *testing.T) {
		t.Parallel()

		output, err := deleteAndCreateAccount(t, zwallet)

		require.NoError(t, err)
		require.Contains(t, output[len(output)-1], "Imported account 0x"+address)

		output, err = zwalletList("eth-list-accounts")

		deleteDefaultAccountInStorage(t, address)
		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], address)
	})
}

func deleteAndCreateAccount(t *testing.T, zwallet func(cmd string, mnemonic string, password string) ([]string, error)) ([]string, error) {
	deleteDefaultAccountInStorage(t, address)

	output, err := zwallet(
		"eth-register-account",
		mnemonic,
		password,
	)

	return output, err
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
