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
		require.Len(t, output, 1)

		allocationID := strings.Fields(output[0])[2]

		output, err = challengePoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Could not fetch challenge pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		challengePool := climodel.ChallengePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &challengePool)
		require.Nil(t, err, "Error unmarshalling challenge pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, challengePool)

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

		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		writePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &writePool)
		require.Nil(t, err, "Error unmarshalling write pool", strings.Join(output, "\n"))
		require.NotEmpty(t, writePool)

		require.Equal(t, allocationID, writePool[0].Id)
		require.Equal(t, 0.8, intToZCN(writePool[0].Balance))
		require.IsType(t, int64(1), writePool[0].ExpireAt)
		require.Equal(t, allocationID, writePool[0].AllocationId)
		require.Less(t, 0, len(writePool[0].Blobber))
		require.Equal(t, true, writePool[0].Locked)

		totalBalanceInBlobbers := float64(0)
		for _, blobber := range writePool[0].Blobber {
			t.Logf("Blobber [%v] balance is [%v]", blobber.BlobberID, intToZCN(blobber.Balance))
			totalBalanceInBlobbers += intToZCN(blobber.Balance)
		}
		require.Equalf(t, 0.8, totalBalanceInBlobbers,"Sum of balances should be [%v] but was [%v]", 0.8, totalBalanceInBlobbers)
	})
}

func writePoolInfo(t *testing.T, cliConfigFilename string, retry bool) ([]string, error) {
	t.Logf("Getting write pool info...")
	cmd := "./zbox wp-info --json --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getUploadCostInUnit(t *testing.T, cliConfigFilename, allocationID, localpath string) ([]string, error) {
	t.Logf("Getting upload cost...")
	output, err := cliutils.RunCommand(t, "./zbox get-upload-cost --allocation "+allocationID+" --localpath "+localpath+" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	require.Nil(t, err, "error getting upload cost in unit", strings.Join(output, "\n"))
	require.Len(t, output, 1)
	return output, err
}

func challengePoolInfo(t *testing.T, cliConfigFilename, allocationID string) ([]string, error) {
	t.Logf("Getting challenge pool info...")
	return cliutils.RunCommand(t, "./zbox cp-info --allocation "+allocationID+" --json --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func intToZCN(balance int64) float64 {
	return float64(balance) / tokenUnit
}
