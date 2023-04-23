package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

const (
	newStakeWallet = "newStakeWallet"
)

func TestStakePool(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	_, err := registerWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", err)

	// get the list of blobbers
	blobbersList := getBlobbersList(t)
	require.Greater(t, len(blobbersList), 0, "No blobbers found")

	t.RunSequentiallyWithTimeout("total stake in a blobber can never be less than it's used capacity", 8*time.Minute, func(t *test.SystemTest) {
		_, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", err)

		// select the blobber with min staked capacity
		var minStakedCapacityBlobber climodel.BlobberInfo

		// set maxStakedCapacity to max uint64 value
		maxStakedCapacity := uint64(math.MaxUint64)

		for _, blobber := range blobbersList {
			output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))
			require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

			var blInfo climodel.BlobberInfo
			err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
			require.Nil(t, err, "error unmarshalling blobber info")

			stakedCapacity := uint64(float64(blInfo.TotalStake) * GB / float64(blInfo.Terms.WritePrice))

			require.GreaterOrEqual(t, stakedCapacity, uint64(blobber.Allocated), "Staked capacity should be greater than allocated capacity")

			stakedCapacity -= uint64(blobber.Allocated)

			if stakedCapacity < maxStakedCapacity {
				maxStakedCapacity = stakedCapacity
				minStakedCapacityBlobber = blInfo
			}
		}

		fmt.Println("maxStakedCapacity", maxStakedCapacity)

		// check total offers of the blobber
		blobber := minStakedCapacityBlobber

		output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))
		require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

		var blInfo climodel.BlobberInfo
		err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffers := blInfo.TotalOffers

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		delegates := stakePool.Delegate

		lenDelegates := len(delegates)

		// stake 1 token to all the blobbers
		for _, blobber := range blobbersList {
			_, err := executeFaucetWithTokens(t, configPath, 9)
			require.Nil(t, err, "Error executing faucet with tokens", err)

			_, err = stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
			require.Nil(t, err, "Error staking tokens", err)
		}

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew := len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew, lenDelegates+1, "Number of delegates should be greater")

		lenDelegates = lenDelegatesNew

		allocSize := maxStakedCapacity*3 + uint64(30*GB)
		allocSize -= allocSize / uint64(1*MB)

		options := map[string]interface{}{"cost": "", "size": allocSize, "data": len(blobbersList) / 2, "parity": len(blobbersList) - len(blobbersList)/2, "write_price": "0.1-0.2", "read_price": "0-0.1"}

		fmt.Println("options", options)

		output, err = createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		fmt.Println("allocation cost", output)

		allocationCost, err := getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		// Matching the wallet balance to allocationCost by executing faucet with tokens
		// As max limit of faucet is 9 tokens we are executing faucet with 9 tokens multiple times till wallet balance is equal to allocationCost
		for i := float64(0); i <= (allocationCost/9 + 2); i++ {
			_, err = executeFaucetWithTokens(t, configPath, 9)
			require.Nil(t, err, "Error executing faucet with tokens", err)
		}

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": (allocationCost + 1) * 2,
			"data":   len(blobbersList) / 2,
			"parity": len(blobbersList) - len(blobbersList)/2,
			"lock":   allocationCost + 1,
		})

		// check total offers new value and compare
		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))
		require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

		err = json.Unmarshal([]byte(output[0]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffersNew := blInfo.TotalOffers
		require.Greater(t, totalOffersNew, totalOffers, "Total Offers should Increase")

		_, err = registerWalletForName(t, configPath, newStakeWallet)
		require.Nil(t, err, "Error registering wallet", err)

		_, err = executeFaucetWithTokensForWallet(t, newStakeWallet, configPath, 9)
		require.Nil(t, err, "Error executing faucet with tokens", err)

		_, err = stakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
		require.Nil(t, err, "Error staking tokens", err)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew, lenDelegates+1, "Number of delegates should be greater")

		lenDelegates = lenDelegatesNew

		_, err = unstakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id}))
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", blobber.Id)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "Number of delegates should be greater")

		lenDelegates = lenDelegatesNew

		_, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id}))
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", blobber.Id)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew, lenDelegates, "delegates should be equal")

		lenDelegates = lenDelegatesNew

		// Cancel the allocation
		_, err = cancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, "error canceling allocation")

		_, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id}))
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", blobber.Id)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "Number of delegates should be greater")
	})
}
