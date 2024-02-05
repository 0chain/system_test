package cli_tests

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestShutdownBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	// Commeneted till fixed: t.SetSmokeTests("shutdowned blobber is not available for allocations")

	// Shutdowning a blobber should make it unavalable for any new allocations,
	// and stake pools should be slashed by an amount given by the "stakepool.shutdown_slash" setting
	t.RunSequentially("shutdowned blobber is not available for allocations", func(t *test.SystemTest) {
		output, err := createWalletForName(t, configPath, scOwnerWallet)
		require.NoError(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		startBlobbers := getBlobbers(t)
		var blobberToShutdown string
		var activeBlobbers int
		for i := range startBlobbers {
			if !startBlobbers[i].IsKilled && !startBlobbers[i].IsShutdown && !startBlobbers[i].NotAvailable {
				activeBlobbers++
				blobberToShutdown = startBlobbers[i].ID
			}
		}
		require.NotEqual(t, blobberToShutdown, "", "all active blobbers have been shutdowned")
		require.True(t, activeBlobbers > 1, "need at least two active blobbers")
		dataShards := 1
		parityShards := activeBlobbers - 1

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   5.0,
			"size":   "10000",
		}))
		require.NoError(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"),
			output[0], "Allocation creation output did not match expected")
		allocationID, err := getAllocationID(output[0])
		require.NoError(t, err)
		createAllocationTestTeardown(t, allocationID)

		_, err = executeFaucetWithTokens(t, configPath, 100.0)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		_, err = stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberToShutdown, "tokens": 100}), true)
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", blobberToShutdown)

		spBefore := getStakePoolInfo(t, blobberToShutdown)
		output, err = shutdownBlobber(t, scOwnerWallet, configPath, createParams(map[string]interface{}{
			"id": blobberToShutdown,
		}), true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		cliutils.Wait(t, 5*time.Second)

		spAfter := getStakePoolInfo(t, blobberToShutdown)
		deadBlobber := getBlobber(t, blobberToShutdown)

		settings := getStorageConfigMap(t)
		shutdownSlash := settings.Numeric["stakepool.kill_slash"] / 2
		require.True(t, deadBlobber.IsShutdown)
		for poolIndex := range spAfter.Delegate {
			t.Log("poolIndex", poolIndex)
			t.Log("delegateID", spAfter.Delegate[poolIndex].DelegateID)
			t.Log("spBefore", spBefore.Delegate[poolIndex].Balance)
			t.Log("spAfter", spAfter.Delegate[poolIndex].Balance)
			t.Log("shutdownSlash", shutdownSlash)
			require.InEpsilon(t, float64(spBefore.Delegate[poolIndex].Balance)*(1-shutdownSlash), float64(spAfter.Delegate[poolIndex].Balance), 0.05,
				"stake pools should be slashed by %f", shutdownSlash) // 5% error margin because there can be challenge penalty
		}

		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberToShutdown}), true)
		require.NoError(t, err, "should be able to unstake tokens from a shutdowned blobber")
		t.Log(strings.Join(output, "\n"))

		balanceBefore := getBalanceFromSharders(t, blobberOwnerWallet)

		output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   blobberToShutdown,
			"fee":           "0.15",
		}), blobberOwnerWallet, true)
		require.NoError(t, err, output)

		balanceAfter := getBalanceFromSharders(t, blobberOwnerWallet)

		require.Greater(t, balanceAfter, balanceBefore, "should have collected rewards")

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   4.0,
			"size":   "10000",
		}))
		require.Error(t, err, "should fail to create allocation")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "not enough blobbers to honor the allocation"),
			"after shutdowning a blobber there should no longer be enough blobbers for this allocation")
	})

	t.RunSequentially("shutdown blobber by non-smartcontract owner should fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

		startBlobbers := getBlobbers(t)
		var blobberToShutdown string
		for i := range startBlobbers {
			if !startBlobbers[i].IsKilled && !startBlobbers[i].IsShutdown {
				blobberToShutdown = startBlobbers[i].ID
				break
			}
		}

		output, err := shutdownBlobber(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"id": blobberToShutdown,
		}), true)
		require.Error(t, err, "shutdown blobber by non-smartcontract owner should fail")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "unauthorized access - only the owner can access"), "")
	})
}

func shutdownBlobber(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("shutdown blobber...")
	cmd := fmt.Sprintf("./zbox shutdown-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)

	fmt.Println(cmd)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
