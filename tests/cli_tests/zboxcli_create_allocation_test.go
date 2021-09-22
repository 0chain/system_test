package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation without providing any additional parameters Should Work", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID)
		require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with smallest expiry (5m) Should Work", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "5m", "size": "256000", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID)
		require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with smallest possible size (1024) Should Work", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "1h", "size": "1024", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID)
		require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with parity 1 Should Work", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "1h", "size": "1024", "parity": "1", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID)
		require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with data shard 20 Should Work", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "1h", "size": "128000", "data": "20", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID)
		require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with read price range 0-0.03 Should Work", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "1h", "size": "128000", "read_price": "0-0.03", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID)
		require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with write price range 0-0.03 Should Work", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "1h", "size": "128000", "write_price": "0-0.03", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))

		output, err = cancelAllocation(t, configPath, allocationID)
		require.Nil(t, err, "error cancelling allocation", strings.Join(output, "\n"))
	})

	t.Run("Create allocation with too large parity (Greater than the number of blobbers) Should Fail", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"parity": "99", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: [txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with read price range 0-0 Should Fail", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"read_price": "0-0", "lock": "0.5", "size": 1024, "expire": "1h"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: [txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with size smaller than limit (size < 1024) Should Fail", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"size": 256, "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: [txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with expire smaller than limit (expire < 5m) Should Fail", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "3m", "lock": "0.5", "size": 1024}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: [txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create allocation with no parameter (missing lock) Should Fail", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "missing required 'lock' argument", output[len(output)-1])
	})

	t.Run("Create allocation with invalid expiry Should Fail", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "-1", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "invalid argument \"-1\" for \"--expire\" flag: time: missing unit in duration -1", output[len(output)-1])
	})

	t.Run("Create allocation by providing expiry in wrong format (expire 1hour) Should Fail", func(t *testing.T) {
		t.Parallel()

		_, err := setupWallet(t, configPath)
		require.Nil(t, err)

		options := map[string]interface{}{"expire": "1hour", "lock": "0.5"}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "invalid argument \"1hour\" for \"--expire\" flag: time: unknown unit hour in duration 1hour", output[len(output)-1])
	})

}

func setupWallet(t *testing.T, configPath string) ([]string, error) {
	output, err := registerWallet(t, configPath)
	if err != nil {
		cli_utils.Logger.Errorf(err.Error())
		return nil, err
	}

	_, err = executeFaucetWithTokens(t, configPath, 1)
	if err != nil {
		cli_utils.Logger.Errorf(err.Error())
		return nil, err
	}
	_, err = getBalance(t, configPath)
	if err != nil {
		cli_utils.Logger.Errorf(err.Error())
		return nil, err
	}

	return output, nil
}

func createNewAllocation(t *testing.T, cliConfigFilename string, params string) ([]string, error) {
	return cli_utils.RunCommand(fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
		escapedTestName(t)+"_allocation.txt"))
}
