package tokenomics_tests

import (
	"fmt"
	"github.com/0chain/system_test/tests/cli_tests"
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

func TestCancelEnterpriseAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Cancel allocation immediately should work")

	t.Parallel()

	t.Run("Cancel allocation immediately should work", func(t *test.SystemTest) {
		allocationID := cli_tests.setupAllocation(t, cli_tests.configPath)

		output, err := cancelAllocation(t, cli_tests.configPath, allocationID, true)
		require.NoError(t, err, "cancel allocation failed but should succeed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		cli_tests.assertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])
	})

	t.RunWithTimeout("Cancel allocation after upload should work", 5*time.Minute, func(t *test.SystemTest) {
		allocationID := cli_tests.setupAllocation(t, cli_tests.configPath)

		filename := cli_tests.generateRandomTestFileName(t)
		err := cli_tests.createFileWithSize(filename, 1*cli_tests.MB)
		require.Nil(t, err)

		output, err := cli_tests.uploadFile(t, cli_tests.configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		time.Sleep(1 * time.Minute)

		output, err = cancelAllocation(t, cli_tests.configPath, allocationID, true)
		require.NoError(t, err, "cancel allocation failed but should succeed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		cli_tests.assertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])
	})

	t.Run("No allocation param should fail", func(t *test.SystemTest) {
		// create wallet
		cli_tests.createWallet(t)

		cmd := fmt.Sprintf(
			"./zbox alloc-cancel --silent "+
				"--wallet %s --configDir ./config --config %s",
			cli_tests.escapedTestName(t)+"_wallet.json",
			cli_tests.configPath,
		)

		output, err := cliutils.RunCommandWithoutRetry(cmd)
		require.Error(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("Cancel Other's Allocation Should Fail", func(t *test.SystemTest) {
		otherAllocationID := cli_tests.setupAllocationWithWallet(t, cli_tests.escapedTestName(t)+"_other_wallet.json", cli_tests.configPath)

		cli_tests.createWallet(t)
		// otherAllocationID should not be cancelable from this level
		output, err := cancelAllocation(t, cli_tests.configPath, otherAllocationID, false)

		require.Error(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: only owner can cancel an allocation", output[len(output)-1])
	})

	t.Run("Cancel Non-existent Allocation Should Fail", func(t *test.SystemTest) {
		cli_tests.createWallet(t)

		allocationID := "123abc"

		output, err := cancelAllocation(t, cli_tests.configPath, allocationID, false)

		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.Equal(t, "Error canceling allocation:alloc_cancel_failed: value not present", output[0])
	})
}

func cancelAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Canceling allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-cancel --allocation %s --silent "+
			"--wallet %s --configDir ./config --config %s",
		allocationID,
		cli_tests.escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
