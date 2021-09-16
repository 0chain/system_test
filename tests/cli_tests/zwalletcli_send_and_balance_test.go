package cli_tests

import (
	"github.com/0chain/system_test/internal/cli/cli_model"
	"github.com/0chain/system_test/internal/cli/cli_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

func TestSendAndBalance(t *testing.T) {
	t.Run("Balance checks before and after ZCN sent", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			t.Errorf("Registering sender and target wallets failed: %v", err)
			return
		}

		requireFaucetSuccess(t)

		// Before send balance checks
		assertGetBalanceSuccess(t, senderWallet(t), "1.000")
		assertGetBalanceFailure(t, targetWallet(t))

		// Transfer ZCN from sender to target wallet
		assertSendSuccess(t, target.ClientId, "1", "transfer")

		// After send balance checks
		assertGetBalanceFailure(t, senderWallet(t))
		assertGetBalanceSuccess(t, targetWallet(t), "1.000")
	})

	t.Run("Send with description", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			t.Errorf("Registering sender and target wallets failed: %v", err)
			return
		}

		requireFaucetSuccess(t)

		assertSendSuccess(t, target.ClientId, "1", "rich description")

		// cannot verify transaction payload at this moment due to transaction hash not being printed.
	})

	t.Run("Send without description", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			t.Errorf("Registering sender and target wallets failed: %v", err)
			return
		}

		requireFaucetSuccess(t)

		assertSendSuccess(t, target.ClientId, "1", "")

		// cannot verify transaction payload at this moment due to transaction hash not being printed.
	})

	t.Run("Send attempt on zero ZCN wallet", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			return
		}

		wantFailureMsg := "Send tokens failed. {\"error\": \"verify transaction failed\"}"
		assertSendFailure(t, target.ClientId, "1", "", wantFailureMsg)
	})

	t.Run("Send attempt to invalid address", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.NoError(t, err, "Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))

		requireFaucetSuccess(t)

		invalidClientID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabb" // more than 64 chars

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
			"\"error\":\"invalid_request: Invalid request (to client id must be a hexadecimal hash)\"}"
		assertSendFailure(t, invalidClientID, "1", "invalid address", wantFailureMsg)
	})

	t.Run("Send attempt to exceeding balance", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			return
		}

		requireFaucetSuccess(t)

		wantFailureMsg := "Send tokens failed. {\"error\": \"verify transaction failed\"}"
		assertSendFailure(t, target.ClientId, "1.5", "exceed bal", wantFailureMsg)
	})

	t.Run("Send attempt with negative token", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			return
		}

		requireFaucetSuccess(t)

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
			"\"error\":\"invalid_request: Invalid request (value must be greater than or equal to zero)\"}"
		assertSendFailure(t, target.ClientId, "-1", "negative token", wantFailureMsg)
	})

	t.Run("Send attempt with very long description", func(t *testing.T) {
		t.Parallel()

		target, err := registerSenderAndTargetWallets(t)
		if err != nil {
			return
		}

		requireFaucetSuccess(t)

		b := make([]rune, 100000)
		for i := range b {
			b[i] = 'a'
		}
		longDesc := string(b)

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"txn_exceed_max_payload\"," +
			"\"error\":\"txn_exceed_max_payload: transaction payload exceeds the max payload (98304)\"}"
		assertSendFailure(t, target.ClientId, "1", longDesc, wantFailureMsg)
	})

	t.Run("Send attempt to self", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.NoError(t, err, "Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		if err != nil {
			t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
		}

		requireFaucetSuccess(t)

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
			"\"error\":\"invalid_request: Invalid request (from and to client should be different)\"}"
		assertSendFailure(t, wallet.ClientId, "1", "send self", wantFailureMsg)
	})
}

func registerSenderAndTargetWallets(t *testing.T) (target *cli_model.Wallet, err error) {
	// Sender Wallet
	if output, err := registerWallet(t, configPath); err != nil {
		require.NoError(t, err, "Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
		return nil, err
	}

	// Target Wallet
	if output, err := registerWalletForName(configPath, targetWallet(t)); err != nil {
		require.NoError(t, err, "Unexpected register wallet failure: Output %v", strings.Join(output, "\n"))
		return nil, err
	}

	target, err = getWalletForName(t, configPath, targetWallet(t))
	if err != nil {
		t.Errorf("Error occured when retrieving target wallet due to error: %v", err)
	}

	return target, nil
}

func assertGetBalanceFailure(t *testing.T, walletName string) {
	output, err := getBalanceForWallet(configPath, walletName)
	if err == nil {
		t.Errorf("Missing expected balance check failure for wallet %s: Output %v", walletName, strings.Join(output, "\n"))
		return
	}

	assert.Equal(t, 1, len(output))
	assert.Equal(t, "Failed to get balance:", output[0])
}

func assertGetBalanceSuccess(t *testing.T, walletName string, balance string) {
	output, err := getBalanceForWallet(configPath, walletName)
	if err != nil {
		t.Errorf("Unexpected balance check failure for wallet %s: Output %v", walletName, strings.Join(output, "\n"))
	}

	assert.Equal(t, 1, len(output))
	assert.Regexp(t, regexp.MustCompile("Balance: "+balance+" ZCN \\([0-9.]+ USD\\)$"), output[0])
}

func assertSendFailure(t *testing.T, toClientID string, tokens string, desc string, failureMsg string) {
	output, err := send(t, configPath, toClientID, tokens, desc)
	if err == nil {
		t.Errorf("Missing expected send failure: Output %v", strings.Join(output, "\n"))
		return
	}

	assert.Len(t, output, 1)
	assert.Equal(t, failureMsg, output[0])
}

func assertSendSuccess(t *testing.T, toClientID string, tokens string, desc string) {
	output, err := send(t, configPath, toClientID, tokens, desc)
	if err != nil {
		t.Errorf("Unexpected send failure: Output %v", strings.Join(output, "\n"))
		return
	}

	assert.Len(t, output, 1)
	assert.Equal(t, "Send tokens success", output[0])
}

func requireFaucetSuccess(t *testing.T) {
	output, err := executeFaucet(t, configPath)
	require.NoError(t, err, "Unexpected faucet failure: Output %v", strings.Join(output, "\n"))
}

func senderWallet(t *testing.T) string {
	return escapedTestName(t)
}

func targetWallet(t *testing.T) string {
	return escapedTestName(t) + "_TARGET"
}

func send(t *testing.T, cliConfigFilename string, toClientID string, tokens string, desc string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet send --silent --tokens " + tokens +
		" --desc \"" + desc + "\"" +
		" --to_client_id " + toClientID +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}
