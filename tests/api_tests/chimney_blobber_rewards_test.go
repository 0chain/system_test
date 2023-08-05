package api_tests

import (
	"fmt"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestChimneyBlobberRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Replace blobber in allocation, should work")

	startBlock := chimneyClient.GetLatestFinalizedBlock(t, client.HttpOkStatus)

	const (
		allocSize = 1073741824
		fileSize  = 1024 * 1024 * 5
		sleepTime = 0 * time.Minute

		standardErrorMargin = 0.05
		extraErrorMargin    = 0.1
	)

	type Reward int
	const (
		MinLockDemandReward Reward = iota
		BlockRewardMiner
		BlockRewardSharder
		BlockRewardBlobber
		FeeRewardMiner
		FeeRewardAuthorizer
		FeeRewardSharder
		ValidationReward
		FileDownloadReward
		ChallengePassReward
		ChallengeSlashPenalty
		CancellationChargeReward
		NumOfRewards
	)

	var (
		expectedAllocationCost            float64
		expectedCancellationCharge        float64
		expectedWritePoolBalance          float64
		expectedMovedToChallenge          float64
		expectedChallengeRewards          float64
		expectedBlobberChallengeRewards   float64
		expectedValidatorChallengeRewards float64

		allocCreatedAt                  int64
		allocExpiredAt                  int64
		actualCancellationCharge        float64
		actualWritePoolBalance          float64
		actualMovedToChallenge          float64
		actualChallengeRewards          float64
		actualBlobberChallengeRewards   float64
		actualValidatorChallengeRewards float64
		actualMinLockDemandReward       float64

		totalBlockRewardPerRound float64
		totalRounds              int64

		expectedBlockReward float64
		actualBlockReward   float64
	)

	chimneyClient.ExecuteFaucetWithTokens(t, sdkWallet, 9000, client.TxSuccessfulStatus)

	allBlobbers, resp, err := chimneyClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	for _, blobber := range allBlobbers {
		// stake tokens to this blobber
		chimneyClient.CreateStakePool(t, sdkWallet, 3, blobber.ID, client.TxSuccessfulStatus)
	}

	allBlobbers, resp, err = chimneyClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	t.Cleanup(func() {
		for _, blobber := range allBlobbers {
			// unstake tokens from this blobber
			chimneyClient.UnlockStakePool(t, sdkWallet, 3, blobber.ID, client.TxSuccessfulStatus)
		}
	})

	lenBlobbers := int64(len(allBlobbers))

	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	blobberRequirements.DataShards = (lenBlobbers + 1) / 2
	blobberRequirements.ParityShards = lenBlobbers / 2
	blobberRequirements.Size = allocSize

	allocationBlobbers := chimneyClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
	allocationID := chimneyClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 5000, client.TxSuccessfulStatus)

	uploadOp := sdkClient.AddUploadOperation(t, allocationID, fileSize)
	sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

	time.Sleep(sleepTime)

	prevAlloc := chimneyClient.GetAllocation(t, allocationID, client.HttpOkStatus)

	chimneyClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

	alloc := chimneyClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	require.Equal(t, true, alloc.Canceled, "Allocation should be cancelled")
	require.Equal(t, true, alloc.Finalized, "Allocation should be finalized")

	alloc.Blobbers = prevAlloc.Blobbers

	endBlock := chimneyClient.GetLatestFinalizedBlock(t, client.HttpOkStatus)

	t.RunWithTimeout("Challenge Rewards", 1*time.Hour, func(t *test.SystemTest) {
		allocCreatedAt = alloc.StartTime
		allocExpiredAt = alloc.Expiration
		actualWritePoolBalance = float64(alloc.WritePool)
		actualMovedToChallenge = float64(alloc.MovedToChallenge)

		allocDuration := allocExpiredAt - allocCreatedAt
		durationInTimeUnits := float64(allocDuration*1e9) / float64(alloc.TimeUnit)
		t.Logf("Alloc duration: %v", durationInTimeUnits)

		// Calculating expected allocation cost
		totalWritePrice := int64(0)
		for _, blobber := range alloc.Blobbers {
			expectedAllocationCost += float64(blobber.Terms.WritePrice) * sizeInGB(int64(allocSize/alloc.DataShards))
			totalWritePrice += blobber.Terms.WritePrice
		}

		// Calculating expected cancellation charge
		expectedCancellationCharge = expectedAllocationCost * 0.2
		expectedWritePoolBalance = 5e13

		//expectedMovedToChallenge = (expectedAllocationCost * fileSize) / allocSize
		for _, blobber := range alloc.Blobbers {
			expectedMovedToChallenge += float64(blobber.Terms.WritePrice) * sizeInGB(int64(fileSize/alloc.DataShards))
		}

		// Assert moved to challenge
		require.InEpsilon(t, expectedMovedToChallenge, actualMovedToChallenge, standardErrorMargin, "Expected moved to challenge is not equal to actual")

		// Reduce expected write pool
		expectedWritePoolBalance -= actualMovedToChallenge

		// Calculating expected challenge rewards
		expectedChallengeRewards = actualMovedToChallenge * durationInTimeUnits
		expectedBlobberChallengeRewards = expectedChallengeRewards * 0.975
		expectedValidatorChallengeRewards = expectedChallengeRewards - expectedBlobberChallengeRewards

		// Compare Cancellation Charges
		for _, blobber := range alloc.Blobbers {
			cancellationChargeQuery := fmt.Sprintf("allocation_id = '%s' AND provider_id = '%s' AND reward_type = %d", allocationID, blobber.ID, CancellationChargeReward)

			queryReward := chimneyClient.GetRewardsByQuery(t, cancellationChargeQuery, client.HttpOkStatus)
			actualCancellationChargeForBlobber := queryReward.TotalReward

			// Updating total values
			actualCancellationCharge += actualCancellationChargeForBlobber

			expectedCancellationChargeForBlobber := expectedCancellationCharge * writePriceWeight(blobber.Terms.WritePrice, totalWritePrice)

			require.InEpsilon(t, expectedCancellationChargeForBlobber, actualCancellationChargeForBlobber, standardErrorMargin, "Expected cancellation charge for blobber is not equal to actual")
			require.InEpsilon(t, queryReward.TotalReward*blobber.StakePoolSettings.ServiceCharge, queryReward.TotalProviderReward, standardErrorMargin, "Expected provider reward is not equal to actual")
			require.InEpsilon(t, queryReward.TotalReward*(1.0-blobber.StakePoolSettings.ServiceCharge), queryReward.TotalDelegateReward, standardErrorMargin, "Expected delegate reward is not equal to actual")
		}

		require.InEpsilon(t, expectedCancellationCharge, actualCancellationCharge, standardErrorMargin, "Expected cancellation charge is not equal to actual")

		// Compare Challenge Rewards
		for _, blobber := range alloc.Blobbers {
			challengeRewardQuery := fmt.Sprintf("allocation_id = '%s' AND provider_id = '%s' AND reward_type = %d", allocationID, blobber.ID, ChallengePassReward)

			queryReward := chimneyClient.GetRewardsByQuery(t, challengeRewardQuery, client.HttpOkStatus)
			actualChallengeRewardForBlobber := queryReward.TotalReward

			// Updating total values
			actualBlobberChallengeRewards += actualChallengeRewardForBlobber
			actualChallengeRewards += actualChallengeRewardForBlobber

			expectedChallengeRewardForBlobber := expectedChallengeRewards * writePriceWeight(blobber.Terms.WritePrice, totalWritePrice)

			require.InEpsilon(t, expectedChallengeRewardForBlobber, actualChallengeRewardForBlobber, standardErrorMargin, "Expected challenge reward for blobber is not equal to actual")
			require.InEpsilon(t, queryReward.TotalReward*blobber.StakePoolSettings.ServiceCharge, queryReward.TotalProviderReward, standardErrorMargin, "Expected provider reward is not equal to actual")
			require.InEpsilon(t, queryReward.TotalReward*(1.0-blobber.StakePoolSettings.ServiceCharge), queryReward.TotalDelegateReward, standardErrorMargin, "Expected delegate reward is not equal to actual")
		}

		require.InEpsilon(t, expectedBlobberChallengeRewards, actualBlobberChallengeRewards, standardErrorMargin, "Expected challenge rewards is not equal to actual")

		// Compare Validator Challenge Rewards
		validatorChallengeRewardQuery := fmt.Sprintf("allocation_id = '%s' AND reward_type = %d", allocationID, ValidationReward)

		queryValidatorReward := chimneyClient.GetRewardsByQuery(t, validatorChallengeRewardQuery, client.HttpOkStatus)
		actualValidatorChallengeRewards = queryValidatorReward.TotalReward
		actualChallengeRewards += actualValidatorChallengeRewards

		require.InEpsilon(t, expectedValidatorChallengeRewards, actualValidatorChallengeRewards, standardErrorMargin, "Expected validator challenge rewards is not equal to actual")

		// Compare Total Challenge Rewards
		require.InEpsilon(t, expectedChallengeRewards, actualChallengeRewards, standardErrorMargin, "Expected total challenge rewards is not equal to actual")
		require.InEpsilon(t, actualMovedToChallenge-actualChallengeRewards, float64(alloc.MovedBack), standardErrorMargin, "Tokens are not moved back properly")

		// Reduce expected write pool
		expectedWritePoolBalance -= actualChallengeRewards

		// Total Min Lock Demand Reward
		minLockDemandQuery := fmt.Sprintf("allocation_id = '%s' AND reward_type = %d", allocationID, MinLockDemandReward)
		queryMinLockDemand := chimneyClient.GetRewardsByQuery(t, minLockDemandQuery, client.HttpOkStatus)
		actualMinLockDemandReward = queryMinLockDemand.TotalReward

		// Reduce expected write pool
		expectedWritePoolBalance -= actualMinLockDemandReward

		// Compare Write Pool Balance
		require.InEpsilon(t, expectedWritePoolBalance, actualWritePoolBalance, standardErrorMargin, "Expected write pool balance is not equal to actual")
	})

	t.RunWithTimeout("Block Rewards", 1*time.Hour, func(t *test.SystemTest) {
		startBlockRound := startBlock.Round
		endBlockRound := endBlock.Round
		totalRounds = endBlockRound - startBlockRound

		calculateTotalBlockRewardPerRound := func() float64 {
			query := fmt.Sprintf("block_number = %d AND reward_type = %d", startBlockRound, BlockRewardBlobber)
			queryReward := chimneyClient.GetRewardsByQuery(t, query, client.HttpOkStatus)
			return queryReward.TotalReward
		}

		totalBlockRewardPerRound = calculateTotalBlockRewardPerRound()
		expectedBlockReward = totalBlockRewardPerRound * float64(totalRounds)

		getZeta := func(wp, rp float64) float64 {

			i := 1.0
			k := 0.9
			mu := 0.2

			if wp == 0 {
				return 0
			}

			return i - (k * (rp / (rp + (mu * wp))))
		}

		getBlobberBlockRewardWeight := func(blobber *model.SCRestGetBlobberResponse) float64 {
			zeta := getZeta(float64(blobber.Terms.WritePrice), float64(blobber.Terms.ReadPrice))

			return (zeta + 1) * float64(blobber.TotalStake)
		}

		totalBlobberBlockRewardWeight := float64(0)
		for _, blobber := range allBlobbers {
			totalBlobberBlockRewardWeight += getBlobberBlockRewardWeight(blobber)
		}

		// Collecting partitions size frequency data
		partitionSizeFrequency := chimneyClient.GetPartitionSizeFrequency(t, startBlockRound, endBlockRound, client.HttpOkStatus)

		probabilityOfBlobberSelected := make(map[float64]float64)
		for size, frequency := range partitionSizeFrequency {
			probabilityOfBlobberSelected[size] = (frequency * size) / float64(lenBlobbers)
		}

		// Compare blobber block rewards
		for _, blobber := range allBlobbers {
			blobberBlockRewardWeight := getBlobberBlockRewardWeight(blobber)
			expectedBlobberBlockReward := 0.0

			for size, probability := range probabilityOfBlobberSelected {
				if size == 1 {
					expectedBlobberBlockReward += totalBlockRewardPerRound * probability
				} else {
					expectedBlobberBlockReward += ((blobberBlockRewardWeight * float64(lenBlobbers-1)) / (totalBlobberBlockRewardWeight * (size - 1))) * totalBlockRewardPerRound * probability
				}
			}

			// Calculate actual block reward
			blockRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d AND block_number >= %d AND block_number < %d", blobber.ID, BlockRewardBlobber, startBlockRound, endBlockRound)
			actualBlockRewardForBlobber := chimneyClient.GetRewardsByQuery(t, blockRewardQuery, client.HttpOkStatus)
			actualBlockReward += actualBlockRewardForBlobber.TotalReward

			require.InEpsilon(t, expectedBlobberBlockReward, actualBlockRewardForBlobber.TotalReward, extraErrorMargin, "Expected block reward for blobber is not equal to actual")
			require.InEpsilon(t, actualBlockRewardForBlobber.TotalReward*blobber.StakePoolSettings.ServiceCharge, actualBlockRewardForBlobber.TotalProviderReward, standardErrorMargin, "Expected provider reward is not equal to actual")
			require.InEpsilon(t, actualBlockRewardForBlobber.TotalReward*(1.0-blobber.StakePoolSettings.ServiceCharge), actualBlockRewardForBlobber.TotalDelegateReward, standardErrorMargin, "Expected delegate reward is not equal to actual")
		}

		require.InEpsilon(t, expectedBlockReward, actualBlockReward, standardErrorMargin, "Expected block reward is not equal to actual")
	})
}

// size in gigabytes
func sizeInGB(size int64) float64 {
	return float64(size) / float64(1024*1024*1024)
}

func writePriceWeight(writePrice int64, totalWritePrice int64) float64 {
	return float64(writePrice) / float64(totalWritePrice)
}
