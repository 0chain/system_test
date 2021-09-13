package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/cli/cli_utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestMultiSigWalletRegisterAndBalanceOperations(t *testing.T) {

	t.Run("Wallet Creation should fail when threshold is 0", func(t *testing.T) {
		t.Parallel()
		numSigners, threshold := 3, 0

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold)
		assert.NotNil(t, err)

		// This is true for the first round only since the wallet is created here
		assert.Equal(t, "ZCN wallet created!!", output[1])
		assert.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		assert.Equal(t, "Read pool created successfully", output[3])

		assert.NotEqual(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should not fail when 0 < threshold <= num-signers", func(t *testing.T) {
		t.Parallel()
		numSigners, threshold := 3, 2

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold)
		assert.Nil(t, err)

		// Total registered wallets = numsigners + 1 (additional wallet for multi-sig)
		msg := fmt.Sprintf("registering %d wallets ", numSigners+1)
		assert.Equal(t, msg, output[0])
		assert.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should not fail when threshold is equal to num-signers", func(t *testing.T) {
		t.Parallel()
		numSigners, threshold := 3, 3

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold)
		assert.Nil(t, err)

		// Total registered wallets = numsigners + 1 (additional wallet for multi-sig)
		msg := fmt.Sprintf("registering %d wallets ", numSigners+1)
		assert.Equal(t, msg, output[0])
		assert.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should fail when threshold is greater than num-signers", func(t *testing.T) {
		t.Parallel()
		numSigners, threshold := 3, 4

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold)
		assert.NotNil(t, err)

		assert.NotEqual(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])

		// Check the error when threshold is greater than signers
		errMsg := fmt.Sprintf(
			"Error: given threshold (%d) is too high. Threshold has to be less than or equal to numsigners (%d)",
			threshold, numSigners,
		)
		assert.Equal(t, errMsg, output[len(output)-1])
	})

	t.Run("Balance call fails due to zero ZCN in wallet", func(t *testing.T) {
		t.Parallel()

		numSigners, threshold := 3, 3

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold)
		assert.Nil(t, err)

		output, err = getBalance(t, configPath)
		if err == nil {
			t.Errorf("Expected initial getBalance operation to fail but was successful with output %v", strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Failed to get balance:", output[0])
	})

	// Since at least 2 test-cases create the multi-sig wallet, we can check it's contents
	t.Run("Get wallet outputs expected", func(t *testing.T) {
		t.Parallel()

		numSigners, threshold := 3, 3

		_, err := createMultiSigWallet(t, configPath, numSigners, threshold)
		assert.Nil(t, err)

		wallet, err := getWallet(t, configPath)

		if err != nil {
			t.Errorf("Error occured when retreiving wallet due to error: %v", err)
		}

		assert.NotNil(t, wallet.ClientId)
		assert.NotNil(t, wallet.ClientPublicKey)
		assert.NotNil(t, wallet.EncryptionPublicKey)
	})

	t.Run("Balance increases by 1 after faucet execution", func(t *testing.T) {
		t.Parallel()

		numSigners, threshold := 3, 3

		_, err := createMultiSigWallet(t, configPath, numSigners, threshold)
		assert.Nil(t, err)

		output, err := executeFaucet(t, configPath)

		if err != nil {
			t.Errorf("Faucet execution failed due to error: %v", err)
		}

		assert.Equal(t, 1, len(output))
		matcher := regexp.MustCompile("Execute faucet smart contract success with txn : {2}([a-f0-9]{64})$")

		assert.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")

		output, err = getBalance(t, configPath)

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1.000 ZCN \\([0-9.]+ USD\\)$"), output[0])
	})
}

func createMultiSigWallet(t *testing.T, cliConfigFilename string, numSigners, threshold int) ([]string, error) {
	return cli_utils.RunCommand(fmt.Sprintf(
		"./zwallet createmswallet --numsigners %d --threshold %d --silent --wallet %s --configDir ./config --config %s",
		numSigners, threshold,
		strings.Replace(t.Name(), "/", "-", -1)+"_wallet.json",
		cliConfigFilename))
}
