package tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/config"
	"github.com/0chain/system_test/internal/model"
	"github.com/0chain/system_test/internal/utils"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestMultiSigWalletRegisterAndBalanceOperations(t *testing.T) {
	msWalletConfigFilename := "wallet_TestMultiSigWalletRegisterAndBalanceOperations_" + utils.RandomAlphaNumericString(10) + ".json"
	cliConfigFilename := "config_TestMultiSigWalletRegisterAndBalanceOperations_" + utils.RandomAlphaNumericString(10) + ".yaml"

	systemTestConfig := GetConfig(t)

	cliConfig := model.Config{
		BlockWorker:             *systemTestConfig.DNSHostName + "/dns",
		SignatureScheme:         "bls0chain",
		MinSubmit:               50,
		MinConfirmation:         50,
		ConfirmationChainLength: 3,
		MaxTxnQuery:             5,
		QuerySleepTime:          5,
	}
	err := config.WriteConfig(cliConfigFilename, cliConfig)
	if err != nil {
		t.Errorf("Error when writing CLI config: %v", err)
	}

	t.Run("Wallet Creation should fail when threshold is 0", func(t *testing.T) {
		numSigners, threshold := 3, 0

		output, err := utils.CreateMultiSigWallet(msWalletConfigFilename, cliConfigFilename, numSigners, threshold)
		assert.NotNil(t, err)

		// This is true for the first round only since the wallet is created here
		assert.Equal(t, "ZCN wallet created!!", output[1])
		assert.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		assert.Equal(t, "Read pool created successfully", output[3])

		assert.NotEqual(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should not fail when 0 < threshold <= num-signers", func(t *testing.T) {
		numSigners, threshold := 3, 2

		output, err := utils.CreateMultiSigWallet(msWalletConfigFilename, cliConfigFilename, numSigners, threshold)
		assert.Nil(t, err)

		// Total registered wallets = numsigners + 1 (additional wallet for multi-sig)
		msg := fmt.Sprintf("registering %d wallets ", numSigners+1)
		assert.Equal(t, msg, output[0])
		assert.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should not fail when threshold is equal to num-signers", func(t *testing.T) {
		numSigners, threshold := 3, 3

		output, err := utils.CreateMultiSigWallet(msWalletConfigFilename, cliConfigFilename, numSigners, threshold)
		assert.Nil(t, err)

		// Total registered wallets = numsigners + 1 (additional wallet for multi-sig)
		msg := fmt.Sprintf("registering %d wallets ", numSigners+1)
		assert.Equal(t, msg, output[0])
		assert.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should fail when threshold is greater than num-signers", func(t *testing.T) {
		numSigners, threshold := 3, 4

		output, err := utils.CreateMultiSigWallet(msWalletConfigFilename, cliConfigFilename, numSigners, threshold)
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
		output, err := utils.GetBalance(msWalletConfigFilename, cliConfigFilename)
		if err == nil {
			t.Errorf("Expected initial getBalance operation to fail but was successful with output %v", strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Get balance failed.", output[0])
	})

	// Since at least 2 test-cases create the multi-sig wallet, we can check it's contents
	t.Run("Get wallet outputs expected", func(t *testing.T) {
		wallet, err := utils.GetWallet(t, msWalletConfigFilename, cliConfigFilename)

		if err != nil {
			t.Errorf("Error occured when retreiving wallet due to error: %v", err)
		}

		assert.NotNil(t, wallet.Client_id)
		assert.NotNil(t, wallet.Client_public_key)
		assert.NotNil(t, wallet.Encryption_public_key)
	})

	t.Run("Balance increases by 1 after faucet execution", func(t *testing.T) {
		testFaucet(t, msWalletConfigFilename, cliConfigFilename)
	})
}
