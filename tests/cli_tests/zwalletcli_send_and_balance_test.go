package cli_tests

import (
	"github.com/0chain/system_test/internal/cli/cli_utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

func TestSendAndBalance(t *testing.T) {
	t.Run("Balance checks before and after ZCN sent", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		// Before send balance checks
		output, err := getBalance(t, configPath)
		if err != nil {
			t.Errorf("Unexpected balance check failure for wallet %s: Output %v", escapedTestName(t), strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1.000 ZCN \\([0-9.]+ USD\\)$"), output[0])

		output, err = getBalanceForWallet(configPath, targetWallet)
		if err == nil {
			t.Errorf("Missing expected balance check failure for wallet %s: Output %v", targetWallet, strings.Join(output, "\n"))
			return
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Failed to get balance:", output[0])

		// Transfer ZCN from sender to target wallet
		output, err = sendZCN(t, configPath, target.ClientId, "1", "transfer")
		if err != nil {
			t.Errorf("Unexpected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, "Send tokens success", output[0])

		// After send balance checks
		output, err = getBalance(t, configPath)
		if err == nil {
			t.Errorf("Missing expected balance check failure for wallet %s: Output %v", escapedTestName(t), strings.Join(output, "\n"))
			return
		}

		assert.Equal(t, 1, len(output))
		assert.Equal(t, "Failed to get balance:", output[0])

		output, err = getBalanceForWallet(configPath, targetWallet)
		if err != nil {
			t.Errorf("Unexpected balance check failure for wallet %s: Output %v", targetWallet, strings.Join(output, "\n"))
		}

		assert.Equal(t, 1, len(output))
		assert.Regexp(t, regexp.MustCompile("Balance: 1.000 ZCN \\([0-9.]+ USD\\)$"), output[0])
	})

	t.Run("Send with description", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		output, err := sendZCN(t, configPath, target.ClientId, "1", "rich description")
		if err != nil {
			t.Errorf("Unexpected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, "Send tokens success", output[0])
		// cannot verify transaction payload at this moment due to transaction hash not being printed.
	})

	t.Run("Send without description", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		output, err := sendZCN(t, configPath, target.ClientId, "1", "")
		if err != nil {
			t.Errorf("Unexpected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, "Send tokens success", output[0])
		// cannot verify transaction payload at this moment due to transaction hash not being printed.
	})

	t.Run("Send attempt on zero ZCN wallet", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		wantFailureMsg := "Send tokens failed. {\"error\": \"verify transaction failed\"}"

		output, err := sendZCN(t, configPath, target.ClientId, "1", "")
		if err == nil {
			t.Errorf("Missing expected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send attempt to invalid address", func(t *testing.T) {
		t.Parallel()

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		invalidClientID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabb" // more than 64 chars
		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
			"\"error\":\"invalid_request: Invalid request (to client id must be a hexadecimal hash)\"}"

		output, err := sendZCN(t, configPath, invalidClientID, "1", "invalid address")
		if err == nil {
			t.Errorf("Missing expected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send with zero token", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		output, err := sendZCN(t, configPath, target.ClientId, "0", "negative token")
		if err != nil {
			t.Errorf("Unexpected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, "Send tokens success", output[0])
	})

	t.Run("Send attempt to exceeding balance", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		wantFailureMsg := "Send tokens failed. {\"error\": \"verify transaction failed\"}"

		output, err := sendZCN(t, configPath, target.ClientId, "1.5", "exceed bal")
		if err == nil {
			t.Errorf("Missing expected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send attempt with negative token", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
			"\"error\":\"invalid_request: Invalid request (value must be greater than or equal to zero)\"}"

		output, err := sendZCN(t, configPath, target.ClientId, "-1", "negative token")
		if err == nil {
			t.Errorf("Missing expected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send attempt with very long description", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		if output, err := registerWalletForName(configPath, targetWallet); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		target, err := getWalletForName(t, configPath, targetWallet)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
			return
		}

		if output, err := executeFaucet(t, configPath); err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		b := make([]rune, 100000)
		for i := range b {
			b[i] = 'a'
		}
		longDesc := string(b)

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"txn_exceed_max_payload\"," +
			"\"error\":\"txn_exceed_max_payload: transaction payload exceeds the max payload (98304)\"}"

		output, err := sendZCN(t, configPath, target.ClientId, "1", longDesc)
		if err == nil {
			t.Errorf("Missing expected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send attempt to self", func(t *testing.T) {
		t.Parallel()

		if output, err := registerWallet(t, configPath); err != nil {
			t.Errorf("Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		wallet, err := getWallet(t, configPath)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
		}

		output, err := executeFaucet(t, configPath)
		if err != nil {
			t.Errorf("Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
			return
		}

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
			"\"error\":\"invalid_request: Invalid request (from and to client should be different)\"}"

		output, err = sendZCN(t, configPath, wallet.ClientId, "1", "send self")
		if err == nil {
			t.Errorf("Missing expected send failure: Output %v", strings.Join(output, "\n"))
			return
		}

		assert.Len(t, output, 1)
		assert.Equal(t, wantFailureMsg, output[0])
	})
}

func sendZCN(t *testing.T, cliConfigFilename string, toClientID string, tokens string, desc string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet send --silent --tokens " + tokens +
		" --desc \"" + desc + "\"" +
		" --to_client_id " + toClientID +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}
