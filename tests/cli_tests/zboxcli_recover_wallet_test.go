package cli_tests

import (
	"github.com/0chain/system_test/internal/cli/cli_utils"
	"github.com/stretchr/testify/assert"
	"strings"
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

	t.Run("Recover wallet invalid mnemonic", func(t *testing.T) {
		t.Parallel()
		inValidMnemonic := "floor crop best weasel suit solid gown filter kitten loan absent noodle nation potato planet demise online ten affair rich panel rent sell"
		output, err := recoverWalletFromMnemonic(t, configPath, inValidMnemonic)
		assert.NotNil(t, err)
		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Error: Invalid mnemonic", output[0])
	})
}

func recoverWalletFromMnemonic(t *testing.T, configPath string, mnemonic string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet recoverwallet --silent --wallet " + strings.Replace(t.Name(), "/", "-", -1) + "_wallet.json" + " --configDir ./config --config " + configPath + " --mnemonic \"" + mnemonic + "\"")
}
