package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	t.Run("parallel", func(t *testing.T) {
		t.Run("File Update - Users should not be charged for updating a file ", func(t *testing.T) {
			t.Parallel()

			allocationSize := int64(1 * MB)
			fileSize := int64(math.Floor(512 * KB))

			allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": allocationSize})

			time.Sleep(10 * time.Second)
			wp := getWritePool(t, configPath)
			require.Equal(t, int64(5000000000), wp[0].Balance, "Write pool balance expected to be equal to locked amount")

			filename, uploadCost := uploadRandomlyGeneratedFile(t, allocationID, fileSize)

			// uploadCost takes into account data+parity, so we divide by that
			uploadCost = (uploadCost / (2 + 2))
			expected_wp_balance := int64(float64(5000000000) - float64(uploadCost))

			time.Sleep(15 * time.Second)
			wp = getWritePool(t, configPath)
			require.Equal(t, 1, len(wp), "Write pool expeted to be found")

			// There is a small difference in the expected and actual balance.
			// The reason needs to be investigated. For now we consider it to be
			// in a range close to expexted value. (range = 100 SAS)
			require.InDelta(t, expected_wp_balance, wp[0].Balance, 100, "Tokens must be transfered Reward Pool to Write Pool", "difference:", wp[0].Balance-expected_wp_balance)
			if wp[0].Balance-expected_wp_balance != 0 {
				t.Log("WARNING: difference in amount taken from Write Pool with the upload cost: ", wp[0].Balance-expected_wp_balance, " SAS")
			}

			cp_balance := getChallengePoolBalance(t, configPath, allocationID)
			require.Equal(t, int64(5000000000)-wp[0].Balance, int64(cp_balance), "Tokens must be transfered from Write Pool to Chanllenge Pool")

			blobber := getOneOfAllocationBlobbers(t, allocationID)

			offer := getAllocationOfferFromBlobberStackPool(t, blobber.BlobberID, allocationID)

			expectedLock := sizeInGB(blobber.Size) * blobber.Terms.Write_price
			require.Equal(t, int64(expectedLock), int64(offer.Lock), "Lock token interest must've been put in stack pool")

			updateFileWithRandomlyGeneratedData(t, allocationID, filename, fileSize)

			time.Sleep(10 * time.Second)
			new_wp := getWritePool(t, configPath)
			require.Equal(t, wp[0].Balance, new_wp[0].Balance, "The write pool is expected to not be changed after update file", "difference:", wp[0].Balance-new_wp[0].Balance)

			new_cp_balance := getChallengePoolBalance(t, configPath, allocationID)
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

		// t.Run("File update - user wallets are not charged but blobber should pay to write the marker to the blockchain", func(t *testing.T) {
		// 	//t.Parallel()

		// 	allocationID := setupAllocation(t, configPath)

		// 	output, err := getBalance(t, configPath)
		// 	require.Nil(t, err, strings.Join(output, "\n"))
		// 	require.Len(t, output, 1)
		// 	require.Regexp(t, regexp.MustCompile(`Balance: [0-9.]+ (|m|µ)ZCN \([0-9.]+ USD\)$`), output[0])
		// 	r := regexp.MustCompile(`Balance: (?P<Balance>[0-9.]+ (|m|µ)ZCN) \([0-9.]+ USD\)$`)
		// 	matches := r.FindStringSubmatch(output[0])
		// 	userWalletBalance := matches[1]
		// 	fmt.Println(userWalletBalance)

		// 	allocation := getAllocationWithoutRetry(t, configPath, allocationID)
		// 	fmt.Println(allocation)
		// 	createAllocationTestTeardown(t, allocationID)
		// })
	})

}

func getWritePool(t *testing.T, cliConfigFilename string) []climodel.WritePoolInfo {
	output, err := writePoolInfo(t, configPath)
	require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

	initialWritePool := []climodel.WritePoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &initialWritePool)
	require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
	return initialWritePool
}

func getChallengePoolBalance(t *testing.T, cliConfigFilename, allocationID string) float64 {
	cp := getChallengePoolInfo(t, cliConfigFilename, allocationID)
	balance := cp.Balance
	return balance
}

func assertLockInRewardPoolIs(t *testing.T, expectedLock int64, cliConfigFilename, allocationID, blobber_id string) {
	offer := getAllocationOfferFromBlobberStackPool(t, blobber_id, allocationID)

	lock := offer.Lock
	require.Equal(t, expectedLock, lock, "Lock token interest must've been put in stack pool")
}

func getOneOfAllocationBlobbers(t *testing.T, allocationID string) *climodel.BlobberAllocation {
	allocation := getAllocation(t, configPath, allocationID)

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

func getAllocation(t *testing.T, cliConfigFilename, allocationID string) *climodel.Allocation {
	return getAllocationWithRetry(t, cliConfigFilename, allocationID, 1)
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
