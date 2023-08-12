package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var (
	createAllocationRegex = regexp.MustCompile(`^Allocation created: (.+)$`)
	updateAllocationRegex = regexp.MustCompile(`^Allocation updated with txId : [a-f0-9]{64}$`)
	repairCompletednRegex = regexp.MustCompile(`Repair file completed, Total files repaired: {2}1`)
)

func TestUpdateAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Update Expiry Should Work")

	t.Parallel()

	t.RunWithTimeout("Update Expiry Should Work", 15*time.Minute, func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		ac := getAllocation(t, allocationID)
		require.Less(t, allocationBeforeUpdate.ExpirationDate, ac.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", allocationBeforeUpdate.ExpirationDate, "After:", ac.ExpirationDate),
		)
	})

	t.Run("Update Size Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(256)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation "+
			"due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, "After:", ac.Size),
		)
	})

	t.Run("Update All Parameters Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(2048)

		params := createParams(map[string]interface{}{
			"allocation":   allocationID,
			"extend":       true,
			"size":         size,
			"update_terms": true,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Less(t, allocationBeforeUpdate.ExpirationDate, ac.ExpirationDate)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size)
	})

	t.Run("Update Negative Size Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(-256)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
		)
	})

	t.Run("Update All Negative Parameters Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(-512)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
		)
	})

	t.Run("Update Size to less than occupied size should fail", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath) // alloc size is 10000

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 2048) // uploading a file of size 2048
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 3)

		size := int64(-9000) // reducing it by 9000 should fail since 2048 is being used
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error updating allocation:allocation_updating_failed: new allocation size is too small: 1000 < 1024")

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
		) // size should be unaffected
	})

	// FIXME extend or size should be required params - should not bother sharders with an empty update
	t.Run("Update Nothing Should Fail", func(t *test.SystemTest) {
		_, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed")

		allocationID := setupAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: update allocation changes nothing", output[0])
	})

	t.Run("Update Non-existent Allocation Should Fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		allocationID := "123abc"

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"extend":     true,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:couldnt_find_allocation: Couldn't find the allocation required for update", output[0])
	})

	t.Run("Update Size To Less Than 1024 Should Fail", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := -allocationBeforeUpdate.Size + 1023

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       fmt.Sprintf("\"%d\"", size),
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output "+
			"length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: new allocation size is too small: 1023 < 1024", output[0])
	})

	t.RunWithTimeout("Update Other's Allocation Should Fail", 5*time.Minute, func(t *test.SystemTest) { // todo: too slow
		_, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed")

		myAllocationID := setupAllocation(t, configPath)

		targetWalletName := escapedTestName(t) + "_TARGET"
		output, err := createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		size := int64(2048)

		// First try updating with myAllocationID: should work
		params := createParams(map[string]interface{}{
			"allocation": myAllocationID,
			"size":       size,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Then try updating with otherAllocationID: should not work
		params = createParams(map[string]interface{}{
			"allocation": myAllocationID,
			"size":       size,
		})
		output, err = updateAllocationWithWallet(t, targetWalletName, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error updating allocation:allocation_updating_failed: only owner can update the allocation", output[0])
	})

	t.Run("Update Mistake Size Parameter Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		size := "ab"

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at "+
			"least 1", strings.Join(output, "\n"))
		expected := fmt.Sprintf(
			`Error: invalid argument "%v" for "--size" flag: strconv.ParseInt: parsing "%v": invalid syntax`,
			size, size,
		)
		require.Equal(t, expected, output[0])
	})

	t.RunWithTimeout("Update Allocation flags for forbid and allow file_options should succeed", 8*time.Minute, func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)
		_, err = executeFaucetWithTokens(t, configPath, 9)
		require.NoError(t, err)

		allocationID := setupAllocation(t, configPath)

		// Forbid upload
		params := createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<0))

		// Forbid delete
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_delete": nil,
		})
		t.Logf("forbidden delete")
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<1))

		// Forbid update
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_update": nil,
		})
		t.Logf("forbidden update")
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<2))

		// Forbid move
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_move": nil,
		})
		t.Logf("forbidden move")
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<3))

		// Forbid copy
		t.Logf("forbidden copy")
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_copy": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<4)) // 63 - 31 = 32 = 00100000

		// Forbid rename
		t.Logf("forbidden rename")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_rename": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions&(1<<5))

		// Allow upload
		t.Logf("allow upload")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": false,
		})

		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(1), alloc.FileOptions) // 0 + 1 = 1 = 00000001

		// Allow delete
		t.Logf("allow delete")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_delete": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(2), alloc.FileOptions&(1<<1))

		// Allow update
		t.Logf("allow update")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_update": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(4), alloc.FileOptions&(1<<2)) // 3 + 4 = 7 = 00000111

		// Allow move
		t.Logf("allow move")
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_move": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(8), alloc.FileOptions&(1<<3))

		// Allow copy
		t.Logf("allow copy")
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_copy": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(16), alloc.FileOptions&(1<<4))

		// Allow rename
		t.Logf("allow rename")
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_rename": false,
		})
		output, err = updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Contains(t, err.Error(), "update allocation changes nothing")
		} else {
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(32), alloc.FileOptions&(1<<5))
	})

	t.Run("Updating same file options twice should fail", func(w *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		// Forbid upload
		params := createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
			"forbid_delete": nil,
			"forbid_move":   nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Forbid upload
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
			"forbid_delete": nil,
			"forbid_move":   nil,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "changes nothing")
	})

	t.Run("Update allocation set_third_party_extendable flag should work", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		// set third party extendable
		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)
	})

	t.Run("Update allocation set_third_party_extendable flag should fail if third_party_extendable is already true", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		// set third party extendable
		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		// set third party extendable
		params = createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "changes nothing")
	})

	t.Run("Update allocation expand by third party if third_party_extendable = false should fail", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       1,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.False(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err = createWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// expand allocation
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, true)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")
	})

	t.Run("Update allocation expand by third party if third_party_extendable = true should succeed", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})

		output, err := updateAllocation(t, configPath, params, true)
		if err != nil {
			require.Equal(t, output[0], "Error updating allocation:allocation_updating_failed: update allocation changes nothing")
		} else {
			require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		}

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err = createWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))
		_, err = executeFaucetWithTokensForWallet(t, nonAllocOwnerWallet, configPath, 3.0)
		require.Nil(t, err)

		// expand allocation
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2,
			"extend":     true,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))

		// get allocation
		allocUpdated := getAllocation(t, allocationID)
		require.Equal(t, alloc.Size+2, allocUpdated.Size)

		require.Nil(t, err)
		require.Less(t, alloc.ExpirationDate, allocUpdated.ExpirationDate)
	})

	t.RunWithTimeout("Update allocation any other action than expand by third party regardless of third_party_extendable should fail", 7*time.Minute, func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err = createWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))
		_, err = executeFaucetWithTokensForWallet(t, nonAllocOwnerWallet, configPath, 3.0)
		require.Nil(t, err)

		// reduce allocation should fail
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       -100,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// set file_options or third_party_extendable should fail
		params = createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"forbid_upload":              nil,
			"forbid_update":              nil,
			"forbid_delete":              nil,
			"forbid_rename":              nil,
			"forbid_move":                nil,
			"forbid_copy":                nil,
			"set_third_party_extendable": nil,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// add blobber should fail
		params = createParams(map[string]interface{}{
			"allocation":     allocationID,
			"add_blobber":    "new_blobber_id",
			"remove_blobber": "blobber_id",
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// set update_term should fail
		params = createParams(map[string]interface{}{
			"allocation":   allocationID,
			"update_terms": false,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// set lock should fail
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"lock":       100,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// get allocation
		updatedAlloc := getAllocation(t, allocationID)

		// Note: the zboxcli `getallocation` calls '/storagesc/allocation' API to get allocation and related blobbers,
		// but we can't rely on the result to assert that nothing changed as the API get fresh blobber data from
		// blobbers table each time the API is called. And because other tests cases could change blobbers,
		// so we can't assert that the blobber info is not changed.
		// Anyway, we should be able to assert that the allocation itself is not changed

		// assert that allocation size is not changed
		require.Equal(t, alloc.Size, updatedAlloc.Size)
		// assert that allocation file options is not changed
		require.Equal(t, alloc.FileOptions, updatedAlloc.FileOptions)
		// assert that no more blobber was added
		require.Equal(t, len(alloc.Blobbers), len(updatedAlloc.Blobbers))
	})

	t.Run("Update allocation with add blobber should succeed", func(t *test.SystemTest) {
		// setup allocation and upload a file
		allocSize := int64(4096)
		fileSize := int64(1024)

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		// faucet tokens
		_, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed")

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		remotePath := "/dir" + filename
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", escapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)
		blobberID, err := GetBlobberNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
			"add_blobber":                blobberID,
		})

		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		assertOutputMatchesAllocationRegex(t, repairCompletednRegex, output[len(output)-1])
		fref, err := VerifyFileRefFromBlobber(walletFile, configFile, allocationID, blobberID, remotePath)
		require.Nil(t, err)
		require.NotNil(t, fref) // not nil when the file exists
	})

	t.Run("Update allocation with replace blobber should succeed", func(t *test.SystemTest) {
		// setup allocation and upload a file
		allocSize := int64(4096)
		fileSize := int64(1024)

		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
		})

		// faucet tokens
		_, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed")

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		remotePath := "/dir" + filename
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", escapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		addBlobber, err := GetBlobberNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)
		removeBlobber, err := GetRandomBlobber(walletFile, configFile, allocationID, addBlobber)
		require.Nil(t, err)
		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
			"add_blobber":                addBlobber,
			"remove_blobber":             removeBlobber,
		})

		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		assertOutputMatchesAllocationRegex(t, repairCompletednRegex, output[len(output)-1])
		fref, err := VerifyFileRefFromBlobber(walletFile, configFile, allocationID, addBlobber, remotePath)
		require.Nil(t, err)
		require.NotNil(t, fref) // not nil when the file exists
	})
}

func setupAndParseAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) (string, climodel.Allocation) {
	allocationID := setupAllocation(t, cliConfigFilename, extraParams...)

	for i := 0; i < 2; i++ {
		_, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed")
	}

	allocations := parseListAllocations(t, cliConfigFilename)
	allocation, ok := allocations[allocationID]
	require.True(t, ok, "current allocation not found", allocationID, allocations)

	return allocationID, allocation
}

func parseListAllocations(t *test.SystemTest, cliConfigFilename string) map[string]climodel.Allocation {
	output, err := listAllocations(t, cliConfigFilename)
	require.Nil(t, err, "list allocations failed", err, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var allocations []*climodel.Allocation
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&allocations)
	require.Nil(t, err, "error deserializing JSON", err)

	allocationMap := make(map[string]climodel.Allocation)

	for _, ac := range allocations {
		allocationMap[ac.ID] = *ac
	}

	return allocationMap
}

func setupAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	return setupAllocationWithWallet(t, escapedTestName(t), cliConfigFilename, extraParams...)
}

func setupAllocationWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	faucetTokens := 2.0
	// Then create new allocation
	options := map[string]interface{}{"size": "1000000000", "lock": "5"}

	// Add additional parameters if available
	// Overwrite with new parameters when available
	for _, params := range extraParams {
		// Extract parameters unrelated to upload
		if tokenStr, ok := params["tokens"]; ok {
			token, err := strconv.ParseFloat(fmt.Sprintf("%v", tokenStr), 64)
			require.Nil(t, err)
			faucetTokens = token
			delete(params, "tokens")
		}
		for k, v := range params {
			options[k] = v
		}
	}
	// First create a wallet and run faucet command
	output, err := createWalletForName(t, cliConfigFilename, walletName)
	require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

	output, err = executeFaucetWithTokensForWallet(t, walletName, cliConfigFilename, faucetTokens)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	output, err = createNewAllocationForWallet(t, walletName, cliConfigFilename, createParams(options))
	require.NoError(t, err, "create new allocation failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	// Get the allocation ID and return it
	allocationID, err := getAllocationID(output[0])
	require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

	return allocationID
}

func assertOutputMatchesAllocationRegex(t *test.SystemTest, re *regexp.Regexp, str string) {
	match := re.FindStringSubmatch(str)
	require.True(t, len(match) > 0, "expected allocation to match regex", re, str)
}

func getAllocationID(str string) (string, error) {
	match := createAllocationRegex.FindStringSubmatch(str)
	if len(match) < 2 {
		return "", errors.New("allocation match not found")
	}
	return match[1], nil
}

func getAllocationCost(str string) (float64, error) {
	allocationCostInOutput, err := strconv.ParseFloat(strings.Fields(str)[5], 64)
	if err != nil {
		return 0.0, err
	}

	unit := strings.Fields(str)[6]
	allocationCostInZCN := unitToZCN(allocationCostInOutput, unit)

	return allocationCostInZCN, nil
}

func createParams(params map[string]interface{}) string {
	var builder strings.Builder

	for k, v := range params {
		if v == nil {
			_, _ = builder.WriteString(fmt.Sprintf("--%s ", k))
		} else if reflect.TypeOf(v).String() == "bool" {
			_, _ = builder.WriteString(fmt.Sprintf("--%s=%v ", k, v))
		} else {
			_, _ = builder.WriteString(fmt.Sprintf("--%s %v ", k, v))
		}
	}
	return strings.TrimSpace(builder.String())
}

func createKeyValueParams(params map[string]string) string {
	keys := "--keys \""
	values := "--values \""
	first := true
	for k, v := range params {
		if first {
			first = false
		} else {
			keys += ","
			values += ","
		}
		keys += " " + k
		values += " " + v
	}
	keys += "\""
	values += "\""
	return keys + " " + values
}

func updateAllocation(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return updateAllocationWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func updateAllocationWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Updating allocation...")
	cmd := fmt.Sprintf(
		"./zbox updateallocation %s --silent --wallet %s "+
			"--configDir ./config --config %s --lock 0.2",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func listAllocations(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Listing allocations...")
	cmd := fmt.Sprintf(
		"./zbox listallocations --json --silent "+
			"--wallet %s --configDir ./config --config %s",
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

// executeFaucetWithTokens executes faucet command with given tokens.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeFaucetWithTokens(t *test.SystemTest, cliConfigFilename string, tokens float64) ([]string, error) {
	return executeFaucetWithTokensForWallet(t, escapedTestName(t), cliConfigFilename, tokens)
}

// executeFaucetWithTokensForWallet executes faucet command with given tokens and wallet.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeFaucetWithTokensForWallet(t *test.SystemTest, wallet, cliConfigFilename string, tokens float64) ([]string, error) {
	t.Logf("Executing faucet...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet faucet --methodName "+
		"pour --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
		tokens,
		wallet,
		cliConfigFilename,
	), 3, time.Second*5)
}
