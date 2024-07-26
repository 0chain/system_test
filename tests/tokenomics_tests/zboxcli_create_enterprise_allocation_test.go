package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/tests/cli_tests"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCreateEnterpriseAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create enterprise allocation for locking cost equal to the cost calculated should work")

	t.Parallel()

	t.Run("Create enterprise allocation for locking cost equal to the cost calculated should work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{
			"cost":        "",
			"size":        "10000",
			"read_price":  "0-1",
			"write_price": "0-1",
		}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		allocationCost, err := cli_tests.getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		options = map[string]interface{}{
			"lock":        allocationCost,
			"size":        "10000",
			"read_price":  "0-1",
			"write_price": "0-1",
		}
		output, err = createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation for locking cost less than minimum cost should not work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{
			"cost":        "",
			"read_price":  "0-1",
			"write_price": "0-1",
			"size":        10000,
		}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")

		allocationCost, err := cli_tests.getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		mustFailCost := allocationCost * 0.8
		options = map[string]interface{}{"lock": mustFailCost}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "not enough tokens to honor the allocation")
	})

	t.Run("Create enterprise allocation for locking negative cost should not work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{
			"cost":        "",
			"read_price":  "0-1",
			"write_price": "0-1",
			"size":        10000,
		}
		mustFailCost := -1
		options = map[string]interface{}{"lock": mustFailCost}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
	})

	t.Run("Create enterprise allocation with smallest possible size (1024) Should Work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"size": "1024", "lock": "0.5"}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation for another owner should Work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)
		targetWalletName := cli_tests.escapedTestName(t) + "_TARGET"
		cli_tests.createWalletForName(targetWalletName)

		targetWallet, err := cli_tests.getWalletForName(t, cli_tests.configPath, targetWalletName)
		require.Nil(t, err, "could not get target wallet")

		options := map[string]interface{}{
			"lock":             "0.5",
			"owner":            targetWallet.ClientID,
			"owner_public_key": targetWallet.ClientPublicKey,
		}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		// upload file for designated owner should work
		file := cli_tests.generateRandomTestFileName(t)
		fileSize := int64(102400) // this is big enough to cause problem with download
		err = cli_tests.createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": "/",
			"encrypt":    "",
		}
		output, err = cli_tests.uploadFileForWallet(t, targetWalletName, cli_tests.configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		// upload for creating wallet should fail
		output, err = cli_tests.uploadFile(t, cli_tests.configPath, uploadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, ""), "Operation needs to be performed by the owner or the payer of the allocation")

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with parity specified Should Work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"size": "1024", "parity": "1", "lock": "0.5"}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with data specified Should Work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"size": "1024", "data": "1", "lock": "0.5"}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with read price range Should Work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"size": "1024", "read_price": "0-9999", "lock": "0.5"}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with write price range Should Work", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"size": "1024", "write_price": "0-9999", "lock": "0.5"}
		output, err := createNewEnterpriseAllocation(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))
		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with too large parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"parity": "99", "lock": "0.5", "size": 1024}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create enterprise allocation with too large data (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"data": "99", "lock": "0.5", "size": 1024}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create enterprise allocation with too large data and parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"data": "30", "parity": "20", "lock": "0.5", "size": 1024}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create enterprise allocation with read price range 0-0 Should Fail", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"read_price": "0-0", "lock": "0.5", "size": 1024}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: not enough blobbers to honor the allocation", strings.Join(output, "\n"))
	})

	t.Run("Create enterprise allocation with size smaller than limit (size < 1024) Should Fail", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"size": 256, "lock": "0.5"}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: allocation_creation_failed: invalid request: insufficient allocation size", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create enterprise allocation with no parameter (missing lock) Should Fail", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "missing required 'lock' argument", output[len(output)-1])
	})

	t.Run("Create enterprise allocation should have all file options permitted by default", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		options := map[string]interface{}{"lock": "0.5", "size": 1024}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err)

		// get allocation
		var alloc *climodel.Allocation

		output, err = cli_tests.getAllocationWithRetry(t, cli_tests.configPath, allocationID, 10)
		require.Nil(t, err, "error fetching allocation")
		require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
		err = json.Unmarshal([]byte(output[0]), &alloc)
		require.Nil(t, err, "error unmarshalling allocation json")
		require.Equal(t, uint16(63), alloc.FileOptions)
		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with some forbidden file options flags should pass and show in allocation", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		// Forbid upload
		options := map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_upload": nil}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err)
		alloc := cli_tests.getAllocation(t, allocationID)
		require.Equal(t, uint16(62), alloc.FileOptions) // 63 - 1 = 62 (upload mask = 1)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		// Forbid delete
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_delete": nil}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = cli_tests.getAllocationID(output[0])
		require.Nil(t, err)
		alloc = cli_tests.getAllocation(t, allocationID)
		require.Equal(t, uint16(61), alloc.FileOptions) // 63 - 2 = 62 (delete mask = 2)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		// Forbid update
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_update": nil}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = cli_tests.getAllocationID(output[0])
		require.Nil(t, err)
		alloc = cli_tests.getAllocation(t, allocationID)
		require.Equal(t, uint16(59), alloc.FileOptions) // 63 - 4 = 59 (update mask = 4)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		// Forbid move
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_move": nil}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = cli_tests.getAllocationID(output[0])
		require.Nil(t, err)
		alloc = cli_tests.getAllocation(t, allocationID)
		require.Equal(t, uint16(55), alloc.FileOptions) // 63 - 8 = 55 (move mask = 8)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		// Forbid copy
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_copy": nil}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = cli_tests.getAllocationID(output[0])
		require.Nil(t, err)
		alloc = cli_tests.getAllocation(t, allocationID)
		require.Equal(t, uint16(47), alloc.FileOptions) // 63 - 16 = 47 (copy mask = 8)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		// Forbid rename
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_rename": nil}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = cli_tests.getAllocationID(output[0])
		require.Nil(t, err)
		alloc = cli_tests.getAllocation(t, allocationID)
		require.Equal(t, uint16(31), alloc.FileOptions) // 63 - 32 = 31 (rename mask = 32)

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation third_party_extendable should be false by default and change with flags", func(t *test.SystemTest) {
		_ = cli_tests.setupWallet(t, cli_tests.configPath)

		// Forbid update, rename and delete
		options := map[string]interface{}{"lock": "0.5", "size": 1024}
		output, err := createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := cli_tests.getAllocationID(output[0])
		require.Nil(t, err)

		// get allocation
		var alloc *climodel.Allocation

		output, err = cli_tests.getAllocationWithRetry(t, cli_tests.configPath, allocationID, 10)
		require.Nil(t, err, "error fetching allocation")
		require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
		err = json.Unmarshal([]byte(output[0]), &alloc)
		require.Nil(t, err, "error unmarshalling allocation json")
		require.Equal(t, false, alloc.ThirdPartyExtendable) // 63 - (2 + 4 + 32) = 25 (update mask = 2, rename = 32, delete = 4)
		createEnterpriseAllocationTestTeardown(t, allocationID)

		// Forbid upload, move and copy
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "third_party_extendable": nil}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, cli_tests.configPath, cli_tests.createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = cli_tests.getAllocationID(output[0])
		require.Nil(t, err)

		// get allocation
		output, err = cli_tests.getAllocationWithRetry(t, cli_tests.configPath, allocationID, 10)
		require.Nil(t, err, "error fetching allocation")
		require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
		err = json.Unmarshal([]byte(output[0]), &alloc)
		require.Nil(t, err, "error unmarshalling allocation json")
		require.Equal(t, true, alloc.ThirdPartyExtendable) // 63 - (1 + 8 + 16) = 38 (upload mask = 1, move = 8, copy = 16)
		createEnterpriseAllocationTestTeardown(t, allocationID)
	})
}

func createNewEnterpriseAllocation(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return createNewEnterpriseAllocationForWallet(t, cli_tests.escapedTestName(t), cliConfigFilename, params)
}

func createNewEnterpriseAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Creating new allocation...")
	return cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
		wallet+"_allocation.txt"), 3, time.Second*5)
}

func createNewEnterpriseAllocationWithoutRetry(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommandWithoutRetry(fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		cli_tests.escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
		cli_tests.escapedTestName(t)+"_allocation.txt"))
}

func createEnterpriseAllocationTestTeardown(t *test.SystemTest, allocationID string) {
	t.Cleanup(func() {
		_, _ = cli_tests.cancelAllocation(t, cli_tests.configPath, allocationID, false)
	})
}
