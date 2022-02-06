package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFinalizeAllocation(t *testing.T) {
	t.Parallel()

	//FIXME: POSSIBLE BUG: Error obtained on finalizing allocation says the allocation has not expired but it has
	t.Run("Finalize Expired Allocation Should Work", func(t *testing.T) {
		t.Parallel()

		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		expDuration := int64(-3) // In hours

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     fmt.Sprintf("%dh", expDuration),
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.LessOrEqual(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate)

		output, err = finalizeAllocation(t, configPath, allocationID, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: allocation is not expired yet, or waiting a challenge completion", output[0])
	})

	t.Run("Finalize Non-Expired Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := finalizeAllocation(t, configPath, allocationID, false)
		// Error should not be nil since finalize is not working
		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: allocation is not expired yet, or waiting a challenge completion", output[0])
	})

	t.Run("Finalize Other's Allocation Should Fail", func(t *testing.T) {
		t.Parallel()

		var otherAllocationID = setupAllocationWithWallet(t, escapedTestName(t)+"_other_wallet.json", configPath)

		// Then try updating with otherAllocationID: should not work
		output, err := finalizeAllocation(t, configPath, otherAllocationID, false)

		// Error should not be nil since finalize is not working
		require.NotNil(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: not allowed, unknown finalization initiator", output[len(output)-1])
	})

	t.Run("No allocation param should fail", func(t *testing.T) {
		t.Parallel()

		cmd := fmt.Sprintf(
			"./zbox alloc-fini --silent "+
				"--wallet %s --configDir ./config --config %s",
			escapedTestName(t)+"_wallet.json",
			configPath,
		)

		output, err := cliutils.RunCommandWithoutRetry(cmd)
		require.Error(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
		require.Len(t, output, 4)
		require.Equal(t, "Error: allocation flag is missing", output[len(output)-1])
	})
}

func finalizeAllocation(t *testing.T, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Finalizing allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-fini --allocation %s "+
			"--silent --wallet %s --configDir ./config --config %s",
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
