package tests

import (
	"encoding/json"
	"github.com/0chain/system_test/internal/model"
	"github.com/0chain/system_test/internal/utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestWalletRegisterAndBalanceOperations(t *testing.T) {
	output, err := registerWallet()

	t.Run("CLI output matches expected", func(t *testing.T) {
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 4, len(output))
		assert.Equal(t, "ZCN wallet created", output[0])
		assert.Equal(t, "Creating related read pool for storage smart-contract...", output[1])
		assert.Equal(t, "Read pool created successfully", output[2])
		assert.Equal(t, "Wallet registered", output[3])
	})

	var wallet model.Wallet
	t.Run("Get wallet outputs expected", func(t *testing.T) {
		err := getWallet(t, wallet)

		if err != nil {
			t.Error(err)
		}

		assert.NotNil(t, wallet.Client_id)
		assert.NotNil(t, wallet.Client_public_key)
		assert.NotNil(t, wallet.Encryption_public_key)
	})

	t.Run("Zero Balance is returned", func(t *testing.T) {
		output, err := getBalance()
		if err == nil {
			t.Error("Expected initial getBalance operation to fail but was successful with output " + strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Get balance failed.", output[0])
	})

	t.Run("Non-zero Balance is returned after faucet execution", func(t *testing.T) {
		output, err := executeFaucet()
		if err != nil {
			t.Error("Faucet execution failed : ", err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Execute faucet smart contract success with txn :  ([a-f0-9]{64})$"), output[0], "Faucet execution output did not match expected")

		output, err = getBalance()
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1 \\([0-9.]+ USD\\)$"), output[0])
	})
}

func registerWallet() ([]string, error) {
	return utils.RunCommand("./zbox register --silent")
}

func getBalance() ([]string, error) {
	return utils.RunCommand("./zwallet getbalance --silent")
}

func getWallet(t *testing.T, wallet model.Wallet) error {
	output, err := utils.RunCommand("./zbox getwallet --json --silent")

	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 1, len(output))

	return json.Unmarshal([]byte(output[0]), &wallet)
}

func executeFaucet() ([]string, error) {
	return utils.RunCommand("./zwallet faucet --methodName pour --tokens 1 --input {} --silent")
}
