package cli_tests

import (
	"encoding/json"
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

var blobbersList []climodel.BlobberInfo

func TestStakePool(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.TestSetup("register wallet and get blobbers", func() {
		createWallet(t)

		// get the list of blobbers
		blobbersList = getBlobbersList(t)
		require.Greater(t, len(blobbersList), 0, "No blobbers found")
	})

	t.RunSequentiallyWithTimeout("Total stake in a blobber can never be less than it's used capacity", 800*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		// stake 10 tokens on all blobbers
		stakeTokensToAllBlobbers(t, 1)

		// select the blobber with minimum available stake capacity
		minAvailableCapacityBlobber, minAvailableCapacity, err := getMinStakedCapacityBlobber(t)
		require.Nil(t, err, "Error fetching blobber with minimum available capacity")

		// Tracking total offers
		totalOffers := minAvailableCapacityBlobber.TotalOffers

		lenDelegates, err := countDelegates(t, minAvailableCapacityBlobber.Id)
		require.Nil(t, err, "error counting delegates")

		// Create an allocation of maximum size that all blobbers can honor.
		// This requires creating an allocation of capacity = available capacity of blobber which has minimum
		// available capacity. For example, if 3 blobbers have 4 GB, 5 GB and 6 GB available,
		// the max allocation they all can honor is of 4 GB.
		allocationId := createAllocationOfMaxSizeBlobbersCanHonour(t, minAvailableCapacity, len(blobbersList))
		t.Cleanup(func() {
			// Cancel the allocation irrespective of test result
			_, _ = cancelAllocation(t, configPath, allocationId, true)
		})

		// check total offers new value and compare
		output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": minAvailableCapacityBlobber.Id}))
		require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

		err = json.Unmarshal([]byte(output[len(output)-1]), &minAvailableCapacityBlobber)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffersNew := minAvailableCapacityBlobber.TotalOffers
		require.Greater(t, totalOffersNew, totalOffers, "Total Offers should Increase")

		// Stake 1 token from new wallet
		createWalletAndStakeTokensForWallet(t, &minAvailableCapacityBlobber)

		lenDelegates = assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates+1)

		// Unstake tokens from new wallet and check if number of delegates decreases
		_, err = unstakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}), true)
		require.NoErrorf(t, err, "error unstaking tokens from new wallet for blobber %s", minAvailableCapacityBlobber.Id)

		lenDelegates = assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates-1)

		// Unstake tokens from old wallet (should return error and number of delegate should not decrease)
		_, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}), false)
		require.Error(t, err, "No error in unstaking tokens from old wallet for blobber %s", minAvailableCapacityBlobber.Id)

		lenDelegates = assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates)

		// Cancel the allocation
		_, err = cancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, "error canceling allocation")

		// Unstake tokens from old wallet (should be successful and number of delegate should decrease)
		_, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}), true)
		require.NoErrorf(t, err, "error unstaking tokens from blobber %s", minAvailableCapacityBlobber.Id)

		assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates-1)
	})
}

func getMinStakedCapacityBlobber(t *test.SystemTest) (climodel.BlobberInfo, int64, error) {
	var minAvailableCapacityBlobber climodel.BlobberInfo
	minAvailableCapacity := int64(math.MaxInt64)

	for i := range blobbersList {
		blobber := blobbersList[i]

		if blobber.IsKilled || blobber.IsShutdown {
			blobbersList = removeFromBlobberList(blobbersList, i)
			continue
		}

		output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))
		require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

		var blInfo climodel.BlobberInfo
		err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		stakedCapacity := int64(float64(blInfo.TotalStake-blInfo.TotalOffers) * GB / float64(blInfo.Terms.WritePrice))

		require.GreaterOrEqual(t, stakedCapacity, blobber.Allocated, "Staked capacity should be greater than allocated capacity")

		if stakedCapacity < minAvailableCapacity {
			minAvailableCapacity = stakedCapacity
			minAvailableCapacityBlobber = blInfo
		}
	}

	output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": minAvailableCapacityBlobber.Id}))
	require.Nil(t, err, "Error fetching blobber info", strings.Join(output, "\n"))

	var blInfo climodel.BlobberInfo
	err = json.Unmarshal([]byte(output[len(output)-1]), &blInfo)
	require.Nil(t, err, "error unmarshalling blobber info")

	return blInfo, minAvailableCapacity, nil
}

func countDelegates(t *test.SystemTest, blobberId string) (int, error) {
	output, err := stakePoolInfo(t, configPath, createParams(map[string]interface{}{
		"blobber_id": blobberId,
		"json":       "",
	}))
	if err != nil {
		return 0, err
	}

	stakePool := climodel.StakePoolInfo{}
	err = json.Unmarshal([]byte(output[len(output)-1]), &stakePool)
	if err != nil {
		return 0, err
	}

	return len(stakePool.Delegate), nil
}

func createAllocationOfMaxSizeBlobbersCanHonour(t *test.SystemTest, minAvailableCapacity int64, numBlobbers int) string {
	allocSize := minAvailableCapacity - 10*MB
	output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
		"cost":        "",
		"data":        1,
		"parity":      numBlobbers - 1,
		"size":        allocSize,
		"read_price":  "0-0.1",
		"write_price": "0-0.1",
	}))
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
	allocationCost, err := getAllocationCost(output[0])
	require.Nil(t, err, "could not get allocation cost")

	// Create an allocation of maximum size that all blobbers can honor.
	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
		"size":        allocSize,
		"data":        1,
		"parity":      numBlobbers - 1,
		"lock":        allocationCost,
		"read_price":  "0-0.1",
		"write_price": "0-0.1",
	}))
	require.Nil(t, err, "Error creating new allocation", err)

	allocationId, err := getAllocationID(output[len(output)-1])
	require.Nil(t, err, "Error getting allocation ID", err)

	return allocationId
}

func createWalletAndStakeTokensForWallet(t *test.SystemTest, blobber *climodel.BlobberInfo) {
	// Stake 1 token from new wallet
	createWalletForName(newStakeWallet)

	_, err := stakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
	require.Nil(t, err, "Error staking tokens", err)
}

func assertNumberOfDelegates(t *test.SystemTest, blobberId string, expectedDelegates int) int {
	lenDelegates, err := countDelegates(t, blobberId)
	require.Nil(t, err, "error counting delegates")

	require.Equal(t, expectedDelegates, lenDelegates, "Number of delegates should be equal")

	return lenDelegates
}

func stakeTokensToAllBlobbers(t *test.SystemTest, tokens int64) {
	// get the list of blobbers
	blobbers := getBlobbersList(t)
	require.Greater(t, len(blobbers), 0, "No blobbers found")

	for i := range blobbers {
		blobber := blobbers[i]
		if blobber.IsKilled || blobber.IsShutdown {
			continue
		}
		_, err := stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": tokens}), true)
		require.Nil(t, err, "Error staking tokens", err)
	}
}

func removeFromBlobberList(slice []climodel.BlobberInfo, s int) []climodel.BlobberInfo {
	return append(slice[:s], slice[s+1:]...)
}
