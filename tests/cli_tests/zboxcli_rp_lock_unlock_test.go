package cli_tests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestReadPoolLockUnlock(t *testing.T) {
	t.Parallel()

	t.Run("Locking read pool tokens moves tokens from wallet to read pool", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		allocParams := createParams(map[string]interface{}{
			"expire": "5m",
			"size":   "1024",
			"lock":   "0.5",
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     1,
			"duration":   "5m",
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
		require.InEpsilon(t, 1, intToZCN(readPool[0].Balance), epsilon, "Read pool balance did not match amount locked")
		require.GreaterOrEqual(t, time.Now().Add(time.Minute*5).Unix(), readPool[0].ExpireAt)
		require.Equal(t, allocationID, readPool[0].AllocationId)
		require.Less(t, 0, len(readPool[0].Blobber))
		require.Equal(t, true, readPool[0].Locked)

		for i := 0; i < len(readPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), readPool[0].Blobber[i].BlobberID)
			require.InEpsilon(t, 0.25, intToZCN(readPool[0].Blobber[i].Balance), epsilon)
		}
	})
}
