package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

// address of minersc
const MINER_SC_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"

func TestSendAndBalance(t *testing.T) {
	t.Parallel()

	t.Run("Send with description", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = sendZCN(t, configPath, target.ClientID, "1", "rich description", createParams(map[string]interface{}{}), true)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Send tokens success:  [a-f0-9]{64}"), output[0])
		// cannot verify transaction payload at this moment due to transaction hash not being printed.
	})

	t.Run("Send with json flag", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = sendZCN(t, configPath, target.ClientID, "1", "rich description", createParams(map[string]interface{}{
			"json": "",
		}), true)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		sendTxnOutput := &climodel.SendTransaction{}
		err = json.Unmarshal([]byte(output[0]), sendTxnOutput)
		require.Nil(t, err, "error unmarshalling send txn json response")
		require.Equal(t, "success", sendTxnOutput.Status)
		require.NotEmpty(t, sendTxnOutput.Txn)
		require.NotEmpty(t, sendTxnOutput.Nonce)
	})

	t.Run("Balance checks before and after ZCN sent", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		successfulBalanceOutputRegex := regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`)

		// Before send balance checks
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Unexpected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, successfulBalanceOutputRegex, output[0])

		output, err = getBalanceForWallet(t, configPath, targetWallet)
		ensureZeroBalance(t, output, err)

		// Transfer ZCN from sender to target wallet
		output, err = sendZCN(t, configPath, target.ClientID, "1", "", createParams(map[string]interface{}{}), true)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Send tokens success:  [a-f0-9]{64}"), output[0])

		// After send balance checks
		output, err = getBalance(t, configPath)
		ensureZeroBalance(t, output, err)

		output, err = getBalanceForWallet(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected balance check failure for wallet", targetWallet, strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, successfulBalanceOutputRegex, output[0])
	})

	t.Run("Send without description should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = cliutils.RunCommandWithoutRetry("./zwallet send --silent --tokens 1" +
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

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		wantFailureMsg := "Insufficient balance for this transaction."

		output, err = sendZCN(t, configPath, target.ClientID, "1", "", createParams(map[string]interface{}{}), false)
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

		output, err = sendZCN(t, configPath, invalidClientID, "1", "", createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send with zero token should fail", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		//FIXME: This passes when fees are disabled but should be rejected once they are enabled
		output, err = sendZCN(t, configPath, target.ClientID, "0", "", createParams(map[string]interface{}{}), true)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Send tokens success:  [a-f0-9]{64}"), output[0])
	})

	t.Run("Send attempt to exceeding balance should fail", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		wantFailureMsg := "Insufficient balance for this transaction."

		output, err = sendZCN(t, configPath, target.ClientID, "1.5", "", createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send attempt with negative token should fail", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		wantFailureMsg := "Send tokens failed. submit transaction failed. {\"code\":\"invalid_request\"," +
			"\"error\":\"invalid_request: Invalid request (value must be greater than or equal to zero)\"}"

		output, err = sendZCN(t, configPath, target.ClientID, "-1", "", createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, wantFailureMsg, output[0])
	})

	t.Run("Send attempt with very long description should fail", func(t *testing.T) {
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
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

		output, err = sendZCN(t, configPath, target.ClientID, "1", longDesc, createParams(map[string]interface{}{}), false)
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

		output, err = sendZCN(t, configPath, wallet.ClientID, "1", "", createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, wantFailureMsg, output[0])
	})
}

func sendZCN(t *testing.T, cliConfigFilename, toClientID, tokens, desc, params string, retry bool) ([]string, error) {
	t.Logf("Sending ZCN...")
	cmd := "./zwallet send --silent --tokens " + tokens +
		" --desc \"" + desc + "\"" +
		" --to_client_id " + toClientID + " " + params +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func sendTokens(t *testing.T, cliConfigFilename, toClientID string, tokens float64, desc string, fee float64) ([]string, error) {
	return sendTokensFromWallet(t, cliConfigFilename, toClientID, tokens, desc, fee, escapedTestName(t))
}

func sendTokensFromWallet(t *testing.T, cliConfigFilename, toClientID string, tokens float64, desc string, fee float64, wallet string) ([]string, error) {
	t.Logf("Sending ZCN...")
	cmd := fmt.Sprintf(`./zwallet send --silent --tokens %v --desc %q --to_client_id %s `, tokens, desc, toClientID)

	if fee > 0 {
		cmd += fmt.Sprintf(" --fee %v ", fee)
	}

	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", wallet+"_wallet.json", cliConfigFilename)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func getRoundBlockFromASharder(t *testing.T, round int64) apimodel.Block {
	sharders := getShardersList(t)
	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]
	sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)

	// Get round details
	res, err := apiGetBlock(sharderBaseUrl, round)
	require.Nil(t, err, "Error retrieving block %d", round)
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get block %d details: %d", round, res.StatusCode)
	require.NotNil(t, res.Body, "Balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body: %v", err)

	var block apimodel.Block
	err = json.Unmarshal(resBody, &block)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)
	return block
}

func getNodeBalanceFromASharder(t *testing.T, client_id string) *apimodel.Balance {
	sharders := getShardersList(t)
	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]
	sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)
	// Get the starting balance for miner's delegate wallet.
	res, err := apiGetBalance(sharderBaseUrl, client_id)
	require.Nil(t, err, "Error retrieving client %s balance", client_id)
	if res.StatusCode == 400 {
		return &apimodel.Balance{
			Txn:     "",
			Round:   0,
			Balance: 0,
		}
	}
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to check client %s balance: %d", client_id, res.StatusCode)
	require.NotNil(t, res.Body, "Balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body")
	strBody := string(resBody)
	var startBalance apimodel.Balance
	err = json.Unmarshal(resBody, &startBalance)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strBody, err)

	return &startBalance
}

func getShardersList(t *testing.T) map[string]climodel.Sharder {
	return getShardersListForWallet(t, escapedTestName(t))
}

func getShardersListForWallet(t *testing.T, wallet string) map[string]climodel.Sharder {
	// Get sharder list.
	output, err := getShardersForWallet(t, configPath, wallet)
	found := false
	for index, line := range output {
		if line == "MagicBlock Sharders" {
			found = true
			output = output[index:]
			break
		}
	}
	require.True(t, found, "MagicBlock Sharders not found in getShardersForWallet output")
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	return sharders
}

func getMinersDetail(t *testing.T, miner_id string) *climodel.Node {
	// Get miner's node details (this has the total_stake and pools populated).
	output, err := getNode(t, configPath, miner_id)
	require.Nil(t, err, "get node %s failed", miner_id, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var nodeRes climodel.Node
	err = json.Unmarshal([]byte(strings.Join(output, "")), &nodeRes)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
	require.NotEmpty(t, nodeRes, "No node found: %v", strings.Join(output, "\n"))
	return &nodeRes
}

func getMinersList(t *testing.T) *climodel.NodeList {
	// Get miner list.
	output, err := getMiners(t, configPath)
	require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, "Expected output to have length of at least 1")

	var miners climodel.NodeList
	err = json.Unmarshal([]byte(output[len(output)-1]), &miners)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[len(output)-1], err)
	require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))
	return &miners
}

func getMinerSCConfiguration(t *testing.T) map[string]float64 {
	// Get MinerSC Global Config
	output, err := getMinerSCConfig(t, configPath, true)
	require.Nil(t, err, "get miners sc config failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)

	_, returnVal := keyValuePairStringToMap(t, output)
	return returnVal
}
