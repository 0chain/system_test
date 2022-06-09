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

func TestCollectRewards(t *testing.T) {
	t.Parallel()

	t.Run("Test Collect Reward Successfully", func(t *testing.T) {
		t.Parallel()

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
		require.Regexp(t, regexp.MustCompile("tokens locked, pool id: ([a-f0-9]{64})"), output[0])
		stakePoolID := strings.Fields(output[0])[4]
		require.Nil(t, err, "Error extracting pool Id from sp-lock output", strings.Join(output, "\n"))

		stackPoolOutputBefore, err := stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		stakePoolBefore := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(stackPoolOutputBefore[0]), &stakePoolBefore)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(stackPoolOutputBefore, "\n"))
		require.NotEmpty(t, stakePoolBefore)

		rewardsBefore := int64(0)
		for _, poolDelegateInfo := range stakePoolBefore.Delegate {
			if poolDelegateInfo.DelegateID == wallet.ClientID {
				rewardsBefore = poolDelegateInfo.TotalReward
				break
			}
		}

		cliutils.Wait(t, 40*time.Second)
		output, err = collectRewards(t, configPath, stakePoolID, true)
		require.Equal(t, "transferred reward tokens", output[0])
		require.Nil(t, err, "Error collecting rewards", strings.Join(output, "\n"))

		cliutils.Wait(t, 40*time.Second)
		stackPoolOutputAfter, err := stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		stakePoolAfter := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(stackPoolOutputAfter[0]), &stakePoolAfter)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(stackPoolOutputAfter, "\n"))
		require.NotEmpty(t, stakePoolAfter)

		rewardsAfter := int64(0)
		for _, poolDelegateInfo := range stakePoolAfter.Delegate {
			if poolDelegateInfo.DelegateID == wallet.ClientID {
				rewardsAfter = poolDelegateInfo.TotalReward
				break
			}
		}
		require.Greater(t, rewardsAfter, rewardsBefore)
	})
}

func collectRewards(t *testing.T, cliConfigFilename, poolId string, retry bool) ([]string, error) {
	t.Log("collecting rewards...")
	cmd := fmt.Sprintf("./zbox collect-reward --pool_id %s --provider_type blobber --silent --wallet %s_wallet.json --configDir ./config --config %s", poolId, escapedTestName(t), cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
