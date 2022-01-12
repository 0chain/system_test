package cli_tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
		require.Regexp(t, regexp.MustCompile(`Balance: 300.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create Allocation - Blobbers must lock appropriate amount of tokens in stake pool", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
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

		allocation := getAllocation(t, allocationID)

		// Each blobber should lock (size of allocation on that blobber * write_price of blobber) in stake pool
		cliutils.Wait(t, 2*time.Minute)
		for _, blobber_detail := range allocation.BlobberDetails {
			output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id": blobber_detail.BlobberID,
				"json":       "",
			}))
			assert.Nil(t, err, "Error fetching stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			stakePool := climodel.StakePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &stakePool)
			assert.Nil(t, err, "Error unmarshalling stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			t.Logf(
				"Expected blobber id [%v] to lock [%v] but it actually locked [%v], size [%v], write price [%v]",
				blobber_detail.BlobberID,
				int64(blobber_detail.Size*int64(blobber_detail.Terms.Write_price)),
				int64(stakePool.OffersTotal),
				int64(blobber_detail.Size),
				int64(blobber_detail.Terms.Write_price),
			)
			assert.Equal(
				t,
				int64(sizeInGB(blobber_detail.Size)*float64(blobber_detail.Terms.Write_price)),
				int64(stakePool.OffersTotal),
			)
		}
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("Update Allocation - Blobbers' lock in stake pool must increase according to updated size", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
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

		// Updated allocation params
		allocParams = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2 * MB,
		})
		output, err = updateAllocation(t, configPath, allocParams, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation updated with txId : ([a-f0-9]{64})"), output[0])

		allocation := getAllocation(t, allocationID)

		// Each blobber should lock (updated size of allocation on that blobber * write_price of blobber) in stake pool
		cliutils.Wait(t, 2*time.Minute)
		for _, blobber_detail := range allocation.BlobberDetails {
			output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id": blobber_detail.BlobberID,
				"json":       "",
			}))
			assert.Nil(t, err, "Error fetching stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			stakePool := climodel.StakePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &stakePool)
			assert.Nil(t, err, "Error unmarshalling stake pool info for blobber id: ", blobber_detail.BlobberID, "\n", strings.Join(output, "\n"))

			t.Logf(
				"Expected blobber id [%v] to lock [%v] but it actually locked [%v]",
				blobber_detail.BlobberID, int64(blobber_detail.Size*int64(blobber_detail.Terms.Write_price)),
				stakePool.OffersTotal,
			)
			assert.Equal(
				t,
				int64(sizeInGB(blobber_detail.Size)*float64(blobber_detail.Terms.Write_price)),
				stakePool.OffersTotal,
			)
		}

		createAllocationTestTeardown(t, allocationID)
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
	require.Equal(t, 2, len(output))
	require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])
	return filename
}

func moveAllocationFile(t *testing.T, allocationID, remotepath, destination string) {
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
	require.GreaterOrEqual(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
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

// size in gigabytes
func sizeInGB(size int64) float64 {
	return float64(size) / GB
}

// ConvertToToken converts the value to ZCN tokens
func ConvertToToken(value int64) float64 {
	return float64(value) / float64(TOKEN_UNIT)
}

// ConvertToValue converts ZCN tokens to value
func ConvertToValue(token float64) int64 {
	return int64(token * float64(TOKEN_UNIT))
}
