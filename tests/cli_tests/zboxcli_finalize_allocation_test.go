package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFinalizeAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Finalize Expired Allocation Should Work after challenge completion time + expiry", func(t *testing.T) {
		t.Parallel()

		expDuration := int64(15) // In secs
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"expire": fmt.Sprintf("\"%ds\"", expDuration),
		})

		// Wait
		cliutils.Wait(t, 4*time.Minute+time.Duration(expDuration)*time.Second)

		output, err := finalizeAllocation(t, configPath, allocationID, false)

		require.Nil(t, err, "unexpected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		matcher := regexp.MustCompile("Allocation finalized with txId .*$")
		require.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")
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
