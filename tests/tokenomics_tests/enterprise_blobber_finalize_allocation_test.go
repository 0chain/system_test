package tokenomics_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestFinalizeAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// Finalise success
	// You need to use same tests cases we have in cancel allocation here. But it's not a priority.

	t.Run("Finalize Non-Expired Allocation Should Fail", func(t *test.SystemTest) {
		allocationID := utils.SetupEnterpriseAllocation(t, configPath)

		output, err := finalizeAllocation(t, configPath, allocationID, false)
		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: allocation is not expired yet", output[0])
	})

	t.Run("Finalize Other's Allocation Should Fail", func(t *test.SystemTest) {
		var otherAllocationID = utils.SetupEnterpriseAllocationWithWallet(t, utils.EscapedTestName(t)+"_other", configPath)
		//var otherAllocationID = setupAllocationWithWallet(t, utils.EscapedTestName(t)+"_other_wallet.json", configPath)

		// create wallet
		_, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error creating wallet")

		// Then try updating with otherAllocationID: should not work
		output, err := finalizeAllocation(t, configPath, otherAllocationID, false)

		// Error should not be nil since finalize is not working
		require.NotNil(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error finalizing allocation:fini_alloc_failed: not allowed, unknown finalization initiator", output[len(output)-1])
	})

	t.Run("No allocation param should fail", func(t *test.SystemTest) {
		_, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err)

		cmd := fmt.Sprintf(
			"./zbox alloc-fini --silent "+
				"--wallet %s --configDir ./config --config %s",
			utils.EscapedTestName(t)+"_wallet.json",
			configPath,
		)

		output, err := cliutils.RunCommandWithoutRetry(cmd)
		require.Error(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: allocation flag is missing", output[len(output)-1])
	})
}

func finalizeAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Finalizing allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-fini --allocation %s "+
			"--silent --wallet %s --configDir ./config --config %s",
		allocationID,
		utils.EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
