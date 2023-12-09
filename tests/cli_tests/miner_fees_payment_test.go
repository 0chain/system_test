package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerFeesPayment(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Skip("Skip till chain-side bugs are resolved")

	t.Skip("Skipped till re-done")

	var mnconfig map[string]float64
	var minerShare float64
	t.TestSetup("Get miner config", func() {
		mnconfig = getMinerSCConfiguration(t)
		minerShare = mnconfig["share_ratio"]

		miners := getMinersList(t)
		miner := getMinersDetail(t, miners.Nodes[0].SimpleNode.ID).SimpleNode
		require.NotEmpty(t, miner)
	})

	t.RunSequentiallyWithTimeout("Send ZCN between wallets with Fee flag - Fee must be paid to miners", 60*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error getting target wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBlock := getLatestFinalizedBlock(t)

		fee := 0.1
		output, err = sendTokens(t, configPath, targetWallet.ClientID, 0.5, escapedTestName(t), fee)
		require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBlock := getLatestFinalizedBlock(t)

		block := getBlockContainingTransaction(t, startBlock, endBlock, wallet, escapedTestName(t))
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.RunSequentiallyWithTimeout("Vp-add with fee should pay fee to the miners", 60*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error getting target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBlock := getLatestFinalizedBlock(t)

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
		endBlock := getLatestFinalizedBlock(t)

		block := getBlockContainingTransaction(t, startBlock, endBlock, wallet, "vestingpool")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.RunSequentiallyWithTimeout("rp-Lock and rp-unlock command with fee flag - fees must be paid to the miners", 60*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		startBlock := getLatestFinalizedBlock(t)

		fee := 0.1
		readPoolParams := createParams(map[string]interface{}{
			"tokens": 0.5,
			"fee":    fee,
		})
		output, err = readPoolLock(t, configPath, readPoolParams, true)
		require.Nil(t, err, "error locking read pool tokens", strings.Join(output, "\n"))

		lockTimer := time.NewTimer(time.Minute)
		cliutils.Wait(t, 30*time.Second)

		endBlock := getLatestFinalizedBlock(t)

		block := getBlockContainingTransaction(t, startBlock, endBlock, wallet, "read_pool_lock")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		output, err = readPoolInfo(t, configPath)
		require.Nil(t, err, "error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		readPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &readPool)
		require.Nil(t, err, "error unmarshalling read pool", strings.Join(output, "\n"))

		<-lockTimer.C

		startBlock = getLatestFinalizedBlock(t)

		output, err = readPoolUnlock(t, configPath, createParams(map[string]interface{}{}), true)
		require.Nil(t, err, "error unlocking read pool", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)

		endBlock = getLatestFinalizedBlock(t)

		block = getBlockContainingTransaction(t, startBlock, endBlock, wallet, "read_pool_unlock")
		blockMinerId = block.Block.MinerId
		blockMiner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.RunSequentiallyWithTimeout("wp-lock and wp-unlock command with fee flag - fee must be paid to the miners", 60*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 7)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		allocationId := setupAllocation(t, configPath)

		startBlock := getLatestFinalizedBlock(t)

		// Lock 1 token in Write pool amongst all blobbers
		fee := 0.1
		output, err = writePoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"tokens":     1,
			"fee":        fee,
		}), true)
		require.Nil(t, err, "Failed to lock write tokens", strings.Join(output, "\n"))

		lockTimer := time.NewTimer(time.Minute * 2)
		cliutils.Wait(t, 30*time.Second)

		endBlock := getLatestFinalizedBlock(t)

		block := getBlockContainingTransaction(t, startBlock, endBlock, wallet, "write_pool_lock")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		<-lockTimer.C

		startBlock = getLatestFinalizedBlock(t)

		// output, err = writePoolUnlock(t, configPath, createParams(map[string]interface{}{
		//	"pool_id": allocationId,
		//	"fee":     fee,
		//}), true)
		//require.Nil(t, err, "Unable to unlock tokens", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBlock = getLatestFinalizedBlock(t)

		block = getBlockContainingTransaction(t, startBlock, endBlock, wallet, "write_pool_unlock")
		blockMinerId = block.Block.MinerId
		blockMiner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})

	t.RunSequentiallyWithTimeout("sp-lock and sp-unlock with fee flag - fees must be paid to the miners", 60*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 7)
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
		startBlock := getLatestFinalizedBlock(t)

		// Stake tokens against this blobber
		fee := 0.1
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.5,
			"fee":        fee,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		cliutils.Wait(t, 30*time.Second)
		endBlock := getLatestFinalizedBlock(t)

		block := getBlockContainingTransaction(t, startBlock, endBlock, wallet, "stake_pool_lock")
		blockMinerId := block.Block.MinerId
		blockMiner := getMinersDetail(t, blockMinerId)

		expectedMinerFee := getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly := verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")

		// Unstake with fee
		startBlock = getLatestFinalizedBlock(t)

		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"fee":        fee,
		}), true)
		require.Nil(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)
		endBlock = getLatestFinalizedBlock(t)

		block = getBlockContainingTransaction(t, startBlock, endBlock, wallet, "stake_pool_unlock")
		blockMinerId = block.Block.MinerId
		blockMiner = getMinersDetail(t, blockMinerId)

		expectedMinerFee = getExpectedMinerFees(t, fee, minerShare, blockMiner)
		areMinerFeesPaidCorrectly = verifyMinerFeesPayment(t, &block, expectedMinerFee)
		require.True(t, areMinerFeesPaidCorrectly, "Test Failed due to transfer from MinerSC to generator miner not found")
	})
}

func getBlockContainingTransaction(
	t *test.SystemTest,
	startBlock, endBlock *climodel.LatestFinalizedBlock,
	wallet *climodel.Wallet,
	txnData string,
) (block climodel.Block) {
	for round := startBlock.Round + 1; round <= endBlock.Round; round++ {
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

func apiGetLatestFinalized(sharderBaseURL string) (*http.Response, error) {
	return http.Get(sharderBaseURL + "/v1/block/get/latest_finalized")
}

func getLatestFinalizedBlock(t *test.SystemTest) *climodel.LatestFinalizedBlock {
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

	sharders := getShardersList(t)
	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]
	sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)

	res, err := apiGetLatestFinalized(sharderBaseUrl)
	require.Nil(t, err, "Error retrieving latest block")
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get latest block: %d", res.StatusCode)
	require.NotNil(t, res.Body, "Latest block API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body")

	var block climodel.LatestFinalizedBlock
	err = json.Unmarshal(resBody, &block)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

	return &block
}

func getExpectedMinerFees(t *test.SystemTest, fee, minerShare float64, blockMiner *climodel.Node) (expectedMinerFee int64) {
	// Expected miner fee is calculating using this formula:
	// Fee * minerShare * miner.ServiceCharge
	// Stakeholders' reward is:
	// Fee * minerShare * (1 - miner.ServiceCharge)
	// In case of no stakeholders, miner gets:
	// Fee * minerShare
	minerFee := ConvertToValue(fee * minerShare)
	minerServiceCharge := int64(float64(minerFee) * blockMiner.Settings.ServiceCharge)
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

func verifyMinerFeesPayment(t *test.SystemTest, block *climodel.Block, expectedMinerFee int64) bool {
	for _, txn := range block.Block.Transactions {
		if strings.Contains(txn.TransactionData, "payFees") && strings.Contains(txn.TransactionData, fmt.Sprintf("%d", block.Block.Round)) {
			var transfers []climodel.Transfer
			err := json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.ReplaceAll(txn.TransactionOutput, "}{", "},{"))), &transfers)
			require.Nil(t, err, "Cannot unmarshal the transfers from transaction output: %v\n, txn data: %v\n txn status: %v", txn.TransactionOutput, txn.TransactionData, txn.TransactionStatus)

			for _, transfer := range transfers {
				// Transfer needs to be from Miner Smart contract to Generator miner
				if transfer.From != MINER_SC_ADDRESS || transfer.To != block.Block.MinerId {
					continue
				}
				t.Logf("--- FOUND IN ROUND: %d ---", block.Block.Round)
				require.NotNil(t, transfer, "The transfer of fee to miner could not be found")
				// Transfer fee must be equal to miner fee
				t.Log("Actual fee transfer: ", transfer.Amount, "Expected fee transfer:", expectedMinerFee)
				require.Equal(t, expectedMinerFee, transfer.Amount, "Transfer fee must be equal to miner fee")
				return true
			}
		}
	}
	return false
}
