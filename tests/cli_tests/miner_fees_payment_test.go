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
	require.NotEmpty(t, miner)

	t.Run("Send ZCN between wallets with Fee flag - Fee must be paid to miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		delegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, minerNodeDelegateWalletName, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		fee := 0.1
		output, err = sendTokensFromWallet(t, configPath, targetWallet.ClientID, 0.5, escapedTestName(t), fee, minerNodeDelegateWalletName)
		require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, escapedTestName(t))
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
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

		output, err = registerWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		delegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, minerNodeDelegateWalletName, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		fee := 0.1
		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":           targetWallet.ClientID + ":0.1",
			"lock":        0.1,
			"duration":    "10m",
			"fee":         fee,
			"description": "vestingpool",
		}), true, minerNodeDelegateWalletName)
		require.Nil(t, err, "error adding vesting pool", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "vestingpool")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("zwallet lock and unlock command with fee flag - Fees must be paid to the miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = registerWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		delegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, minerNodeDelegateWalletName, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		// lock with fee
		fee := 0.1
		output, err = lockInterestForWallet(t, configPath, createParams(map[string]interface{}{
			"durationMin": 1,
			"tokens":      0.1,
			"fee":         fee,
		}), true, minerNodeDelegateWalletName)
		require.Nil(t, err, "error locking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		lockId := regexp.MustCompile("[a-f0-9]{64}").FindString(output[1])

		lockTimer := time.NewTimer(time.Minute)
		cliutils.Wait(t, 30*time.Second)

		endBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "lock")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		<-lockTimer.C

		// Unlock with fee
		startBalance = getNodeBalanceFromASharder(t, miner.ID)

		output, err = unlockInterest(t, configPath, createParams(map[string]interface{}{
			"pool_id": lockId,
			"fee":     fee,
		}), true)
		require.Nil(t, err, "error unlocking tokens", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)

		endBalance = getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "unlock")
		blockMinerId = block.Block.MinerId
		blockMiner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("rp-Lock and rp-unlock command with fee flag - fees must be paid to the miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = registerWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		delegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, minerNodeDelegateWalletName, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		allocationId := setupAllocationWithWallet(t, minerNodeDelegateWalletName, configPath)

		startBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		fee := 0.1
		output, err = readPoolLockWithWallet(t, minerNodeDelegateWalletName, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"tokens":     0.5,
			"duration":   "1m",
			"fee":        fee,
		}), true)
		require.Nil(t, err, "error locking read pool tokens", strings.Join(output, "\n"))

		lockTimer := time.NewTimer(time.Minute)
		cliutils.Wait(t, 30*time.Second)

		endBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "read_pool_lock")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		output, err = readPoolInfo(t, configPath, allocationId)
		require.Nil(t, err, "error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		readPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &readPool)
		require.Nil(t, err, "error unmarshalling read pool", strings.Join(output, "\n"))

		<-lockTimer.C

		startBalance = getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		output, err = readPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": readPool[0].Id,
			"fee":     fee,
		}), true)
		require.Nil(t, err, "error unlocking read pool", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)

		endBalance = getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "read_pool_unlock")
		blockMinerId = block.Block.MinerId
		blockMiner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("wp-lock and wp-unlock command with fee flag - fee must be paid to the miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokensForWallet(t, minerNodeDelegateWalletName, configPath, 7)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		delegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		allocationId := setupAllocation(t, configPath)

		startBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		// Lock 1 token in Write pool amongst all blobbers
		fee := 0.1
		output, err = writePoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"duration":   "2m",
			"tokens":     1,
			"fee":        fee,
		}), true)
		require.Nil(t, err, "Failed to lock write tokens", strings.Join(output, "\n"))

		lockTimer := time.NewTimer(time.Minute * 2)
		cliutils.Wait(t, 30*time.Second)

		endBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "write_pool_lock")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		output, err = writePoolInfo(t, configPath, true)
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		writePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &writePool)
		require.Nil(t, err, "error unmarshalling write pool", strings.Join(output, "\n"))

		<-lockTimer.C

		startBalance = getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		output, err = writePoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": writePool[0].Id,
			"fee":     fee,
		}), true)
		require.Nil(t, err, "Unable to unlock tokens", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBalance = getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "write_pool_unlock")
		blockMinerId = block.Block.MinerId
		blockMiner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.Run("sp-lock and sp-unlock with fee flag - fees must be paid to the miners", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = registerWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		delegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, minerNodeDelegateWalletName, configPath, 7)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Get miner's start balance
		startBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		// Stake tokens against this blobber
		fee := 0.1
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.5,
			"fee":        fee,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		stakePoolID := regexp.MustCompile("[a-f0-9]{64}").FindString(output[0])

		cliutils.Wait(t, 30*time.Second)
		endBalance := getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block := getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "stake_pool_lock")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		// Unstake with fee
		startBalance = getNodeBalanceFromASharder(t, delegateWallet.ClientID)

		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"pool_id":    stakePoolID,
			"fee":        fee,
		}))
		require.Nil(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBalance = getNodeBalanceFromASharder(t, delegateWallet.ClientID)
		require.Greaterf(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)
		require.Greaterf(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)

		block = getBlockContainingTransaction(t, startBalance, endBalance, wallet, &miner, "stake_pool_unlock")
		blockMinerId = block.Block.MinerId
		blockMiner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, delegateWallet.ClientID, expectedMinerFee)
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

func getExpectedMinerFees(t *testing.T, fee, minerShare float64, blockMiner *climodel.Node) (expectedMinerFee int64) {
	// Expected miner fee is calculating using this formula:
	// Fee * minerShare * miner.ServiceCharge
	// Stakeholders' reward is:
	// Fee * minerShare * (1 - miner.ServiceCharge)
	// In case of no stakeholders, miner gets:
	// Fee * minerShare
	minerFee := ConvertToValue(fee * minerShare)
	minerServiceCharge := int64(float64(minerFee) * blockMiner.SimpleNode.ServiceCharge)
	expectedMinerFee = minerServiceCharge
	minerFeeRemaining := minerFee - minerServiceCharge

	// If there is no stake, the miner gets entire fee.
	// Else "Remaining" portion would be distributed to stake holders
	// And hence not go the miner
	if blockMiner.TotalStake == 0 {
		expectedMinerFee += minerFeeRemaining
	}
	return expectedMinerFee
}

func verifyMinerFeesPayment(t *testing.T, block *apimodel.Block, delegateWallet string, expectedMinerFee int64) bool {
	for _, txn := range block.Block.Transactions {
		if strings.Contains(txn.TransactionData, "payFees") && strings.Contains(txn.TransactionData, fmt.Sprintf("%d", block.Block.Round)) {
			var transfers []apimodel.Transfer
			err := json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Replace(txn.TransactionOutput, "}{", "},{", -1))), &transfers)
			require.Nil(t, err, "Cannot unmarshal the transfers from transaction output: %v\n, txn data: %v\n txn status: %v", txn.TransactionOutput, txn.TransactionData, txn.TransactionStatus)

			for _, transfer := range transfers {
				// Transfer needs to be from Miner Smart contract to Generator miner
				if transfer.From != MINER_SC_ADDRESS || transfer.To != block.Block.MinerId && transfer.To != delegateWallet {
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
