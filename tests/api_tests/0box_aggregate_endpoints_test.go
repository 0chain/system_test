package api_tests

import (
	"log"
	"math"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"

	"github.com/0chain/system_test/internal/api/util/tokenomics"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

//nolint:gocyclo
func Test0boxGraphAndTotalEndpoints(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	// Faucet the used wallets

	ownerBalance := apiClient.GetWalletBalance(t, ownerWallet, client.HttpOkStatus)
	t.Logf("ZboxOwner balance: %v", ownerBalance)
	blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
	t.Logf("Blobber owner balance: %v", blobberOwnerBalance)
	ownerWallet.Nonce = int(ownerBalance.Nonce)
	blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)

	testWallet := initialisedWallets[walletIdx]
	walletIdx++
	balance := apiClient.GetWalletBalance(t, testWallet, client.HttpOkStatus)
	testWallet.Nonce = int(balance.Nonce)

	// Stake 6 blobbers, each with 1 token
	targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 6, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
	require.Len(t, targetBlobbers, 6)
	for _, blobber := range targetBlobbers {
		confHash := apiClient.CreateStakePool(t, testWallet, 3, blobber.ID, client.TxSuccessfulStatus, 10.0) // 3zcn from sdkwallet
		require.NotEmpty(t, confHash)
	}

	// Create the free allocation marker (ownerWallet -> sdkWallet)
	apiClient.AddFreeStorageAssigner(t, ownerWallet, client.TxSuccessfulStatus) // 0.1 ZCN 1 ZCN = 1e10 from owner wallet
	marker := config.CreateFreeStorageMarker(t, testWallet.ToSdkWallet(testWallet.Mnemonics), ownerWallet.ToSdkWallet(ownerWalletMnemonics))
	t.Logf("Free allocation marker: %v", marker)

	t.RunSequentiallyWithTimeout("test multi allocation overall graph data", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		// Get initial total challenge pools
		PrintBalance(t, ownerWallet, blobberOwnerWallet, wallet)
		data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		allocatedStorage := (*data)[0]

		data, resp, err = zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		totalChallengePools := (*data)[0]

		data, resp, err = zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		usedStorage := (*data)[0]

		challengeData, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(challengeData.TotalChallenges)))
		require.Equal(t, 1, len([]int64(challengeData.SuccessfulChallenges)))
		totalChallenges, _ := challengeData.TotalChallenges[0], challengeData.SuccessfulChallenges[0]

		type GraphAllocTestCase struct {
			DataShards               int64
			ParityShards             int64
			ExpectedAllocatedStorage int64
			Fsize                    int64
			UpdatedFsize             int64
		}

		graphAllocTestCases := []GraphAllocTestCase{
			{DataShards: 4, ParityShards: 2, ExpectedAllocatedStorage: 13107200 * 6 / 4, Fsize: 65536 * 6},
			{DataShards: 2, ParityShards: 1, ExpectedAllocatedStorage: 13107200 * 3 / 2, Fsize: 65536 * 3},
			{DataShards: 2, ParityShards: 2, ExpectedAllocatedStorage: 13107200 * 4 / 2, Fsize: 65536 * 4},
			{DataShards: 3, ParityShards: 1, ExpectedAllocatedStorage: 13107200 * 4 / 3, Fsize: 65536 * 4},
			{DataShards: 3, ParityShards: 2, ExpectedAllocatedStorage: 13107200 * 5 / 3, Fsize: 65536 * 5},
		}

		const tolerance = int64(10)

		var allocationIds []string

		for i, testCase := range graphAllocTestCases {
			log.Println("Test case: ", i)
			blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
			blobberRequirements.DataShards = testCase.DataShards
			blobberRequirements.ParityShards = testCase.ParityShards
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
			allocationIds = append(allocationIds, allocationID)
			fpath, fsize := sdkClient.UploadFile(t, allocationID, 65536*testCase.DataShards)

			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				allocatedStorageAfterAllocation := (*data)[0]
				latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				expectedTotalAllocationStorage := allocatedStorage + testCase.ExpectedAllocatedStorage
				t.Logf("Allocated storage check: graph=%d, total=%d, expected=%d, initial=%d",
					allocatedStorageAfterAllocation, int64(*latest), expectedTotalAllocationStorage, allocatedStorage)
				// Check if graph data matches expected (within tolerance)
				cond1 := math.Abs(float64(allocatedStorageAfterAllocation-expectedTotalAllocationStorage)) < float64(tolerance)
				// Check if graph and total are close (allow some difference for async updates)
				cond2 := math.Abs(float64(allocatedStorageAfterAllocation-int64(*latest))) < float64(tolerance*100) // Allow larger tolerance for sync
				cond := cond1 && cond2
				if cond {
					allocatedStorage = allocatedStorageAfterAllocation
					t.Logf("Allocated storage updated successfully: %d", allocatedStorage)
				}
				return cond
			})

			// Add blobber to the allocation
			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
			require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
			apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, "", allocationID, client.TxSuccessfulStatus)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				allocatedStorageAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := allocatedStorageAfter > allocatedStorage && allocatedStorageAfter == int64(*latest)
				if cond {
					allocatedStorage = allocatedStorageAfter
				}
				return cond
			})

			var totalChallengePoolsAfterAllocation int64
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				// Get total challenge pools
				data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalChallengePoolsAfterAllocation = (*data)[0]
				cond := totalChallengePoolsAfterAllocation > totalChallengePools
				if cond {
					totalChallengePools = totalChallengePoolsAfterAllocation
				}

				return cond
			})

			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorageAfter-usedStorage >= testCase.Fsize-tolerance) &&
					(usedStorageAfter-usedStorage <= testCase.Fsize+tolerance)

				if cond {
					usedStorage = usedStorageAfter
				}
				return cond
			})

			// Update with a bigger file
			fpath, newFsize := sdkClient.UpdateFileWithParams(t, allocationID, fsize*2, fpath)
			t.Logf("Filename after update bigger : %v", fpath)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorageAfter-usedStorage >= testCase.Fsize-tolerance) &&
					(usedStorageAfter-usedStorage <= testCase.Fsize+tolerance)

				if cond {
					usedStorage = usedStorageAfter
					fsize = newFsize
				}
				return cond
			})

			// Update with a smaller file
			fpath, newFsize = sdkClient.UpdateFileWithParams(t, allocationID, newFsize/2, fpath)
			t.Logf("Filename after update smaller : %v", fpath)

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorage-usedStorageAfter >= testCase.Fsize-tolerance) &&
					(usedStorage-usedStorageAfter <= testCase.Fsize+tolerance)

				if cond {
					usedStorage = usedStorageAfter
					fsize = newFsize
				}

				return cond
			})

			// Remove a file
			sdkClient.DeleteFile(t, allocationID, fpath)

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorage-usedStorageAfter >= testCase.Fsize-tolerance) &&
					(usedStorage-usedStorageAfter <= testCase.Fsize+tolerance)

				if cond {
					usedStorage = usedStorageAfter
				}
				return cond
			})

			// Upload another file
			_, fsize = sdkClient.UploadFile(t, allocationID, 65536*testCase.DataShards)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorageAfter-usedStorage >= testCase.Fsize-tolerance) &&
					(usedStorageAfter-usedStorage <= testCase.Fsize+tolerance)

				if cond {
					usedStorage = usedStorageAfter
				}
				return cond
			})

			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(data.TotalChallenges)))
				totalChallengesAfter := data.TotalChallenges[0]
				successfulChallengesAfter := data.SuccessfulChallenges[0]
				latestTotal, resp, err := zboxClient.GetTotalChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				latestSuccessful, resp, err := zboxClient.GetSuccessfulChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalChallengesAfter > totalChallenges && int64(*latestTotal) == totalChallengesAfter && int64(*latestSuccessful) == successfulChallengesAfter
				if cond {
					totalChallenges = totalChallengesAfter
				}

				return cond
			})
		}

		totalChallengesForAllocations := 0

		for i, allocationID := range allocationIds {
			testCase := graphAllocTestCases[i]
			apiClient.CancelAllocation(t, wallet, allocationID, client.TxSuccessfulStatus)

			// Check decreased + consistency
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				allocatedStorageAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := (allocatedStorageAfter == allocatedStorage) && (allocatedStorageAfter == int64(*latest)) //nolint
				allocatedStorage = allocatedStorageAfter

				// get all blobbers
				allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				expectedAllocatedStorage := calculateExpectedAllocated(allBlobbers)
				cond = cond && (allocatedStorageAfter == expectedAllocatedStorage)

				return cond
			})

			var totalChallengePoolsAfterAllocation int64
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				// Get total challenge pools
				data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalChallengePoolsAfterAllocation = (*data)[0]
				cond := totalChallengePoolsAfterAllocation < totalChallengePools
				if cond {
					totalChallengePools = totalChallengePoolsAfterAllocation
				}

				return cond
			})

			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorage-usedStorageAfter >= testCase.Fsize-tolerance) &&
					(usedStorage-usedStorageAfter <= testCase.Fsize+tolerance)

				if cond {
					usedStorage = usedStorageAfter
				}
				return cond
			})

			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			totalChallengesForAllocations += int(allocation.Stats.TotalChallenges)
		}

		challengeData, resp, err = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(challengeData.TotalChallenges)))
		totalGraphChallenges := challengeData.TotalChallenges[0]

		require.InEpsilon(t, totalChallengesForAllocations, totalGraphChallenges, 0.1, "Total challenges for allocations and total challenges from graph should be equal")
	})

	t.RunSequentially("test graph data ( test /v2/graph-total-challenge-pools )", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		PrintBalance(t, ownerWallet, blobberOwnerWallet, wallet)
		// Get initial total challenge pools
		data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		totalChallengePools := (*data)[0]

		// Create a new allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		// Upload a file
		sdkClient.UploadFile(t, allocationID, 65536)

		var totalChallengePoolsAfterAllocation int64
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			// Get total challenge pools
			data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			totalChallengePoolsAfterAllocation = (*data)[0]
			return totalChallengePoolsAfterAllocation > totalChallengePools
		})

		// Cancel the second allocation
		apiClient.CancelAllocation(t, wallet, allocationID, client.TxSuccessfulStatus)

		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			// Get total challenge pools
			data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			totalChallengePoolsAfterCancel := (*data)[0]
			return totalChallengePoolsAfterCancel < totalChallengePoolsAfterAllocation
		})
	})

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-allocated-storage )", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		// Get initial total challenge pools
		PrintBalance(t, ownerWallet, blobberOwnerWallet, wallet)
		data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		allocatedStorage := (*data)[0]

		// Create a new allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		// Check increased
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			allocatedStorageAfterAllocation := (*data)[0]
			latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			cond := allocatedStorageAfterAllocation > allocatedStorage && allocatedStorageAfterAllocation == int64(*latest)
			if cond {
				allocatedStorage = allocatedStorageAfterAllocation
			}
			return cond
		})

		// Add blobber to the allocation
		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, "", allocationID, client.TxSuccessfulStatus)

		// Check increased
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			allocatedStorageAfter := (*data)[0]
			latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			cond := allocatedStorageAfter > allocatedStorage && allocatedStorageAfter == int64(*latest)
			if cond {
				allocatedStorage = allocatedStorageAfter
			}
			return cond
		})

		// Cancel allocation
		apiClient.CancelAllocation(t, wallet, allocationID, client.TxSuccessfulStatus)

		// Check decreased + consistency
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			allocatedStorageAfter := (*data)[0]
			latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			cond := (allocatedStorageAfter == allocatedStorage) && (allocatedStorageAfter == int64(*latest)) //nolint
			allocatedStorage = allocatedStorageAfter

			// get all blobbers
			allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			expectedAllocatedStorage := calculateExpectedAllocated(allBlobbers)
			cond = cond && (allocatedStorageAfter == expectedAllocatedStorage)

			return cond
		})
	})

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-used-storage )", 30*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		PrintBalance(t, ownerWallet, blobberOwnerWallet, wallet)
		// Get initial used storage
		data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		usedStorage := (*data)[0]

		// Create a new allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		fpath, fsize := sdkClient.UploadFile(t, allocationID, 65536)

		// Check increased
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			usedStorageAfter := (*data)[0]
			cond := (usedStorageAfter - usedStorage) == fsize*2

			if cond {
				usedStorage = usedStorageAfter
			}
			return cond
		})

		// Update with a bigger file
		fpath, newFsize := sdkClient.UpdateFileWithParams(t, allocationID, fsize*2, fpath)
		t.Logf("Filename after update bigger : %v", fpath)

		// Check increased
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			usedStorageAfter := (*data)[0]
			cond := (usedStorageAfter - usedStorage) == (newFsize-fsize)*2

			if cond {
				usedStorage = usedStorageAfter
				fsize = newFsize
			}
			return cond
		})

		// Update with a smaller file
		fpath, newFsize = sdkClient.UpdateFileWithParams(t, allocationID, newFsize/2, fpath)
		t.Logf("Filename after update smaller : %v", fpath)

		// Check decreased
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			usedStorageAfter := (*data)[0]
			cond := (usedStorage - usedStorageAfter) == (fsize-newFsize)*2

			if cond {
				usedStorage = usedStorageAfter
				fsize = newFsize
			}

			return cond
		})

		// Remove a file
		sdkClient.DeleteFile(t, allocationID, fpath)

		// Check decreased
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			usedStorageAfter := (*data)[0]
			cond := (usedStorage - usedStorageAfter) == fsize*2
			if cond {
				usedStorage = usedStorageAfter
			}
			return cond
		})

		// Upload another file
		_, fsize = sdkClient.UploadFile(t, allocationID, 65536)

		// Check increased
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			usedStorageAfter := (*data)[0]
			cond := (usedStorageAfter - usedStorage) == fsize*2
			if cond {
				usedStorage = usedStorageAfter
			}
			return cond
		})

		// Cancel the allocation
		apiClient.CancelAllocation(t, wallet, allocationID, client.TxSuccessfulStatus)

		// Check decreased + consistency
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			// Get total challenge pools
			data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			usedStorageAfter := (*data)[0]
			cond := usedStorage == usedStorageAfter
			usedStorage = usedStorageAfter

			// get all blobbers
			allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())

			expectedSavedData := calculateExpectedSavedData(allBlobbers)
			cond = cond && usedStorageAfter == expectedSavedData

			return cond
		})
	})

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-write-price )", 5*time.Minute, func(t *test.SystemTest) {
		data, resp, err := zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))
		priceBeforeStaking := (*data)[0]

		targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 2, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, targetBlobbers, 2)

		targetBlobbers[0].Capacity += 10 * 1024 * 1024 * 1024
		targetBlobbers[1].Capacity -= 10 * 1024 * 1024 * 1024

		targetBlobbers[0].Terms.WritePrice += *tokenomics.IntToZCN(0.1)
		targetBlobbers[1].Terms.WritePrice += *tokenomics.IntToZCN(0.1)

		t.Log("Blobber : ", blobberOwnerWallet.Id)

		// Ensure blobberOwnerWallet has sufficient balance and updated nonce
		blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transactions (0.1 ZCN value + fees)")
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)

		// Update nonce before second update
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transactions (0.1 ZCN value + fees)")
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)

		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			// get all blobbers
			allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			printBlobbers(t, "After Update", allBlobbers)

			expectedAWP := calculateExpectedAvgWritePrice(allBlobbers)
			roundingError := int64(1000)

			data, resp, err := zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			priceAfterStaking := (*data)[0]

			latest, resp, err := zboxClient.GetAverageWritePrice(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())

			diff := priceAfterStaking - expectedAWP
			latestDiff := priceAfterStaking - int64(*latest)
			t.Logf("priceBeforeStaking: %d, priceAfterStaking: %d, expectedAWP: %d, diff: %d, latest: %d, latestDiff: %d", priceBeforeStaking, priceAfterStaking, expectedAWP, diff, int64(*latest), latestDiff)
			// Allow tolerance for both expectedAWP and latest value due to timing differences in graph updates
			return priceAfterStaking != priceBeforeStaking && diff >= -roundingError && diff <= roundingError && latestDiff >= -roundingError && latestDiff <= roundingError
		})

		// Cleanup: Revert write price to 0.1
		targetBlobbers[0].Terms.WritePrice = *tokenomics.IntToZCN(0.1)
		targetBlobbers[1].Terms.WritePrice = *tokenomics.IntToZCN(0.1)
		// Update nonce before first cleanup update
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transactions (0.1 ZCN value + fees)")
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)
		// Update nonce before second cleanup update
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transactions (0.1 ZCN value + fees)")
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)
	})

	t.RunSequentiallyWithTimeout("/v2/graph-total-staked", 5*time.Minute, func(t *test.SystemTest) {
		t.RunSequentially("endpoint parameters ( test /v2/graph-total-staked )", graphEndpointTestCases(zboxClient.GetGraphTotalStaked))

		t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-total-staked )", 5*time.Minute, func(t *test.SystemTest) {
			wallet := initialisedWallets[walletIdx]
			walletIdx++
			balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
			wallet.Nonce = int(balance.Nonce)

			PrintBalance(t, ownerWallet, blobberOwnerWallet, wallet)
			data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			totalStaked := (*data)[0]

			// Stake a blobbers
			targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, targetBlobbers, 1)
			confHash := apiClient.CreateStakePool(t, wallet, 3, targetBlobbers[0].ID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := (totalStakedAfter-totalStaked) == *(tokenomics.IntToZCN(1)) && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})

			// Get blobber balance before unlocking
			blobberBalanceBefore := getClientStakeForSSCProvider(t, wallet, targetBlobbers[0].ID)

			// Unlock a stake pool => should decrease
			restake := unstakeBlobber(t, wallet, targetBlobbers[0].ID)
			defer restake()

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := (totalStaked-totalStakedAfter) == blobberBalanceBefore && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})

			// Stake a validator
			vs, resp, err := apiClient.V1SCRestGetAllValidators(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, vs)
			validatorId := vs[0].ValidatorID
			confHash = apiClient.CreateStakePool(t, wallet, 4, validatorId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStakedAfter-totalStaked == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})

			// Unstake the validator
			confHash = apiClient.UnlockStakePool(t, wallet, 4, validatorId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStaked-totalStakedAfter == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})

			// Stake a miner
			miners, resp, err := apiClient.V1SCRestGetAllMiners(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, miners)
			minerId := miners[0].SimpleNodeResponse.ID
			t.Logf("Staking miner %s", minerId)
			confHash = apiClient.CreateMinerStakePool(t, wallet, 1, minerId, 1.0, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStakedAfter-totalStaked == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})

			// Unstake the miner
			confHash = apiClient.UnlockMinerStakePool(t, wallet, 1, minerId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStaked-totalStakedAfter == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})

			// Stake a sharder
			sharders, resp, err := apiClient.V1SCRestGetAllSharders(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, sharders)
			sharderId := sharders[0].SimpleNodeResponse.ID
			confHash = apiClient.CreateMinerStakePool(t, wallet, 2, sharderId, 1.0, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStakedAfter-totalStaked == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})

			// Unstake the sharder
			confHash = apiClient.UnlockMinerStakePool(t, wallet, 2, sharderId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStaked-totalStakedAfter == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				if cond {
					totalStaked = totalStakedAfter
				}
				return cond
			})
		})
	})

	t.RunSequentiallyWithTimeout("/v2/graph-challenges", 5*time.Minute, func(t *test.SystemTest) {
		t.RunSequentially("endpoint parameters ( test /v2/graph-challenges )", func(t *test.SystemTest) {
			// should fail for invalid parameters
			_, resp, _ := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "invalid from param")

			_, resp, _ = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "10", To: "AX", DataPoints: "5"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "invalid to param")

			_, resp, _ = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "10", To: "20", DataPoints: "AX"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "invalid data-points query param")

			// should pass for valid parameters (end - start = points)
			data, resp, _ := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "1000", To: "1010", DataPoints: "10"})
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 10, len([]int64(data.TotalChallenges)))
			require.Equal(t, 10, len([]int64(data.SuccessfulChallenges)))

			// should fail for invalid parameters (end < start)
			_, resp, _ = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "10000", To: "1000", DataPoints: "10"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "to 1000 less than from 10000")

			// should succeed in case of 1 point
			data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(data.TotalChallenges)))
			require.Equal(t, 1, len([]int64(data.SuccessfulChallenges)))

			// should succeed in case of multiple points
			minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
			require.NoError(t, err)
			latestRound := minerStats.LastFinalizedRound
			data, resp, err = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: strconv.FormatInt(latestRound-int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 10, len([]int64(data.TotalChallenges)))
			require.Equal(t, 10, len([]int64(data.SuccessfulChallenges)))
		})

		t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-challenges )", 5*time.Minute, func(t *test.SystemTest) {
			wallet := initialisedWallets[walletIdx]
			walletIdx++
			balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
			wallet.Nonce = int(balance.Nonce)

			sdkClient.SetWallet(t, wallet)

			// Get initial graph data
			data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(data.TotalChallenges)))
			require.Equal(t, 1, len([]int64(data.SuccessfulChallenges)))
			totalChallenges, successfulChallenges := data.TotalChallenges[0], data.SuccessfulChallenges[0]

			sdkWalletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
			t.Logf("sdk wallet balance: %v", sdkWalletBalance.Balance)
			wallet.Nonce = int(sdkWalletBalance.Nonce)

			// Create an allocation
			blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

			// Upload a file
			sdkClient.UploadFile(t, allocationID, 65536)

			// Check total challenges increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(data.TotalChallenges)))
				totalChallengesAfter := data.TotalChallenges[0]
				successfulChallengesAfter := data.SuccessfulChallenges[0]
				latestTotal, resp, err := zboxClient.GetTotalChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				latestSuccessful, resp, err := zboxClient.GetSuccessfulChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalChallengesAfter > totalChallenges && int64(*latestTotal) == totalChallengesAfter && int64(*latestSuccessful) == successfulChallengesAfter
				if cond {
					totalChallenges = totalChallengesAfter
					successfulChallenges = data.SuccessfulChallenges[0]
				}

				return cond
			})

			// Add blobber to the allocation
			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
			require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
			apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID,
				allocation.Blobbers[0].ID, allocationID, client.TxSuccessfulStatus)

			// Check total challenges increase + successful challenges increase because time has passed since the upload
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(data.TotalChallenges)))
				totalChallengesAfter := data.TotalChallenges[0]
				cond := totalChallengesAfter > totalChallenges
				totalChallenges = totalChallengesAfter
				successfulChallengesAfter := data.SuccessfulChallenges[0]
				cond = cond && successfulChallengesAfter > successfulChallenges
				successfulChallenges = successfulChallengesAfter
				latestTotal, resp, err := zboxClient.GetTotalChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				latestSuccessful, resp, err := zboxClient.GetSuccessfulChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond = cond && int64(*latestTotal) == totalChallengesAfter && int64(*latestSuccessful) == successfulChallengesAfter
				return cond
			})
		})
	})

	t.RunSequentiallyWithTimeout("test /v2/total-blobber-capacity", 5*time.Minute, func(t *test.SystemTest) {
		// Get initial
		data, resp, err := zboxClient.GetTotalBlobberCapacity(t)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		totalBlobberCapacity := int64(*data)

		// Increase capacity of 2 blobber
		targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 2, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, targetBlobbers, 2)

		// Ensure blobberOwnerWallet has sufficient balance and updated nonce
		blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transactions (0.1 ZCN value + fees)")

		targetBlobbers[0].Capacity += 10 * 1024 * 1024 * 1024
		targetBlobbers[1].Capacity += 5 * 1024 * 1024 * 1024
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)

		// Update nonce before second update
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)

		// Check increase
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetTotalBlobberCapacity(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			totalBlobberCapacityAfter := int64(*data)
			cond := totalBlobberCapacityAfter > totalBlobberCapacity
			totalBlobberCapacity = totalBlobberCapacityAfter
			return cond
		})

		// Decrease them back
		// Update nonce before first decrease
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transactions (0.1 ZCN value + fees)")

		targetBlobbers[0].Capacity -= 10 * 1024 * 1024 * 1024
		targetBlobbers[1].Capacity -= 5 * 1024 * 1024 * 1024
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)

		// Update nonce before second decrease
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)

		// Check decrease
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetTotalBlobberCapacity(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			totalBlobberCapacityAfter := int64(*data)
			totalBlobberCapacity = totalBlobberCapacityAfter

			blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			expectedCapacity := calculateCapacity(blobbers)
			cond := expectedCapacity == totalBlobberCapacityAfter
			return cond
		})
	})

	t.Run("endpoint parameters ( test /v2/graph-write-price )", graphEndpointTestCases(zboxClient.GetGraphWritePrice))
	t.Run("endpoint parameters ( test /v2/graph-total-challenge-pools )", graphEndpointTestCases(zboxClient.GetGraphTotalChallengePools))
	t.Run("endpoint parameters ( test /v2/graph-allocated-storage )", graphEndpointTestCases(zboxClient.GetGraphAllocatedStorage))
	t.Run("endpoint parameters ( test /v2/graph-used-storage )", graphEndpointTestCases(zboxClient.GetGraphUsedStorage))
	t.Run("endpoint parameters ( test /v2/graph-total-locked )", graphEndpointTestCases(zboxClient.GetGraphTotalLocked))
}

//nolint:gocyclo
func Test0boxGraphBlobberEndpoints(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	testWallet := initialisedWallets[walletIdx]
	walletIdx++
	balance := apiClient.GetWalletBalance(t, testWallet, client.HttpOkStatus)
	testWallet.Nonce = int(balance.Nonce)

	// Faucet the used initialisedWallets
	blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
	t.Logf("Blobber owner balance: %v", blobberOwnerBalance)
	blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)

	// Stake 6 blobbers, each with 1 token
	targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 6, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
	require.Len(t, targetBlobbers, 6)
	for _, blobber := range targetBlobbers {
		confHash := apiClient.CreateStakePool(t, testWallet, 3, blobber.ID, client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)
	}

	// Get a single blobber to use in graph parameters test
	blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
	require.Len(t, blobbers, 1)
	require.NotNil(t, blobbers[0].ID)

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-blobber-challenges-passed and /v2/graph-blobber-challenges-completed )", 3*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		// Create allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		alloc, _ := sdk.GetAllocation(allocationID)

		targetBlobber := alloc.BlobberDetails[0].BlobberID

		// Get initial value of one of the blobbers
		data, resp, err := zboxClient.GetGraphBlobberChallengesPassed(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, *data, 1)
		challnegesPassed := (*data)[0]

		data, resp, err = zboxClient.GetGraphBlobberChallengesCompleted(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, *data, 1)
		challnegesCompleted := (*data)[0]

		data, resp, err = zboxClient.GetGraphBlobberChallengesOpen(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, *data, 1)

		// Upload file
		fpath, fsize := sdkClient.UploadFile(t, allocationID, 65536)
		require.NotEmpty(t, fpath)
		require.NotZero(t, fsize)

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberChallengesPassed(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			challnegesPassedAfter := (*data)[0]
			cond := challnegesPassedAfter > challnegesPassed

			data, resp, err = zboxClient.GetGraphBlobberChallengesCompleted(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			challnegesCompletedAfter := (*data)[0]
			cond = cond && challnegesCompletedAfter > challnegesCompleted

			data, resp, err = zboxClient.GetGraphBlobberChallengesOpen(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)

			if cond {
				challnegesPassed = challnegesPassedAfter
				challnegesCompleted = challnegesCompletedAfter
			}
			return cond
		})
	})

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-blobber-write-price )", 5*time.Minute, func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		targetBlobber := blobbers[0]

		// Get initial value of one of the blobbers
		data, resp, err := zboxClient.GetGraphBlobberWritePrice(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, *data, 1)
		writePrice := (*data)[0]
		t.Logf("Initial write price: %d", writePrice)

		// Ensure blobberOwnerWallet has sufficient balance and updated nonce
		blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transaction (0.1 ZCN value + fees)")

		// Increase write price
		targetBlobber.Terms.WritePrice += 1000000000
		t.Logf("Updating blobber write price to: %d", targetBlobber.Terms.WritePrice)
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberWritePrice(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			t.Logf("Current write price: %d, expected > %d", afterValue, writePrice)
			cond := afterValue > writePrice
			if cond {
				writePrice = afterValue
				t.Logf("Write price increased successfully to: %d", writePrice)
			}
			return cond
		})

		// Decrease write price
		// Update nonce before second update
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transaction (0.1 ZCN value + fees)")

		targetBlobber.Terms.WritePrice -= 1000000000
		t.Logf("Updating blobber write price back to: %d", targetBlobber.Terms.WritePrice)
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

		// Check decreased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberWritePrice(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			t.Logf("Current write price: %d, expected < %d", afterValue, writePrice)
			cond := afterValue < writePrice
			if cond {
				writePrice = afterValue
				t.Logf("Write price decreased successfully to: %d", writePrice)
			}
			return cond
		})
	})

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-blobber-capacity )", 5*time.Minute, func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		targetBlobber := blobbers[0]

		// Get initial value of one of the blobbers
		data, resp, err := zboxClient.GetGraphBlobberCapacity(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, *data, 1)
		capacity := (*data)[0]

		// Ensure blobberOwnerWallet has sufficient balance and updated nonce
		blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transaction (0.1 ZCN value + fees)")

		// Increase capacity
		targetBlobber.Capacity += 1000000000
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberCapacity(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue > capacity
			if cond {
				capacity = afterValue
			}
			return cond
		})

		// Decrease capacity
		// Update nonce before second update
		blobberOwnerBalance = apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
		blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)
		require.GreaterOrEqual(t, blobberOwnerBalance.Balance, int64(200000000), "blobberOwnerWallet must have at least 0.2 ZCN to pay for update transaction (0.1 ZCN value + fees)")

		targetBlobber.Capacity -= 1000000000
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

		// Check decreased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberCapacity(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue < capacity
			if cond {
				capacity = afterValue
			}
			return cond
		})
	})

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-blobber-allocated )", 5*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		// Get allocated of all blobbers
		blobberAllocated := make(map[string]int64)

		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		for _, blobber := range blobbers {
			blobberAllocated[blobber.ID] = blobber.Allocated
		}

		// Create allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

		alloc, _ := sdk.GetAllocation(allocationID)

		targetBlobber := alloc.BlobberDetails[0].BlobberID

		allocated := blobberAllocated[targetBlobber]

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberAllocated(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue > allocated
			return cond
		})

		// Cancel the allocation
		confHash := apiClient.CancelAllocation(t, wallet, allocationID, client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)

		// Check decreased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberAllocated(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := allocated == afterValue
			return cond
		})
	})

	t.RunSequentiallyWithTimeout("test graph data ( test /v2/graph-blobber-saved-data )", 5*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		// Get saved data of all blobbers
		blobberSavedData := make(map[string]int64)

		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		for _, blobber := range blobbers {
			blobberSavedData[blobber.ID] = blobber.SavedData
		}

		// Create allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

		alloc, _ := sdk.GetAllocation(allocationID)

		targetBlobber := alloc.BlobberDetails[0].BlobberID
		savedData := blobberSavedData[targetBlobber]

		// Upload a file
		fpath, fsize := sdkClient.UploadFile(t, allocationID, 65536)

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			increase := afterValue - savedData
			t.Logf("Saved data: before=%d, after=%d, increase=%d, expected=%d", savedData, afterValue, increase, fsize)
			// Allow for small variance due to encoding overhead or rounding
			const tolerance = int64(1024) // 1KB tolerance
			cond := increase >= fsize-tolerance && increase <= fsize+tolerance
			if cond {
				savedData = afterValue
			}
			return cond
		})

		// Delete the file
		sdkClient.DeleteFile(t, allocationID, fpath)

		// Check decreased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			decrease := savedData - afterValue
			t.Logf("Saved data: before=%d, after=%d, decrease=%d, expected=%d", savedData, afterValue, decrease, fsize)
			// Allow for small variance due to encoding overhead or rounding
			const tolerance = int64(1024) // 1KB tolerance
			cond := decrease >= fsize-tolerance && decrease <= fsize+tolerance
			if cond {
				savedData = afterValue
			}
			return cond
		})

		// Upload another file
		_, fsize = sdkClient.UploadFile(t, allocationID, 65536)

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			increase := afterValue - savedData
			t.Logf("Saved data: before=%d, after=%d, increase=%d, expected=%d", savedData, afterValue, increase, fsize)
			// Allow for small variance due to encoding overhead or rounding
			const tolerance = int64(1024) // 1KB tolerance
			cond := increase >= fsize-tolerance && increase <= fsize+tolerance
			if cond {
				savedData = afterValue
			}
			return cond
		})

		// Cancel the allocation
		confHash := apiClient.CancelAllocation(t, wallet, allocationID, client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)

		// Check decreased for the same  blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			decrease := savedData - afterValue
			t.Logf("Saved data: before=%d, after=%d, decrease=%d, expected=%d", savedData, afterValue, decrease, fsize)
			// Allow for small variance due to encoding overhead or rounding
			const tolerance = int64(1024) // 1KB tolerance
			cond := decrease >= fsize-tolerance && decrease <= fsize+tolerance
			if cond {
				savedData = afterValue
			}
			return cond
		})
	})

	t.RunSequentially("test graph data ( test /v2/graph-blobber-read-data )", func(t *test.SystemTest) {
		t.Skipf("Skipping test as it is failing due to the issue in the code")
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		// Get read data of all blobbers
		blobberReadData := make(map[string]int64)

		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		for _, blobber := range blobbers {
			blobberReadData[blobber.ID] = blobber.ReadData
		}

		// Create allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

		alloc, _ := sdk.GetAllocation(allocationID)

		targetBlobber := alloc.BlobberDetails[0].BlobberID
		readData := blobberReadData[targetBlobber]

		// Upload a file
		fpath, fsize := sdkClient.UploadFile(t, allocationID, 65536)

		// Download the file
		sdkClient.DownloadFile(t, allocationID, fpath, "temp/")
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				t.Logf("error removing file %v: %v", name, err)
			}
		}(path.Join("temp/", fpath))

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberReadData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue-readData == fsize
			if cond {
				readData = afterValue
			}
			return cond
		})
	})

	t.RunSequentially("test graph data ( test /v2/graph-blobber-offers-total )", func(t *test.SystemTest) {
		wallet := createWallet(t)

		// Get offers of all blobbers
		blobberOffersTotal := make(map[string]int64)

		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		for _, blobber := range blobbers {
			data, resp, err := apiClient.V1SCRestGetStakePoolStat(t, model.SCRestGetStakePoolStatRequest{
				ProviderType: "3",
				ProviderID:   blobber.ID,
			}, client.HttpOkStatus)
			t.Logf("SP for blobber %v: %+v", blobber.ID, data)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			blobberOffersTotal[blobber.ID] = data.OffersTotal
		}

		// Create allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 3
		blobberRequirements.ParityShards = 3
		blobberRequirements.Size = 1024 * 1024 * 1024
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 1, client.TxSuccessfulStatus)

		// Value before allocation
		alloc, _ := sdk.GetAllocation(allocationID)

		targetBlobber := alloc.BlobberDetails[0].BlobberID
		offersTotal := blobberOffersTotal[targetBlobber]

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberOffersTotal(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue > offersTotal
			if cond {
				offersTotal = afterValue
			}
			return cond
		})

		// Cancel the allocation
		confHash := apiClient.CancelAllocation(t, wallet, allocationID, client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)

		// Check decreased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberOffersTotal(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue < offersTotal
			if cond {
				offersTotal = afterValue
			}
			return cond
		})
	})

	t.RunSequentially("test graph data ( test /v2/graph-blobber-stake-total )", func(t *test.SystemTest) {
		wallet := createWallet(t)

		targetBlobber := blobbers[0].ID
		data, resp, err := apiClient.V1SCRestGetStakePoolStat(t, model.SCRestGetStakePoolStatRequest{
			ProviderType: "3",
			ProviderID:   targetBlobber,
		}, client.HttpOkStatus)
		t.Logf("SP for blobber %v: %+v", targetBlobber, data)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		stakeTotal := data.Balance

		// Stake the blobber
		confHash := apiClient.CreateStakePool(t, wallet, 3, targetBlobber, client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)

		// Check stake increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberTotalStake(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue > stakeTotal
			if cond {
				stakeTotal = afterValue
			}
			return cond
		})

		// Unstake the blobber
		confHash = apiClient.UnlockStakePool(t, wallet, 3, targetBlobber, client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)

		// Check unstake increased and stake decrease for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberTotalStake(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue < stakeTotal

			return cond
		})
	})

	t.RunSequentially("test graph data ( test /v2/graph-blobber-total-rewards )", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		// Get read data of all blobbers
		blobberRewards := make(map[string]int64)

		blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		for _, blobber := range blobbers {
			sp, resp, err := apiClient.V1SCRestGetStakePoolStat(t, model.SCRestGetStakePoolStatRequest{
				ProviderType: "3",
				ProviderID:   blobber.ID,
			}, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			blobberRewards[blobber.ID] = sp.Rewards
		}

		// Create allocation
		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		blobberRequirements.DataShards = 1
		blobberRequirements.ParityShards = 1
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, wallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

		// Value before allocation
		alloc, _ := sdk.GetAllocation(allocationID)

		targetBlobber := alloc.BlobberDetails[0].BlobberID
		rewards := blobberRewards[targetBlobber]

		// Upload a file
		sdkClient.UploadFile(t, allocationID, 65536)

		// Check increased for the same blobber
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetGraphBlobberTotalRewards(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			afterValue := (*data)[0]
			cond := afterValue > rewards
			if cond {
				rewards = afterValue
			}
			return cond
		})
	})

	t.Run("endpoint parameters ( test /v2/graph-blobber-inactive-rounds )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberInactiveRounds, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-write-price )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberWritePrice, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-capacity )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberCapacity, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-allocated )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberAllocated, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-saved-data )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberSavedData, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-read-data )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberReadData, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-offers-total )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberOffersTotal, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-stake-total )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberTotalStake, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-total-rewards )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberTotalRewards, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-challenges-passed and /v2/graph-blobber-challenges-completed )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberChallengesPassed, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-challenges-passed and /v2/graph-blobber-challenges-completed )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberChallengesCompleted, blobbers[0].ID))
	t.Run("endpoint parameters ( test /v2/graph-blobber-challenges-passed and /v2/graph-blobber-challenges-completed )", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberChallengesOpen, blobbers[0].ID))
}

func graphEndpointTestCases(endpoint model.ZboxGraphEndpoint) func(*test.SystemTest) {
	return func(t *test.SystemTest) {
		// should fail for invalid parameters
		_, resp, err := endpoint(t, &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid from param")

		_, resp, err = endpoint(t, &model.ZboxGraphRequest{From: "10", To: "AX", DataPoints: "5"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid to param")

		_, resp, err = endpoint(t, &model.ZboxGraphRequest{From: "10", To: "20", DataPoints: "AX"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid data-points query param")

		// should not fail for valid parameters (end - start = points)
		data, resp, err := endpoint(t, &model.ZboxGraphRequest{From: "1000", To: "1010", DataPoints: "10"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 10, len([]int64(*data)))

		// should fail for invalid parameters (end < start)
		_, resp, err = endpoint(t, &model.ZboxGraphRequest{From: "10000", To: "1000", DataPoints: "10"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "to 1000 less than from 10000")

		// should succeed in case of 1 point
		data, resp, err = endpoint(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))

		// should succeed in case of multiple points
		minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
		require.NoError(t, err)
		latestRound := minerStats.LastFinalizedRound
		data, resp, err = endpoint(t, &model.ZboxGraphRequest{From: strconv.FormatInt(latestRound-int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 10, len([]int64(*data)))
	}
}

func graphBlobberEndpointTestCases(endpoint model.ZboxGraphBlobberEndpoint, blobberId string) func(*test.SystemTest) {
	return func(t *test.SystemTest) {
		// should fail for invalid parameters
		_, resp, _ := endpoint(t, "", &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "provider id not provided")

		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid from param")

		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "10", To: "AX", DataPoints: "5"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid to param")

		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "10", To: "20", DataPoints: "AX"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid data-points query param")

		// should fail for invalid parameters (end < start)
		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "10000", To: "1000", DataPoints: "10"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "to 1000 less than from 10000")

		// should succeed in case of 1 point
		data, resp, _ := endpoint(t, blobberId, &model.ZboxGraphRequest{DataPoints: "1"})
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))

		// should succeed in case of multiple points
		minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
		require.NoError(t, err)
		latestRound := minerStats.LastFinalizedRound
		time.Sleep(5 * time.Second)
		data, resp, err = endpoint(t, blobberId, &model.ZboxGraphRequest{From: strconv.FormatInt(latestRound-int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 10, len([]int64(*data)))
	}
}

func PrintBalance(t *test.SystemTest, ownerWallet, blobberOwnerWallet, sdkWallet *model.Wallet) {
	ownerBalance := apiClient.GetWalletBalance(t, ownerWallet, client.HttpOkStatus)
	t.Logf("ZboxOwner balance: %v", ownerBalance)
	blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
	t.Logf("Blobber owner balance: %v", blobberOwnerBalance)
	sdkWalletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
	t.Logf("Blobber owner balance: %v", sdkWalletBalance)
}

func printBlobbers(t *test.SystemTest, tag string, blobbers []*model.SCRestGetBlobberResponse) {
	t.Logf("%v: \n", tag)
	for _, blobber := range blobbers {
		t.Logf("ID: %s", blobber.ID)
		t.Logf("URL: %s", blobber.BaseURL)
		t.Logf("ReadPrice: %+v", blobber.Terms.ReadPrice)
		t.Logf("WritePrice: %+v", blobber.Terms.WritePrice)
		t.Logf("Capacity: %+v", blobber.Capacity)
		t.Logf("Allocated: %+v", blobber.Allocated)
		t.Logf("LastHealthCheck: %+v", blobber.LastHealthCheck)

		t.Logf("TotalStake: %+v", blobber.TotalStake)
		t.Logf("DelegateWallet: %+v", blobber.StakePoolSettings.DelegateWallet)
		t.Logf("MinStake: %+v", blobber.StakePoolSettings.MinStake)
		t.Logf("MaxStake: %+v", blobber.StakePoolSettings.MaxStake)
		t.Logf("NumDelegates: %+v", blobber.StakePoolSettings.NumDelegates)
		t.Logf("ServiceCharge: %+v", blobber.StakePoolSettings.ServiceCharge)
		t.Logf("----------------------------------")
	}
}

func calculateExpectedAvgWritePrice(blobbers []*model.SCRestGetBlobberResponse) (expectedAvgWritePrice int64) {
	var totalWritePrice int64

	totalStakedStorage := int64(0)
	stakedStorage := make([]int64, 0, len(blobbers))
	for _, blobber := range blobbers {
		ss := (float64(blobber.TotalStake) / float64(blobber.Terms.WritePrice)) * model.GB
		stakedStorage = append(stakedStorage, int64(ss))
		totalStakedStorage += int64(ss)
	}

	for i, blobber := range blobbers {
		totalWritePrice += int64((float64(stakedStorage[i]) / float64(totalStakedStorage)) * float64(blobber.Terms.WritePrice))
	}
	return totalWritePrice
}

func calculateExpectedAllocated(blobbers []*model.SCRestGetBlobberResponse) int64 {
	var totalAllocatedData int64

	for _, blobber := range blobbers {
		totalAllocatedData += blobber.Allocated
	}
	return totalAllocatedData
}

func calculateExpectedSavedData(blobbers []*model.SCRestGetBlobberResponse) int64 {
	var totalSavedData int64

	for _, blobber := range blobbers {
		totalSavedData += blobber.SavedData
	}
	return totalSavedData
}

func calculateCapacity(blobbers []*model.SCRestGetBlobberResponse) int64 {
	var totalCapacity int64

	for _, blobber := range blobbers {
		totalCapacity += blobber.Capacity
	}
	return totalCapacity
}

func unstakeBlobber(t *test.SystemTest, wallet *model.Wallet, blobberId string) func() {
	confHash := apiClient.UnlockStakePool(t, wallet, 3, blobberId, client.TxSuccessfulStatus)
	require.NotEmpty(t, confHash)
	return func() {
		// Re-stake the blobber
		confHash := apiClient.CreateStakePool(t, wallet, 3, blobberId, client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)
	}
}

// getClientStakeForSSCProvider returns the stake of the client for the given Storage Smart Contract provider (Blobber/Validator)
func getClientStakeForSSCProvider(t *test.SystemTest, wallet *model.Wallet, providerId string) int64 {
	stake, resp, err := apiClient.V1SCRestGetUserStakePoolStat(t, model.SCRestGetUserStakePoolStatRequest{
		ClientId: wallet.Id,
	}, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
	require.NotEmpty(t, stake)

	providerStake := (*stake.Pools[providerId])[0].Balance
	return providerStake
}
