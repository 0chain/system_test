package cli_tests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEthRegisterAccount(t *testing.T) {
	t.Parallel()

	t.Run("Register ethereum account in local key storage", func(t *testing.T) {
		t.Parallel()

		deleteDefaultAccountInStorage(t)

		output, err := registerAccountInStorage(
			t,
			"tag volcano eight thank tide danger coast health above argue embrace heavy",
			"password",
		)

		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Imported account 0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947")
	})
}

func TestEthListAccounts(t *testing.T) {
	t.Parallel()

	t.Run("List Ethereum account registered in local key chain", func(t *testing.T) {
		t.Parallel()

		output, err := registerAccountInStorage(
			t,
			"tag volcano eight thank tide danger coast health above argue embrace heavy",
			"password",
		)

		require.NoError(t, err)
		require.Contains(t, output[len(output)-1], "Imported account 0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947")

		output, err = listAccounts(t)

		deleteDefaultAccountInStorage(t)
		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947")
	})
}
