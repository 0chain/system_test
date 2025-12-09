package cli_tests

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestKillBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	// Commeneted till fixed: t.SetSmokeTests("killed blobber is not available for allocations")

	// Killing a blobber should make it unavalable for any new allocations,
	// and stake pools should be slashed by an amount given by the "stakepool.kill_slash" setting
	t.RunSequentiallyWithTimeout("killed blobber is not available for allocations", 10*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		startBlobbers := getBlobbers(t)
		var blobberToKill string
		var activeBlobbers int
		for i := range startBlobbers {
			if !startBlobbers[i].IsKilled && !startBlobbers[i].IsShutdown && !startBlobbers[i].NotAvailable {
				activeBlobbers++
				if !startBlobbers[i].IsKilled && blobberToKill == "" {
					blobberToKill = startBlobbers[i].ID
				}
			}
		}
		require.NotEqual(t, blobberToKill, "", "all active blobbers have been killed")
		require.True(t, activeBlobbers > 1, "need at least two active blobbers")
		// Use a conservative shard configuration that works with available blobbers
		// Use 2 data + 1 parity = 3 total to ensure we have enough blobbers even after kill
		// This is more conservative than 4+2=6 to avoid "not enough blobbers" errors
		dataShards := 2
		parityShards := 1
		// Only use larger configuration if we have plenty of blobbers
		if activeBlobbers >= 7 {
			dataShards = 4
			parityShards = 2
		}
		// Final safety check - ensure we don't exceed available blobbers
		if dataShards+parityShards > activeBlobbers {
			dataShards = 2
			parityShards = 1
		}
		t.Logf("blobberToKill: %s, activeBlobbers: %d, dataShards: %d, parityShards: %d, total needed: %d",
			blobberToKill, activeBlobbers, dataShards, parityShards, dataShards+parityShards)

		t.Log("blobberToKill", blobberToKill)

		_, err := stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberToKill, "tokens": 100}), true)
		require.NoErrorf(t, err, "error staking tokens to blobber %s", blobberToKill)

		time.Sleep(2 * time.Minute)

		allocationParams := createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   5.0,
			"size":   "2097152", // 2MB to ensure it's above min_alloc_size
		})
		t.Logf("Creating allocation with params: %s (expecting %d blobbers: %d data + %d parity)",
			allocationParams, dataShards+parityShards, dataShards, parityShards)
		output, err := createNewAllocation(t, configPath, allocationParams)
		require.NoError(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"),
			output[0], "Allocation creation output did not match expected")
		allocationID, err := getAllocationID(output[0])
		require.NoError(t, err)
		createAllocationTestTeardown(t, allocationID)

		spBefore := getStakePoolInfo(t, blobberToKill)
		output, err = killBlobber(t, scOwnerWallet, configPath, createParams(map[string]interface{}{
			"id": blobberToKill,
		}), true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		cliutils.Wait(t, 5*time.Second)

		spAfter := getStakePoolInfo(t, blobberToKill)
		deadBlobber := getBlobber(t, blobberToKill)

		settings := getStorageConfigMap(t)
		killSlash := settings.Numeric["stakepool.kill_slash"]
		require.True(t, deadBlobber.IsKilled)
		for poolIndex := range spAfter.Delegate {
			t.Log("poolIndex", poolIndex)
			t.Log("delegateID", spAfter.Delegate[poolIndex].DelegateID)
			t.Log("spBefore", spBefore.Delegate[poolIndex].Balance)
			t.Log("spAfter", spAfter.Delegate[poolIndex].Balance)
			t.Log("killSlash", killSlash)
			require.InEpsilon(t, float64(spBefore.Delegate[poolIndex].Balance)*(1-killSlash), float64(spAfter.Delegate[poolIndex].Balance), 0.05,
				"stake pools should be slashed by %f", killSlash) // 5% error margin because there can be challenge penalty
		}

		stakingWalletModel, err := getWalletForName(t, configPath, stakingWallet)
		require.NoError(t, err, "error fetching wallet")

		balanceBefore := getBalanceFromSharders(t, stakingWalletModel.ClientID)

		output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   blobberToKill,
			"fee":           "0.15",
		}), stakingWallet, true)
		require.NoError(t, err, output)

		balanceAfter := getBalanceFromSharders(t, stakingWalletModel.ClientID) + 1500000000 // Txn fee
		require.GreaterOrEqual(t, balanceAfter, balanceBefore, "should have collected rewards")

		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberToKill}), true)
		require.NoError(t, err, "should be able to unstake tokens from a killed blobber")
		t.Log(strings.Join(output, "\n"))

		blobberDelegateWallet, err := getWalletForName(t, configPath, blobberOwnerWallet)
		require.NoError(t, err, "Error getting wallet for blobber owner")

		balanceBefore = getBalanceFromSharders(t, blobberDelegateWallet.ClientID)

		output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   blobberToKill,
			"fee":           "0.15",
		}), blobberOwnerWallet, true)
		require.NoError(t, err, output)

		balanceAfter = getBalanceFromSharders(t, blobberDelegateWallet.ClientID) + 1500000000 // Txn fee
		require.Greater(t, balanceAfter, balanceBefore, "should have collected rewards")

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   4.0,
			"size":   "2097152", // 2MB to ensure it's above min_alloc_size
		}))
		require.Error(t, err, "should fail to create allocation")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "not enough blobbers to honor the allocation"),
			"after killing a blobber there should no longer be enough blobbers for this allocation")
	})

	t.RunSequentially("kill blobber by non-smartcontract owner should fail", func(t *test.SystemTest) {
		createWallet(t)

		startBlobbers := getBlobbers(t)
		var blobberToKill string
		for i := range startBlobbers {
			if !startBlobbers[i].IsKilled && !startBlobbers[i].IsShutdown {
				blobberToKill = startBlobbers[i].ID
				break
			}
		}

		output, err := killBlobber(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"id": blobberToKill,
		}), true)
		require.Error(t, err, "kill blobber by non-smartcontract owner should fail")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "unauthorized access - only the owner can access"), "")
	})

	t.RunSequentiallyWithTimeout("shutdowned blobber is not available for allocations", 10*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

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
		// Use a conservative shard configuration that works with available blobbers
		// Use 2 data + 1 parity = 3 total to ensure we have enough blobbers even after shutdown
		// This is more conservative than 4+2=6 to avoid "not enough blobbers" errors
		dataShards := 2
		parityShards := 1
		// Only use larger configuration if we have plenty of blobbers
		if activeBlobbers >= 7 {
			dataShards = 4
			parityShards = 2
		}
		// Final safety check - ensure we don't exceed available blobbers
		if dataShards+parityShards > activeBlobbers {
			dataShards = 2
			parityShards = 1
		}

		t.Logf("blobberToShutdown: %s, activeBlobbers: %d, dataShards: %d, parityShards: %d, total needed: %d",
			blobberToShutdown, activeBlobbers, dataShards, parityShards, dataShards+parityShards)

		_, err := stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberToShutdown, "tokens": 100}), true)
		require.NoErrorf(t, err, "error staking tokens to blobber %s", blobberToShutdown)

		time.Sleep(2 * time.Minute)

		allocationParams := createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   5.0,
			"size":   "2097152", // 2MB to ensure it's above min_alloc_size
		})
		t.Logf("Creating allocation with params: %s (expecting %d blobbers: %d data + %d parity)",
			allocationParams, dataShards+parityShards, dataShards, parityShards)
		output, err := createNewAllocation(t, configPath, allocationParams)
		require.NoError(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"),
			output[0], "Allocation creation output did not match expected")
		allocationID, err := getAllocationID(output[0])
		require.NoError(t, err)
		createAllocationTestTeardown(t, allocationID)

		_, err = stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberToShutdown, "tokens": 100}), true)
		require.NoErrorf(t, err, "error staking tokens to blobber %s", blobberToShutdown)

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

		stakingWalletModel, err := getWalletForName(t, configPath, stakingWallet)
		require.NoError(t, err, "error fetching wallet")

		balanceBefore := getBalanceFromSharders(t, stakingWalletModel.ClientID)

		output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   blobberToShutdown,
			"fee":           "0.15",
		}), stakingWallet, true)
		require.NoError(t, err, output)

		balanceAfter := getBalanceFromSharders(t, stakingWalletModel.ClientID) + 1500000000 // Txn fee
		require.GreaterOrEqual(t, balanceAfter, balanceBefore, "should have collected rewards")

		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberToShutdown}), true)
		require.NoError(t, err, "should be able to unstake tokens from a shutdowned blobber")
		t.Log(strings.Join(output, "\n"))

		blobberDelegateWallet, err := getWalletForName(t, configPath, blobberOwnerWallet)
		require.NoError(t, err, "Error getting wallet for blobber owner")

		balanceBefore = getBalanceFromSharders(t, blobberDelegateWallet.ClientID)

		output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   blobberToShutdown,
			"fee":           "0.15",
		}), blobberOwnerWallet, true)
		require.NoError(t, err, output)

		balanceAfter = getBalanceFromSharders(t, blobberDelegateWallet.ClientID) + 1500000000 // Txn fee
		require.Greater(t, balanceAfter, balanceBefore, "should have collected rewards")

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   4.0,
			"size":   "2097152", // 2MB to ensure it's above min_alloc_size
		}))
		require.Error(t, err, "should fail to create allocation")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "not enough blobbers to honor the allocation"),
			"after shutdowning a blobber there should no longer be enough blobbers for this allocation")
	})

	t.RunSequentially("shutdown blobber by non-smartcontract owner should fail", func(t *test.SystemTest) {
		createWallet(t)

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

func killBlobber(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("kill blobber...")
	cmd := fmt.Sprintf("./zbox kill-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getBlobbers(t *test.SystemTest) []model.BlobberDetails {
	var blobbers []model.BlobberDetails
	output, err := listBlobbers(t, configPath, "--json")
	require.NoError(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.True(t, len(output) > 0, "no output to ls-blobbers")
	err = json.Unmarshal([]byte(output[len(output)-1]), &blobbers)
	require.NoError(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")
	return blobbers
}

func getBlobber(t *test.SystemTest, id string) model.BlobberDetails {
	output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": id}))
	require.NoError(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var blobberInfo model.BlobberDetails
	err = json.Unmarshal([]byte(output[0]), &blobberInfo)
	require.Nil(t, err, strings.Join(output, "\n"))

	return blobberInfo
}

func getStakePoolInfo(t *test.SystemTest, blobberId string) model.StakePoolInfo {
	// Use sp-info to check the staked tokens in blobber's stake pool
	output, err := stakePoolInfo(t, configPath, createParams(map[string]interface{}{
		"blobber_id": blobberId,
		"json":       "",
	}))
	require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	stakePool := model.StakePoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &stakePool)
	require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
	require.NotEmpty(t, stakePool)

	sort.Slice(stakePool.Delegate, func(i, j int) bool {
		return stakePool.Delegate[i].DelegateID < stakePool.Delegate[j].DelegateID
	})

	return stakePool
}

func shutdownBlobber(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("shutdown blobber...")
	cmd := fmt.Sprintf("./zbox shutdown-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)

	log.Println(cmd)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint:deadcode,unused
func collectRewards(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return collectRewardsForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func collectRewardsForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("collecting rewards...")
	cmd := fmt.Sprintf("./zbox collect-reward %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
