package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCreateAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.RunWithTimeout("Create allocation with name Should Work", 90*time.Second, func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		name := cliutils.RandomAlphaNumericString(10)

		options := map[string]interface{}{
			"lock": "0.5",
			"name": name,
		}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		alloc := getAllocation(t, allocationID)

		require.Equal(t, name, alloc.Name, "allocation name is not created properly")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation without providing any additional parameters Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.RunWithTimeout("Create allocation with smallest expiry (5m) Should Work", 120*time.Second, func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "5m", "size": "256000", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with smallest possible size (1024) Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "1h", "size": "1024", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with parity specified Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "1h", "size": "1024", "parity": "1", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with data specified Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "1h", "size": "1024", "data": "1", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with read price range Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "1h", "size": "1024", "read_price": "0-9999", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with write price range Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "1h", "size": "1024", "write_price": "0-9999", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with too large parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"parity": "99", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: allocation_creation_failed: Too many blobbers selected, max available \\d*"), output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with too large data (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"data": "99", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: allocation_creation_failed: Too many blobbers selected, max available \\d*"), output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with too large data and parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"data": "30", "parity": "20", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: allocation_creation_failed: Too many blobbers selected, max available \\d*"), output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with read price range 0-0 Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"read_price": "0-0", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: not enough blobbers to honor the allocation", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with size smaller than limit (size < 1024) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": 256, "lock": "0.5"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: allocation_creation_failed: invalid request: insufficient allocation size", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with expire smaller than limit (expire < 5m) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "3m", "lock": "0.5", "size": 1024}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: allocation_creation_failed: invalid request: insufficient allocation duration", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with no parameter (missing lock) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "missing required 'lock' argument", output[len(output)-1])
	})

	t.Run("Create allocation with invalid expiry Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "-1", "lock": "0.5"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "invalid argument \"-1\" for \"--expire\" flag: time: missing unit in duration \"-1\"", output[len(output)-1])
	})

	t.Run("Create allocation by providing expiry in wrong format (expire 1hour) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "1hour", "lock": "0.5"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "invalid argument \"1hour\" for \"--expire\" flag: time: unknown unit \"hour\" in duration \"1hour\"", output[len(output)-1])
	})
}

func setupWallet(t *test.SystemTest, configPath string) []string {
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	output, err = executeFaucetWithTokens(t, configPath, 1)
	require.Nil(t, err, strings.Join(output, "\n"))

	output, err = getBalance(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	return output
}

func createNewAllocation(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return createNewAllocationForWallet(t, escapedTestName(t), cliConfigFilename, params)
}

func createNewAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Creating new allocation...")
	return cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
		wallet+"_allocation.txt"), 3, time.Second*5)
}

func createNewAllocationWithoutRetry(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommandWithoutRetry(fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
		escapedTestName(t)+"_allocation.txt"))
}

func createAllocationTestTeardown(t *test.SystemTest, allocationID string) {
	t.Cleanup(func() {
		_, _ = cancelAllocation(t, configPath, allocationID, false)
	})
}
