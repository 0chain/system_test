package tests

import (
	"github.com/0chain/system_test/internal/config"
	"github.com/0chain/system_test/internal/model"
	"github.com/0chain/system_test/internal/utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestWalletRegisterAndBalanceOperations(t *testing.T) {
	walletConfigFilename := "wallet_TestWalletRegisterAndBalanceOperations_" + utils.RandomAlphaNumericString(10) + ".json"
	cliConfigFilename := "config_TestWalletRegisterAndBalanceOperations_" + utils.RandomAlphaNumericString(10) + ".yaml"

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

	t.Run("CLI output matches expected", func(t *testing.T) {
		output, err := utils.RegisterWallet(walletConfigFilename, cliConfigFilename)
		if err != nil {
			t.Errorf("An error occured registering a wallet due to error: %v", err)
		}

		assert.Equal(t, 4, len(output))
		assert.Equal(t, "ZCN wallet created", output[0])
		assert.Equal(t, "Creating related read pool for storage smart-contract...", output[1])
		assert.Equal(t, "Read pool created successfully", output[2])
		assert.Equal(t, "Wallet registered", output[3])
	})

	t.Run("Get wallet outputs expected", func(t *testing.T) {
		wallet, err := utils.GetWallet(t, walletConfigFilename, cliConfigFilename)

		if err != nil {
			t.Errorf("Error occured when retreiving wallet due to error: %v", err)
		}

		assert.NotNil(t, wallet.Client_id)
		assert.NotNil(t, wallet.Client_public_key)
		assert.NotNil(t, wallet.Encryption_public_key)
	})

	t.Run("Balance call fails due to zero ZCN in wallet", func(t *testing.T) {
		output, err := utils.GetBalance(walletConfigFilename, cliConfigFilename)
		if err == nil {
			t.Errorf("Expected initial getBalance operation to fail but was successful with output %v", strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Get balance failed.", output[0])
	})

	t.Run("Balance of 1 is returned after faucet execution", func(t *testing.T) {
		testFaucet(t, walletConfigFilename, cliConfigFilename)
	})
}

func testFaucet(t *testing.T, walletConfigFilename string, cliConfigFilename string) {

	t.Run("Execute Faucet", func(t *testing.T) {
		output, err := utils.ExecuteFaucet(walletConfigFilename, cliConfigFilename)
		if err != nil {
			t.Errorf("Faucet execution failed due to error: %v", err)
		}

		assert.Equal(t, 1, len(output))
		matcher := regexp.MustCompile("Execute faucet smart contract success with txn : {2}([a-f0-9]{64})$")
		assert.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")
		txnId := matcher.FindAllStringSubmatch(output[0], 1)[0][1]

		t.Run("Faucet Execution Verified", func(t *testing.T) {
			output, err = utils.VerifyTransaction(walletConfigFilename, cliConfigFilename, txnId)
			if err != nil {
				t.Errorf("Faucet verification failed due to error: %v", err)
			}

			assert.Equal(t, 1, len(output))
			assert.Equal(t, "Transaction verification success", output[0])
			t.Log("Faucet executed successful with txn id [" + txnId + "]")
		})
	})

	output, err := utils.GetBalance(walletConfigFilename, cliConfigFilename)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 1, len(output))

	assert.Regexp(t, regexp.MustCompile(`Balance: 1 \([0-9.]+ USD\)$`), output[0])
}
