package cli_tests

import (
	"github.com/0chain/system_test/internal/cli/cli_utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWalletRecoveryUsingMnemonics(t *testing.T) {

	t.Run("Recover wallet valid mnemonic", func(t *testing.T) {
		t.Parallel()
		validMnemonic := "pull floor crop best weasel suit solid gown filter kitten loan absent noodle nation potato planet demise online ten affair rich panel rent sell"
		output, err := recoverWalletFromMnemonic(t, configPath, validMnemonic)
		if err != nil {
			t.Errorf("An error occured recovering a wallet due to error: %v", err)
		}

		message := output[len(output)-1]
		assert.Equal(t, "Wallet recovered!!", message)
	})

	//FIXME: POSSIBLE BUG: Blank wallet created if mnemonic is invalid
	t.Run("Recover wallet invalid mnemonic", func(t *testing.T) {
		t.Parallel()
		inValidMnemonic := "floor crop best weasel suit solid gown filter kitten loan absent noodle nation potato planet demise online ten affair rich panel rent sell"
		output, err := recoverWalletFromMnemonic(t, configPath, inValidMnemonic)
		assert.NotNil(t, err)
		assert.Equal(t, 5, len(output))
		assert.Equal(t, "No wallet in path  ./config/TestWalletRecoveryUsingMnemonics-Recover_wallet_invalid_mnemonic_wallet.json found. Creating wallet...", output[0])
		assert.Equal(t, "ZCN wallet created!!", output[1])
		assert.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		assert.Equal(t, "Read pool created successfully", output[3])
		assert.Equal(t, "Error: Invalid mnemonic", output[4])
	})

	//FIXME: POSSIBLE BUG: Blank wallet created if mnemonic is missing
	t.Run("Recover wallet no mnemonic", func(t *testing.T) {
		t.Parallel()
		output, err := cli_utils.RunCommand("./zwallet recoverwallet --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + configPath)
		assert.NotNil(t, err)
		assert.Equal(t, 5, len(output))
		assert.Equal(t, "No wallet in path  ./config/TestWalletRecoveryUsingMnemonics-Recover_wallet_no_mnemonic_wallet.json found. Creating wallet...", output[0])
		assert.Equal(t, "ZCN wallet created!!", output[1])
		assert.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		assert.Equal(t, "Read pool created successfully", output[3])
		assert.Equal(t, "Error: Mnemonic not provided", output[4])
	})
}

func recoverWalletFromMnemonic(t *testing.T, configPath string, mnemonic string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet recoverwallet --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + configPath + " --mnemonic \"" + mnemonic + "\"")
}
