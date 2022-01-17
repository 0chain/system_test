package cli_tests

import (
	"fmt"
	"strings"
	"testing"

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

		output, err = listAccounts(t)

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
