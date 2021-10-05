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

func TestCommonUserFunctions(t *testing.T) {
	//t.Parallel()
	t.Run("parallel", func(t *testing.T) {
		t.Run("Create Allocation - Lock token interest must've been put in stack pool", func(t *testing.T) {
			//t.Parallel()

			allocationID := setupAllocation(t, configPath)

			output, err := getBalance(t, configPath)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Regexp(t, regexp.MustCompile(`Balance: [0-9.]+ (|m|µ)ZCN \([0-9.]+ USD\)$`), output[0])
			// r := regexp.MustCompile(`Balance: (?P<Balance>[0-9.]+ (|m|µ)ZCN) \([0-9.]+ USD\)$`)
			// matches := r.FindStringSubmatch(output[0])
			// userWalletBalance := matches[1]
			// fmt.Println(userWalletBalance)

			allocation := getAllocation(t, configPath, allocationID)

			blobbersI := allocation["blobbers"].([]interface{})

			require.GreaterOrEqual(t, len(blobbersI), 1, "Allocation must've been stored at least on one blobber", strings.Join(output, "\n"))

			blobbers := make([]map[string]interface{}, len(blobbersI))
			for i, blobber := range blobbersI {
				blobbers[i] = blobber.(map[string]interface{})
			}

			// We can also select a blobber randomly or select the first one
			blobber := blobbers[0]
			blobber_id := blobber["id"].(string)

			sp_info := getStackPoolInfo(t, configPath, blobber_id)
			offersI := sp_info["offers"].([]interface{})

			require.GreaterOrEqual(t, len(offersI), 1, "Blobbers offers must not be empty")

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
			lock := offer["lock"].(float64)

			require.Equal(t, lock, float64(643), "Lock token interest must've been put in stack pool")

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

func getAllocation(t *testing.T, cliConfigFilename, allocationID string) map[string]interface{} {
	return getAllocationWithRetry(t, cliConfigFilename, allocationID, 1)
}

func getAllocationWithRetry(t *testing.T, cliConfigFilename, allocationID string, retry int) map[string]interface{} {
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
