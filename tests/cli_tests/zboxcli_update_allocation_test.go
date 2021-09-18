package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
)

var reAllocation = regexp.MustCompile(`^Allocation created: (.+)$`)
var reUpdateAllocation = regexp.MustCompile(`^Allocation updated with txId : [a-f0-9]{64}$`)
var reCancelAllocation = regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`)

func TestUpdateAllocation(t *testing.T) {

	t.Run("Update Nothing", func(t *testing.T) {
		t.Parallel()

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
		})

		output, err := updateAllocation(t, configPath, params)
		// Error should not be nil
		assert.NotNil(t, err)

		assert.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
	})

	t.Run("Update Non-existent Allocation", func(t *testing.T) {
		t.Parallel()

		allocationID := "123abc"

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "1h",
		})

		output, err := updateAllocation(t, configPath, params)
		// Error should not be nil
		assert.NotNil(t, err)

		assert.Equal(t, "Error updating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[len(output)-3])
	})

	t.Run("Update Check Expiry", func(t *testing.T) {
		t.Parallel()

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		allocations, err := parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		allocationBeforeUpdate, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		expDuration := int64(1) // In hours

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     fmt.Sprintf("%dh", expDuration),
		})

		output, err := updateAllocation(t, configPath, params)
		if err != nil {
			t.Errorf("Could not update allocation due to error: %v", err)
		}
		if len(output) != 1 {
			t.Error("Unexpected outputs:", output)
		}
		if err := checkAllocationRegex(reUpdateAllocation, output[0]); err != nil {
			t.Error("Error on checking allocation:", err)
		}

		allocations, err = parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		ac, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		assert.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: Before:", allocationBeforeUpdate.ExpirationDate, "After:", ac.ExpirationDate),
		)
	})

	t.Run("Update Check Size", func(t *testing.T) {
		t.Parallel()

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		allocations, err := parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		allocationBeforeUpdate, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		size := int64(2048)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})

		output, err := updateAllocation(t, configPath, params)
		if err != nil {
			t.Errorf("Could not update allocation due to error: %v", err)
		}
		if len(output) != 1 {
			t.Error("Unexpected outputs:", output)
		}
		if err := checkAllocationRegex(reUpdateAllocation, output[0]); err != nil {
			t.Error("Error on checking allocation:", err)
		}

		allocations, err = parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		ac, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		assert.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, "After:", ac.Size),
		)
	})

	t.Run("Update Check All Parameters", func(t *testing.T) {
		t.Parallel()

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		allocations, err := parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		allocationBeforeUpdate, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		expDuration := int64(1) // In hours
		size := int64(512)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     fmt.Sprintf("%dh", expDuration),
			"size":       size,
		})

		output, err := updateAllocation(t, configPath, params)
		if err != nil {
			t.Errorf("Could not update allocation due to error: %v", err)
		}
		if len(output) != 1 {
			t.Error("Unexpected outputs:", output)
		}
		if err := checkAllocationRegex(reUpdateAllocation, output[0]); err != nil {
			t.Error("Error on checking allocation:", err)
		}

		allocations, err = parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		ac, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		assert.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: Before:", allocationBeforeUpdate.ExpirationDate, "After:", ac.ExpirationDate),
		)
		assert.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, "After:", ac.Size),
		)
	})

	t.Run("Cancel Allocation", func(t *testing.T) {
		t.Parallel()

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		output, err := cancelAllocation(t, configPath, allocationID)
		if err != nil {
			t.Errorf("Could not cancel allocation due to error: %v", err)
		}
		if len(output) != 1 {
			t.Error("Unexpected outputs:", output)
		}

		if err := checkAllocationRegex(reCancelAllocation, output[0]); err != nil {
			t.Error("Error on checking allocation:", err)
		}
	})

	//FIXME: POSSIBLE BUG: Error obtained on finalizing allocation
	t.Run("Finalize Allocation", func(t *testing.T) {
		t.Parallel()

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		output, err := finalizeAllocation(t, configPath, allocationID)
		// Error should not be nil since finalize is not working
		assert.NotNil(t, err)

		assert.Equal(t, "Error finalizing allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
	})
}

func parseListAllocations(t *testing.T, cliConfigFilename string) (map[string]cli_model.Allocation, error) {

	output, err := listAllocations(t, cliConfigFilename)
	if err != nil {
		return nil, err
	}
	if len(output) != 1 {
		return nil, fmt.Errorf("unexpected output: %v", output)
	}

	var allocations []cli_model.Allocation
	if err := json.NewDecoder(strings.NewReader(output[0])).Decode(&allocations); err != nil {
		return nil, err
	}

	allocationMap := make(map[string]cli_model.Allocation)

	for _, ac := range allocations {
		allocationMap[ac.ID] = ac
	}

	return allocationMap, nil
}

func setupAllocation(t *testing.T, cliConfigFilename string) (string, error) {
	// First create a wallet and run faucet command
	_, err := registerWallet(t, cliConfigFilename)
	if err != nil {
		return "", fmt.Errorf("registering wallet failed: %v", err)
	}

	_, err = executeFaucetWithTokens(t, cliConfigFilename, 9)
	if err != nil {
		return "", fmt.Errorf("faucet execution failed: %v", err)
	}

	// Then create new allocation
	allocParam := createParams(map[string]interface{}{
		"lock":   0.5,
		"size":   2048,
		"expire": "1h",
	})
	output, err := createNewAllocation(t, cliConfigFilename, allocParam)
	if err != nil {
		return "", fmt.Errorf("new allocation failed: %v", err)
	}

	// Get the allocation ID and return it
	allocationID, err := getAllocationID(output[0])
	if err != nil {
		return "", fmt.Errorf("could not get allocation ID: %v", err)
	}

	return allocationID, nil
}

func checkAllocationRegex(re *regexp.Regexp, str string) error {
	match := re.FindStringSubmatch(str)
	if len(match) < 1 {
		return fmt.Errorf("unexpected format: %v", str)
	}
	return nil
}

func getAllocationID(str string) (string, error) {
	match := reAllocation.FindStringSubmatch(str)
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
