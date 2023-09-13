package cli_tests

import (
	"encoding/json"
	"fmt"
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
	t.RunSequentially("killed blobber is not available for allocations", func(t *test.SystemTest) {
		output, err := createWalletForName(t, configPath, scOwnerWallet)
		require.NoError(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

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

		spBefore := getStakePoolInfo(t, blobberToKill)
		output, err = killBlobber(t, scOwnerWallet, configPath, createParams(map[string]interface{}{
			"id": blobberToKill,
		}), true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		time.Sleep(5 * time.Second)

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
			require.InEpsilon(t, float64(spBefore.Delegate[poolIndex].Balance)*killSlash, float64(spAfter.Delegate[poolIndex].Balance), 0.05,
				"stake pools should be slashed by %f", killSlash)
		}

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   4.0,
			"size":   "10000",
		}))
		require.Error(t, err, "should fail to create allocation")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "not enough blobbers to honor the allocation"),
			"after killing a blobber there should no longer be enough blobbers for this allocation")
	})

	t.RunSequentially("kill blobber by non-smartcontract owner should fail", func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.NoError(t, err)

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
