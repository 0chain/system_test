package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileDownloadTokenMovement(t *testing.T) {
	t.Parallel()

	balance := 0.4 // 400.000 mZCN
	t.Run("Read pool must have no tokens locked for a newly created allocation", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   balance,
			"size":   10485760,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "no tokens locked", output[0])
	})

	t.Run("Locked read pool tokens should equal total blobber balance in read pool", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   balance,
			"size":   10485760,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		output, err = readPoolLock(t, configPath, allocationID, 0.4)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		readPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &readPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), readPool[0].Id)
		require.InEpsilon(t, 0.4, intToZCN(readPool[0].Balance), epsilon, "Read pool balance [%v] did not match amount locked [%v]", intToZCN(readPool[0].Balance), 0.4)
		require.IsType(t, int64(1), readPool[0].ExpireAt)
		require.Equal(t, allocationID, readPool[0].AllocationId)
		require.Less(t, 0, len(readPool[0].Blobber))
		require.Equal(t, true, readPool[0].Locked)

		balanceInTotal := float64(0)
		for i := 0; i < len(readPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), readPool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), readPool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] read pool balance is [%v]", i, intToZCN(readPool[0].Blobber[i].Balance))
			balanceInTotal += intToZCN(readPool[0].Blobber[i].Balance)
		}

		require.InEpsilon(t, 0.4, balanceInTotal, epsilon, "Combined balance of blobbers [%v] did not match expected [%v]", balanceInTotal, 0.4)
	})

	t.Run("Each blobber's read pool balance should reduce by download cost", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   balance,
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
		output, err = readPoolLock(t, configPath, allocationID, 0.4)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Read pool before download
		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		initialReadPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Id)
		require.InEpsilon(t, 0.4, intToZCN(initialReadPool[0].Balance), epsilon, "read pool balance did not match expected")
		require.IsType(t, int64(1), initialReadPool[0].ExpireAt)
		require.Equal(t, allocationID, initialReadPool[0].AllocationId)
		require.Less(t, 0, len(initialReadPool[0].Blobber))
		require.Equal(t, true, initialReadPool[0].Locked)

		for i := 0; i < len(initialReadPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), initialReadPool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] balance is [%v]", i, intToZCN(initialReadPool[0].Blobber[i].Balance))
		}

		output, err = getDownloadCostInUnit(t, configPath, allocationID, "/"+filename)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))

		expectedDownloadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN = unitToZCN(expectedDownloadCostInZCN, unit)

		// Download the file
		output, err = downloadFile(t, configPath, allocationID, "../../internal/dummy_file/five_MB_test_file_dowloaded", "/"+filename)
		require.Nil(t, err, "Downloading the file failed", strings.Join(output, "\n"))

		defer os.Remove("../../internal/dummy_file/five_MB_test_file_dowloaded")

		require.Len(t, output, 2)
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filename, output[1])

		// Read pool after download
		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		finalReadPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Id)
		require.Less(t, intToZCN(finalReadPool[0].Balance), 0.4)
		require.IsType(t, int64(1), finalReadPool[0].ExpireAt)
		require.Equal(t, allocationID, finalReadPool[0].AllocationId)
		require.Equal(t, len(initialReadPool[0].Blobber), len(finalReadPool[0].Blobber))
		require.True(t, finalReadPool[0].Locked)

		for i := 0; i < len(finalReadPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), finalReadPool[0].Blobber[i].Balance)

			// amount deducted
			diff := intToZCN(initialReadPool[0].Blobber[i].Balance) - intToZCN(finalReadPool[0].Blobber[i].Balance)
			t.Logf("blobber [%v] read pool was deducted by [%v]", i, diff)
			require.InEpsilon(t, expectedDownloadCostInZCN, diff, epsilon, "blobber [%v] read pool was deducted by [%v] rather than the expected [%v]", i, diff, expectedDownloadCostInZCN)
		}
	})
}

func readPoolInfo(t *testing.T, cliConfigFilename, allocationID string) ([]string, error) {
	time.Sleep(30 * time.Second) // TODO replace with poller
	t.Logf("Getting read pool info...")
	return cliutils.RunCommand("./zbox rp-info --allocation " + allocationID + " --json --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func readPoolLock(t *testing.T, cliConfigFilename, allocationID string, tokens float64) ([]string, error) {
	t.Logf("Locking read tokens...")
	return cliutils.RunCommand(fmt.Sprintf("./zbox rp-lock --allocation %s --tokens %v --duration 900s --silent --wallet %s_wallet.json --configDir ./config --config %s", allocationID, tokens, escapedTestName(t), cliConfigFilename))
}

func getDownloadCostInUnit(t *testing.T, cliConfigFilename, allocationID, remotepath string) ([]string, error) {
	t.Logf("Getting download cost...")
	return cliutils.RunCommand("./zbox get-download-cost --allocation " + allocationID + " --remotepath " + remotepath + " --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config ")
}

func downloadFile(t *testing.T, cliConfigFilename, allocation, localpath, remotepath string) ([]string, error) {
	t.Logf("Downloading file...")
	return cliutils.RunCommand("./zbox download --allocation " + allocation + " --localpath " + localpath + " --remotepath " + remotepath + " --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
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
	case "ZCN", "zcn":
		unitCost /= 1e0
		return unitCost
	}
	return unitCost
}
