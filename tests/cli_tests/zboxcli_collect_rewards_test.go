package cli_tests

import (
	"encoding/json"
	"fmt"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestCollectRewards(t *testing.T) {
	t.Run("Test Collect Reward", func(t *testing.T) {
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

		balanceBefore := getBalanceFromSharders(t, wallet.ClientID)

		cliutils.Wait(t, 2*time.Minute)
		output, err = collectRewards(t, configPath, stakePoolID, true)
		require.Equal(t, "transferred reward tokens", output[0])
		require.Nil(t, err, "Error collecting rewards", strings.Join(output, "\n"))

		balanceAfter := getBalanceFromSharders(t, wallet.ClientID)
		require.Greater(t, balanceAfter, balanceBefore)

		//// Wallet balance should decrease by locked amount
		//output, err = getBalance(t, configPath)
		//require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		//require.Len(t, output, 1)
		//require.Regexp(t, regexp.MustCompile(`Balance: 500.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		//cliutils.Wait(t, 20*time.Second)
		//output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobber.Id,
		//	"json":       "",
		//}))
		//stakePoolAfter := climodel.StakePoolInfo{}
		//err = json.Unmarshal([]byte(output[0]), &stakePoolAfter)
		//require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		//require.NotEmpty(t, stakePoolAfter)
		//require.Greater(t, stakePoolAfter.Rewards, stakePoolBefore.Rewards)
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
