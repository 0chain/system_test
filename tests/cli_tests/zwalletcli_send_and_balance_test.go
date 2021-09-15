package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/cli/cli_model"
	"github.com/0chain/system_test/internal/cli/cli_utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestSendAndBalance(t *testing.T) {
	t.Run("Balance checks before and after ZCN sent", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			return
		}

		output, err := executeFaucet(t, configPath)
		if err != nil {
			t.Errorf("Faucet execution failed due to error: %v", err)
			return
		}

		// Before send balance checks
		output, err = getBalance(t, configPath)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1.000 ZCN \\([0-9.]+ USD\\)$"), output[0])

		output, err = getBalanceForWallet(configPath, targetWallet(t))
		if err == nil {
			t.Errorf("Expected balance check to fail for target wallet: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Failed to get balance:", output[0])

		// TODO delete this
		fmt.Printf("target wallet: %v", target)
		fmt.Printf("target wallet: %v", target.ClientId)

		output, err = send(t, configPath, target.ClientId, "1", "")
		if err != nil {
			t.Errorf("Send failed due to error: Output %v", strings.Join(output, "\n"))
			return
		}

		// TODO delete this
		fmt.Printf("Send output: %v", output)

		// TODO verify send transaction and message

		// After send balance checks
		output, err = getBalance(t, configPath)
		if err == nil {
			t.Errorf("Expected balance check to fail for sender wallet: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Failed to get balance:", output[0])

		output, err = getBalanceForWallet(configPath, targetWallet(t))
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1.000 ZCN \\([0-9.]+ USD\\)$"), output[0])
	})

	t.Run("Send with description", func(t *testing.T) {
		// TODO
	})

	t.Run("Send attempt on zero ZCN wallet", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			return
		}

		output, err := send(t, configPath, target.ClientId, "1", "")
		if err == nil {
			t.Errorf("Expected send to fail due to zero ZCN wallet: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Equal(t, "Send tokens failed. {\"error\": \"verify transaction failed\"}", output[len(output)-1])
	})

	t.Run("Send attempt to invalid address", func(t *testing.T) {
		t.Parallel()

		if _, err := registerWallet(t, configPath); err != nil {
			t.Errorf("An error occured registering a wallet due to error: %v", err)
			return
		}

		output, err := executeFaucet(t, configPath)
		if err != nil {
			t.Errorf("Faucet execution failed due to error: %v", err)
			return
		}

		output, err = send(t, configPath, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "1", "")
		if err == nil {
			t.Errorf("Expected send to fail due to invalid target wallet: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Equal(t, "Send tokens failed. {\"error\": \"verify transaction failed\"}", output[len(output)-1])
	})

	t.Run("Send attempt to exceeding balance", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			return
		}

		output, err := executeFaucet(t, configPath)
		if err != nil {
			t.Errorf("Faucet execution failed due to error: %v", err)
			return
		}

		output, err = send(t, configPath, target.ClientId, "1.5", "")
		if err == nil {
			t.Errorf("Expected send to fail due to low ZCN balance: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Equal(t, "Send tokens failed. {\"error\": \"verify transaction failed\"}", output[len(output)-1])
	})
}

func registerSenderAndTargetWallets(t *testing.T) (target *cli_model.Wallet, err error) {
	// Sender Wallet
	if _, err := registerWallet(t, configPath); err != nil {
		t.Errorf("An error occured registering a wallet due to error: %v", err)
		return nil, err
	}

	// Target Wallet
	if _, err := registerWalletForName(configPath, targetWallet(t)); err != nil {
		t.Errorf("An error occured registering a wallet due to error: %v", err)
		return nil, err
	}

	target, err = getWalletForName(t, configPath, targetWallet(t))
	if err != nil {
		t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
	}

	return target, err
}

func targetWallet(t *testing.T) string {
	return escapedTestName(t) + "_TARGET"
}

func send(t *testing.T, cliConfigFilename string, toClientID string, tokens string, desc string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet send --tokens " + tokens +
		" --desc \"" + desc + "\"" +
		" --to_client_id " + toClientID +
		" --wallet " + escapedTestName(t) + " --configDir ./config --config " + cliConfigFilename)
}
