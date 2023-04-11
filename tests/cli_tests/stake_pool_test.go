package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestStakePool(testSetup *testing.T) {

	t := test.NewSystemTest(testSetup)

	_, err := registerWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", err)

	// get the list of blobbers
	blobbersList := getBlobbersList(t)
	require.Greater(testSetup, len(blobbersList), 0, "No blobbers found")

	t.RunSequentiallyWithTimeout("total stake in a blobber can never be lesser than it's used capacity", 8*time.Minute, func(t *test.SystemTest) {

		// select the blobber with min staked capacity
		var minStakedCapacityBlobber climodel.BlobberInfoDetailed

		maxStakedCapacity := uint64(9999999999999999999999)

		for _, blobber := range blobbersList {
			output, _ := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))

			var blInfo climodel.BlobberInfoDetailed
			err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
			require.Nil(t, err, "error unmarshalling blobber info")

			stakedCapacity := uint64(((float64(blInfo.TotalStake) / 10000000000.0) * (10 * GB)) - float64(blInfo.Allocated))

			if stakedCapacity < maxStakedCapacity {
				maxStakedCapacity = stakedCapacity
				minStakedCapacityBlobber = blInfo
			}

			fmt.Println("Blobber ID : ", blobber.Id, " Staked Capacity : ", stakedCapacity, " Allocated : ", blInfo.Allocated, " Total Stake : ", blInfo.TotalStake)
		}

		// select any random blobber and check total offers
		blobber := minStakedCapacityBlobber
		output, _ := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))

		var blInfo climodel.BlobberInfoDetailed
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
		require.NotEmpty(t, stakePool)

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
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew := len(delegates)

		require.Equal(t, lenDelegatesNew, lenDelegates+1, "delegates should be greater")

		lenDelegates = lenDelegatesNew

		// create an allocation of capacity

		allocSize := maxStakedCapacity*3 + uint64(30*GB)
		allocSize -= allocSize / uint64(1000)

		fmt.Println("Allocation Size : ", allocSize)

		options := map[string]interface{}{"cost": "", "size": allocSize, "data": 3, "parity": 3}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		allocationCost, err := getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		for i := float64(0); i <= (allocationCost/9+2)*2; i++ {
			_, err = executeFaucetWithTokens(t, configPath, 9)
			require.Nil(t, err, "Error executing faucet with tokens", err)
		}

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": (allocationCost + 1) * 2,
			"data":   3,
			"parity": 3,
			"lock":   allocationCost + 1,
		})

		// check total offers new value and compare
		output, _ = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))

		err = json.Unmarshal([]byte(output[0]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffersNew := blInfo.TotalOffers
		require.Greater(t, totalOffersNew, totalOffers, "Total Offers should Increase")

		newStakeWallet := "new_stake_wallet"

		_, err = registerWalletForName(t, configPath, newStakeWallet)
		if err != nil {
			return
		}

		// stake 1 more token to blobbers
		for _, blobber := range blobbersList {
			_, err := executeFaucetWithTokensForWallet(t, newStakeWallet, configPath, 9)
			require.Nil(t, err, "Error executing faucet with tokens", err)

			_, err = stakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
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
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew, lenDelegates+1, "delegates should be greater")

		lenDelegates = lenDelegatesNew

		// Try to unstake tokens from the blobbers
		for _, blobber := range blobbersList {
			_, err := unstakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id}))

			require.Nil(t, err, "error should not be there")
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
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "delegates should be greater")

		lenDelegates = lenDelegatesNew

		// Unstaking tokens from the blobbers
		for _, blobber := range blobbersList {
			output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id}))
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
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew, lenDelegates, "delegates should be equal")

		lenDelegates = lenDelegatesNew

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, "error cancelling allocation")

		// Try to unstake tokens from the blobbers
		for _, blobber := range blobbersList {
			_, err := unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id}))
			require.Nil(t, err, "error should not be there")
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
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "delegates should be greater")
	})
}
