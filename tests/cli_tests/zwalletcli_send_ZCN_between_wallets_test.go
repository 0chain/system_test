package cli_tests

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSendZCNBetweenWallets(t *testing.T) {
	t.Run("Send ZCN between wallets - Fee must be paid to miners", func(t *testing.T) {
		t.Parallel()

		_, targetWallet := setupTransferWallets(t)

		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		// Set a random fee in range [0.01, 0.02) (crypto/rand used for linting fix)
		random, err := rand.Int(rand.Reader, big.NewInt(1))
		var randomF big.Float
		randomBigFloat := *randomF.SetInt(random)
		randomFloat, _ := randomBigFloat.Float64()
		require.Nil(t, err, "error generating random number from crypto/rand")
		send_fee := 0.01 + randomFloat*0.01

		output, err := sendTokens(t, configPath, targetWallet.ClientID, 0.5, "{}", send_fee)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		wait(t, 120*time.Second)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)
		for endBalance.Round == startBalance.Round {
			time.Sleep(10 * time.Second)
			endBalance = getNodeBalanceFromASharder(t, miner.ID)
		}

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		var block_miner *climodel.Node
		var block_miner_id string
		var feeTransfer apimodel.Transfer
		var transactionRound int64

		// Expected miner fee is calculating using this formula:
		// Fee * minerShare * miner.ServiceCharge
		var expected_miner_fee int64

		// Find the miner who has processed the transaction,
		// After finding the miner id, search for the fee payment to that miner in "payFee" transaction output
	out:
		for round := startBalance.Round + 1; round <= endBalance.Round; round++ {
			block := getRoundBlockFromASharder(t, round)

			for _, txn := range block.Block.Transactions {
				if block_miner_id == "" {
					if txn.ToClientId == targetWallet.ClientID {
						block_miner_id = block.Block.MinerId
						transactionRound = block.Block.Round
						block_miner = getMinersDetail(t, minerNode.ID)
						expected_miner_fee = ConvertToValue(send_fee * minerShare * block_miner.SimpleNode.ServiceCharge)
					}
				} else {
					data := fmt.Sprintf("{\"name\":\"payFees\",\"input\":{\"round\":%d}}", transactionRound)
					if txn.TransactionData == data {
						var transfers []apimodel.Transfer

						err = json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Replace(txn.TransactionOutput, "}{", "},{", -1))), &transfers)
						require.Nil(t, err, "Cannot unmarshal the transfers from transaction output")

						for _, tr := range transfers {
							if tr.To == block_miner_id && tr.Amount == int64(expected_miner_fee) {
								feeTransfer = tr
								break out
							}
						}
					}
				}
			}
		}

		require.NotNil(t, feeTransfer, "The transfer of fee to miner could not be found")
		require.Equal(t, expected_miner_fee, feeTransfer.Amount, "Transfer fee must be equal to miner fee")
	})
}

func sendTokens(t *testing.T, cliConfigFilename, toClientID string, tokens float64, desc string, fee float64) ([]string, error) {
	t.Logf("Sending ZCN...")
	cmd := fmt.Sprintf("./zwallet send --silent --tokens %v --desc \"%s\" --to_client_id %s ", tokens, desc, toClientID)

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
	for _, o := range output {
		configPair := strings.Split(o, "\t")
		val, err := strconv.ParseFloat(strings.TrimSpace(configPair[1]), 64)
		require.Nil(t, err, "config val %s for %s is unexpected not float64: %s", configPair[1], configPair[0], strings.Join(output, "\n"))
		mconfig[strings.TrimSpace(configPair[0])] = val
	}
	return mconfig
}

func setupTransferWallets(t *testing.T) (client, wallet *climodel.Wallet) {
	targetWallet := escapedTestName(t) + "_TARGET"

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

	output, err = registerWalletForName(configPath, targetWallet)
	require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

	client, err = getWalletForName(t, configPath, escapedTestName(t))
	require.Nil(t, err, "Error occurred when retrieving client wallet")

	target, err := getWalletForName(t, configPath, targetWallet)
	require.Nil(t, err, "Error occurred when retrieving target wallet")

	output, err = executeFaucetWithTokens(t, configPath, 1)
	require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

	return client, target
}
