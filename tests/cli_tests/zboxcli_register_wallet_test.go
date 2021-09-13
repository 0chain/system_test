package cli_tests

import (
	"encoding/json"
	"github.com/0chain/system_test/internal/cli/cli_model"
	"github.com/0chain/system_test/internal/cli/cli_utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestWalletRegisterAndBalanceOperations(t *testing.T) {
	t.Run("Register wallet outputs expected", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
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
		t.Parallel()
		_, err := registerWallet(t, configPath)
		if err != nil {
			t.Errorf("An error occured registering a wallet due to error: %v", err)
		}

		wallet, err := getWallet(t, configPath)

		if err != nil {
			t.Errorf("Error occured when retreiving wallet due to error: %v", err)
		}

		assert.NotNil(t, wallet.ClientId)
		assert.NotNil(t, wallet.ClientPublicKey)
		assert.NotNil(t, wallet.EncryptionPublicKey)
	})

	t.Run("Balance call fails due to zero ZCN in wallet", func(t *testing.T) {
		t.Parallel()
		_, err := registerWallet(t, configPath)
		if err != nil {
			t.Errorf("An error occured registering a wallet due to error: %v", err)
		}

		output, err := getBalance(t, configPath)
		if err == nil {
			t.Errorf("Expected initial getBalance operation to fail but was successful with output %v", strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Failed to get balance:", output[0])
	})

	t.Run("Balance of 1 is returned after faucet execution", func(t *testing.T) {
		t.Parallel()
		_, err := registerWallet(t, configPath)
		if err != nil {
			t.Errorf("An error occured registering a wallet due to error: %v", err)
		}

		output, err := executeFaucet(t, configPath)

		if err != nil {
			t.Errorf("Faucet execution failed due to error: %v", err)
		}

		assert.Equal(t, 1, len(output))
		matcher := regexp.MustCompile("Execute faucet smart contract success with txn : {2}([a-f0-9]{64})$")

		assert.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")

		txnId := matcher.FindAllStringSubmatch(output[0], 1)[0][1]
		output, err = verifyTransaction(t, configPath, txnId)

		if err != nil {
			t.Errorf("Faucet verification failed due to error: %v", err)
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Transaction verification success", output[0])

		t.Log("Faucet executed successful with txn id [" + txnId + "]")

		output, err = getBalance(t, configPath)

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1.000 ZCN \\([0-9.]+ USD\\)$"), output[0])
	})

}

func registerWallet(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cli_utils.RunCommand("./zbox register --silent --wallet " + strings.Replace(t.Name(), "/", "-", -1) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)

}

func getBalance(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet getbalance --silent --wallet " + strings.Replace(t.Name(), "/", "-", -1) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func getWallet(t *testing.T, cliConfigFilename string) (*cli_model.Wallet, error) {
	output, err := cli_utils.RunCommand("./zbox getwallet --json --silent --wallet " + strings.Replace(t.Name(), "/", "-", -1) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)

	if err != nil {
		return nil, err
	}

	assert.Equal(t, 1, len(output))

	var wallet *cli_model.Wallet

	err = json.Unmarshal([]byte(output[0]), &wallet)
	if err != nil {
		t.Errorf("failed to unmarshal the result into wallet")
		return nil, err
	}

	return wallet, err
}

func executeFaucet(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet faucet --methodName pour --tokens 1 --input {} --silent --wallet " + strings.Replace(t.Name(), "/", "-", -1) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func verifyTransaction(t *testing.T, cliConfigFilename string, txn string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet verify --silent --wallet " + escapedTestName(t) + ".json" + " --hash " + txn + " --configDir ./config --config " + cliConfigFilename)
}

func escapedTestName(t *testing.T) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "<", "-", ">", "-", "|", "-", "*", "-", "?", "-")
	return replacer.Replace(t.Name())
}
