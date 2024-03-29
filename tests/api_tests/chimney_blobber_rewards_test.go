package api_tests

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test1ChimneyBlobberRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Replace blobber in allocation, should work")
	t.Skip()

	const (
		allocSize = 20 * GB
		fileSize  = 10 * GB
		sleepTime = 30 * time.Minute

		standardErrorMargin = 0.05
		extraErrorMargin    = 0.15
	)

	type Reward int
	const (
		BlockRewardMiner Reward = iota
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

		totalBlockRewardPerRound float64
		totalRounds              int64

		expectedBlockReward float64
		actualBlockReward   float64
	)

	sdkWallet := initialisedWallets[walletIdx]
	walletIdx++

	allBlobbers, resp, err := chimneyClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	for _, blobber := range allBlobbers {
		// stake tokens to this blobber
		chimneyClient.CreateStakePool(t, sdkWallet, 3, blobber.ID, client.TxSuccessfulStatus, 20.0)
	}

	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	blobberRequirements.DataShards = 1
	blobberRequirements.ParityShards = 1
	blobberRequirements.Size = allocSize

	allocationBlobbers := chimneyClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)

	lenAvailableBlobbers := len(*allocationBlobbers.Blobbers)

	blobberRequirements.DataShards = int64((lenAvailableBlobbers-1)/2 + 1)
	blobberRequirements.ParityShards = int64(lenAvailableBlobbers) - blobberRequirements.DataShards

	beforeWallet := chimneyClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)

	allocationBlobbers = chimneyClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
	allocationID := chimneyClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 5000, client.TxSuccessfulStatus)

	time.Sleep(1 * time.Minute)

	beforeAlloc := chimneyClient.GetAllocation(t, allocationID, client.HttpOkStatus)

	afterWallet := chimneyClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
	require.InEpsilon(t, beforeWallet.Balance-afterWallet.Balance, beforeAlloc.WritePool, extraErrorMargin, "Write pool is not equal to wallet balance difference")
	beforeWallet = afterWallet

	uploadOp := chimneySdkClient.AddUploadOperationForBigFile(t, allocationID, fileSize/GB) // 10gb
	chimneySdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

	startBlock := chimneyClient.GetLatestFinalizedBlock(t, client.HttpOkStatus)

	allBlobbers, resp, err = chimneyClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	time.Sleep(sleepTime)

	prevAlloc := chimneyClient.GetAllocation(t, allocationID, client.HttpOkStatus)

	chimneyClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

	endBlock := chimneyClient.GetLatestFinalizedBlock(t, client.HttpOkStatus)

	time.Sleep(2 * time.Minute)

	alloc := chimneyClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	require.Equal(t, true, alloc.Canceled, "Allocation should be canceled")
	require.Equal(t, true, alloc.Finalized, "Allocation should be finalized")

	alloc.Blobbers = prevAlloc.Blobbers

	t.RunWithTimeout("Challenge Rewards", 1*time.Hour, func(t *test.SystemTest) {
		allocCreatedAt = alloc.StartTime
		allocExpiredAt = alloc.Expiration
		actualWritePoolBalance = float64(alloc.WritePool)
		actualMovedToChallenge = float64(alloc.MovedToChallenge)

		allocDuration := allocExpiredAt - allocCreatedAt - 300
		prevAllocDuration := prevAlloc.Expiration - prevAlloc.StartTime
		durationInTimeUnits := float64(allocDuration) / float64(prevAllocDuration)
		t.Logf("Alloc duration: %v", durationInTimeUnits)

		// Calculating expected allocation cost
		totalWritePrice := int64(0)
		for _, blobber := range alloc.Blobbers {
			expectedAllocationCost += float64(blobber.Terms.WritePrice) * sizeInGB(int64(allocSize/alloc.DataShards))
			totalWritePrice += blobber.Terms.WritePrice
		}

		// Calculating expected cancellation charge
		expectedCancellationCharge = expectedAllocationCost*0.2 - float64(alloc.MovedToChallenge-alloc.MovedBack)
		expectedWritePoolBalance = 5e13

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
			cancellationChargeQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobber.ID, CancellationChargeReward)

			queryReward := chimneyClient.GetRewardsByQuery(t, cancellationChargeQuery, client.HttpOkStatus)
			actualCancellationChargeForBlobber := queryReward.TotalReward
			totalDelegateReward := queryReward.TotalDelegateReward

			challengesCountQuery := fmt.Sprintf("blobber_id='%s' AND allocation_id='%s'", blobber.ID, allocationID)
			blobberChallengeCount := chimneyClient.GetChallengesCountByQuery(t, challengesCountQuery, client.HttpOkStatus)
			t.Log("Blobber ID: ", blobber.ID)
			t.Log("Blobber Challenge Count: ", blobberChallengeCount)

			// Updating total values
			actualCancellationCharge += actualCancellationChargeForBlobber

			expectedCancellationChargeForBlobber := expectedCancellationCharge * writePriceWeight(blobber.Terms.WritePrice, totalWritePrice)
			expectedCancellationChargeForBlobber *= float64(blobberChallengeCount["passed"]+blobberChallengeCount["open"]) / float64(blobberChallengeCount["total"])

			t.Log("Expected Challenge Reward: ", expectedCancellationChargeForBlobber)
			t.Log("Actual Challenge Reward: ", actualCancellationChargeForBlobber)

			require.InEpsilon(t, expectedCancellationChargeForBlobber, actualCancellationChargeForBlobber, standardErrorMargin, "Expected cancellation charge for blobber is not equal to actual")

			t.Log("Service Charge : ", blobber.StakePoolSettings.ServiceCharge)
			if blobber.StakePoolSettings.ServiceCharge == 0.0 {
				require.Equal(t, float64(0), queryReward.TotalProviderReward, "Total delegate reward should be 0")
				require.Equal(t, queryReward.TotalReward, queryReward.TotalDelegateReward, "Total delegate reward should be equal to total reward")
				continue
			}

			require.InEpsilon(t, queryReward.TotalReward*blobber.StakePoolSettings.ServiceCharge, queryReward.TotalProviderReward, standardErrorMargin, "Expected provider reward is not equal to actual")
			require.InEpsilon(t, queryReward.TotalReward*(1.0-blobber.StakePoolSettings.ServiceCharge), queryReward.TotalDelegateReward, standardErrorMargin, "Expected delegate reward is not equal to actual")

			// Compare Stakepool Rewards
			blobberStakePools, err := sdk.GetStakePoolInfo(sdk.ProviderBlobber, blobber.ID)
			require.NoError(t, err, "Error while getting blobber stake pool info")

			totalStakePoolBalance := float64(blobberStakePools.Balance)

			blobberDelegateRewardsQuery := fmt.Sprintf("allocation_id='%s' AND provider_id='%s' AND reward_type=%d", allocationID, blobber.ID, CancellationChargeReward)
			blobberDelegateRewards := chimneyClient.GetDelegateRewardsByQuery(t, blobberDelegateRewardsQuery, client.HttpOkStatus)

			for _, blobberStakePool := range blobberStakePools.Delegate {
				delegateID := blobberStakePool.DelegateID
				delegateStakePoolBalance := float64(blobberStakePool.Balance)
				delegateReward := float64(blobberDelegateRewards[string(delegateID)])

				require.InEpsilon(t, delegateStakePoolBalance/totalStakePoolBalance, delegateReward/totalDelegateReward, standardErrorMargin, "Expected delegate reward is not in proportion to stake pool balance")
			}
		}

		require.InEpsilon(t, expectedCancellationCharge, actualCancellationCharge, standardErrorMargin, "Expected cancellation charge is not equal to actual")

		// Compare Challenge Rewards
		for _, blobber := range alloc.Blobbers {
			challengeRewardQuery := fmt.Sprintf("allocation_id = '%s' AND provider_id = '%s' AND reward_type = %d", allocationID, blobber.ID, ChallengePassReward)

			queryReward := chimneyClient.GetRewardsByQuery(t, challengeRewardQuery, client.HttpOkStatus)
			actualChallengeRewardForBlobber := queryReward.TotalReward

			challengesCountQuery := fmt.Sprintf("blobber_id='%s' AND allocation_id='%s'", blobber.ID, allocationID)
			blobberChallengeCount := chimneyClient.GetChallengesCountByQuery(t, challengesCountQuery, client.HttpOkStatus)

			t.Log("Blobber ID: ", blobber.ID)
			t.Log("Blobber Challenge Count: ", blobberChallengeCount)

			// Updating total values
			actualBlobberChallengeRewards += actualChallengeRewardForBlobber
			actualChallengeRewards += actualChallengeRewardForBlobber

			expectedChallengeRewardForBlobber := expectedBlobberChallengeRewards * writePriceWeight(blobber.Terms.WritePrice, totalWritePrice)
			expectedChallengeRewardForBlobber *= float64(blobberChallengeCount["passed"]+blobberChallengeCount["open"]) / float64(blobberChallengeCount["total"])

			t.Log("Expected Challenge Reward: ", expectedChallengeRewardForBlobber)
			t.Log("Actual Challenge Reward: ", actualChallengeRewardForBlobber)

			require.InEpsilon(t, expectedChallengeRewardForBlobber, actualChallengeRewardForBlobber, extraErrorMargin, "Expected challenge reward for blobber is not equal to actual")

			t.Log("Service Charge : ", blobber.StakePoolSettings.ServiceCharge)
			if blobber.StakePoolSettings.ServiceCharge == 0.0 {
				require.Equal(t, float64(0), queryReward.TotalProviderReward, "Total delegate reward should be 0")
				require.Equal(t, queryReward.TotalReward, queryReward.TotalDelegateReward, "Total delegate reward should be equal to total reward")
				continue
			}

			require.InEpsilon(t, queryReward.TotalReward*blobber.StakePoolSettings.ServiceCharge, queryReward.TotalProviderReward, standardErrorMargin, "Expected provider reward is not equal to actual")
			require.InEpsilon(t, queryReward.TotalReward*(1.0-blobber.StakePoolSettings.ServiceCharge), queryReward.TotalDelegateReward, standardErrorMargin, "Expected delegate reward is not equal to actual")
		}

		require.InEpsilon(t, expectedBlobberChallengeRewards, actualBlobberChallengeRewards, extraErrorMargin, "Expected challenge rewards is not equal to actual")

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

		afterWallet = chimneyClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		require.InEpsilon(t, afterWallet.Balance-beforeWallet.Balance, expectedWritePoolBalance, extraErrorMargin, "Expected write pool balance is not equal to actual")

		// Compare Write Pool Balance
		require.Equal(t, float64(0), actualWritePoolBalance, "Writepool should be 0")
	})

	t.RunWithTimeout("Block Rewards", 1*time.Hour, func(t *test.SystemTest) {
		totalChallengesPassedInRoundsDiff := float64(0)

		var eligibleBlobbers []*model.SCRestGetBlobberResponse
		for _, blobber := range allBlobbers {
			updatedBlobber := chimneyClient.GetBlobber(t, blobber.ID, client.HttpOkStatus)

			if updatedBlobber.ChallengesPassed > blobber.ChallengesPassed {
				updatedBlobber.ChallengesPassed -= blobber.ChallengesPassed
				totalChallengesPassedInRoundsDiff += float64(updatedBlobber.ChallengesPassed)

				eligibleBlobbers = append(eligibleBlobbers, updatedBlobber)
			}
		}

		startBlockRound := startBlock.Round
		endBlockRound := endBlock.Round
		totalRounds = endBlockRound - startBlockRound

		meanRound := (startBlockRound + endBlockRound) / 2

		calculateTotalBlockRewardPerRound := func() float64 {
			query := fmt.Sprintf("block_number > %d AND block_number <= %d AND reward_type = %d", meanRound, meanRound+29, BlockRewardBlobber)
			queryReward := chimneyClient.GetRewardsByQuery(t, query, client.HttpOkStatus)
			return queryReward.TotalReward
		}

		totalBlockRewardPerRound = calculateTotalBlockRewardPerRound()
		expectedBlockReward = totalBlockRewardPerRound * float64(totalRounds/30)

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

			return (zeta + 1) * float64(blobber.TotalStake) * float64(blobber.ChallengesPassed)
		}

		getTotalWeightOfRandomBlobbersSize := func(size int) float64 {
			totalWeight := float64(0)
			selectedIndexes := make(map[int]bool)
			lenBlobbers := len(eligibleBlobbers)

			for i := 0; i < size; i++ {
				// Generate a random index within the range of available blobbers.
				randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(lenBlobbers)))
				if err != nil {
					// Handle the error if any.
					// For simplicity, you can log the error or return an error value.
					return 0
				}
				index := int(randomIndex.Int64())

				// Check if the index has already been selected. If yes, generate a new random index.
				for selectedIndexes[index] {
					randomIndex, err = rand.Int(rand.Reader, big.NewInt(int64(lenBlobbers)))
					if err != nil {
						// Handle the error if any.
						// For simplicity, you can log the error or return an error value.
						return 0
					}
					index = int(randomIndex.Int64())
				}

				// Mark the current index as selected.
				selectedIndexes[index] = true

				totalWeight += getBlobberBlockRewardWeight(eligibleBlobbers[index])
			}
			return totalWeight
		}

		totalBlobberBlockRewardWeight := float64(0)
		for _, blobber := range eligibleBlobbers {
			totalBlobberBlockRewardWeight += getBlobberBlockRewardWeight(blobber)
		}

		// Collecting partitions size frequency data
		partitionSizeFrequency := chimneyClient.GetPartitionSizeFrequency(t, startBlockRound, endBlockRound, client.HttpOkStatus)
		blobberPartitionSelection := chimneyClient.GetBlobberPartitionSelectionFrequency(t, startBlockRound, endBlockRound, client.HttpOkStatus)

		probabilityOfBlobberSelected := make(map[float64]float64)
		for size, frequency := range partitionSizeFrequency {
			probabilityOfBlobberSelected[size] = customRound(frequency * size)
		}

		// Compare blobber block rewards
		for _, blobber := range eligibleBlobbers {
			blobberBlockRewardWeight := getBlobberBlockRewardWeight(blobber)
			expectedBlobberBlockReward := 0.0

			maxSize := 0.0
			count := 0

			for size, probability := range probabilityOfBlobberSelected {
				probability *= float64(blobber.ChallengesPassed) / totalChallengesPassedInRoundsDiff
				count += int(probability)
				if size > maxSize {
					maxSize = size
				}
				if size == 1 {
					expectedBlobberBlockReward += totalBlockRewardPerRound * probability
				} else {
					weightRatio := blobberBlockRewardWeight / (blobberBlockRewardWeight + getTotalWeightOfRandomBlobbersSize(int(size)-1))
					expectedBlobberBlockReward += weightRatio * totalBlockRewardPerRound * probability
				}
			}

			weightRatio := blobberBlockRewardWeight / (blobberBlockRewardWeight + getTotalWeightOfRandomBlobbersSize(int(maxSize)-1))
			expectedBlobberBlockReward += weightRatio * totalBlockRewardPerRound * float64(blobberPartitionSelection[blobber.ID]-int64(count))

			// Calculate actual block reward
			blockRewardQuery := fmt.Sprintf("provider_id = '%s' AND reward_type = %d AND block_number >= %d AND block_number < %d", blobber.ID, BlockRewardBlobber, startBlockRound, endBlockRound)
			actualBlockRewardForBlobber := chimneyClient.GetRewardsByQuery(t, blockRewardQuery, client.HttpOkStatus)
			actualBlockReward += actualBlockRewardForBlobber.TotalReward

			t.Log("Blobber ID: ", blobber.ID)
			t.Log("Expected Block Reward: ", expectedBlobberBlockReward)
			t.Log("Actual Block Reward: ", actualBlockRewardForBlobber.TotalReward)

			require.InEpsilon(t, expectedBlobberBlockReward, actualBlockRewardForBlobber.TotalReward, extraErrorMargin, "Expected block reward for blobber is not equal to actual")

			t.Log("Service Charge : ", blobber.StakePoolSettings.ServiceCharge)
			if blobber.StakePoolSettings.ServiceCharge == 0.0 {
				require.Equal(t, float64(0), actualBlockRewardForBlobber.TotalProviderReward, "Total delegate reward should be 0")
				require.Equal(t, actualBlockRewardForBlobber.TotalReward, actualBlockRewardForBlobber.TotalDelegateReward, "Total delegate reward should be equal to total reward")
				continue
			}

			require.InEpsilon(t, actualBlockRewardForBlobber.TotalReward*blobber.StakePoolSettings.ServiceCharge, actualBlockRewardForBlobber.TotalProviderReward, standardErrorMargin, "Expected provider reward is not equal to actual")
			require.InEpsilon(t, actualBlockRewardForBlobber.TotalReward*(1.0-blobber.StakePoolSettings.ServiceCharge), actualBlockRewardForBlobber.TotalDelegateReward, standardErrorMargin, "Expected delegate reward is not equal to actual")
		}

		require.InEpsilon(t, expectedBlockReward, actualBlockReward, extraErrorMargin, "Expected block reward is not equal to actual")

		// Check Blobber Partitions are selected evenly
		for blobberId, frequncy := range blobberPartitionSelection {
			require.Greater(t, frequncy, totalRounds/90, "Blobber %s is not selected for partitions evenly", blobberId)
		}
	})
}

// size in gigabytes
func sizeInGB(size int64) float64 {
	return float64(size) / float64(1024*1024*1024)
}

func writePriceWeight(writePrice, totalWritePrice int64) float64 {
	return float64(writePrice) / float64(totalWritePrice)
}

func customRound(number float64) float64 {
	integerPart := math.Floor(number)
	decimalPart := number - integerPart

	if decimalPart >= 0.65 {
		return math.Ceil(number)
	} else {
		return math.Floor(number)
	}
}
