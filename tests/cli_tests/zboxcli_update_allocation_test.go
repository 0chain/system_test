package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

var (
	reCreateAllocation = regexp.MustCompile(`^Allocation created: (.+)$`)
	reUpdateAllocation = regexp.MustCompile(`^Allocation updated with txId : [a-f0-9]{64}$`)
	reCancelAllocation = regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`)
)

func TestUpdateAllocation(t *testing.T) {

	t.Run("Parallel", func(t *testing.T) {
		t.Parallel()

		t.Run("Success Scenarios", func(t *testing.T) {
			t.Parallel()

			t.Run("Update Expiry Should Work", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				expDuration := int64(1) // In hours

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("%dh", expDuration),
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate,
					fmt.Sprint("Expiration Time doesn't match: Before:", allocationBeforeUpdate.ExpirationDate, "After:", ac.ExpirationDate),
				)
			})

			t.Run("Update Size Should Work", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				size := int64(256)

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"size":       size,
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
					fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, "After:", ac.Size),
				)
			})

			t.Run("Update All Parameters Should Work", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				expDuration := int64(1) // In hours
				size := int64(512)

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("%dh", expDuration),
					"size":       size,
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate)
				require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size)
			})

			t.Run("Update Negative Expiry Should Work", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				expDuration := int64(-30) // In minutes

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("\"%dm\"", expDuration),
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*60, ac.ExpirationDate,
					fmt.Sprint("Expiration Time doesn't match: Before:", allocationBeforeUpdate.ExpirationDate, " After:", ac.ExpirationDate),
				)
			})

			t.Run("Update Negative Size Should Work", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				size := int64(-256)

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"size":       size,
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
					fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
				)
			})

			t.Run("Update All Negative Parameters Should Work", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				expDuration := int64(-30) // In minutes
				size := int64(-512)

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("%dm", expDuration),
					"size":       size,
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*60, ac.ExpirationDate,
					fmt.Sprint("Expiration Time doesn't match: Before:", allocationBeforeUpdate.ExpirationDate, " After:", ac.ExpirationDate),
				)
				require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
					fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
				)
			})

			t.Run("Cancel Allocation Should Work", func(t *testing.T) {
				t.Parallel()

				allocationID := setupAllocation(t, configPath)

				output, err := cancelAllocation(t, configPath, allocationID)

				require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reCancelAllocation, output[0])
			})

		})

		t.Run("Failure Scenarios", func(t *testing.T) {
			t.Parallel()

			//FIXME expiry or size should be required params - should not bother sharders with an empty update
			t.Run("Update Nothing Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID := setupAllocation(t, configPath)

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
				})
				output, err := updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			t.Run("Update Non-existent Allocation Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID := "123abc"

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     "1h",
				})
				output, err := updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 3, "expected output length be at least 4", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[len(output)-3])
			})

			t.Run("Update Expired Allocation Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				expDuration := int64(-1) // In hours

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("\"%dh\"", expDuration),
				})

				output, err := updateAllocation(t, configPath, params)
				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.LessOrEqual(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate)

				// Update the expired allocation's Expiration time

				expDuration = int64(1) // In hours

				params = createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("%dh", expDuration),
				})
				output, err = updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])

				// Update the expired allocation's size
				size := int64(2048)

				params = createParams(map[string]interface{}{
					"allocation": allocationID,
					"size":       size,
				})

				output, err = updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])

			})

			t.Run("Update Size To Less Than 1024 Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				size := -allocationBeforeUpdate.Size + 1023

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"size":       fmt.Sprintf("\"%d\"", size),
				})
				output, err := updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			t.Run("Cancel Expired Allocation Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				expDuration := int64(-1) // In hours

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("%dh", expDuration),
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.LessOrEqual(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate)

				// Cancel the expired allocation
				output, err = cancelAllocation(t, configPath, allocationID)
				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))

				//FIXME: POSSIBLE BUG: Error message shows error in creating instead of error in canceling
				require.Equal(t, "Error creating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			//FIXME: POSSIBLE BUG: Error obtained on finalizing allocation
			t.Run("Finalize Allocation Should Have Worked", func(t *testing.T) {
				t.Parallel()

				allocationID := setupAllocation(t, configPath)

				output, err := finalizeAllocation(t, configPath, allocationID)
				// Error should not be nil since finalize is not working
				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error finalizing allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			//FIXME: POSSIBLE BUG: Error obtained on finalizing allocation (both expired and non-expired)
			t.Run("Finalize Expired Allocation Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
				expDuration := int64(-1) // In hours

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("%dh", expDuration),
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				allocations := parseListAllocations(t, configPath)
				ac, ok := allocations[allocationID]
				require.True(t, ok, "current allocation not found", allocationID, allocations)
				require.LessOrEqual(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate)

				// Update the expired allocation's Expiration time
				expDuration = int64(1) // In hours

				params = createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     fmt.Sprintf("%dh", expDuration),
				})
				output, err = updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])

				// Update the expired allocation's size
				size := int64(2048)

				params = createParams(map[string]interface{}{
					"allocation": allocationID,
					"size":       size,
				})
				output, err = updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			t.Run("Update Other's Allocation Should Fail", func(t *testing.T) {
				t.Parallel()

				var otherAllocationID string

				myAllocationID := setupAllocation(t, configPath)

				// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
				t.Run("Get Other Allocation ID", func(t *testing.T) {
					otherAllocationID = setupAllocation(t, configPath)

					// Updating the otherAllocationID should work here
					size := int64(2048)

					params := createParams(map[string]interface{}{
						"allocation": otherAllocationID,
						"size":       size,
					})
					output, err := updateAllocation(t, configPath, params)

					require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
					require.Equal(t, 1, len(output), "output length did not match expected", strings.Join(output, "\n"))
					assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])
				})

				// otherAllocationID should not be updatable from this level
				size := int64(2048)

				// First try updating with myAllocationID: should work
				params := createParams(map[string]interface{}{
					"allocation": myAllocationID,
					"size":       size,
				})
				output, err := updateAllocation(t, configPath, params)

				require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))
				assertOutputMatchesAllocationRegex(t, reUpdateAllocation, output[0])

				// Then try updating with otherAllocationID: should not work
				params = createParams(map[string]interface{}{
					"allocation": otherAllocationID,
					"size":       size,
				})
				output, err = updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			t.Run("Cancel Other's Allocation Should Fail", func(t *testing.T) {
				t.Parallel()

				var otherAllocationID string
				// This test creates a separate wallet and allocates there, test nesting needed to create other wallet json
				t.Run("Get Other Allocation ID", func(t *testing.T) {
					otherAllocationID = setupAllocation(t, configPath)
				})
				myAllocationID := setupAllocation(t, configPath)

				// otherAllocationID should not be cancelable from this level

				// First try canceling with myAllocationID: should work

				output, err := cancelAllocation(t, configPath, myAllocationID)
				require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
				require.Equal(t, 1, len(output), strings.Join(output, "\n"))

				assertOutputMatchesAllocationRegex(t, reCancelAllocation, output[0])

				// Then try canceling with otherAllocationID: should not work
				output, err = cancelAllocation(t, configPath, otherAllocationID)

				require.NotNil(t, err, "expected error cancelling allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				//FIXME: POSSIBLE BUG: Error message shows error in creating instead of error in canceling
				require.Equal(t, "Error creating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			//FIXME: POSSIBLE BUG: Error obtained on finalizing allocation (both owned and others)
			t.Run("Finalize Other's Allocation Should Fail", func(t *testing.T) {
				t.Parallel()

				var otherAllocationID string
				// This test creates a separate wallet and allocates there, test nesting needed to create other wallet json
				t.Run("Get Other Allocation ID", func(t *testing.T) {
					otherAllocationID = setupAllocation(t, configPath)
				})
				myAllocationID := setupAllocation(t, configPath)

				// First try updating with myAllocationID: should work but it's buggy now
				output, err := finalizeAllocation(t, configPath, myAllocationID)
				require.NotNil(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error finalizing allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])

				// Then try updating with otherAllocationID: should not work
				output, err = finalizeAllocation(t, configPath, otherAllocationID)

				// Error should not be nil since finalize is not working
				require.NotNil(t, err, "expected error finalizing allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				require.Equal(t, "Error finalizing allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
			})

			t.Run("Update Mistake Expiry Parameter Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID := setupAllocation(t, configPath)
				expiry := 1

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"expiry":     expiry,
				})
				output, err := updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				expected := fmt.Sprintf(
					`Error: invalid argument "%v" for "--expiry" flag: time: missing unit in duration "%v"`,
					expiry, expiry,
				)
				require.Equal(t, expected, output[0])
			})

			t.Run("Update Mistake Size Parameter Should Fail", func(t *testing.T) {
				t.Parallel()

				allocationID := setupAllocation(t, configPath)
				size := "ab"

				params := createParams(map[string]interface{}{
					"allocation": allocationID,
					"size":       size,
				})
				output, err := updateAllocation(t, configPath, params)

				require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
				require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
				expected := fmt.Sprintf(
					`Error: invalid argument "%v" for "--size" flag: strconv.ParseInt: parsing "%v": invalid syntax`,
					size, size,
				)
				require.Equal(t, expected, output[0])
			})
		})
	})
}

func setupAndParseAllocation(t *testing.T, cliConfigFilename string) (string, cli_model.Allocation) {
	allocationID := setupAllocation(t, cliConfigFilename)

	allocations := parseListAllocations(t, cliConfigFilename)
	allocation, ok := allocations[allocationID]
	require.True(t, ok, "current allocation not found", allocationID, allocations)

	return allocationID, allocation
}

func parseListAllocations(t *testing.T, cliConfigFilename string) map[string]cli_model.Allocation {

	output, err := listAllocations(t, cliConfigFilename)
	require.Nil(t, err, "list allocations failed", err, strings.Join(output, "\n"))
	require.Equal(t, 1, len(output), "unexpected output", strings.Join(output, "\n"))

	var allocations []cli_model.Allocation
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&allocations)
	require.Nil(t, err, "error deserializing JSON", err)

	allocationMap := make(map[string]cli_model.Allocation)

	for _, ac := range allocations {
		allocationMap[ac.ID] = ac
	}

	return allocationMap
}

func setupAllocation(t *testing.T, cliConfigFilename string) string {
	// First create a wallet and run faucet command
	output, err := registerWallet(t, cliConfigFilename)
	require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

	output, err = executeFaucetWithTokens(t, cliConfigFilename, 1)
	require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

	// Then create new allocation
	allocParam := createParams(map[string]interface{}{
		"lock":   0.5,
		"size":   2048,
		"expire": "1h",
	})
	output, err = createNewAllocation(t, cliConfigFilename, allocParam)
	require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

	// Get the allocation ID and return it
	allocationID, err := getAllocationID(output[0])
	require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

	return allocationID
}

func assertOutputMatchesAllocationRegex(t *testing.T, re *regexp.Regexp, str string) {
	match := re.FindStringSubmatch(str)
	require.True(t, len(match) > 0, "expected allocation to match regex", re, str)
}

func getAllocationID(str string) (string, error) {
	match := reCreateAllocation.FindStringSubmatch(str)
	if len(match) < 2 {
		return "", errors.New("allocation match not found")
	}
	return match[1], nil
}

func createParams(params map[string]interface{}) string {
	var builder strings.Builder

	for k, v := range params {
		builder.WriteString(fmt.Sprintf("--%s %v ", k, v))
	}
	return strings.TrimSpace(builder.String())
}

func updateAllocation(t *testing.T, cliConfigFilename string, params string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox updateallocation %s --silent --wallet %s --configDir ./config --config %s",
		params,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cli_utils.RunCommand(cmd)
}

func createNewAllocation(t *testing.T, cliConfigFilename string, params string) ([]string, error) {
	return cli_utils.RunCommand(fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
		escapedTestName(t)+"_allocation.txt"))
}

func listAllocations(t *testing.T, cliConfigFilename string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox listallocations --json --silent --wallet %s --configDir ./config --config %s",
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cli_utils.RunCommand(cmd)
}

func cancelAllocation(t *testing.T, cliConfigFilename string, allocationID string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox alloc-cancel --allocation %s --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cli_utils.RunCommand(cmd)
}

// executeFaucetWithTokens executes faucet command with given tokens.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeFaucetWithTokens(t *testing.T, cliConfigFilename string, tokens float64) ([]string, error) {
	return cli_utils.RunCommand(
		fmt.Sprintf("./zwallet faucet --methodName pour --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
			tokens,
			escapedTestName(t),
			cliConfigFilename,
		))
}

func finalizeAllocation(t *testing.T, cliConfigFilename string, allocationID string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox alloc-fini --allocation %s --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cli_utils.RunCommand(cmd)
}
