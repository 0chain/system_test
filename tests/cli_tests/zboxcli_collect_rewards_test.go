package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCollectRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.RunWithTimeout("Test collect reward with valid pool and blobber id should pass", 90*time.Second, func(t *test.SystemTest) { //TODO slow

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.5,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		balanceBefore := getBalanceFromSharders(t, wallet.ClientID)

		// Upload and download a file so blobber can accumulate rewards
		allocSize := int64(2048)
		filesize := int64(256)
		remotepath := "/"

		// Use all 6 blobbers
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 1,
			"data":   5,
			"parity": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "error getting stake pool info")
		require.Len(t, output, 1)
		stakePoolAfter := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePoolAfter)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePoolAfter)

		rewards := int64(0)
		for _, poolDelegateInfo := range stakePoolAfter.Delegate {
			if poolDelegateInfo.DelegateID == wallet.ClientID {
				rewards = poolDelegateInfo.TotalReward
				break
			}
		}
		require.Greater(t, rewards, int64(0))

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   blobber.Id,
		}), true)
		require.Nil(t, err, "Error collecting rewards", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "transferred reward tokens", output[0])

		balanceAfter := getBalanceFromSharders(t, wallet.ClientID)
		require.GreaterOrEqual(t, balanceAfter, balanceBefore+rewards) // greater or equal since more rewards can accumulate after we check stakepool
	})

	t.Run("Test collect reward with invalid blobber id should fail", func(t *test.SystemTest) {

		t.Skip("piers")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.5,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   "invalid-blobber-id",
		}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "collect_reward_failed")
	})

	t.Run("Test collect reward with invalid provider type should fail", func(t *test.SystemTest) {

		t.Skip("piers")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.5,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "invalid-provider",
			"provider_id":   blobber.Id,
		}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "provider type must be blobber or validator")
	})

	t.Run("Test collect reward with no provider id or type should fail", func(t *test.SystemTest) {

		t.Skip("piers")
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "missing tokens flag")
	})
}

func collectRewards(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("collecting rewards...")
	cmd := fmt.Sprintf("./zbox collect-reward %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
