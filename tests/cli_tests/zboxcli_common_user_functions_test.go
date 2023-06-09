package cli_tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

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

func TestCommonUserFunctions(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create Allocation - Locked amount must've been withdrawn from user wallet")

	t.Parallel()

	t.Run("Create Allocation - Locked amount must've been withdrawn from user wallet", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		_, err = executeFaucetWithTokensForWallet(t, escapedTestName(t), configPath, 9)
		require.Nil(t, err)

		// Lock tokens for allocation
		allocParams := createParams(map[string]interface{}{
			"lock":   "5",
			"size":   1 * MB,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// Wallet balance should decrease by locked amount
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 8.99, balance) // lock - fee

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Update Allocation by locking more tokens - Locked amount must be withdrawn from user wallet", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		_, err = executeFaucetWithTokensForWallet(t, escapedTestName(t), configPath, 9)
		require.Nil(t, err)

		// get wallet balance
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Lock 5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock":   "5",
			"size":   1 * MB,
			"expire": "5m",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		// get balance after creating allocation
		balanceAfterAllocation, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Wallet balance should decrease by locked amount and txn fee
		require.Less(t, balanceAfterAllocation, balance-0.5)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "30m",
			"lock":       1,
		})
		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "Error updating allocation due to", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation updated with txId : ([a-f0-9]{64})"), output[0])

		balanceAfterUpdate, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Wallet balance should decrease by locked amount and txn fee
		require.Less(t, balanceAfterUpdate, balanceAfterAllocation-0.2)

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create Allocation - Blobbers must lock appropriate amount of tokens in stake pool", func(t *test.SystemTest) {
		t.Skip("To be covered after addition of stakePool table to eventsDB")
	})

	t.Run("Update Allocation - Blobbers' lock in stake pool must increase according to updated size", func(t *test.SystemTest) {
		t.Skip("To be covered after addition of stakePool table to eventsDB")
	})
}

func uploadRandomlyGeneratedFile(t *test.SystemTest, allocationID, remotePath string, fileSize int64) string {
	return uploadRandomlyGeneratedFileWithWallet(t, escapedTestName(t), allocationID, remotePath, fileSize)
}

func uploadRandomlyGeneratedFileWithWallet(t *test.SystemTest, walletName, allocationID, remotePath string, fileSize int64) string {
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

func moveAllocationFile(t *test.SystemTest, allocationID, remotepath, destination string) { // nolint
	output, err := moveFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destpath":   "/" + destination,
	}, true)
	require.Nil(t, err, "error in moving the file: ", strings.Join(output, "\n"))
}

func renameAllocationFile(t *test.SystemTest, allocationID, remotepath, newName string) {
	output, err := renameFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destname":   newName,
	}, true)
	require.Nil(t, err, "error in renaming the file: ", strings.Join(output, "\n"))
}

func updateFileWithRandomlyGeneratedData(t *test.SystemTest, allocationID, remotepath string, size int64) string {
	return updateFileWithRandomlyGeneratedDataWithWallet(t, escapedTestName(t), allocationID, remotepath, size)
}

func updateFileWithRandomlyGeneratedDataWithWallet(t *test.SystemTest, walletName, allocationID, remotepath string, size int64) string {
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

func renameFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
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

func updateFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return updateFileWithWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func updateFileWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
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

func getAllocation(t *test.SystemTest, allocationID string) (allocation climodel.Allocation) {
	output, err := getAllocationWithRetry(t, configPath, allocationID, 1)
	require.Nil(t, err, "error fetching allocation")
	require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")

	fmt.Println(output)

	err = json.Unmarshal([]byte(output[0]), &allocation)
	require.Nil(t, err, "error unmarshalling allocation json")
	return
}

func getAllocationWithRetry(t *test.SystemTest, cliConfigFilename, allocationID string, retry int) ([]string, error) {
	t.Logf("Get Allocation...")
	output, err := cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox getallocation --allocation %s --json --silent --wallet %s --configDir ./config --config %s",
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
