package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

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

	t.Run("File Update with a different size - Blobbers should be paid for the extra file size", func(t *testing.T) {
		t.Parallel()

		// Logic: Upload a 2 MB file and get the upload cost. Update the 2 MB file with a 4 MB file
		// and see that blobber's write pool balances are deduced again for the cost of uploading extra
		// 2 MBs.

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 10 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		fileSize := int64(2 * MB)

		// Get expected upload cost for 2 MB
		localpath := uploadRandomlyGeneratedFile(t, allocationID, fileSize)
		output, err = getUploadCostInUnit(t, configPath, allocationID, localpath)
		require.Nil(t, err, "Could not get upload cost", strings.Join(output, "\n"))
		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))
		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := (expectedUploadCostInZCN / (2 + 2))

		// Wait for write pool blobber balances to be deduced for 2 MB
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

		remotepath := filepath.Base(localpath)
		updateFileWithRandomlyGeneratedData(t, allocationID, remotepath, int64(4*MB))

		// Wait before fetching final write pool
		wait(t, time.Minute)

		// Get the new Write Pool info after update
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

		// Blobber pool balance should reduce by expected cost of 2 MB for each blobber
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

	t.Run("File Update with same size - Users should not be charged, blobber should not be paid", func(t *testing.T) {
		t.Parallel()

		// Logic: Upload a 1 MB file, get the write pool info. Update said file with another file
		// of size 1 MB. Get write pool info and check nothing has been deducted.

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

		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Update with same size
		remotepath := filepath.Base(localpath)
		updateFileWithRandomlyGeneratedData(t, allocationID, remotepath, fileSize)

		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

		// Get final write pool, no deduction should have been made
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

	t.Run("File Rename - Users should not be charged for renaming a file", func(t *testing.T) {
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
		require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Rename file
		remotepath := filepath.Base(localpath)
		renameAllocationFile(t, allocationID, remotepath, remotepath+"_renamed")

		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

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
		require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		// Move file
		remotepath := filepath.Base(localpath)
		moveAllocationFile(t, allocationID, remotepath, "newDir")

		wait(t, 10*time.Second)
		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

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

	t.Run("Update Allocation - Lock token interest must've been put in stack pool", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		assertBalanceIs(t, "500.000 mZCN")

		blobber := getOneOfAllocationBlobbers(t, allocationID)

		offer := getAllocationOfferFromBlobberStakePool(t, blobber.BlobberID, allocationID)

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

		offer = getAllocationOfferFromBlobberStakePool(t, blobber.BlobberID, allocationID)

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

	t.Run("Create Allocation - Lock token interest must've been put in stake pool", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		assertBalanceIs(t, "500.000 mZCN")

		blobber := getOneOfAllocationBlobbers(t, allocationID)

		offer := getAllocationOfferFromBlobberStakePool(t, blobber.BlobberID, allocationID)

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

func uploadRandomlyGeneratedFile(t *testing.T, allocationID string, fileSize int64) string {
	filename := generateRandomTestFileName(t)
	err := createFileWithSize(filename, fileSize)
	require.Nil(t, err)

	output, err := uploadFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/",
		"localpath":  filename,
	})
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Equal(t, 2, len(output))
	require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])
	return filename
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

func getAllocationOfferFromBlobberStakePool(t *testing.T, blobber_id, allocationID string) *climodel.StakePoolOfferInfo {
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

// ConvertToToken converts the value to ZCN tokens
func ConvertToToken(value int64) float64 {
	return float64(value) / float64(TOKEN_UNIT)
}

// ConvertToValue converts ZCN tokens to value
func ConvertToValue(token float64) int64 {
	return int64(token * float64(TOKEN_UNIT))
}

func wait(t *testing.T, duration time.Duration) {
	t.Logf("Waiting %s", duration)
	time.Sleep(duration)
}
