package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0chain/common/core/common"
	"github.com/0chain/gosdk/core/zcncrypto"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCreateAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create allocation for locking cost equal to the cost calculated should work")

	t.TestSetup("register wallet and get blobbers", func() {
		createWallet(t)

		// get the list of blobbers
		blobbersList = getBlobbersList(t)
		require.Greater(t, len(blobbersList), 0, "No blobbers found")
	})

	t.Parallel()

	t.RunSequentiallyWithTimeout("Create allocation with invalid blobber auth ticket should fail", 10*time.Minute, func(t *test.SystemTest) {
		// Update blobber config to make restricted blobbers to true
		blobber := blobbersList[0]
		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobber.Id,
			"is_restricted": "true",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))

		t.Cleanup(func() {
			// Reset blobber config to make restricted blobbers to false
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber.Id,
				"is_restricted": "false",
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		// Setup wallet and create allocation
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "1024", "data": "3", "parity": "3", "lock": "0.5"}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		//require.Equal(t, "missing required 'lock' argument", output[len(output)-1])

		options = map[string]interface{}{"size": "1024", "data": "3", "parity": "3", "lock": "0.5", "preferred_blobbers": blobber.Id, "blobber_auth_tickets": "invalid"}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "invalid blobber auth ticket")
	})

	t.Skip()

	t.Run("Create allocation for locking cost equal to the cost calculated should work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{
			"cost":        "",
			"size":        "10000",
			"read_price":  "0-1",
			"write_price": "0-1",
		}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		allocationCost, err := getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		options = map[string]interface{}{
			"lock":        allocationCost,
			"size":        "10000",
			"read_price":  "0-1",
			"write_price": "0-1",
		}
		output, err = createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation for locking cost less than minimum cost should not work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{
			"cost":        "",
			"read_price":  "0-1",
			"write_price": "0-1",
			"size":        10000,
		}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")

		allocationCost, err := getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		mustFailCost := allocationCost * 0.8
		options = map[string]interface{}{"lock": mustFailCost}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "not enough tokens to honor the allocation")
	})

	t.Run("Create allocation for locking negative cost should not work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{
			"cost":        "",
			"read_price":  "0-1",
			"write_price": "0-1",
			"size":        10000,
		}
		mustFailCost := -1
		options = map[string]interface{}{"lock": mustFailCost}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
	})

	t.Run("Create allocation with smallest possible size (1024) Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "1024", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation for another owner should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)
		targetWalletName := escapedTestName(t) + "_TARGET"
		createWalletForName(targetWalletName)

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "could not get target wallet")

		options := map[string]interface{}{
			"lock":             "0.5",
			"owner":            targetWallet.ClientID,
			"owner_public_key": targetWallet.ClientPublicKey,
		}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		// upload file for designated owner should work
		file := generateRandomTestFileName(t)
		fileSize := int64(102400) // this is big enough to cause problem with download
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": "/",
			"encrypt":    "",
		}
		output, err = uploadFileForWallet(t, targetWalletName, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		// upload for creating wallet should fail
		output, err = uploadFile(t, configPath, uploadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, ""), "Operation needs to be performed by the owner or the payer of the allocation")

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create allocation with parity specified Should Work", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "1024", "parity": "1", "lock": "0.5"}
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

		options := map[string]interface{}{"size": "1024", "data": "1", "lock": "0.5"}
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

		options := map[string]interface{}{"size": "1024", "read_price": "0-9999", "lock": "0.5"}
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

		options := map[string]interface{}{"size": "1024", "write_price": "0-9999", "lock": "0.5"}
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

		options := map[string]interface{}{"parity": "99", "lock": "0.5", "size": 1024}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create allocation with too large data (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"data": "99", "lock": "0.5", "size": 1024}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create allocation with too large data and parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"data": "30", "parity": "20", "lock": "0.5", "size": 1024}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Too many blobbers selected")
	})

	t.Run("Create allocation with read price range 0-0 Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"read_price": "0-0", "lock": "0.5", "size": 1024}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: not enough blobbers to honor the allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with size smaller than limit (size < 1024) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": 256, "lock": "0.5"}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: allocation_creation_failed: invalid request: insufficient allocation size", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with no parameter (missing lock) Should Fail", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{}
		output, err := createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "missing required 'lock' argument", output[len(output)-1])
	})

	t.Run("Create allocation should have all file options permitted by default", func(t *test.SystemTest) {
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"lock": "0.5", "size": 1024}
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

		// Forbid upload
		options := map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_upload": nil}
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
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_delete": nil}
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
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_update": nil}
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
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_move": nil}
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
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_copy": nil}
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
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_rename": nil}
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
		options := map[string]interface{}{"lock": "0.5", "size": 1024}
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
		options = map[string]interface{}{"lock": "0.5", "size": 1024, "third_party_extendable": nil}
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

	t.RunSequentiallyWithTimeout("Create allocation on restricted blobbers should pass with correct auth tickets", 10*time.Minute, func(t *test.SystemTest) {
		// Update blobber config to make restricted blobbers to true
		blobber1 := blobbersList[0]
		blobber2 := blobbersList[1]
		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobber1.Id,
			"is_restricted": "true",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobber2.Id,
			"is_restricted": "true",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))

		t.Cleanup(func() {
			// Reset blobber config to make restricted blobbers to false
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber1.Id,
				"is_restricted": "false",
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber2.Id,
				"is_restricted": "false",
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		// Setup wallet and create allocation
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "1024", "data": "3", "parity": "3", "lock": "0.5"}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		//require.Equal(t, "missing required 'lock' argument", output[len(output)-1])

		// Retry with auth ticket
		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "could not get wallet")

		blobber1AuthTicket, err := getBlobberAuthTicket(t, blobber1.Id, blobber1.Url, wallet.ClientID)
		require.Nil(t, err, "could not get blobber1 auth ticket")
		blobber2AuthTicket, err := getBlobberAuthTicket(t, blobber2.Id, blobber2.Url, wallet.ClientID)
		require.Nil(t, err, "could not get blobber2 auth ticket")

		options = map[string]interface{}{"size": "1024", "data": "3", "parity": "2", "lock": "0.5", "preferred_blobbers": blobber1.Id + "," + blobber2.Id, "blobber_auth_tickets": blobber1AuthTicket + "," + blobber2AuthTicket}
		output, err = createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createAllocationTestTeardown(t, allocationID)
	})
}

func setupWallet(t *test.SystemTest, configPath string) []string {
	createWallet(t)

	output, err := getBalance(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	return output
}

func createNewAllocation(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return createNewAllocationForWallet(t, escapedTestName(t), cliConfigFilename, params)
}

func createNewAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {
	print(fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
		wallet+"_allocation.txt"))
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

func getBlobberAuthTicket(t *test.SystemTest, blobberID, blobberUrl, clientID string) (string, error) {
	zboxWallet, err := getWalletForName(t, configPath, zboxTeamWallet)
	require.Nil(t, err, "could not get zbox wallet")

	var authTicket string
	signatureScheme := zcncrypto.NewSignatureScheme("bls0chain")
	_ = signatureScheme.SetPrivateKey("85e2119f494cd40ca524f6342e8bdb7bef2af03fe9a08c8d9c1d9f14d6c64f14")
	_ = signatureScheme.SetPublicKey(zboxWallet.ClientPublicKey)

	signature, err := signatureScheme.Sign(hex.EncodeToString([]byte(zboxWallet.ClientPublicKey)))
	if err != nil {
		return authTicket, err
	}

	//sigScheme := zcncrypto.NewSignatureScheme("bls0chain")
	//if err := sigScheme.SetPublicKey(zboxWallet.ClientPublicKey); err != nil {
	//	return "", err
	//}
	//
	//success, err := sigScheme.Verify(signature, hex.EncodeToString([]byte(zboxWallet.ClientPublicKey)))
	//if err != nil || !success {
	//	return "", err
	//}

	url := blobberUrl + "/v1/auth/generate?client_id=" + clientID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return authTicket, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Zbox-Signature", signature)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return authTicket, err
	}
	defer resp.Body.Close()
	var responseMap map[string]string
	err = json.NewDecoder(resp.Body).Decode(&responseMap)
	if err != nil {
		return "", err
	}
	authTicket = responseMap["auth_ticket"]
	if authTicket == "" {
		return "", common.NewError("500", "Error getting auth ticket from blobber")
	}

	return authTicket, nil
}
