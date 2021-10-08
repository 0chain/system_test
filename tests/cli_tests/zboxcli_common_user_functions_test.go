package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	KB = 1024      // kilobyte
	MB = 1024 * KB // megabyte
	GB = 1024 * MB // gigabyte
)

func getOneOfAllocationBlobbers(t *testing.T, allocationID string) map[string]interface{} {
	allocation := getAllocation(t, configPath, allocationID)

	blobbersI := allocation["blobber_details"].([]interface{})

	require.GreaterOrEqual(t, len(blobbersI), 1, "Allocation must've been stored at least on one blobber")

	blobbers := make([]map[string]interface{}, len(blobbersI))
	for i, blobber := range blobbersI {
		blobbers[i] = blobber.(map[string]interface{})
	}

	// We can also select a blobber randomly or select the first one
	blobber := blobbers[0]

	return blobber
}

func assertBalanceIs(t *testing.T, balance string) {
	userWalletBalance := getWalletBalance(t, configPath)
	require.Equal(t, balance, userWalletBalance, "User wallet balance mismatch")
}

func TestCommonUserFunctions(t *testing.T) {
	t.Parallel()
	t.Run("parallel", func(t *testing.T) {
		t.Run("Update Allocation - Lock token interest must've been put in stack pool", func(t *testing.T) {
			t.Parallel()

			allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

			assertBalanceIs(t, "500.000 mZCN")

			blobber := getOneOfAllocationBlobbers(t, allocationID)

			blobber_id := blobber["blobber_id"].(string)

			size := blobber["size"].(float64)
			terms := blobber["terms"].(map[string]interface{})
			write_price := terms["write_price"].(float64)

			offer := getAllocationOfferFromBlobberStackPool(t, blobber_id, allocationID)

			lock := int64(offer["lock"].(float64))
			expectedLock := int64(sizeInGB(size) * write_price)
			require.Equal(t, expectedLock, lock, "Lock token interest must've been put in stack pool")

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

			blobber_id = blobber["blobber_id"].(string)

			size = blobber["size"].(float64)
			terms = blobber["terms"].(map[string]interface{})
			write_price = terms["write_price"].(float64)

			offer = getAllocationOfferFromBlobberStackPool(t, blobber_id, allocationID)

			lock = int64(offer["lock"].(float64))
			expectedLock = int64(sizeInGB(size) * write_price)
			require.Equal(t, expectedLock, lock, "Lock token interest must've been put in stack pool")

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

			blobber_id := blobber["blobber_id"].(string)

			size := blobber["size"].(float64)
			terms := blobber["terms"].(map[string]interface{})
			write_price := terms["write_price"].(float64)

			offer := getAllocationOfferFromBlobberStackPool(t, blobber_id, allocationID)

			lock := int64(offer["lock"].(float64))
			expectedLock := int64(sizeInGB(size) * write_price)
			require.Equal(t, expectedLock, lock, "Lock token interest must've been put in stack pool")

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

func getAllocationOfferFromBlobberStackPool(t *testing.T, blobber_id, allocationID string) map[string]interface{} {
	sp_info := getStackPoolInfo(t, configPath, blobber_id)
	offersI := sp_info["offers"].([]interface{})

	require.GreaterOrEqual(t, len(offersI), 1, "Blobbers offers must not be empty")

	// Find the offer related to this allocation
	offers := make([]map[string]interface{}, len(offersI))
	n := 0
	for _, o := range offersI {
		offer := o.(map[string]interface{})
		if offer["allocation_id"].(string) == allocationID {
			offers[n] = offer
			n++
		}
	}

	require.GreaterOrEqual(t, n, 1, "The allocation offer expected to be found on blobber stack pool information")

	offer := offers[0]
	return offer
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

func getAllocation(t *testing.T, cliConfigFilename, allocationID string) map[string]interface{} {
	return getAllocationWithRetry(t, cliConfigFilename, allocationID, 1)
}

func getAllocationWithRetry(t *testing.T, cliConfigFilename, allocationID string, retry int) map[string]interface{} {
	t.Logf("Get Allocation...")
	output, err := cliutils.RunCommandWithRetry(t, fmt.Sprintf(
		"./zbox get --allocation %s --json --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename), retry, time.Second*5)
	require.Nil(t, err, "Failed to get allocation", strings.Join(output, "\n"))
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[0]), &jsonMap)
	require.Nil(t, err, "Error unmarshalling allocation", strings.Join(output, "\n"))

	return jsonMap
}

func getStackPoolInfo(t *testing.T, cliConfigFilename, blobberId string) map[string]interface{} {
	t.Logf("Get Stack Pool...")
	output, err := cliutils.RunCommand(fmt.Sprintf(
		"./zbox sp-info --blobber_id %s --json --silent --wallet %s --configDir ./config --config %s",
		blobberId,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename))
	require.Nil(t, err, "Failed to get blobber stack pool information", strings.Join(output, "\n"))
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[0]), &jsonMap)
	require.Nil(t, err, "Error unmarshalling blobber stack information", strings.Join(output, "\n"))

	return jsonMap
}

// size in gigabytes
func sizeInGB(size float64) float64 {
	return float64(size) / GB
}
