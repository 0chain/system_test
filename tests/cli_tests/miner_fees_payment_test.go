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

	t.Run("rp-Lock and rp-unlock command with fee flag - fees must be paid to the miners", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 7)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

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
		fee := 3.0

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

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

		wait(t, 2*time.Minute)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		readPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &readPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "read_pool_lock")
		blockMinerId := block.Block.MinerId
		block_miner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		startBalance = getNodeBalanceFromASharder(t, miner.ID)

		params = createParams(map[string]interface{}{
			"pool_id": readPool[0].Id,
			"fee":     fee,
		})
		output, err = readPoolUnlock(t, configPath, params, true)
		require.Nil(t, err, "Unable to unlock tokens", strings.Join(output, "\n"))
		require.Equal(t, "unlocked", output[0])

		wait(t, time.Minute)
		endBalance = getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "read_pool_unlock")
		blockMinerId = block.Block.MinerId
		block_miner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("wp-lock and wp-unlock command with fee flag - fee must be paid to the miners", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 7)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

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
		fee := 3.0

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		// Lock 1 token in Write pool amongst all blobbers
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"duration":   "2m",
			"tokens":     1,
			"fee":        fee,
		})
		output, err = writePoolLock(t, configPath, params, true)
		lockTimer := time.NewTimer(time.Minute * 2)
		require.Nil(t, err, "Failed to lock write tokens", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0])

		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		writePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &writePool)
		require.Nil(t, err, "error unmarshalling write pool", strings.Join(output, "\n"))

		wait(t, time.Minute)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "write_pool_lock")
		blockMinerId := block.Block.MinerId
		block_miner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		<-lockTimer.C

		startBalance = getNodeBalanceFromASharder(t, miner.ID)

		params = createParams(map[string]interface{}{
			"pool_id": writePool[0].Id,
			"fee":     fee,
		})
		output, err = writePoolUnlock(t, configPath, params, true)
		require.Nil(t, err, "Unable to unlock tokens", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "unlocked", output[0])

		wait(t, time.Minute)
		endBalance = getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "write_pool_unlock")
		blockMinerId = block.Block.MinerId
		block_miner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("sp-lock and sp-unlock with fee flag - fees must be paid to the miners", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 7)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Get miner's start balance
		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		// Stake tokens against this blobber
		fee := 3.0
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.5,
			"fee":        fee,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, pool id: ([a-f0-9]{64})"), output[0])
		stakePoolID := strings.Fields(output[0])[4]
		require.Nil(t, err, "Error extracting pool Id from sp-lock output", strings.Join(output, "\n"))

		wait(t, 2*time.Minute)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "stake_pool_lock")
		blockMinerId := block.Block.MinerId
		block_miner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		startBalance = getNodeBalanceFromASharder(t, miner.ID)

		// Unstake the tokens
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"pool_id":    stakePoolID,
			"fee":        fee,
		}))
		require.Nil(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "tokens has unlocked, pool deleted", output[0])

		wait(t, time.Minute)
		endBalance = getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "stake_pool_unlock")
		blockMinerId = block.Block.MinerId
		block_miner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("zwallet lock and unlock command with fee flag - Fees must be paid to the miners", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 7)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

		// Get miner's start balance
		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		// lock tokens
		fee := 3.0
		params := createParams(map[string]interface{}{
			"durationMin": 1,
			"fee":         fee,
			"tokens":      0.5,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Tokens (0.500000) locked successfully", output[0])

		lockTimer := time.NewTimer(time.Minute)

		wait(t, 2*time.Minute)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)
		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		output, err = getLockedTokens(t, configPath)
		require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Locked tokens:", output[0])

		var stats climodel.LockedInterestPoolStats
		err = json.Unmarshal([]byte(output[1]), &stats)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "lock")
		blockMinerId := block.Block.MinerId
		block_miner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		// Wait until lock expires.
		<-lockTimer.C

		startBalance = getNodeBalanceFromASharder(t, miner.ID)

		// unlock
		output, err = unlockInterest(t, configPath, createParams(map[string]interface{}{
			"pool_id": stats.Stats[0].ID,
			"fee":     fee,
		}), true)
		require.Nil(t, err, "unlock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Unlock tokens success", output[0])

		wait(t, time.Minute)
		endBalance = getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &minerNode, "unlock")
		blockMinerId = block.Block.MinerId
		block_miner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, block_miner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, expectedMinerFee)
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
