package cli_tests

import (
	"encoding/json"
	"fmt"
	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func Test___FlakyScenariosMinerFees(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	const chunksize = 64 * 1024

	// originally miner_fees_payment.go
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

}

func Test___FlakyScenariosCommonUserFunctions(t *testing.T) {
	t.Parallel()

	t.Run("Update Allocation - Blobbers' lock in stake pool must increase according to updated size", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 1 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Updated allocation params
		allocParams = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2 * MB,
		})
		output, err = updateAllocation(t, configPath, allocParams, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation updated with txId : ([a-f0-9]{64})"), output[0])

		allocation := getAllocation(t, allocationID)

		// Each blobber should lock (updated size of allocation on that blobber * write_price of blobber) in stake pool
		wait(t, 2*time.Minute)
		for _, blobber_detail := range allocation.BlobberDetails {
			output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id": blobber_detail.BlobberID,
				"json":       "",
			}))
			assert.Nil(t, err, "Error fetching stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			stakePool := climodel.StakePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &stakePool)
			assert.Nil(t, err, "Error unmarshalling stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			allocationOffer := climodel.StakePoolOfferInfo{}
			for _, offer := range stakePool.Offers {
				if offer.AllocationID == allocationID {
					allocationOffer = *offer
				}
			}

			t.Logf("Expected blobber id [%v] to lock [%v] but it actually locked [%v]", blobber_detail.BlobberID, int64(blobber_detail.Size*int64(blobber_detail.Terms.Write_price)), int64(allocationOffer.Lock))
			assert.Equal(t, int64(sizeInGB(blobber_detail.Size)*float64(blobber_detail.Terms.Write_price)), int64(allocationOffer.Lock))
		}

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create Allocation - Blobbers' must lock appropriate amount of tokens in stake pool", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 1 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		allocation := getAllocation(t, allocationID)

		// Each blobber should lock (size of allocation on that blobber * write_price of blobber) in stake pool
		wait(t, 2*time.Minute)
		for _, blobber_detail := range allocation.BlobberDetails {
			output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id": blobber_detail.BlobberID,
				"json":       "",
			}))
			assert.Nil(t, err, "Error fetching stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			stakePool := climodel.StakePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &stakePool)
			assert.Nil(t, err, "Error unmarshalling stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			allocationOffer := climodel.StakePoolOfferInfo{}
			for _, offer := range stakePool.Offers {
				if offer.AllocationID == allocationID {
					allocationOffer = *offer
				}
			}

			t.Logf("Expected blobber id [%v] to lock [%v] but it actually locked [%v]", blobber_detail.BlobberID, int64(blobber_detail.Size*int64(blobber_detail.Terms.Write_price)), int64(allocationOffer.Lock))
			assert.Equal(t, int64(sizeInGB(blobber_detail.Size)*float64(blobber_detail.Terms.Write_price)), int64(allocationOffer.Lock))
		}
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File move - Users should not be charged for moving a file ", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 4 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		fileSize := int64(math.Floor(1 * MB))

		// Upload 1 MB file
		localpath := uploadRandomlyGeneratedFile(t, allocationID, fileSize)

		// Get initial write pool
		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Move file
		remotepath := filepath.Base(localpath)
		moveAllocationFile(t, allocationID, remotepath, "newDir")

		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		// Get final write pool, no deduction should have been done
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Equal(t, initialWritePool[0].Balance, finalWritePool[0].Balance, "Write pool balance expected to be unchanged")

		for i := 0; i < len(finalWritePool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
			t.Logf("Initital blobber[%v] balance: [%v], final balance: [%v]", i, initialWritePool[0].Blobber[i].Balance, finalWritePool[0].Blobber[i].Balance)
			require.Equal(t, finalWritePool[0].Blobber[i].Balance, initialWritePool[0].Blobber[i].Balance, epsilon)
		}
		createAllocationTestTeardown(t, allocationID)
	})
}

func Test___FlakyScenariosFileStats(t *testing.T) {
	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	const chunksize = 64 * 1024

	t.Parallel()

	t.Run("get file stats before and after download", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		remotepath := "/"
		filesize := int64(256)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

		output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var stats map[string]climodel.FileStats

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			require.Equal(t, int64(1), data.NumOfUpdates)
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			//FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		wait(t, 2*time.Minute)
		// get file stats after download
		output, err = getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &stats)
		require.Nil(t, err)

		for _, data := range stats {
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, int64(1), data.NumOfBlockDownloads)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, int64(1), data.NumOfBlockDownloads)
			require.Equal(t, int64(1), data.NumOfUpdates)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			//FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}
	})
}

func Test___FlakyScenariosSendAndBalance(t *testing.T) {
	t.Run("Send ZCN between wallets - Fee must be paid to miners", func(t *testing.T) {
		t.Parallel()

		targetWalletName := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWalletName)
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
}

func Test___FlakyScenariosTransferAllocation(t *testing.T) {
	t.Parallel()

	t.Run("transfer allocation by owner should fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(2048),
		})

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err := registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Error adding curator:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("transfer allocation accounting test", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(1024000),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, 204800)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))

		newOwner := escapedTestName(t) + "_NEW_OWNER"

		output, err = registerWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, newOwner, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		output, _ = writePoolInfo(t, configPath)
		require.Len(t, output, 1, "write pool info - Unexpected output", strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Len(t, initialWritePool, 1)

		require.True(t, initialWritePool[0].Locked, strings.Join(output, "\n"))
		require.Equal(t, allocationID, initialWritePool[0].Id, strings.Join(output, "\n"))
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, strings.Join(output, "\n"))

		output, err = transferAllocationOwnership(t, map[string]interface{}{
			"allocation":    allocationID,
			"new_owner_key": newOwnerWallet.ClientPublicKey,
			"new_owner":     newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "transfer allocation - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, newOwnerWallet.ClientID), output[0],
			"transfer allocation - Unexpected output", strings.Join(output, "\n"))

		transferred := pollForAllocationTransferToEffect(t, newOwner, allocationID)
		require.True(t, transferred, "allocation was not transferred to new owner within time allotted")

		// balance of old owner should be unchanged
		// FIXME should this contain the released pool balances given the change of ownership?
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Unexpected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))
		require.Len(t, output, 1, "get balance - Unexpected output", strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile(`Balance: 500.000 mZCN \(\d*\.?\d+ USD\)$`), output[0],
			"get balance - Unexpected output", strings.Join(output, "\n"))

		// balance of new owner should be unchanged
		output, err = getBalanceForWallet(t, configPath, newOwner)
		require.Nil(t, err, "Unexpected balance check failure for wallet", escapedTestName(t), strings.Join(output, "\n"))
		require.Len(t, output, 1, "get balance - Unexpected output", strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0],
			"get balance - Unexpected output", strings.Join(output, "\n"))

		// zero cost to transfer
		expectedTransferCost := int64(0)

		// write lock pool of old owner should remain locked
		// FIXME should this be unlocked given the change of ownership?
		wait(t, 2*time.Minute)
		output, _ = writePoolInfo(t, configPath)
		require.Len(t, output, 1, "write pool info - Unexpected output", strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		require.Len(t, finalWritePool, 1)

		actualCost := initialWritePool[0].Balance - finalWritePool[0].Balance

		require.Equal(t, expectedTransferCost, actualCost)
		require.True(t, finalWritePool[0].Locked, strings.Join(output, "\n"))
		require.Equal(t, allocationID, finalWritePool[0].Id, strings.Join(output, "\n"))
		require.Equal(t, allocationID, finalWritePool[0].AllocationId, strings.Join(output, "\n"))
	})
}

func Test___FlakyScenariosCreateDir(t *testing.T) {
	t.Run("create attempt with invalid dir - no leading slash", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "noleadingslash")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: creating dir with no leading slash must throw error explicitly to not give impression it was success

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 0)
	})

}
