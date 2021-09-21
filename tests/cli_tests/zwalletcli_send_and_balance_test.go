package cli_tests

import (
	"github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

func TestSendAndBalance(t *testing.T) {
	t.Parallel()
	t.Run("parallel", func(t *testing.T) {
		t.Run("Balance checks before and after ZCN sent", func(t *testing.T) {
			t.Parallel()

			targetWallet := escapedTestName(t) + "_TARGET"

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = registerWalletForName(configPath, targetWallet)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			target, err := getWalletForName(t, configPath, targetWallet)
			require.Nil(t, err, "Error occurred when retrieving target wallet")

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			successfulBalanceOutputRegex := regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`)

			// Before send balance checks
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "Unexpected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))

			require.Equal(t, 1, len(output))
			require.Regexp(t, successfulBalanceOutputRegex, output[0])

			output, err = getBalanceForWallet(configPath, targetWallet)
			require.NotNil(t, err, "Missing expected balance check failure for wallet", targetWallet, strings.Join(output, "\n"))

			require.Equal(t, 1, len(output))
			require.Equal(t, "Failed to get balance:", output[0])

			// Transfer ZCN from sender to target wallet
			output, err = sendZCN(t, configPath, target.ClientId, "1", "transfer")
			require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, "Send tokens success", output[0])

			// After send balance checks
			output, err = getBalance(t, configPath)
			require.NotNil(t, err, "Missing expected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))

			require.Equal(t, 1, len(output))
			require.Equal(t, "Failed to get balance:", output[0])

			output, err = getBalanceForWallet(configPath, targetWallet)
			require.Nil(t, err, "Unexpected balance check failure for wallet", targetWallet, strings.Join(output, "\n"))

			require.Equal(t, 1, len(output))
			require.Regexp(t, successfulBalanceOutputRegex, output[0])
		})

		t.Run("Send with description", func(t *testing.T) {
			t.Parallel()

			targetWallet := escapedTestName(t) + "_TARGET"

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = registerWalletForName(configPath, targetWallet)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			target, err := getWalletForName(t, configPath, targetWallet)
			require.Nil(t, err, "Error occurred when retrieving target wallet")

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			output, err = sendZCN(t, configPath, target.ClientId, "1", "rich description")
			require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, "Send tokens success", output[0])
			// cannot verify transaction payload at this moment due to transaction hash not being printed.
		})

		t.Run("Send without description should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = cli_utils.RunCommand("./zwallet send --silent --tokens 1" +
				" --to_client_id 7ec733204418d72b68e3579bdf55881b1528c676850976920de3f73e45d4fafa" +
				" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + configPath,
			)
			require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, "Error: Description flag is missing", output[0])
			// cannot verify transaction payload at this moment due to transaction hash not being printed.
		})

		t.Run("Send attempt on zero ZCN wallet should fail", func(t *testing.T) {
			t.Parallel()

			targetWallet := escapedTestName(t) + "_TARGET"

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = registerWalletForName(configPath, targetWallet)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			target, err := getWalletForName(t, configPath, targetWallet)
			require.Nil(t, err, "Error occurred when retrieving target wallet")

			wantFailureMsg := "Send tokens failed. {\"error\": \"verify transaction failed\"}"

			output, err = sendZCN(t, configPath, target.ClientId, "1", "")
			require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, wantFailureMsg, output[0])
		})

		t.Run("Send attempt to invalid address should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			invalidClientID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabb" // more than 64 chars
			wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
				"\"error\":\"invalid_request: Invalid request (to client id must be a hexadecimal hash)\"}"

			output, err = sendZCN(t, configPath, invalidClientID, "1", "invalid_address")
			require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, wantFailureMsg, output[0])
		})

		/* FIXME - this and the exceeding balance test takes a long time to run because the CLI sends the txn and has to wait for it to fail
		   it would be more efficient for the CLI to first run a balance check internally before sending the txn in order to fail fast
		   https://github.com/0chain/zwalletcli/issues/52
		*/
		t.Run("Send with zero token should fail", func(t *testing.T) {
			t.Parallel()

			targetWallet := escapedTestName(t) + "_TARGET"

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = registerWalletForName(configPath, targetWallet)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			target, err := getWalletForName(t, configPath, targetWallet)
			require.Nil(t, err, "Error occurred when retrieving target wallet")

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			output, err = sendZCN(t, configPath, target.ClientId, "1", "negative token")
			require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, "Send tokens success", output[0])
		})

		t.Run("Send attempt to exceeding balance should fail", func(t *testing.T) {
			t.Parallel()

			targetWallet := escapedTestName(t) + "_TARGET"

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = registerWalletForName(configPath, targetWallet)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			target, err := getWalletForName(t, configPath, targetWallet)
			require.Nil(t, err, "Error occurred when retrieving target wallet")

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			wantFailureMsg := "Send tokens failed. {\"error\": \"verify transaction failed\"}"

			output, err = sendZCN(t, configPath, target.ClientId, "1.5", "exceed bal")
			require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, wantFailureMsg, output[0])
		})

		t.Run("Send attempt with negative token should fail", func(t *testing.T) {
			t.Parallel()

			targetWallet := escapedTestName(t) + "_TARGET"

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = registerWalletForName(configPath, targetWallet)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			target, err := getWalletForName(t, configPath, targetWallet)
			require.Nil(t, err, "Error occurred when retrieving target wallet")

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
				"\"error\":\"invalid_request: Invalid request (value must be greater than or equal to zero)\"}"

			output, err = sendZCN(t, configPath, target.ClientId, "-1", "negative token")
			require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, wantFailureMsg, output[0])
		})

		t.Run("Send attempt with very long description should fail", func(t *testing.T) {
			t.Parallel()

			targetWallet := escapedTestName(t) + "_TARGET"

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			output, err = registerWalletForName(configPath, targetWallet)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			target, err := getWalletForName(t, configPath, targetWallet)
			require.Nil(t, err, "Error occurred when retrieving target wallet")

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			b := make([]rune, 100000)
			for i := range b {
				b[i] = 'a'
			}
			longDesc := string(b)

			wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"txn_exceed_max_payload\"," +
				"\"error\":\"txn_exceed_max_payload: transaction payload exceeds the max payload (98304)\"}"

			output, err = sendZCN(t, configPath, target.ClientId, "1", longDesc)
			require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, wantFailureMsg, output[0])
		})

		t.Run("Send attempt to self should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

			wallet, err := getWallet(t, configPath)
			require.Nil(t, err, "Get wallet failed", strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

			wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
				"\"error\":\"invalid_request: Invalid request (from and to client should be different)\"}"

			output, err = sendZCN(t, configPath, wallet.ClientId, "1", "send self")
			require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Equal(t, wantFailureMsg, output[0])
		})
	})
}

func sendZCN(t *testing.T, cliConfigFilename, toClientID, tokens, desc string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet send --silent --tokens " + tokens +
		" --desc \"" + desc + "\"" +
		" --to_client_id " + toClientID +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}
