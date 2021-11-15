package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
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

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "900s",
		})
		output, err = readPoolLock(t, configPath, params)
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
}

func readPoolInfo(t *testing.T, cliConfigFilename, allocationID string) ([]string, error) {
	time.Sleep(30 * time.Second) // TODO replace with poller
	t.Logf("Getting read pool info...")
	return cliutils.RunCommand(t, "./zbox rp-info --allocation "+allocationID+" --json --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func readPoolLock(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return readPoolLockWithWallet(t, escapedTestName(t), cliConfigFilename, params)
}

func readPoolLockWithWallet(t *testing.T, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Locking read tokens...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox rp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}

func getDownloadCostInUnit(t *testing.T, cliConfigFilename, allocationID, remotepath string) ([]string, error) {
	t.Logf("Getting download cost...")
	return cliutils.RunCommand(t, "./zbox get-download-cost --allocation "+allocationID+" --remotepath "+remotepath+" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
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
