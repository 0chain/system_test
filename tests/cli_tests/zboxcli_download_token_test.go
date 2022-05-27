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
		t.Skip("Skipped for nonce merge")
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

		path, err := filepath.Abs("tmp")
		require.Nil(t, err)

		filename := cliutils.RandomAlphaNumericString(10) + "_test.txt"
		fullPath := fmt.Sprintf("%s/%s", path, filename)
		err = createFileWithSize(fullPath, 1024*5)
		require.Nil(t, err, "error while generating file: ", err)

		// upload a dummy 5 MB file
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fullPath,
			"remotepath": "/",
		})

		// Lock read pool tokens
		lockedTokens := 0.4
		params := createParams(map[string]interface{}{
			"tokens": lockedTokens,
		})
		output, err = readPoolLock(t, configPath, params, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Read pool before download
		initialReadPool := getReadPoolInfo(t)
		require.Equal(t, ConvertToValue(lockedTokens), initialReadPool.OwnerBalance)

		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filename,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expectedDownloadCost, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN := unitToZCN(expectedDownloadCost, unit)

		// Download the file
		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filename,
			"localpath":  "../../internal/dummy_file/five_MB_test_file_dowloaded",
		}), true)
		require.Nil(t, err, "Downloading the file failed", strings.Join(output, "\n"))

		defer os.Remove("../../internal/dummy_file/five_MB_test_file_dowloaded")

		require.Len(t, output, 2)
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filename, output[1])

		// Wait for blobber to redeem read-tokens
		// Blobber runs worker in the interval of usually 10 seconds.
		time.Sleep(time.Second * 20)

		// Read pool after download
		expectedPoolBalance := ConvertToValue(lockedTokens) - ConvertToValue(expectedDownloadCostInZCN)
		updatedReadPool, err := getReadPoolUpdate(t, initialReadPool, 5)
		require.NoError(t, err)
		require.Equal(t, expectedPoolBalance, updatedReadPool.OwnerBalance, "Read Pool balance must be equal to (initial balance-download cost)")
	})
}

func readPoolInfo(t *testing.T, cliConfigFilename string) ([]string, error) {
	return readPoolInfoWithwallet(t, escapedTestName(t), cliConfigFilename)
}

func readPoolInfoWithwallet(t *testing.T, wallet, cliConfigFilename string) ([]string, error) {
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
