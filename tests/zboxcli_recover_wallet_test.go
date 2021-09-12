package tests

import (
	"github.com/0chain/system_test/internal/config"
	"github.com/0chain/system_test/internal/model"
	"github.com/0chain/system_test/internal/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWalletRecoveryUsingMnemonics(t *testing.T) {
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

	validMnemonic := "pull floor crop best weasel suit solid gown filter kitten loan absent noodle nation potato planet demise online ten affair rich panel rent sell"
	inValidMnemonic := "floor crop best weasel suit solid gown filter kitten loan absent noodle nation potato planet demise online ten affair rich panel rent sell"

	output, err := utils.RecoverWalletFromMnemonic(walletConfigFilename, cliConfigFilename, validMnemonic)
	if err != nil {
		t.Errorf("An error occured recovering a wallet due to error: %v", err)
	}

	message := output[len(output)-1]
	assert.Equal(t, "Wallet recovered!!", message)

	output, err = utils.RecoverWalletFromMnemonic(walletConfigFilename, cliConfigFilename, inValidMnemonic)
	assert.NotNil(t, err)
	assert.Equal(t, 1, len(output))
	assert.Equal(t, "Error: Invalid mnemonic", output[0])
}
