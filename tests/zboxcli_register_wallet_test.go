package tests

import (
	"github.com/0chain/system_test/internal/model"
	"github.com/0chain/system_test/internal/utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestWalletRegisterAndBalanceOperations(t *testing.T) {
	walletConfigFilename := "system_tests_wallet_" + utils.RandomAlphaNumericString(10) + ".json"
	output, err := utils.RegisterWallet(walletConfigFilename)

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
		err := utils.GetWallet(t, wallet, walletConfigFilename)

		if err != nil {
			t.Error(err)
		}

		assert.NotNil(t, wallet.Client_id)
		assert.NotNil(t, wallet.Client_public_key)
		assert.NotNil(t, wallet.Encryption_public_key)
	})

	t.Run("Zero Balance is returned", func(t *testing.T) {
		output, err := utils.GetBalance(walletConfigFilename)
		if err == nil {
			t.Error("Expected initial getBalance operation to fail but was successful with output " + strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Get balance failed.", output[0])
	})

	t.Run("Non-zero Balance is returned after faucet execution", func(t *testing.T) {
		t.Run("Execute Faucet", func(t *testing.T) {
			output, err := utils.ExecuteFaucet(walletConfigFilename)
			if err != nil {
				t.Error("Faucet execution failed : ", err)
			}

			assert.Equal(t, 1, len(output))
			matcher := regexp.MustCompile("Execute faucet smart contract success with txn : {2}([a-f0-9]{64})$")
			assert.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")
			txnId := matcher.FindAllStringSubmatch(output[0], 1)[0][1]

			t.Run("Faucet Execution Verified", func(t *testing.T) {
				output, err = utils.VerifyTransaction(walletConfigFilename, txnId)
				if err != nil {
					t.Error("Faucet verification failed : ", err)
				}

				assert.Equal(t, 1, len(output))
				assert.Equal(t, "Transaction verification success", output[0])
				t.Log("Faucet executed successful with txn id [" + txnId + "]")
			})
		})

		output, err = utils.GetBalance(walletConfigFilename)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1 \\([0-9.]+ USD\\)$"), output[0])
	})
}
