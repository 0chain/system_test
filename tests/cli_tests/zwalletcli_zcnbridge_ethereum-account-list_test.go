package cli_tests

import (
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestEthListAccounts(t *testing.T) {
	t.Parallel()

	t.Run("List Ethereum account registered in local key chain", func(t *testing.T) {
		t.Parallel()

		output, err := ethRegisterAccount(t,
			"tag volcano eight thank tide danger coast health above argue embrace heavy",
			"password",
		)
		require.NoError(t, err)
		require.Contains(t, output[len(output)-1], "Imported account 0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947")

		output, err = ethListAccounts(t)

		DeleteDefaultAccount(t)
		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947")
	})
}

// eth-register-account
func ethListAccounts(t *testing.T) ([]string, error) {
	t.Logf("List Ethereum account registered in local key chain in HOME (~/.zcn) folder")

	cmd := "./zwallet " + "eth-list-accounts"

	return cliutils.RunCommandWithoutRetry(cmd)
}
