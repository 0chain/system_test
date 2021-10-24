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

const epsilon float64 = 1e-01
const tokenUnit float64 = 1e+10

func TestFileUploadTokenMovement(t *testing.T) {
	t.Parallel()

	balance := 0.8 // 800.000 mZCN
	t.Run("Challenge pool should be 0 before any write", func(t *testing.T) {
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

		allocationID := strings.Fields(output[0])[2]

		output, err = challengePoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Could not fetch challenge pool", strings.Join(output, "\n"))

		challengePool := climodel.ChallengePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &challengePool)
		require.Nil(t, err, "Error unmarshalling challenge pool info", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("([a-f0-9]{64}):challengepool:%s", allocationID)), challengePool.Id)
		require.IsType(t, int64(0), challengePool.StartTime)
		require.IsType(t, int64(0), challengePool.Expiration)
		require.False(t, challengePool.Finalized)
		require.Equal(t, float64(0), float64(challengePool.Balance))
	})

	t.Run("Total balance in blobber pool equals locked tokens", func(t *testing.T) {
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
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		output, err = writePoolInfo(t, configPath)
		require.Nil(t, err, "Failed to fetch Write Pool info", strings.Join(output, "\n"))

		writePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &writePool)
		require.Nil(t, err, "Error unmarshalling write pool", strings.Join(output, "\n"))

		require.Equal(t, allocationID, writePool[0].Id)
		require.InDelta(t, 0.8, intToZCN(writePool[0].Balance), epsilon)
		require.IsType(t, int64(1), writePool[0].ExpireAt)
		require.Equal(t, allocationID, writePool[0].AllocationId)
		require.Less(t, 0, len(writePool[0].Blobber))
		require.Equal(t, true, writePool[0].Locked)

		totalBalanceInBlobbers := float64(0)
		for _, blobber := range writePool[0].Blobber {
			t.Logf("Blobber [%v] balance is [%v]", blobber.BlobberID, intToZCN(blobber.Balance))
			totalBalanceInBlobbers += intToZCN(blobber.Balance)
		}
		require.InDelta(t, 0.8, totalBalanceInBlobbers, epsilon, "Sum of balances should be [%v] but was [%v]", 0.8, totalBalanceInBlobbers)
	})
}

func writePoolInfo(t *testing.T, cliConfigFilename string) ([]string, error) {
	time.Sleep(10 * time.Second) // TODO replace with poller
	t.Logf("Getting write pool info...")
	return cliutils.RunCommand("./zbox wp-info --json --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func getUploadCostInUnit(t *testing.T, cliConfigFilename, allocationID, localpath string) ([]string, error) {
	t.Logf("Getting upload cost...")
	return cliutils.RunCommand("./zbox get-upload-cost --allocation " + allocationID + " --localpath " + localpath + " --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func challengePoolInfo(t *testing.T, cliConfigFilename, allocationID string) ([]string, error) {
	t.Logf("Getting challenge pool info...")
	return cliutils.RunCommand("./zbox cp-info --allocation " + allocationID + " --json --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func intToZCN(balance int64) float64 {
	return float64(balance) / tokenUnit
}
