package cli_tests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestRecoverWallet(t *testing.T) {
	t.Parallel()
	//t.Run("parallel", func(t *testing.T) {
	t.Run("Recover wallet valid mnemonic", func(t *testing.T) {
		t.Parallel()
		validMnemonic := "pull floor crop best weasel suit solid gown" +
			" filter kitten loan absent noodle nation potato planet demise" +
			" online ten affair rich panel rent sell"

		output, err := recoverWalletFromMnemonic(t, configPath, validMnemonic)

		require.Nil(t, err, "error occurred recovering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 5)
		require.Equal(t, "Wallet recovered!!", output[len(output)-1])
	})

	//FIXME: POSSIBLE BUG: Blank wallet created if mnemonic is invalid (same issue in missing mnemonic test)
	t.Run("Recover wallet invalid mnemonic", func(t *testing.T) {
		t.Parallel()
		inValidMnemonic := "floor crop best weasel suit solid gown" +
			" filter kitten loan absent noodle nation potato planet demise" +
			" online ten affair rich panel rent sell"

		output, err := recoverWalletFromMnemonic(t, configPath, inValidMnemonic)

		require.NotNil(t, err, "expected error to occur recovering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 5)
		require.Equal(t, "No wallet in path"+
			"  ./config/TestRecoverWallet-Recover_wallet_invalid_mnemonic_wallet.json found."+
			" Creating wallet...", output[0])
		require.Equal(t, "ZCN wallet created!!", output[1])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		require.Equal(t, "Read pool created successfully", output[3])
		require.Equal(t, "Error: Invalid mnemonic", output[4])
	})

	t.Run("Recover wallet no mnemonic", func(t *testing.T) {
		t.Parallel()

		output, err := cliutils.RunCommand("./zwallet recoverwallet --silent " +
			"--wallet " + escapedTestName(t) + "_wallet.json" + " " +
			"--configDir ./config --config " + configPath)

		require.NotNil(t, err, "expected error to occur recovering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 5)
		require.Equal(t, "No wallet in path  ./config/TestRecoverWallet-Recover_wallet_no_mnemonic_wallet.json found. Creating wallet...", output[0])
		require.Equal(t, "ZCN wallet created!!", output[1])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		require.Equal(t, "Read pool created successfully", output[3])
		require.Equal(t, "Error: Mnemonic not provided", output[4])
	})
	//})
}

func recoverWalletFromMnemonic(t *testing.T, configPath, mnemonic string) ([]string, error) {
	return cliutils.RunCommand("./zwallet recoverwallet " +
		"--silent --wallet " + escapedTestName(t) + "_wallet.json" + " " +
		"--configDir ./config --config " + configPath + " --mnemonic \"" + mnemonic + "\"")
}
