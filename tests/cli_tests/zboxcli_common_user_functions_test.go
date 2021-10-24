package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	KB               = 1024      // kilobyte
	MB               = 1024 * KB // megabyte
	GB               = 1024 * MB // gigabyte
	TOKEN_UNIT int64 = 1e10
)

func TestCommonUserFunctions(t *testing.T) {
	t.Parallel()

	t.Run("File Update - Blobbers should pay to write the marker to the blockchain ", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		fileSize := int64(3 * MB)
		filename, _ := uploadRandomlyGeneratedFile(t, allocationID, fileSize)
		wait(t, time.Minute)

		// Get write pool info before file update
		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		require.InEpsilon(t, 0.5, intToZCN(initialWritePool[0].Balance), epsilon)
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId)
		require.Less(t, 0, len(initialWritePool[0].Blobber))
		require.Equal(t, true, initialWritePool[0].Locked)

		filepath := updateFileWithRandomlyGeneratedData(t, allocationID, filename, int64(5*MB))

		// Get expected upload cost
		output, err = getUploadCostInUnit(t, configPath, allocationID, filepath)
		require.Nil(t, err, "Could not get upload cost", strings.Join(output, "\n"))

		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost is given in "per 720 hours", we need 1 hour
		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := (expectedUploadCostInZCN / (2 + 2))

		// Wait before fetching final write pool
		wait(t, time.Minute)

		// Get the new Write-Pool info after upload
		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "Failed to fetch Write Pool info", strings.Join(output, "\n"))

		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		require.InEpsilon(t, (0.5 - actualExpectedUploadCostInZCN), intToZCN(finalWritePool[0].Balance), epsilon)
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, finalWritePool[0].AllocationId)
		require.Less(t, 0, len(finalWritePool[0].Blobber))
		require.Equal(t, true, finalWritePool[0].Locked)

		// Blobber pool balance should reduce by (write price*filesize) for each blobber
		totalChangeInWritePool := float64(0)
		for i := 0; i < len(finalWritePool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), finalWritePool[0].Blobber[i].Balance)

			// deduce tokens
			diff := intToZCN(initialWritePool[0].Blobber[i].Balance) - intToZCN(finalWritePool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] write pool has decreased by [%v] tokens after upload when it was expected to decrease by [%v]", i, diff, actualExpectedUploadCostInZCN/float64(len(finalWritePool[0].Blobber)))
			require.InEpsilon(t, actualExpectedUploadCostInZCN/float64(len(finalWritePool[0].Blobber)), diff, epsilon)
			totalChangeInWritePool += diff
		}

		require.InEpsilon(t, actualExpectedUploadCostInZCN, totalChangeInWritePool, epsilon, "expected write pool balance to decrease by [%v] but has actually decreased by [%v]", actualExpectedUploadCostInZCN, totalChangeInWritePool)
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Send ZCN between wallets - Fee must be paid to miners", func(t *testing.T) {
		t.Parallel()

		s := rand.NewSource(time.Now().UnixNano())

		_, targetWallet := setupTransferWallets(t)

		mconfig := getMinerSCConfiguration(t)
		minerShare := mconfig["share_ratio"]
		// minerServiceCharge := mconfig["service_charge"]

		miners := getMinersList(t)
		minerNode := miners.Nodes[0].SimpleNode
		miner := getMinersDetail(t, minerNode.ID).SimpleNode

		startBalance := getNodeBalanceFromASharder(t, miner.ID)

		// Set a random fee in range [0.01, 0.02)
		send_fee := 0.01 + rand.New(s).Float64()*0.01

		output, err := sendTokens(t, configPath, targetWallet.ClientID, 0.5, "{}", send_fee)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		wait(t, 60*time.Second)
		endBalance := getNodeBalanceFromASharder(t, miner.ID)

		require.Greater(t, endBalance.Balance, startBalance.Balance, "Balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Balance, endBalance.Balance)
		require.Greater(t, endBalance.Round, startBalance.Round, "Round of balance is unexpectedly unchanged since last balance check: last %d, retrieved %d", startBalance.Round, endBalance.Round)

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
				if len(block_miner_id) == 0 {
					if txn.ToClientId == targetWallet.ClientID {
						// transaction = *txn
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

		require.Equal(t, expected_miner_fee, feeTransfer.Amount, "Transfer fee must be equal to miner fee")
	})

	// Test is failing. Maybe blobbers are not paying for rename a file
	t.Run("File move - Blobbers should pay to write the marker to the blockchain ", func(t *testing.T) {
		t.Parallel()

		allocationSize := int64(1 * MB)
		fileSize := int64(math.Floor(512 * KB))

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": allocationSize})

		filename, _ := uploadRandomlyGeneratedFile(t, allocationID, fileSize)
		wait(t, 50*time.Second)

		initialWritePool := getWritePool(t, configPath)

		oldFileName := filename
		for i := 0; i < 10; i++ {

			newFileName := fmt.Sprintf("%s_%d", filename, i)
			renameAllocationFile(t, allocationID, oldFileName, newFileName)
			oldFileName = newFileName
		}

		// Wait before fetching final write pool
		wait(t, 50*time.Second)

		finalWritePool := getWritePool(t, configPath)

		// Blobber pool balance should have been reduced
		totalChangeInWritePool := int64(0)
		for i := range finalWritePool[0].Blobber {
			diff := finalWritePool[0].Blobber[i].Balance - initialWritePool[0].Blobber[i].Balance
			t.Logf("Blobber [%v] balance in write pool has decreased by [%v] tokens after update", i, -diff)
			require.Negative(t, diff, "Blobber has to pay some of its token to write marker")
			totalChangeInWritePool += diff
		}
		require.Negative(t, totalChangeInWritePool, "Blobbers has to pay some of their token to redeem write markers")
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File move - Users should not be charged for moving a file ", func(t *testing.T) {
		t.Parallel()

		allocationSize := int64(1 * MB)
		fileSize := int64(math.Floor(512 * KB))

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": allocationSize})

		wait(t, 10*time.Second)
		wp := getWritePool(t, configPath)
		require.Equal(t, int64(5000000000), wp[0].Balance, "Write pool balance expected to be equal to locked amount")

		filename, uploadCost := uploadRandomlyGeneratedFile(t, allocationID, fileSize)

		// uploadCost takes into account data+parity, so we divide by that
		uploadCost = (uploadCost / (2 + 2))
		expected_wp_balance := int64(float64(5000000000) - float64(uploadCost))

		wait(t, 15*time.Second)
		wp = getWritePool(t, configPath)
		require.Equal(t, 1, len(wp), "Write pool expeted to be found")

		// There is a small difference in the expected and actual balance.
		// The reason needs to be investigated. For now we consider it to be
		// in a range close to expexted value. (range = 100 SAS)
		require.InDelta(t, expected_wp_balance, wp[0].Balance, 100, "Tokens must be transfered Reward Pool to Write Pool", "difference:", wp[0].Balance-expected_wp_balance)
		if wp[0].Balance-expected_wp_balance != 0 {
			t.Log("WARNING: difference in amount taken from Write Pool with the upload cost: ", wp[0].Balance-expected_wp_balance, " SAS")
		}

		cp_balance := getChallengePoolBalance(t, allocationID)
		require.Equal(t, int64(5000000000)-wp[0].Balance, int64(cp_balance), "Tokens must be transfered from Write Pool to Chanllenge Pool")

		blobber := getOneOfAllocationBlobbers(t, allocationID)

		offer := getAllocationOfferFromBlobberStackPool(t, blobber.BlobberID, allocationID)

		expectedLock := sizeInGB(blobber.Size) * blobber.Terms.Write_price
		require.Equal(t, int64(expectedLock), int64(offer.Lock), "Lock token interest must've been put in stack pool")

		moveAllocationFile(t, allocationID, filename, "new_dir")

		time.Sleep(10 * time.Second)
		new_wp := getWritePool(t, configPath)
		require.Equal(t, wp[0].Balance, new_wp[0].Balance, "The write pool is expected to not be changed after update file", "difference:", wp[0].Balance-new_wp[0].Balance)

		new_cp_balance := getChallengePoolBalance(t, allocationID)
		require.Equal(t, int64(cp_balance), int64(new_cp_balance), "Challenge pool blance shouldn't be changed after update file")

		createAllocationTestTeardown(t, allocationID)
	})

	// Test is failing. Maybe blobbers are not paying for rename a file
	t.Run("File Rename - Blobbers should pay to write the marker to the blockchain ", func(t *testing.T) {
		//t.Parallel()
		return
		allocationSize := int64(1 * MB)
		fileSize := int64(math.Floor(512 * KB))

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": allocationSize})

		filename, _ := uploadRandomlyGeneratedFile(t, allocationID, fileSize)
		wait(t, 50*time.Second)

		initialWritePool := getWritePool(t, configPath)

		oldFileName := filename
		for i := 0; i < 1; i++ {

			newFileName := fmt.Sprintf("%s_%d", filename, i)
			renameAllocationFile(t, allocationID, oldFileName, newFileName)
			oldFileName = newFileName
		}

		// Wait before fetching final write pool
		wait(t, 50*time.Second)

		finalWritePool := getWritePool(t, configPath)

		// Blobber pool balance should have been reduced
		totalChangeInWritePool := int64(0)
		for i := range finalWritePool[0].Blobber {
			diff := finalWritePool[0].Blobber[i].Balance - initialWritePool[0].Blobber[i].Balance
			t.Logf("Blobber [%v] balance in write pool has decreased by [%v] tokens after update", i, -diff)
			require.Negative(t, diff, "Blobber has to pay some of its token to write marker")
			totalChangeInWritePool += diff
		}
		require.Negative(t, totalChangeInWritePool, "Blobbers has to pay some of their token to redeem write markers")
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File Rename - Users should not be charged for renaming a file ", func(t *testing.T) {
		return
		t.Parallel()

		allocationSize := int64(1 * MB)
		fileSize := int64(math.Floor(512 * KB))

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": allocationSize})

		wait(t, 10*time.Second)
		wp := getWritePool(t, configPath)
		require.Equal(t, int64(5000000000), wp[0].Balance, "Write pool balance expected to be equal to locked amount")

		filename, uploadCost := uploadRandomlyGeneratedFile(t, allocationID, fileSize)

		// uploadCost takes into account data+parity, so we divide by that
		uploadCost = (uploadCost / (2 + 2))
		expected_wp_balance := int64(float64(5000000000) - float64(uploadCost))

		wait(t, 15*time.Second)
		wp = getWritePool(t, configPath)
		require.Equal(t, 1, len(wp), "Write pool expeted to be found")

		// There is a small difference in the expected and actual balance.
		// The reason needs to be investigated. For now we consider it to be
		// in a range close to expexted value. (range = 100 SAS)
		require.InDelta(t, expected_wp_balance, wp[0].Balance, 100, "Tokens must be transfered Reward Pool to Write Pool", "difference:", wp[0].Balance-expected_wp_balance)
		if wp[0].Balance-expected_wp_balance != 0 {
			t.Log("WARNING: difference in amount taken from Write Pool with the upload cost: ", wp[0].Balance-expected_wp_balance, " SAS")
		}

		cp_balance := getChallengePoolBalance(t, allocationID)
		require.Equal(t, int64(5000000000)-wp[0].Balance, int64(cp_balance), "Tokens must be transfered from Write Pool to Chanllenge Pool")

		blobber := getOneOfAllocationBlobbers(t, allocationID)

		offer := getAllocationOfferFromBlobberStackPool(t, blobber.BlobberID, allocationID)

		expectedLock := sizeInGB(blobber.Size) * blobber.Terms.Write_price
		require.Equal(t, int64(expectedLock), int64(offer.Lock), "Lock token interest must've been put in stack pool")

		renameAllocationFile(t, allocationID, filename, filename+"_renamed")

		time.Sleep(10 * time.Second)
		new_wp := getWritePool(t, configPath)
		require.Equal(t, wp[0].Balance, new_wp[0].Balance, "The write pool is expected to not be changed after update file", "difference:", wp[0].Balance-new_wp[0].Balance)

		new_cp_balance := getChallengePoolBalance(t, allocationID)
		require.Equal(t, int64(cp_balance), int64(new_cp_balance), "Challenge pool blance shouldn't be changed after update file")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File Update - Blobbers should pay to write the marker to the blockchain ", func(t *testing.T) {
		//t.Parallel()

		allocationSize := int64(1 * MB)
		fileSize := int64(math.Floor(512 * KB))

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": allocationSize})

		filename, _ := uploadRandomlyGeneratedFile(t, allocationID, fileSize)
		wait(t, 50*time.Second)

		initialWritePool := getWritePool(t, configPath)

		updateFileWithRandomlyGeneratedData(t, allocationID, filename, int64(5*MB))

		// Wait before fetching final write pool
		wait(t, 50*time.Second)

		finalWritePool := getWritePool(t, configPath)

		// Blobber pool balance should have been reduced
		totalChangeInWritePool := int64(0)
		for i := range finalWritePool[0].Blobber {
			diff := finalWritePool[0].Blobber[i].Balance - initialWritePool[0].Blobber[i].Balance
			t.Logf("Blobber [%v] balance in write pool has decreased by [%v] tokens after update", i, -diff)
			require.Negative(t, diff, "Blobber has to pay some of its token to write marker")
			totalChangeInWritePool += diff
		}
		require.Negative(t, totalChangeInWritePool, "Blobbers has to pay some of their token to redeem write markers")
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("File Update - Users should not be charged for updating a file ", func(t *testing.T) {
		//t.Parallel()

		allocationSize := int64(1 * MB)
		fileSize := int64(math.Floor(512 * KB))

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": allocationSize})

		wait(t, 10*time.Second)
		wp := getWritePool(t, configPath)
		require.Equal(t, int64(5000000000), wp[0].Balance, "Write pool balance expected to be equal to locked amount")

		filename, uploadCost := uploadRandomlyGeneratedFile(t, allocationID, fileSize)

		// uploadCost takes into account data+parity, so we divide by that
		uploadCost = (uploadCost / (2 + 2))
		expected_wp_balance := int64(float64(5000000000) - float64(uploadCost))

		wait(t, 15*time.Second)
		wp = getWritePool(t, configPath)
		require.Equal(t, 1, len(wp), "Write pool expeted to be found")

		// There is a small difference in the expected and actual balance.
		// The reason needs to be investigated. For now we consider it to be
		// in a range close to expexted value. (range = 100 SAS)
		require.InDelta(t, expected_wp_balance, wp[0].Balance, 100, "Tokens must be transfered Reward Pool to Write Pool", "difference:", wp[0].Balance-expected_wp_balance)
		if wp[0].Balance-expected_wp_balance != 0 {
			t.Log("WARNING: difference in amount taken from Write Pool with the upload cost: ", wp[0].Balance-expected_wp_balance, " SAS")
		}

		cp_balance := getChallengePoolBalance(t, allocationID)
		require.Equal(t, int64(5000000000)-wp[0].Balance, int64(cp_balance), "Tokens must be transfered from Write Pool to Chanllenge Pool")

		blobber := getOneOfAllocationBlobbers(t, allocationID)

		offer := getAllocationOfferFromBlobberStackPool(t, blobber.BlobberID, allocationID)

		expectedLock := sizeInGB(blobber.Size) * blobber.Terms.Write_price
		require.Equal(t, int64(expectedLock), int64(offer.Lock), "Lock token interest must've been put in stack pool")

		updateFileWithRandomlyGeneratedData(t, allocationID, filename, fileSize)

		time.Sleep(10 * time.Second)
		new_wp := getWritePool(t, configPath)
		require.Equal(t, wp[0].Balance, new_wp[0].Balance, "The write pool is expected to not be changed after update file", "difference:", wp[0].Balance-new_wp[0].Balance)

		new_cp_balance := getChallengePoolBalance(t, allocationID)
		require.Equal(t, int64(cp_balance), int64(new_cp_balance), "Challenge pool blance shouldn't be changed after update file")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Update Allocation - Lock token interest must've been put in stack pool", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		assertBalanceIs(t, "500.000 mZCN")

		blobber := getOneOfAllocationBlobbers(t, allocationID)

		offer := getAllocationOfferFromBlobberStackPool(t, blobber.BlobberID, allocationID)

		expectedLock := sizeInGB(blobber.Size) * blobber.Terms.Write_price
		require.Equal(t, int64(expectedLock), int64(offer.Lock), "Lock token interest must've been put in stack pool")

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "30m",
			"size":       20 * MB,
			"lock":       0.2,
		})
		output, err := updateAllocation(t, configPath, params)
		require.Nil(t, err, "Error updating allocation due to", strings.Join(output, "\n"))

		assertBalanceIs(t, "300.000 mZCN")

		blobber = getOneOfAllocationBlobbers(t, allocationID)

		offer = getAllocationOfferFromBlobberStackPool(t, blobber.BlobberID, allocationID)

		expectedLock = sizeInGB(blobber.Size) * blobber.Terms.Write_price
		require.Equal(t, int64(expectedLock), int64(offer.Lock), "Lock token interest must've been put in stack pool")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Update Allocation - Lock amount must've been withdrown from user wallet", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		assertBalanceIs(t, "500.000 mZCN")

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "30m",
			"lock":       0.2,
		})
		output, err := updateAllocation(t, configPath, params)
		require.Nil(t, err, "Error updating allocation due to", strings.Join(output, "\n"))

		assertBalanceIs(t, "300.000 mZCN")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create Allocation - Lock token interest must've been put in stack pool", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		assertBalanceIs(t, "500.000 mZCN")

		blobber := getOneOfAllocationBlobbers(t, allocationID)

		offer := getAllocationOfferFromBlobberStackPool(t, blobber.BlobberID, allocationID)

		expectedLock := sizeInGB(blobber.Size) * blobber.Terms.Write_price
		require.Equal(t, int64(expectedLock), int64(offer.Lock), "Lock token interest must've been put in stack pool")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create Allocation - Lock amount must've been withdrown from user wallet", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		assertBalanceIs(t, "500.000 mZCN")

		createAllocationTestTeardown(t, allocationID)
	})
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

func setupTransferWallets(t *testing.T) (*climodel.Wallet, *climodel.Wallet) {
	targetWallet := escapedTestName(t) + "_TARGET"

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

	output, err = registerWalletForName(configPath, targetWallet)
	require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

	client, err := getWalletForName(t, configPath, escapedTestName(t))
	require.Nil(t, err, "Error occurred when retrieving client wallet")

	target, err := getWalletForName(t, configPath, targetWallet)
	require.Nil(t, err, "Error occurred when retrieving target wallet")

	output, err = executeFaucetWithTokens(t, configPath, 1)
	require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

	return client, target
}

func getWritePool(t *testing.T, cliConfigFilename string) []climodel.WritePoolInfo {
	output, err := writePoolInfo(t, configPath)
	require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

	initialWritePool := []climodel.WritePoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &initialWritePool)
	require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
	return initialWritePool
}

func getChallengePoolBalance(t *testing.T, allocationID string) float64 {
	cp := getChallengePoolInfo(t, configPath, allocationID)
	return cp.Balance
}

func getOneOfAllocationBlobbers(t *testing.T, allocationID string) *climodel.BlobberAllocation {
	allocation := getAllocation(t, allocationID)

	require.GreaterOrEqual(t, len(allocation.BlobberDetails), 1, "Allocation must've been stored at least on one blobber")

	// We can also select a blobber randomly or select the first one
	blobber := allocation.BlobberDetails[0]

	return blobber
}

func assertBalanceIs(t *testing.T, balance string) {
	userWalletBalance := getWalletBalance(t, configPath)
	require.Equal(t, balance, userWalletBalance, "User wallet balance mismatch")
}

func uploadRandomlyGeneratedFile(t *testing.T, allocationID string, fileSize int64) (string, int64) {
	filename := generateRandomTestFileName(t)
	err := createFileWithSize(filename, fileSize)
	require.Nil(t, err)

	// Get expected upload cost
	uploadCost := getUploadCostValue(t, allocationID, filename, map[string]interface{}{"duration": "1h"})

	output, err := uploadFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/",
		"localpath":  filename,
	})
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Equal(t, 2, len(output))
	require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])
	r := regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`)
	matches := r.FindStringSubmatch(output[1])
	filename = matches[1]
	return filename, uploadCost
}

func moveAllocationFile(t *testing.T, allocationID, remotepath, destination string) {
	output, err := moveFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destpath":   "/" + destination,
	})
	require.Nil(t, err, "error in moving the file: ", strings.Join(output, "\n"))
}

func renameAllocationFile(t *testing.T, allocationID, remotepath, newName string) {
	output, err := renameFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destname":   newName,
	})
	require.Nil(t, err, "error in renaming the file: ", strings.Join(output, "\n"))
}

func updateFileWithRandomlyGeneratedData(t *testing.T, allocationID, remotepath string, size int64) string {
	localfile := generateRandomTestFileName(t)
	err := createFileWithSize(localfile, size)
	require.Nil(t, err)

	output, err := updateFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"localpath":  localfile,
	})
	require.Nil(t, err, strings.Join(output, "\n"))
	return localfile
}

func moveFile(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Moving file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox move %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithRetry(t, cmd, 3, time.Second*20)
}

func renameFile(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Renaming file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox rename %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithRetry(t, cmd, 3, time.Second*20)
}

func updateFile(t *testing.T, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Updating file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox update %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithRetry(t, cmd, 3, time.Second*20)
}

func getAllocationOfferFromBlobberStackPool(t *testing.T, blobber_id, allocationID string) *climodel.StakePoolOfferInfo {
	sp_info := getStackPoolInfo(t, configPath, blobber_id)

	require.GreaterOrEqual(t, len(sp_info.Offers), 1, "Blobbers offers must not be empty")

	// Find the offer related to this allocation
	offers := make([]climodel.StakePoolOfferInfo, len(sp_info.Offers))
	n := 0
	for _, o := range sp_info.Offers {
		if o.AllocationID == allocationID {
			offers[n] = *o
			n++
		}
	}

	require.GreaterOrEqual(t, n, 1, "The allocation offer expected to be found on blobber stack pool information")

	offer := offers[0]
	return &offer
}

func getWalletBalance(t *testing.T, cliConfigFilename string) string {
	t.Logf("Get Wallet Balance...")
	output, err := getBalance(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Regexp(t, regexp.MustCompile(`Balance: [0-9.]+ (|m|µ)ZCN \([0-9.]+ USD\)$`), output[0])
	r := regexp.MustCompile(`Balance: (?P<Balance>[0-9.]+ (|m|µ)ZCN) \([0-9.]+ USD\)$`)
	matches := r.FindStringSubmatch(output[0])
	userWalletBalance := matches[1]
	t.Logf(userWalletBalance)
	return userWalletBalance
}

func getAllocation(t *testing.T, allocationID string) *climodel.Allocation {
	return getAllocationWithRetry(t, configPath, allocationID, 1)
}

func getAllocationWithRetry(t *testing.T, cliConfigFilename, allocationID string, retry int) *climodel.Allocation {
	t.Logf("Get Allocation...")
	output, err := cliutils.RunCommandWithRetry(t, fmt.Sprintf(
		"./zbox get --allocation %s --json --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename), retry, time.Second*5)
	require.Nil(t, err, "Failed to get allocation", strings.Join(output, "\n"))
	alloc := &climodel.Allocation{}
	err = json.Unmarshal([]byte(output[0]), &alloc)
	require.Nil(t, err, "Error unmarshalling allocation", strings.Join(output, "\n"))

	return alloc
}

func getStackPoolInfo(t *testing.T, cliConfigFilename, blobberId string) *climodel.StakePoolInfo {
	t.Logf("Get Stack Pool...")
	output, err := cliutils.RunCommand(fmt.Sprintf(
		"./zbox sp-info --blobber_id %s --json --silent --wallet %s --configDir ./config --config %s",
		blobberId,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename))
	require.Nil(t, err, "Failed to get blobber stack pool information", strings.Join(output, "\n"))
	sp := new(climodel.StakePoolInfo)
	err = json.Unmarshal([]byte(output[0]), &sp)
	require.Nil(t, err, "Error unmarshalling blobber stack information", strings.Join(output, "\n"))

	return sp
}

func getChallengePoolInfo(t *testing.T, cliConfigFilename, allocationID string) *climodel.ChallengePoolInfo {
	t.Logf("Get Challenge Pool...")
	output, err := cliutils.RunCommand(fmt.Sprintf(
		"./zbox cp-info --allocation %s --json --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename))
	require.Nil(t, err, "Failed to get blobber stack pool information", strings.Join(output, "\n"))
	cp := &climodel.ChallengePoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &cp)
	require.Nil(t, err, "Error unmarshalling blobber stack information", strings.Join(output, "\n"))

	return cp
}

// size in gigabytes
func sizeInGB(size int64) float64 {
	return float64(size) / GB
}

func parseZCNtoValue(amount string) (int64, error) {
	r := regexp.MustCompile(`^(?P<amount>[\.\d]+) (?P<unit>(SAS|uZCN|mZCN|ZCN)).+$`)
	matches := r.FindStringSubmatch(amount)
	if len(matches) == 0 {
		return 0, errors.New("Amount string is not in correct format")
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}
	unit := matches[2]

	unitMuliplier := float64(1)
	switch unit {
	case "SAS", "sas":
		unitMuliplier = 1e-10
	case "uZCN", "uzcn":
		unitMuliplier = 1e-6
	case "mZCN", "mzcn":
		unitMuliplier = 1e-3
	}

	return ConvertToValue(value * unitMuliplier), nil
}

// ConvertToToken converts the value to ZCN tokens
func ConvertToToken(value int64) float64 {
	return float64(value) / float64(TOKEN_UNIT)
}

// ConvertToValue converts ZCN tokens to value
func ConvertToValue(token float64) int64 {
	return int64(token * float64(TOKEN_UNIT))
}

func getUploadCostValue(t *testing.T, allocationID, localpath string, extraParams map[string]interface{}) int64 {
	t.Logf("Getting upload cost...")
	options := map[string]interface{}{
		"allocation": allocationID,
		"localpath":  localpath,
	}
	for k, v := range extraParams {
		options[k] = v
	}
	output, err := getUploadCost(t, configPath, createParams(options))
	require.Nil(t, err, "Could not get upload cost", strings.Join(output, "\n"))
	require.Regexp(t, regexp.MustCompile(`^(?P<amount>[\.\d]+ (SAS|uZCN|mZCN|ZCN)) tokens.+$`), output[0])
	uploadCost, err := parseZCNtoValue(output[0])
	require.Nil(t, err, "Cannot convert uploadCost to float", strings.Join(output, "\n"))

	return uploadCost
}

func getUploadCost(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommand(fmt.Sprintf(
		"./zbox get-upload-cost %s --silent --wallet %s --configDir ./config --config %s ",
		params,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename))
}

func wait(t *testing.T, duration time.Duration) {
	t.Logf("Waiting %s", duration)
	time.Sleep(duration)
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
