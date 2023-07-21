package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestFileDownloadTokenMovement(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Downloader's readpool balance should reduce by download cost")

	t.Parallel()

	t.RunWithTimeout("Downloader's readpool balance should reduce by download cost", 5*time.Minute, func(t *test.SystemTest) { //TODO: way too slow
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		file := generateRandomTestFileName(t)
		remoteOwnerPath := "/" + filepath.Base(file)
		fileSize := int64(10 * MB) // must upload bigger file to ensure has noticeable cost
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remoteOwnerPath,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// locking 1 read tokens to readPool via wallet
		createWalletForNameAndLockReadTokens(t, configPath, walletOwner)

		output, err = readPoolInfoWithWallet(t, walletOwner, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		initialReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, initialReadPool)

		// staked a total of 1.4*1e10 tokens in readpool
		require.Equal(t, 1.4*1e10, float64(initialReadPool.Balance))

		// download cost functions works fine with no issues.
		output, err = getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteOwnerPath,
		}), true)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expectedDownloadCost, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInSas := unitToZCN(expectedDownloadCost, unit) * 1e10
		t.Logf("Download cost: %v sas", expectedDownloadCostInSas)

		// Download the file (delete local copy first)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"allocation": allocationID,
			"remotepath": remoteOwnerPath,
		})

		// downloading file for wallet
		output, err = downloadFileForWallet(t, walletOwner, configPath, downloadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, "download file - Unexpected output", strings.Join(output, "\n"))
		require.Contains(t, output[1], StatusCompletedCB)
		require.Contains(t, output[1], filepath.Base(file))

		// waiting 60 seconds for blobber to redeem tokens
		cliutils.Wait(t, 60*time.Second)

		output, err = readPoolInfoWithWallet(t, walletOwner, configPath)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		finalReadPool := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, finalReadPool)

		expectedRPBalance := 1.4*1e10 - expectedDownloadCostInSas - 10 // because download cost is till 3 decimal point only and missing the 4th decimal digit
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		// getDownloadCost returns download cost when all the associated blobbers of an allocation are required
		// In current enhancement/verify-download PR, it gets data from minimum blobbers possible.
		// So the download cost will be in between initial balance and expected balance.
		t.Logf("FinalReadPool.Balance:%d\nInitialReadPool.Balance:%d\nExpectedReadPool.Balance:%d\n", finalReadPool.Balance, initialReadPool.Balance, int64(expectedRPBalance))
		require.Equal(t, true,
			finalReadPool.Balance < initialReadPool.Balance &&
				finalReadPool.Balance >= int64(expectedRPBalance))
	})
}

func readPoolInfo(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return readPoolInfoWithWallet(t, escapedTestName(t), cliConfigFilename)
}

func readPoolInfoWithWallet(t *test.SystemTest, wallet, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 30*time.Second) // TODO replace with poller
	t.Logf("Getting read pool info...")
	return cliutils.RunCommand(t, "./zbox rp-info"+" --json --silent --wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func readPoolLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return readPoolLockWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func readPoolLockWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Locking read tokens...")
	cmd := fmt.Sprintf("./zbox rp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getDownloadCost(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return getDownloadCostWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func getDownloadCostWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
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
