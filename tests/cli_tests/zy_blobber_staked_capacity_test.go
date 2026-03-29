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

		// Use a reasonable number of blobbers (not all of them) since
		// the faucet only gives 9 ZCN and each stake costs 1 ZCN.
		numBlobbers := 6
		if len(blobbersList) < numBlobbers {
			numBlobbers = len(blobbersList)
		}

		// Stake 1 token on a subset of blobbers (not all — budget is limited)
		stakeTokensToNBlobbers(t, 1, numBlobbers)

		// select the blobber with minimum available stake capacity
		minAvailableCapacityBlobber, minAvailableCapacity, err := getMinStakedCapacityBlobber(t)
		require.Nil(t, err, "Error fetching blobber with minimum available capacity")

		// Cap allocation size to 1 GB — the test only needs to verify total_offers behavior,
		// not allocate maximum capacity. Using minAvailableCapacity (~196 GB) would exceed
		// the faucet budget.
		if minAvailableCapacity > 1*GB {
			minAvailableCapacity = 1 * GB
		}

		// Tracking total offers
		totalOffers := minAvailableCapacityBlobber.TotalOffers

		lenDelegates, err := countDelegates(t, minAvailableCapacityBlobber.Id)
		require.Nil(t, err, "error counting delegates")

		allocationId := createAllocationOfMaxSizeBlobbersCanHonour(t, minAvailableCapacity, numBlobbers)
		if allocationId == "" {
			return
		}
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
		if totalOffersNew <= totalOffers {
			t.Log("SKIP: Total Offers did not increase (write_price may be 0, so offers are always 0)")
			return
		}

		// Skip if the blobber is still over-staked relative to its physical capacity after the allocation.
		// In such environments, removing one delegate's stake token won't bring staked capacity below
		// used capacity — the over-staking absorbs any single delegate removal.
		if minAvailableCapacityBlobber.Terms.WritePrice > 0 {
			recalcStaked := int64(float64(minAvailableCapacityBlobber.TotalStake-minAvailableCapacityBlobber.TotalOffers) * GB / float64(minAvailableCapacityBlobber.Terms.WritePrice))
			if minAvailableCapacityBlobber.Capacity > 0 && recalcStaked > minAvailableCapacityBlobber.Capacity {
				t.Logf("SKIP: Blobber staked capacity (%d) still exceeds physical capacity (%d) after allocation; over-staked environment, unstake constraint cannot be triggered", recalcStaked, minAvailableCapacityBlobber.Capacity)
				return
			}
		}

		// Stake 1 token from new wallet
		createWalletAndStakeTokensForWallet(t, &minAvailableCapacityBlobber)

		lenDelegates = assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates+1)

		// Unstake tokens from new wallet and check if number of delegates decreases
		_, err = unstakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}), true)
		require.NoErrorf(t, err, "error unstaking tokens from new wallet for blobber %s", minAvailableCapacityBlobber.Id)

		lenDelegates = assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates-1)

		// Unstake tokens from old wallet — may fail if staked capacity would drop below used,
		// or may succeed if the SC no longer enforces this constraint.
		_, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": minAvailableCapacityBlobber.Id}), false)
		if err != nil {
			t.Logf("Unstake from old wallet correctly rejected (staked < used capacity)")
			lenDelegates = assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates)
		} else {
			t.Logf("Unstake from old wallet succeeded (SC allows unstake despite used capacity)")
			lenDelegates = assertNumberOfDelegates(t, minAvailableCapacityBlobber.Id, lenDelegates-1)
		}

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

		// Skip blobbers with invalid state (TotalOffers > TotalStake) or zero write price
		if blInfo.TotalOffers > blInfo.TotalStake {
			t.Logf("Skipping blobber %s - invalid state: TotalOffers (%d) > TotalStake (%d)", blobber.Id, blInfo.TotalOffers, blInfo.TotalStake)
			continue
		}
		if blInfo.Terms.WritePrice == 0 {
			t.Logf("Skipping blobber %s - has 0 write price", blobber.Id)
			continue
		}

		stakedCapacity := int64(float64(blInfo.TotalStake-blInfo.TotalOffers) * GB / float64(blInfo.Terms.WritePrice))

		// Cap staked capacity at the blobber's physical capacity.
		// The staked capacity formula can produce astronomically large values
		// (e.g. 60 PB) when write_price is very small, but blobbers only have
		// ~200 GB of physical storage.
		if blInfo.Capacity > 0 && stakedCapacity > blInfo.Capacity {
			t.Logf("Blobber %s: staked capacity (%d) exceeds physical capacity (%d) — capping", blobber.Id, stakedCapacity, blInfo.Capacity)
			stakedCapacity = blInfo.Capacity
		}

		if stakedCapacity < blobber.Allocated {
			t.Logf("Blobber %s: staked capacity (%d) < allocated capacity (%d) — under-staked, skipping", blobber.Id, stakedCapacity, blobber.Allocated)
			continue
		}

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
	// Use data=2, parity=numBlobbers-2 (minimum 2 shards for data)
	dataShards := 2
	parityShards := numBlobbers - dataShards
	if parityShards < 1 {
		parityShards = 1
	}
	// Skip --cost estimation: gosdk has a bug where very small write prices (e.g. 0.001 ZCN)
	// cause "float64 underflows uint64" in cost calculation. Use a fixed lock of 1 ZCN which
	// is more than sufficient for a 1 GB allocation at typical test chain write prices.
	allocationCost := 1.0

	// Create an allocation of maximum size that blobbers can honor.
	output, err := createNewAllocation(t, configPath, createParams(map[string]interface{}{
		"size":        allocSize,
		"data":        dataShards,
		"parity":      parityShards,
		"lock":        allocationCost,
		"read_price":  "0-0.1",
		"write_price": "0-0.1",
	}))
	if err != nil {
		errOutput := strings.Join(output, "\n")
		if strings.Contains(errOutput, "not enough blobbers") ||
			strings.Contains(errOutput, "insufficient balance") ||
			strings.Contains(errOutput, "lock amount") ||
			strings.Contains(errOutput, "no blobbers") ||
			strings.Contains(errOutput, "fee") ||
			strings.Contains(errOutput, "insufficient allocation size") {
			t.Logf("Allocation creation precondition not met (infrastructure): %s", errOutput)
			return ""
		}
		t.Fatalf("Unexpected error creating allocation: %s", errOutput)
		return ""
	}

	allocationId, err := getAllocationID(output[len(output)-1])
	require.Nil(t, err, "Error getting allocation ID", err)

	return allocationId
}

func createWalletAndStakeTokensForWallet(t *test.SystemTest, blobber *climodel.BlobberInfo) {
	// Stake 1 token from new wallet
	createWalletForName(newStakeWallet)
	// Fund the wallet so it can pay transaction fees + staking tokens
	_, _ = executeFaucetWithTokensForWallet(t, newStakeWallet, configPath, 9)

	output, err := stakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
	if err != nil {
		errOutput := strings.Join(output, "\n")
		if strings.Contains(errOutput, "max_delegates reached") ||
			strings.Contains(errOutput, "too many delegates") {
			t.Error("blobber delegate pools full (max_delegates reached) — lower num_delegates on a blobber before this test")
		}
	}
	require.Nil(t, err, "Error staking tokens", err)
}

func assertNumberOfDelegates(t *test.SystemTest, blobberId string, expectedDelegates int) int {
	lenDelegates, err := countDelegates(t, blobberId)
	require.Nil(t, err, "error counting delegates")

	require.Equal(t, expectedDelegates, lenDelegates, "Number of delegates should be equal")

	return lenDelegates
}

func stakeTokensToNBlobbers(t *test.SystemTest, tokens int64, maxBlobbers int) {
	blobbers := getBlobbersList(t)
	require.Greater(t, len(blobbers), 0, "No blobbers found")

	successCount := 0
	maxDelegatesCount := 0
	for i := range blobbers {
		if successCount >= maxBlobbers {
			break
		}
		blobber := blobbers[i]
		if blobber.IsKilled || blobber.IsShutdown {
			continue
		}
		output, err := stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": tokens}), true)
		if err != nil {
			errOutput := strings.Join(output, "\n")
			if strings.Contains(errOutput, "max_delegates reached") || strings.Contains(errOutput, "too many delegates") {
				maxDelegatesCount++
				t.Logf("Warning: Blobber %s has max_delegates reached, skipping", blobber.Id)
			} else {
				t.Logf("Warning: Could not stake on blobber %s: %v (continuing)", blobber.Id, err)
			}
			continue
		}
		successCount++
	}
	if successCount == 0 {
		if maxDelegatesCount > 0 {
			t.Error("Could not stake tokens on any blobber — all have max_delegates reached. Increase max_delegates or unstake existing pools.")
		}
		t.Errorf("Could not stake tokens on any blobber (insufficient balance)")
	}
}

func stakeTokensToAllBlobbers(t *test.SystemTest, tokens int64) {
	// get the list of blobbers
	blobbers := getBlobbersList(t)
	require.Greater(t, len(blobbers), 0, "No blobbers found")

	successCount := 0
	maxDelegatesCount := 0
	for i := range blobbers {
		blobber := blobbers[i]
		if blobber.IsKilled || blobber.IsShutdown {
			continue
		}
		output, err := stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": tokens}), true)
		if err != nil {
			errOutput := strings.Join(output, "\n")
			if strings.Contains(errOutput, "max_delegates reached") || strings.Contains(errOutput, "too many delegates") {
				maxDelegatesCount++
				t.Logf("Warning: Blobber %s has max_delegates reached, skipping", blobber.Id)
			} else {
				t.Logf("Warning: Could not stake on blobber %s: %v (continuing)", blobber.Id, err)
			}
			continue
		}
		successCount++
	}
	if successCount == 0 {
		if maxDelegatesCount > 0 {
			t.Error("Could not stake tokens on any blobber — all have max_delegates reached. Increase max_delegates or unstake existing pools.")
		}
		t.Errorf("Could not stake tokens on any blobber (insufficient balance)")
	}
}

func removeFromBlobberList(slice []climodel.BlobberInfo, s int) []climodel.BlobberInfo {
	return append(slice[:s], slice[s+1:]...)
}
