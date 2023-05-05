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

	_, err := createWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", err)

	// get the list of blobbers
	blobbersList := getBlobbersList(t)
	require.Greater(t, len(blobbersList), 0, "No blobbers found")

	t.RunSequentiallyWithTimeout("Total stake in a blobber can never be less than it's used capacity", 800*time.Minute, func(t *test.SystemTest) {
		_, err := createWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", err)

		// select the blobber with minimum available stake capacity
		var minAvailableCapacityBlobber climodel.BlobberInfo
		minAvailableCapacity := int64(math.MaxInt64)

		for _, blobber := range blobbersList {
			if blobber.IsKilled || blobber.IsShutdown {
				continue
			}

			output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))
			require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

			var blInfo climodel.BlobberInfo
			err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
			require.Nil(t, err, "error unmarshalling blobber info")

			stakedCapacity := int64(float64(blInfo.TotalStake) * GB / float64(blInfo.Terms.WritePrice))

			require.GreaterOrEqual(t, stakedCapacity, blobber.Allocated, "Staked capacity should be greater than allocated capacity")

			fmt.Println("stakedCapacity", stakedCapacity)

			stakedCapacity -= blobber.Allocated

			if stakedCapacity < minAvailableCapacity {
				minAvailableCapacity = stakedCapacity
				minAvailableCapacityBlobber = blInfo
			}

			fmt.Println("minAvailableCapacity", minAvailableCapacity)
		}

		output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": minAvailableCapacityBlobber.Id}))
		require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

		var blInfo climodel.BlobberInfo
		err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffers := blInfo.TotalOffers

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": minAvailableCapacityBlobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))

		stakePool := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		delegates := stakePool.Delegate
		lenDelegates := len(delegates)

		// stake 1 token on this blobber
		_, err = executeFaucetWithTokens(t, configPath, 9)
		require.Nil(t, err, "Error executing faucet with tokens", err)

		for _, blobber := range blobbersList {
			if blobber.IsKilled || blobber.IsShutdown {
				continue
			}
			_, err = stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
			require.Nil(t, err, "Error staking tokens", err)
		}

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": minAvailableCapacityBlobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew := len(stakePool.Delegate)
		require.Equal(t, lenDelegatesNew, lenDelegates+1, "Number of delegates should increase by")
		lenDelegates = lenDelegatesNew // update lenDelegates to current value

		// Create an allocation of maximum size that all blobbers can honor.
		// This requires creating an allocation of capacity = available capacity of blobber which has minimum
		// available capacity. For example, if 3 blobbers have 4 GB, 5 GB and 6 GB available,
		// the max allocation they all can honor is of 4 GB.
		allocSize := minAvailableCapacity*2 + 20*GB - 200000
		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"cost":        "",
			"data":        2,
			"parity":      2,
			"expire":      "5m",
			"size":        allocSize,
			"read_price":  "0-0.1",
			"write_price": "0-0.1",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		allocationCost, err := getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost")

		fmt.Println("allocationCost", allocationCost)

		// Matching the wallet balance to allocationCost by executing faucet with tokens
		// As max limit of faucet is 9 tokens we are executing faucet with 9 tokens multiple times till wallet balance is equal to allocationCost
		for i := float64(0); i <= (allocationCost/9)+1; i++ {
			_, err = executeFaucetWithTokens(t, configPath, 9)
			require.Nil(t, err, "Error executing faucet with tokens", err)
		}

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"size":        allocSize,
			"data":        2,
			"parity":      2,
			"lock":        allocationCost + 1,
			"expire":      "5m",
			"read_price":  "0-0.1",
			"write_price": "0-0.1",
		}))
		require.Nil(t, err, "Error creating new allocation", err)

		allocationId, _ := getAllocationID(output[len(output)-1])

		// check total offers new value and compare
		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": minAvailableCapacityBlobber.Id}))
		require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

		err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffersNew := blInfo.TotalOffers
		require.Greater(t, totalOffersNew, totalOffers, "Total Offers should Increase")

		_, err = createWalletForName(t, configPath, newStakeWallet)
		require.Nil(t, err, "Error registering wallet", err)

		_, err = executeFaucetWithTokensForWallet(t, newStakeWallet, configPath, 9)
		require.Nil(t, err, "Error executing faucet with tokens", err)

		_, err = stakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id, "tokens": 1}), true)
		require.Nil(t, err, "Error staking tokens", err)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": minAvailableCapacityBlobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew, lenDelegates+1, "Number of delegates should be greater")

		lenDelegates = lenDelegatesNew

		_, err = unstakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}))
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", minAvailableCapacityBlobber.Id)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": minAvailableCapacityBlobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "Number of delegates should be greater")

		lenDelegates = lenDelegatesNew

		_, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}))
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", minAvailableCapacityBlobber.Id)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": minAvailableCapacityBlobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew, lenDelegates, "delegates should be equal")

		lenDelegates = lenDelegatesNew

		// Cancel the allocation
		_, err = cancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, "error canceling allocation")

		_, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}))
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", minAvailableCapacityBlobber.Id)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": minAvailableCapacityBlobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotNil(t, stakePool, "stake pool info should not be empty")

		lenDelegatesNew = len(stakePool.Delegate)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "Number of delegates should be greater")
	})
}
