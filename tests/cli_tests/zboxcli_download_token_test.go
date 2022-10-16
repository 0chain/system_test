package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileDownloadTokenMovement(t *testing.T) {
	t.Parallel()

	t.Run("Each blobber's read pool balance should reduce by download cost", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   0.6,
			"size":   10485760,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		file := generateRandomTestFileName(t)
		remotePath := "/" + filepath.Base(file)
		fileSize := int64(5 * MB) // must upload bigger file to ensure has noticeable cost
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}
		output, err = uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		// Lock read pool tokens
		lockedTokens := 0.4
		readPoolParams := createParams(map[string]interface{}{
			"tokens": lockedTokens,
		})
		output, err = readPoolLock(t, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Read pool before download
		initialReadPool := getReadPoolInfo(t)
		require.Equal(t, ConvertToValue(lockedTokens), initialReadPool.Balance)

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expectedDownloadCost, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		t.Logf("Download cost: %f", expectedDownloadCost)

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN := unitToZCN(expectedDownloadCost, unit)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		})

		output, err = downloadFile(t, configPath, downloadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Nil(t, err, "Downloading the file failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)

		// Wait for blobber to redeem read-tokens
		// Blobber runs worker in the interval of usually 10 seconds.
		time.Sleep(time.Second * 20)

		// Read pool after download
		expectedPoolBalance := ConvertToValue(lockedTokens) - ConvertToValue(expectedDownloadCostInZCN)
		updatedReadPool, err := getReadPoolUpdate(t, initialReadPool, 5)
		require.NoError(t, err)
		require.Equal(t, expectedPoolBalance, updatedReadPool.Balance, "Read Pool balance must be equal to (initial balance-download cost)")
	})
}

func readPoolInfo(t *testing.T, cliConfigFilename string) ([]string, error) {
	return readPoolInfoWithWallet(t, escapedTestName(t), cliConfigFilename)
}

func readPoolInfoWithWallet(t *testing.T, wallet, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 30*time.Second) // TODO replace with poller
	t.Logf("Getting read pool info...")
	return cliutils.RunCommand(t, "./zbox rp-info"+" --json --silent --wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func readPoolLock(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return readPoolLockWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func readPoolLockWithWallet(t *testing.T, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Locking read tokens...")
	cmd := fmt.Sprintf("./zbox rp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getDownloadCost(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return getDownloadCostWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func getDownloadCostWithWallet(t *testing.T, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Getting download cost...")
	cmd := fmt.Sprintf("./zbox get-download-cost %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func unitToZCN(unitCost float64, unit string) float64 {
	switch unit {
	case "SAS", "sas":
		unitCost /= 1e10
		return unitCost
	case "uZCN", "uzcn":
		unitCost /= 1e6
		return unitCost
	case "mZCN", "mzcn":
		unitCost /= 1e3
		return unitCost
	}
	return unitCost
}
