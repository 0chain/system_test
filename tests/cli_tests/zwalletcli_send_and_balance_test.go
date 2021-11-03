package cli_tests

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
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

		require.Len(t, output, 1)
		require.Regexp(t, successfulBalanceOutputRegex, output[0])

		output, err = getBalanceForWallet(configPath, targetWallet)
		require.NotNil(t, err, "Missing expected balance check failure for wallet", targetWallet, strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "Failed to get balance:", output[0])

		// Transfer ZCN from sender to target wallet
		output, err = sendZCN(t, configPath, target.ClientID, "1", "{}")
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "Send tokens success", output[0])

		// After send balance checks
		output, err = getBalance(t, configPath)
		require.NotNil(t, err, "Missing expected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "Failed to get balance:", output[0])

		output, err = getBalanceForWallet(configPath, targetWallet)
		require.Nil(t, err, "Unexpected balance check failure for wallet", targetWallet, strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, successfulBalanceOutputRegex, output[0])
	})

	t.Run("Send ZCN between wallets - Fee must be paid to miners", func(t *testing.T) {
		t.Parallel()

		targetWalletName := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(configPath, targetWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		// Set a random fee in range [0.01, 0.02) (crypto/rand used for linting fix)
		send_fee := 0.01 + getRandomUniformFloat64(t)*0.01

		output, err = sendTokens(t, configPath, targetWallet.ClientID, 0.5, "{}", send_fee)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		wait(t, time.Minute*2)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		var block_miner *climodel.Node
		var block_miner_id string
		var transactionRound int64

		var expectedMinerFee int64

		for round := startBalance.Round + 1; round <= endBalance.Round; round++ {
			block := getRoundBlockFromASharder(t, round)

			for i := range block.Block.Transactions {
				txn := block.Block.Transactions[i]
				// Find the generator miner of the block on which this transaction was recorded
				if block_miner_id == "" {
					if txn.ToClientId == targetWallet.ClientID {
						block_miner_id = block.Block.MinerId // Generator miner
						transactionRound = block.Block.Round
						block_miner = getMinersDetail(t, minerNode.ID)

						// Expected miner fee is calculating using this formula:
						// Fee * minerShare * miner.ServiceCharge
						// Stakeholders' reward is:
						// Fee * minerShare * (1 - miner.ServiceCharge)
						// In case of no stakeholders, miner gets:
						// Fee * minerShare
						minerFee := ConvertToValue(send_fee * minerShare)
						minerServiceCharge := int64(float64(minerFee) * block_miner.SimpleNode.ServiceCharge)
						expectedMinerFee = minerServiceCharge
						minerFeeRemaining := minerFee - minerServiceCharge

						// If there is no stake, the miner gets entire fee.
						// Else "Remaining" portion would be distributed to stake holders
						// And hence not go the miner
						if miner.TotalStake == 0 {
							expectedMinerFee += minerFeeRemaining
						}
						t.Logf("Expected miner fee: %v", expectedMinerFee)
						t.Logf("Miner ID: %v", block_miner_id)
					}
				} else {
					// Search for the fee payment to generator miner in "payFee" transaction output
					if strings.ContainsAny(txn.TransactionData, "payFees") && strings.ContainsAny(txn.TransactionData, fmt.Sprintf("%d", transactionRound)) {
						var transfers []apimodel.Transfer
						err = json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Replace(txn.TransactionOutput, "}{", "},{", -1))), &transfers)
						require.Nil(t, err, "Cannot unmarshal the transfers from transaction output")

						for _, transfer := range transfers {
							// Transfer needs to be from Miner Smart contract to Generator miner
							if transfer.From != MINER_SC_ADDRESS || transfer.To != block_miner_id {
								continue
							} else {
								t.Logf("--- FOUND IN ROUND: %d ---", block.Block.Round)
								require.NotNil(t, transfer, "The transfer of fee to miner could not be found")
								// Transfer fee must be equal to miner fee
								require.InEpsilon(t, expectedMinerFee, transfer.Amount, 0.00000001, "Transfer fee must be equal to miner fee")
								return // test passed
							}
						}
					}
				}
			}
		}
	})

	t.Run("Send without description should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = cliutils.RunCommand("./zwallet send --silent --tokens 1" +
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

		output, err = sendZCN(t, configPath, target.ClientID, "1", "{}")
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

		output, err = sendZCN(t, configPath, invalidClientID, "1", "{}")
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

		output, err = sendZCN(t, configPath, target.ClientID, "1", "{}")
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

		output, err = sendZCN(t, configPath, target.ClientID, "1.5", "{}")
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

		output, err = sendZCN(t, configPath, target.ClientID, "-1", "{}")
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

		output, err = sendZCN(t, configPath, target.ClientID, "1", longDesc)
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

		output, err = sendZCN(t, configPath, wallet.ClientID, "1", "{}")
		require.NotNil(t, err, "Expected send to fail", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, wantFailureMsg, output[0])
	})
}

func sendZCN(t *testing.T, cliConfigFilename, toClientID, tokens, desc string) ([]string, error) {
	t.Logf("Sending ZCN...")
	return cliutils.RunCommand("./zwallet send --silent --tokens " + tokens +
		" --desc \"" + desc + "\"" +
		" --to_client_id " + toClientID +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func getRandomUniformFloat64(t *testing.T) float64 {
	random, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	var randomF big.Float
	randomBigFloat := *randomF.SetInt(random)
	randomFloat, _ := randomBigFloat.Float64()
	randomFloat /= float64(math.MaxInt64)
	require.Nil(t, err, "error generating random number from crypto/rand")
	return randomFloat
}

func sendTokens(t *testing.T, cliConfigFilename, toClientID string, tokens float64, desc string, fee float64) ([]string, error) {
	t.Logf("Sending ZCN...")
	cmd := fmt.Sprintf(`./zwallet send --silent --tokens %v --desc "%s" --to_client_id %s `, tokens, desc, toClientID)

	if fee > 0 {
		cmd += fmt.Sprintf(" --fee %v ", fee)
	}

	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", cliConfigFilename)
	return cliutils.RunCommand(cmd)
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

	resBody, err := ioutil.ReadAll(res.Body)
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
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to check client %s balance: %d", client_id, res.StatusCode)
	require.NotNil(t, res.Body, "Balance API response must not be nil")

	resBody, err := ioutil.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body")

	var startBalance apimodel.Balance
	err = json.Unmarshal(resBody, &startBalance)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)
	require.NotEmpty(t, startBalance.Txn, "Balance txn is unexpectedly empty: %s", string(resBody))
	require.Positive(t, startBalance.Balance, "Balance is unexpectedly zero or negative: %d", startBalance.Balance)
	require.Positive(t, startBalance.Round, "Round of balance is unexpectedly zero or negative: %d", startBalance.Round)
	return &startBalance
}

func getShardersList(t *testing.T) map[string]climodel.Sharder {
	// Get sharder list.
	output, err := getSharders(t, configPath)
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 1)
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
	require.Len(t, output, 1)

	var miners climodel.NodeList
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
	require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))
	return &miners
}

func getMinerSCConfiguration(t *testing.T) map[string]float64 {
	// Get MinerSC Global Config
	output, err := getMinerSCConfig(t, configPath)
	require.Nil(t, err, "get miners sc config failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)

	mconfig := map[string]float64{}
	for i := range output {
		configPair := strings.Split(output[i], "\t")
		val, err := strconv.ParseFloat(strings.TrimSpace(configPair[1]), 64)
		require.Nil(t, err, "config val %s for %s is unexpected not float64: %s", configPair[1], configPair[0], strings.Join(output, "\n"))
		mconfig[strings.TrimSpace(configPair[0])] = val
	}
	return mconfig
}
