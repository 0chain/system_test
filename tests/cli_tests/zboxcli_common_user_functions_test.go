package cli_tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	KB               = 1024      // kilobyte
	MB               = 1024 * KB // megabyte
	GB               = 1024 * MB // gigabyte
	TOKEN_UNIT int64 = 1e10
)

func TestCommonUserFunctions(t *testing.T) {
	t.Parallel()

	t.Run("Create Allocation - Locked amount must've been withdrawn from user wallet", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 1 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance should decrease by locked amount
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Update Allocation by locking more tokens - Locked amount must be withdrawn from user wallet", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 1 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "30m",
			"lock":       0.2,
		})
		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation due to", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation updated with txId : ([a-f0-9]{64})"), output[0])

		// Wallet balance should decrease by locked amount
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 300.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create Allocation - Blobbers must lock appropriate amount of tokens in stake pool", func(t *testing.T) {
		t.Skip("To be covered after addition of stakePool table to eventsDB")
	})

	t.Run("Update Allocation - Blobbers' lock in stake pool must increase according to updated size", func(t *testing.T) {
		t.Skip("To be covered after addition of stakePool table to eventsDB")
	})
}

func uploadRandomlyGeneratedFile(t *testing.T, allocationID, remotePath string, fileSize int64) string {
	return uploadRandomlyGeneratedFileWithWallet(t, escapedTestName(t), allocationID, remotePath, fileSize)
}

func uploadRandomlyGeneratedFileWithWallet(t *testing.T, walletName, allocationID, remotePath string, fileSize int64) string {
	filename := generateRandomTestFileName(t)
	err := createFileWithSize(filename, fileSize)
	require.Nil(t, err)

	if !strings.HasSuffix(remotePath, "/") {
		remotePath += "/"
	}

	output, err := uploadFileForWallet(t, walletName, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remotePath + filepath.Base(filename),
		"localpath":  filename,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)
	require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])
	return filename
}

func moveAllocationFile(t *testing.T, allocationID, remotepath, destination string) { // nolint
	output, err := moveFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destpath":   "/" + destination,
	}, true)
	require.Nil(t, err, "error in moving the file: ", strings.Join(output, "\n"))
}

func renameAllocationFile(t *testing.T, allocationID, remotepath, newName string) {
	output, err := renameFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destname":   newName,
	}, true)
	require.Nil(t, err, "error in renaming the file: ", strings.Join(output, "\n"))
}

func updateFileWithRandomlyGeneratedData(t *testing.T, allocationID, remotepath string, size int64) string {
	return updateFileWithRandomlyGeneratedDataWithWallet(t, escapedTestName(t), allocationID, remotepath, size)
}

func updateFileWithRandomlyGeneratedDataWithWallet(t *testing.T, walletName, allocationID, remotepath string, size int64) string {
	localfile := generateRandomTestFileName(t)
	err := createFileWithSize(localfile, size)
	require.Nil(t, err)

	output, err := updateFileWithWallet(t, walletName, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remotepath,
		"localpath":  localfile,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	return localfile
}

func renameFile(t *testing.T, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Renaming file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox rename %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateFile(t *testing.T, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return updateFileWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func updateFileWithWallet(t *testing.T, walletName, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Updating file...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox update %s --silent --wallet %s --configDir ./config --config %s",
		p,
		walletName+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getAllocation(t *testing.T, allocationID string) (allocation climodel.Allocation) {
	output, err := getAllocationWithRetry(t, configPath, allocationID, 1)
	require.Nil(t, err, "error fetching allocation")
	require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
	err = json.Unmarshal([]byte(output[0]), &allocation)
	require.Nil(t, err, "error unmarshalling allocation json")
	return
}

func getAllocationWithRetry(t *testing.T, cliConfigFilename, allocationID string, retry int) ([]string, error) {
	t.Logf("Get Allocation...")
	output, err := cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox get --allocation %s --json --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename), retry, time.Second*5)
	return output, err
}

// ConvertToToken converts the value to ZCN tokens
func ConvertToToken(value int64) float64 {
	return float64(value) / float64(TOKEN_UNIT)
}

// ConvertToValue converts ZCN tokens to value
func ConvertToValue(token float64) int64 {
	return int64(token * float64(TOKEN_UNIT))
}
