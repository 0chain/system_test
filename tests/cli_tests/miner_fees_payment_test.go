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
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerFeesPayment(t *testing.T) {
	mnconfig := getMinerSCConfiguration(t)
	minerShare := mnconfig["share_ratio"]

	miners := getMinersList(t)
	miner := getMinersDetail(t, miners.Nodes[0].SimpleNode.ID).SimpleNode

	t.Run("Send with fee should pay fees to the miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		fee := 0.1
		output, err = sendTokens(t, configPath, targetWallet.ClientID, 0.5, escapedTestName(t), fee)
		require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, escapedTestName(t))
		blockMinerId := block.Block.MinerId
		block_miner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("Vp-add with fee should pay fee to the miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error getting target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		fee := 0.1
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":           targetWallet.ClientID + ":0.1",
			"lock":        0.1,
			"duration":    "10m",
			"fee":         fee,
			"description": "vestingpool",
		}), true)
		require.Nil(t, err, "error adding vesting pool", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "vestingpool")
		blockMinerId := block.Block.MinerId
		block_miner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("Lock and unlock with fee flag should pay fee to the miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		fee := 0.1
		output, err = lockInterest(t, configPath, createParams(map[string]interface{}{
			"durationMin": 1,
			"tokens":      0.1,
			"fee":         fee,
		}), true)
		require.Nil(t, err, "error locking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		lockId := regexp.MustCompile("[a-f0-9]{64}").FindString(output[1])
		t.Log(lockId)

		cliutils.Wait(t, 30*time.Second)

		endBalance := getNodeBalanceFromASharder(t, miner.ID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "lock")
		blockMinerId := block.Block.MinerId
		block_miner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})
}

func getBlockContainingTransaction(t *testing.T, startBalance, endBalance *apimodel.Balance,
	wallet *climodel.Wallet, minerNode *climodel.SimpleNode, txnData string) (block apimodel.Block) {
	for round := startBalance.Round + 1; round <= endBalance.Round; round++ {
		block := getRoundBlockFromASharder(t, round)

		for i := range block.Block.Transactions {
			txn := block.Block.Transactions[i]
			// Find the generator miner of the block on which this transaction was recorded
			if txn.ClientId == wallet.ClientID && strings.Contains(txn.TransactionData, txnData) {
				return block
			}
		}
	}
	return block
}

func getExpectedMinerFees(t *testing.T, fee, minerShare float64, block_miner *climodel.Node) (expectedMinerFee int64) {
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
	if block_miner.TotalStake == 0 {
		expectedMinerFee += minerFeeRemaining
	}
	return expectedMinerFee
}

func verifyMinerFeesPayment(t *testing.T, block *apimodel.Block, expectedMinerFee int64) bool {
	for _, txn := range block.Block.Transactions {
		if strings.ContainsAny(txn.TransactionData, "payFees") && strings.ContainsAny(txn.TransactionData, fmt.Sprintf("%d", block.Block.Round)) {
			var transfers []apimodel.Transfer
			err := json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Replace(txn.TransactionOutput, "}{", "},{", -1))), &transfers)
			require.Nil(t, err, "Cannot unmarshal the transfers from transaction output:", txn.TransactionOutput)

			for _, transfer := range transfers {
				// Transfer needs to be from Miner Smart contract to Generator miner
				if transfer.From != MINER_SC_ADDRESS || transfer.To != block.Block.MinerId {
					continue
				}
				t.Logf("--- FOUND IN ROUND: %d ---", block.Block.Round)
				require.NotNil(t, transfer, "The transfer of fee to miner could not be found")
				// Transfer fee must be equal to miner fee
				t.Log("Actual fee transfer: ", transfer.Amount, "Expected fee transfer:", expectedMinerFee)
				require.InEpsilon(t, expectedMinerFee, transfer.Amount, 0.00000001, "Transfer fee must be equal to miner fee")
				return true
			}
		}
	}
	return false
}
