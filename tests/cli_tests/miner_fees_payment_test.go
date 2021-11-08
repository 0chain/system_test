package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestMinerFeesPayment(t *testing.T) {
	t.Parallel()

	t.Run("rp-Lock command with fees flag - fees must be paid to the miners", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 2)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		// Create an allocation to use rp-lock on
		allocParams := createParams(map[string]interface{}{
			"expire": "5m",
			"size":   "1024",
			"lock":   "0.5",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		fee := 0.1

		// Use rp-lock with fees
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.5,
			"duration":   "1m",
			"fee":        fee,
		})
		output, err = readPoolLock(t, configPath, params)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		wait(t, time.Minute)
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
					if txn.ToClientId == MINER_SC_ADDRESS && txn.ClientId == wallet.ClientID && strings.Contains(txn.TransactionData, "read_pool_lock") {
						block_miner_id = block.Block.MinerId // Generator miner
						transactionRound = block.Block.Round
						block_miner = getMinersDetail(t, minerNode.ID)

						// Expected miner fee is calculating using this formula:
						// Fee * minerShare * miner.ServiceCharge
						// Stakeholders' reward is:
						// Fee * minerShare * (1 - miner.ServiceCharge)
						// In case of no stakeholders, miner gets:
						// Fee * minerShare
						minerFee := ConvertToValue(fee * minerShare)
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
					if strings.ContainsAny(txn.TransactionData, "read_pool_lock") && strings.ContainsAny(txn.TransactionData, fmt.Sprintf("%d", transactionRound)) {
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
}
