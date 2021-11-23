package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func Test___FlakyScenariosMinerFees(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

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
		output, err = readPoolLock(t, configPath, params, true)
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

		output, err = writePoolInfo(t, configPath, true)
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
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)

		// Get initial write pool
		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Move file
		remotepath := filepath.Base(localpath)
		moveAllocationFile(t, allocationID, remotepath, "newDir")

		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath, true)
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

	t.Run("File Update with a different size - Blobbers should be paid for the extra file size", func(t *testing.T) {
		t.Parallel()

		// Logic: Upload a 0.5 MB file and get the upload cost. Update the 0.5 MB file with a 1 MB file
		// and see that blobber's write pool balances are deduced again for the cost of uploading extra
		// 0.5 MBs.

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock":   "0.5",
			"size":   10 * MB,
			"data":   2,
			"parity": 2,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		fileSize := int64(0.5 * MB)

		// Get expected upload cost for 0.5 MB
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)
		output, _ = getUploadCostInUnit(t, configPath, allocationID, localpath)
		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))
		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := (expectedUploadCostInZCN / (2 + 2))

		// Wait for write pool blobber balances to be deduced for initial 0.5 MB
		wait(t, time.Minute)

		// Get write pool info before file update
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		t.Logf("Write pool Balance after upload expected to be [%v] but was [%v]", 0.5, intToZCN(initialWritePool[0].Balance))
		require.InEpsilonf(t, 0.5-actualExpectedUploadCostInZCN, intToZCN(initialWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		remotepath := "/" + filepath.Base(localpath)
		updateFileWithRandomlyGeneratedData(t, allocationID, remotepath, int64(1*MB))

		// Wait before fetching final write pool
		wait(t, time.Minute)

		// Get the new Write Pool info after update
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		t.Logf("Write pool Balance after upload expected to be [%v] but was [%v]", 0.5-actualExpectedUploadCostInZCN, intToZCN(initialWritePool[0].Balance))
		require.InEpsilon(t, (0.5 - 2*actualExpectedUploadCostInZCN), intToZCN(finalWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5-actualExpectedUploadCostInZCN, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		// Blobber pool balance should reduce by expected cost of 0.5 MB for each blobber
		totalChangeInWritePool := float64(0)
		for i := 0; i < len(finalWritePool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), finalWritePool[0].Blobber[i].Balance)

			// deduce tokens
			diff := intToZCN(initialWritePool[0].Blobber[i].Balance) - intToZCN(finalWritePool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] write pool has decreased by [%v] tokens after upload when it was expected to decrease by [%v]", i, diff, actualExpectedUploadCostInZCN/float64(len(finalWritePool[0].Blobber)))
			assert.InEpsilon(t, actualExpectedUploadCostInZCN/float64(len(finalWritePool[0].Blobber)), diff, epsilon, "Blobber balance should have deduced by expected cost divided number of blobbers")
			totalChangeInWritePool += diff
		}

		require.InEpsilon(t, actualExpectedUploadCostInZCN, totalChangeInWritePool, epsilon, "expected write pool balance to decrease by [%v] but has actually decreased by [%v]", actualExpectedUploadCostInZCN, totalChangeInWritePool)
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
	t.Parallel()

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

func Test___FlakyScenariosUpdateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Cancel Allocation Should Work when blobber fails challenges", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := cancelAllocation(t, configPath, allocationID, true)

		require.Nil(t, err, "error canceling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`), output[0])
	})

	t.Run("Cancel Other's Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		var otherAllocationID string
		// This test creates a separate wallet and allocates there, test nesting needed to create other wallet json
		t.Run("Get Other Allocation ID", func(t *testing.T) {
			otherAllocationID = setupAllocation(t, configPath)
		})

		// otherAllocationID should not be cancelable from this level
		output, err := cancelAllocation(t, configPath, otherAllocationID, false)

		require.NotNil(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		//FIXME: POSSIBLE BUG: Error message shows error in creating instead of error in canceling
		require.Equal(t, "Error creating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, "+
			"but got 0/2 sharders", output[len(output)-1])
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
		}, false)
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
		}), true)
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

		output, _ = writePoolInfo(t, configPath, true)
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
		require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0],
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
		output, _ = writePoolInfo(t, configPath, true)
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

	t.Run("transfer allocation and upload file", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": int64(20480),
		})

		ownerWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving owner wallet")

		output, err := addCurator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"curator":    ownerWallet.ClientID,
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, "add curator - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("%s added %s as a curator to allocation %s", ownerWallet.ClientID, ownerWallet.ClientID, allocationID), output[0],
			"add curator - Unexpected output", strings.Join(output, "\n"))

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, 256)
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
		require.Nil(t, err, "faucet execution failed for non-owner wallet", strings.Join(output, "\n"))

		newOwnerWallet, err := getWalletForName(t, configPath, newOwner)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

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

		output, err = writePoolLockWithWallet(t, newOwner, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.5,
			"duration":   "24h",
		}), false)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "write pool lock - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0], "write pool lock - Unexpected output", strings.Join(output, "\n"))

		output, err = uploadFileForWallet(t, newOwner, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/new" + remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "upload file - Unexpected output", strings.Join(output, "\n"))
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filepath.Base(file), output[1],
			"upload file - Unexpected output", strings.Join(output, "\n"))
	})
}

func Test___FlakyScenariosCreateDir(t *testing.T) {
	t.Parallel()

	t.Run("create dir with no leading slash should work", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		dirName := "noleadingslash"
		output, err := createDir(t, configPath, allocID, dirName, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: creating dir with no leading slash, there should be success message in output
		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 1)
		require.Equal(t, dirName, files[0].Name, "Directory must be created", files)
	})
}

func Test___FlakyScenariosDownload(t *testing.T) {
	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Parallel()

	t.Run("Download Encrypted File Should Work", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Downloading encrypted file is not working")
		}
		t.Parallel()

		allocSize := int64(10 * MB)
		filesize := int64(10)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath + filepath.Base(filename),
			"encrypt":    "",
		})

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		// Downloading encrypted file not working
		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  os.TempDir(),
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
	})

	t.Run("Download File to Existing File Should Fail", func(t *testing.T) {
		t.Parallel()

		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  os.TempDir(),
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := fmt.Sprintf(
			"Download failed. Local file already exists '%s'",
			strings.TrimSuffix(os.TempDir(), "/")+"/"+filepath.Base(filename),
		)
		require.Equal(t, expected, output[0])
	})
}
