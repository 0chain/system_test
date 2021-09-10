package clitests

import (
	"encoding/json"
	zbox_models "github.com/0chain/system_test/internal/zbox/model"
	zbox_utils "github.com/0chain/system_test/internal/zbox/utils"

	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestWalletRegisterAndBalanceOperations(t *testing.T) {
	walletConfigFilename := "wallet_TestWalletRegisterAndBalanceOperations_" + zbox_utils.RandomAlphaNumericString(10) + ".json"

	t.Run("CLI output matches expected", func(t *testing.T) {
		output, err := registerWallet(walletConfigFilename, configPath)
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
		wallet, err := getWallet(t, walletConfigFilename, configPath)

		if err != nil {
			t.Errorf("Error occured when retreiving wallet due to error: %v", err)
		}

		assert.NotNil(t, wallet.ClientId)
		assert.NotNil(t, wallet.ClientPublicKey)
		assert.NotNil(t, wallet.EncryptionPublicKey)
	})

	t.Run("Balance call fails due to zero ZCN in wallet", func(t *testing.T) {
		output, err := getBalance(walletConfigFilename, configPath)
		if err == nil {
			t.Errorf("Expected initial getBalance operation to fail but was successful with output %v", strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Get balance failed.", output[0])
	})

	t.Run("Balance of 1 is returned after faucet execution", func(t *testing.T) {
		output, err := executeFaucet(walletConfigFilename, configPath)

		if err != nil {
			t.Errorf("Faucet execution failed due to error: %v", err)
		}

		assert.Equal(t, 1, len(output))
		matcher := regexp.MustCompile("Execute faucet smart contract success with txn : {2}([a-f0-9]{64})$")

		assert.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")

		txnId := matcher.FindAllStringSubmatch(output[0], 1)[0][1]
		output, err = verifyTransaction(walletConfigFilename, configPath, txnId)

		if err != nil {
			t.Errorf("Faucet verification failed due to error: %v", err)
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Transaction verification success", output[0])

		t.Log("Faucet executed successful with txn id [" + txnId + "]")

		output, err = getBalance(walletConfigFilename, configPath)

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1 \\([0-9.]+ USD\\)$"), output[0])
	})
}

func registerWallet(walletConfigFilename string, cliConfigFilename string) ([]string, error) {
	return zbox_utils.RunCommand("./zbox register --silent --wallet " + walletConfigFilename + " --configDir ./config --config " + cliConfigFilename)
}

func getBalance(walletConfigFilename string, cliConfigFilename string) ([]string, error) {
	return zbox_utils.RunCommand("./zwallet getbalance --silent --wallet " + walletConfigFilename + " --configDir ./config --config " + cliConfigFilename)
}

func getWallet(t *testing.T, walletConfigFilename string, cliConfigFilename string) (*zbox_models.Wallet, error) {
	output, err := zbox_utils.RunCommand("./zbox getwallet --json --silent --wallet " + walletConfigFilename + " --configDir ./config --config " + cliConfigFilename)

	if err != nil {
		return nil, err
	}

	assert.Equal(t, 1, len(output))

	var wallet *zbox_models.Wallet

	err = json.Unmarshal([]byte(output[0]), &wallet)
	if err != nil {
		t.Errorf("failed to unmarshal the result into wallet")
		return nil, err
	}

	return wallet, err
}

func executeFaucet(walletConfigFilename string, cliConfigFilename string) ([]string, error) {
	return zbox_utils.RunCommand("./zwallet faucet --methodName pour --tokens 1 --input {} --silent --wallet " + walletConfigFilename + " --configDir ./config --config " + cliConfigFilename)
}

func verifyTransaction(walletConfigFilename string, cliConfigFilename string, txn string) ([]string, error) {
	return zbox_utils.RunCommand("./zwallet verify --silent --wallet " + walletConfigFilename + " --hash " + txn + " --configDir ./config --config " + cliConfigFilename)
}
