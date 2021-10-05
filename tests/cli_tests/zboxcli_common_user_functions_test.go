package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCommonUserFunctions(t *testing.T) {
	//t.Parallel()
	t.Run("parallel", func(t *testing.T) {
		t.Run("Create Allocation - user wallets are not charged but blobber should pay to write the marker to the blockchain", func(t *testing.T) {
			//t.Parallel()

			allocationID := setupAllocation(t, configPath)

			output, err := getBalance(t, configPath)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Regexp(t, regexp.MustCompile(`Balance: [0-9.]+ (|m|µ)ZCN \([0-9.]+ USD\)$`), output[0])
			r := regexp.MustCompile(`Balance: (?P<Balance>[0-9.]+ (|m|µ)ZCN) \([0-9.]+ USD\)$`)
			matches := r.FindStringSubmatch(output[0])
			userWalletBalance := matches[1]
			fmt.Println(userWalletBalance)

			allocation := getAllocationWithoutRetry(t, configPath, allocationID)
			fmt.Println(allocation)
			createAllocationTestTeardown(t, allocationID)
		})

		t.Run("File update - user wallets are not charged but blobber should pay to write the marker to the blockchain", func(t *testing.T) {
			//t.Parallel()

			allocationID := setupAllocation(t, configPath)

			output, err := getBalance(t, configPath)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Regexp(t, regexp.MustCompile(`Balance: [0-9.]+ (|m|µ)ZCN \([0-9.]+ USD\)$`), output[0])
			r := regexp.MustCompile(`Balance: (?P<Balance>[0-9.]+ (|m|µ)ZCN) \([0-9.]+ USD\)$`)
			matches := r.FindStringSubmatch(output[0])
			userWalletBalance := matches[1]
			fmt.Println(userWalletBalance)

			allocation := getAllocationWithoutRetry(t, configPath, allocationID)
			fmt.Println(allocation)
			createAllocationTestTeardown(t, allocationID)
		})
	})

}

func getAllocationWithoutRetry(t *testing.T, cliConfigFilename, allocationID string) map[string]interface{} {
	output, err := cliutils.RunCommand(fmt.Sprintf(
		"./zbox get --allocation %s --json --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename))
	require.Nil(t, err, "Failed to get allocation", strings.Join(output, "\n"))
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[0]), &jsonMap)
	require.Nil(t, err, "Error unmarshalling allocation", strings.Join(output, "\n"))

	return jsonMap
}

// func createNewAllocation(t *testing.T, cliConfigFilename, params string) ([]string, error) {
// 	return cliutils.RunCommandWithRetry(fmt.Sprintf(
// 		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
// 		params,
// 		escapedTestName(t)+"_wallet.json",
// 		cliConfigFilename,
// 		escapedTestName(t)+"_allocation.txt"), 3, time.Second*5)
// }
