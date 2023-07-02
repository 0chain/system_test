package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var (
	cancelAllocationRegex = regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`)
)

func TestCancelAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Cancel allocation immediately should work")

	t.Parallel()

	t.Run("Cancel allocation immediately should work", func(t *test.SystemTest) {
		output, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		allocationID := setupAllocation(t, configPath)

		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.NoError(t, err, "cancel allocation failed but should succeed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])
	})

	t.Run("No allocation param should fail", func(t *test.SystemTest) {
		// create wallet
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		cmd := fmt.Sprintf(
			"./zbox alloc-cancel --silent "+
				"--wallet %s --configDir ./config --config %s",
			escapedTestName(t)+"_wallet.json",
			configPath,
		)

		output, err := cliutils.RunCommandWithoutRetry(cmd)
		require.Error(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("Cancel Other's Allocation Should Fail", func(t *test.SystemTest) {
		otherAllocationID := setupAllocationWithWallet(t, escapedTestName(t)+"_other_wallet.json", configPath)

		_, err := createWallet(t, configPath)
		require.NoError(t, err)
		// otherAllocationID should not be cancelable from this level
		output, err := cancelAllocation(t, configPath, otherAllocationID, false)

		require.Error(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: only owner can cancel an allocation", output[len(output)-1])
	})

	t.Run("Cancel Non-existent Allocation Should Fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		allocationID := "123abc"

		output, err := cancelAllocation(t, configPath, allocationID, false)

		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: value not present", output[0])
	})

	t.RunWithTimeout("Cancel Expired Allocation Should Fail", 3*time.Minute, func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		output, err := updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "1s",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		t.Cleanup(func() {
			output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]string{
				"time_unit": "1h",
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		allocationID, _ := setupAndParseAllocation(t, configPath, map[string]interface{}{
			"expire": "5s",
		})

		time.Sleep(30 * time.Second)
		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.LessOrEqual(t, ac.ExpirationDate, time.Now().Unix())

		// Cancel the expired allocation
		output, err = cancelAllocation(t, configPath, allocationID, false)
		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))

		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: trying to cancel expired allocation", output[0])
	})
}

func cancelAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Canceling allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-cancel --allocation %s --silent "+
			"--wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
