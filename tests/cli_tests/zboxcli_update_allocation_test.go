package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0chain/system_test/internal/cli/util"
	"regexp"
	"strings"
	"testing"
)

var reAllocation = regexp.MustCompile(`^Allocation created: (.+)$`)
var reUpdateAllocation = regexp.MustCompile(`^Allocation updated with txId : [a-z0-9]+$`)
var reCancelAllocation = regexp.MustCompile(`^Allocation canceled with txId : [a-z0-9]+$`)

func TestAllocation(t *testing.T) {

	t.Run("Update Nothing", func(t *testing.T) {

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		params := createParams(param{
			key:   "allocation",
			value: allocationID,
		})

		_, err = updateAllocation(t, configPath, params)
		// Error should not be nil
		if err == nil {
			t.Errorf("Should have obtained errors")
		}
	})

	t.Run("Update Non-existent Allocation", func(t *testing.T) {

		allocationID := "123abc"

		params := createParams(param{
			key:   "allocation",
			value: allocationID,
		}, param{
			key:   "expiry",
			value: "1h",
		})

		_, err := updateAllocation(t, configPath, params)
		// Error should not be nil
		if err == nil {
			t.Errorf("Should have obtained errors")
		}
	})

	t.Run("Update Check Expiry", func(t *testing.T) {

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		allocations, err := parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		acBefore, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		expDuration := int64(1) // In hours

		params := createParams(param{
			key:   "allocation",
			value: allocationID,
		}, param{
			key:   "expiry",
			value: fmt.Sprintf("%dh", expDuration),
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

		if acBefore.ExpirationDate+expDuration*3600 != ac.ExpirationDate {
			t.Error("Expiration Time doesn't match: Before:", acBefore.ExpirationDate, "After:", ac.ExpirationDate)
		}
	})

	t.Run("Update Check Size", func(t *testing.T) {

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		allocations, err := parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		acBefore, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		size := int64(2048)

		params := createParams(param{
			key:   "allocation",
			value: allocationID,
		}, param{
			key:   "size",
			value: size,
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

		if acBefore.Size+size != ac.Size {
			t.Error("Size doesn't match: Before:", acBefore.Size, "After:", ac.Size)
		}
	})

	t.Run("Update Check All Parameters", func(t *testing.T) {

		allocationID, err := setupAllocation(t, configPath)
		if err != nil {
			t.Errorf("Error in allocation setup: %v", err)
		}

		allocations, err := parseListAllocations(t, configPath)
		if err != nil {
			t.Errorf("Error in listing allocations: %v", err)
		}
		acBefore, ok := allocations[allocationID]
		if !ok {
			t.Error("Current allocation not found")
		}

		expDuration := int64(1) // In hours
		size := int64(512)

		params := createParams(param{
			key:   "allocation",
			value: allocationID,
		}, param{
			key:   "expiry",
			value: fmt.Sprintf("%dh", expDuration),
		}, param{
			key:   "size",
			value: size,
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

		if acBefore.ExpirationDate+expDuration*3600 != ac.ExpirationDate {
			t.Error("Expiration Time doesn't match: Before:", acBefore.ExpirationDate, "After:", ac.ExpirationDate)
		}

		if acBefore.Size+size != ac.Size {
			t.Error("Size doesn't match: Before:", acBefore.Size, "After:", ac.Size)
		}
	})

	t.Run("Cancel Allocation", func(t *testing.T) {

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
}

type allocation struct {
	ID             string `json:"id"`
	ExpirationDate int64  `json:"expiration_date"`
	DataShards     int    `json:"data_shards"`
	ParityShards   int    `json:"parity_shards"`
	Size           int64  `json:"size"`
}

func parseListAllocations(t *testing.T, cliConfigFilename string) (map[string]allocation, error) {
	output, err := listAllocations(t, cliConfigFilename)
	if err != nil {
		return nil, err
	}
	if len(output) != 1 {
		return nil, fmt.Errorf("unexpected output: %v", output)
	}

	var allocations []allocation
	if err := json.NewDecoder(strings.NewReader(output[0])).Decode(&allocations); err != nil {
		return nil, err
	}

	allocationMap := make(map[string]allocation)

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
	allocParam := createParams(param{
		key:   "lock",
		value: 0.5,
	}, param{
		key:   "size",
		value: 2048,
	}, param{
		key:   "expire",
		value: "1h",
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

type param struct {
	key   string
	value interface{}
}

func createParams(params ...param) string {
	var builder strings.Builder

	for i, p := range params {
		builder.WriteString(fmt.Sprintf("--%s %v", p.key, p.value))
		if i != len(params)-1 {
			builder.WriteString(" ")
		}
	}
	return builder.String()
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

func executeFaucetWithTokens(t *testing.T, cliConfigFilename string, tokens float64) ([]string, error) {
	return cli_utils.RunCommand(
		fmt.Sprintf("./zwallet faucet --methodName pour --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
			tokens,
			escapedTestName(t),
			cliConfigFilename,
		))
}
