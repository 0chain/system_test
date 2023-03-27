package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCoreCreateAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

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

	t.Run("Create allocation with smallest expiry (5m) Should Work", func(t *test.SystemTest) {
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
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create allocation with too large data (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"data": "99", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create allocation with too large data and parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"data": "30", "parity": "20", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
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

	t.Run("Create allocation with expire smaller than limit (expire < 1s) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"expire": "1s", "lock": "0.5", "size": 1024}
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

	t.Run("Create allocation should have all file options permitted by default", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err)

		// get allocation
		var alloc *climodel.Allocation

		output, err = getAllocationWithRetry(t, configPath, allocationID, 10)
		require.Nil(t, err, "error fetching allocation")
		require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
		err = json.Unmarshal([]byte(output[0]), &alloc)
		require.Nil(t, err, "error unmarshalling allocation json")
		require.Equal(t, uint16(63), alloc.FileOptions)
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with some forbidden file options flags should pass and show in allocation", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)
		_, err := executeFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err)

		// Forbid upload
		options := map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h", "forbid_upload": nil}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err)
		alloc := getAllocation(t, allocationID)
		require.Equal(t, uint16(62), alloc.FileOptions) // 63 - 1 = 62 (upload mask = 1)

		createAllocationTestTeardown(t, allocationID)

		// Forbid delete
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h", "forbid_delete": nil}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = getAllocationID(output[0])
		require.Nil(t, err)
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(61), alloc.FileOptions) // 63 - 2 = 62 (delete mask = 2)

		createAllocationTestTeardown(t, allocationID)

		// Forbid update
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h", "forbid_update": nil}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = getAllocationID(output[0])
		require.Nil(t, err)
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(59), alloc.FileOptions) // 63 - 4 = 59 (update mask = 4)

		createAllocationTestTeardown(t, allocationID)

		// Forbid move
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h", "forbid_move": nil}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = getAllocationID(output[0])
		require.Nil(t, err)
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(55), alloc.FileOptions) // 63 - 8 = 55 (move mask = 8)

		createAllocationTestTeardown(t, allocationID)

		// Forbid copy
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h", "forbid_copy": nil}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = getAllocationID(output[0])
		require.Nil(t, err)
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(47), alloc.FileOptions) // 63 - 16 = 47 (copy mask = 8)

		createAllocationTestTeardown(t, allocationID)

		// Forbid rename
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h", "forbid_rename": nil}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = getAllocationID(output[0])
		require.Nil(t, err)
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(31), alloc.FileOptions) // 63 - 32 = 31 (rename mask = 32)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation third_party_extendable should be false by default and change with flags", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		// Forbid update, rename and delete
		options := map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err)

		// get allocation
		var alloc *climodel.Allocation

		output, err = getAllocationWithRetry(t, configPath, allocationID, 10)
		require.Nil(t, err, "error fetching allocation")
		require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
		err = json.Unmarshal([]byte(output[0]), &alloc)
		require.Nil(t, err, "error unmarshalling allocation json")
		require.Equal(t, false, alloc.ThirdPartyExtendable) // 63 - (2 + 4 + 32) = 25 (update mask = 2, rename = 32, delete = 4)
		createAllocationTestTeardown(t, allocationID)

		// Forbid upload, move and copy
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "expire": "1h", "third_party_extendable": nil}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = getAllocationID(output[0])
		require.Nil(t, err)

		// get allocation
		output, err = getAllocationWithRetry(t, configPath, allocationID, 10)
		require.Nil(t, err, "error fetching allocation")
		require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
		err = json.Unmarshal([]byte(output[0]), &alloc)
		require.Nil(t, err, "error unmarshalling allocation json")
		require.Equal(t, true, alloc.ThirdPartyExtendable) // 63 - (1 + 8 + 16) = 38 (upload mask = 1, move = 8, copy = 16)
		createAllocationTestTeardown(t, allocationID)
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
