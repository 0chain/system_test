package api_tests

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/blockchain"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	KB = 1024      // kilobyte
	MB = 1024 * KB // megabyte
	GB = 1024 * MB // gigabyte
)

func TestProtocolChallengeTimings(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 900, client.TxSuccessfulStatus)

	allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	lenBlobbers := int64(len(allBlobbers))
	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	blobberRequirements.DataShards = (lenBlobbers + 1) / 2
	blobberRequirements.ParityShards = lenBlobbers / 2

	t.TestSetup("Setup", func() {
		for _, blobber := range allBlobbers {
			// stake tokens to this blobber
			apiClient.CreateStakePool(t, sdkWallet, 3, blobber.ID, client.TxSuccessfulStatus)
		}

		allBlobbers, resp, err = apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		blobberRequirements.Size = 10 * MB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 5000, client.TxSuccessfulStatus)

		fileSize := int64(1 * MB)
		uploadOp := sdkClient.AddUploadOperation(t, allocationID, fileSize)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})
	})

	t.Cleanup(func() {
		for _, blobber := range allBlobbers {
			// unstake tokens from this blobber
			apiClient.UnlockStakePool(t, sdkWallet, 3, blobber.ID, client.TxSuccessfulStatus)
		}
	})

	t.RunWithTimeout("1mb file", 1*time.Hour, func(t *test.SystemTest) {
		blobberRequirements.Size = 2 * MB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 5000, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)

		fileSize := int64(1 * MB)
		uploadOp := sdkClient.AddUploadOperation(t, allocationID, fileSize)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(20 * time.Minute)

		result := getChallengeTimings(t, alloc.Blobbers, allocationID)

		proofGenTime := result[0]
		txnSubmission := result[1]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 50, "It is taking more than 50000 milliseconds to generate proof")
		require.True(t, txnSubmission < 70, "It is taking more than 7000 seconds to submit txn")
		require.True(t, txnVerificationTime < 70, "It is taking more than 7000 seconds to verify txn")
		require.True(t, false)
	})

	t.RunWithTimeout("10mb file", 1*time.Hour, func(t *test.SystemTest) {
		blobberRequirements.Size = 20 * MB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 5000, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)

		fileSize := int64(10 * MB)
		uploadOp := sdkClient.AddUploadOperation(t, allocationID, fileSize)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(20 * time.Minute)

		result := getChallengeTimings(t, alloc.Blobbers, allocationID)

		proofGenTime := result[0]
		txnSubmission := result[1]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 50, "It is taking more than 50000 milliseconds to generate proof")
		require.True(t, txnSubmission < 70, "It is taking more than 7000 seconds to submit txn")
		require.True(t, txnVerificationTime < 70, "It is taking more than 7000 seconds to verify txn")
		require.True(t, false)
	})

	t.RunWithTimeout("100mb file", 1*time.Hour, func(t *test.SystemTest) {
		blobberRequirements.Size = 200 * MB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 5000, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)

		fileSize := int64(100 * MB)
		uploadOp := sdkClient.AddUploadOperation(t, allocationID, fileSize)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(20 * time.Minute)

		result := getChallengeTimings(t, alloc.Blobbers, allocationID)

		proofGenTime := result[0]
		txnSubmission := result[1]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 50, "It is taking more than 50000 milliseconds to generate proof")
		require.True(t, txnSubmission < 70, "It is taking more than 7000 seconds to submit txn")
		require.True(t, txnVerificationTime < 70, "It is taking more than 7000 seconds to verify txn")
		require.True(t, false)
	})

	t.RunWithTimeout("1gb file", 1*time.Hour, func(t *test.SystemTest) {
		blobberRequirements.Size = 2 * GB
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 5000, client.TxSuccessfulStatus)

		alloc, err := sdk.GetAllocation(allocationID)
		require.NoError(t, err)

		fileSize := int64(1 * GB)
		uploadOp := sdkClient.AddUploadOperation(t, allocationID, fileSize)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

		time.Sleep(20 * time.Minute)

		result := getChallengeTimings(t, alloc.Blobbers, allocationID)

		proofGenTime := result[0]
		txnSubmission := result[1]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 50, "It is taking more than 50000 milliseconds to generate proof")
		require.True(t, txnSubmission < 70, "It is taking more than 7000 seconds to submit txn")
		require.True(t, txnVerificationTime < 70, "It is taking more than 7000 seconds to verify txn")
		require.True(t, false)
	})
}

func getChallengeTimings(t *test.SystemTest, blobbers []*blockchain.StorageNode, allocationID string) []int64 {
	blobberUrls := make(map[string]string)

	for i := 0; i < len(blobbers); i++ {
		blobber := blobbers[i]
		blobberUrls[blobber.ID] = blobber.Baseurl
	}

	var proofGenTimes, txnSubmissions, txnVerifications []int64
	var floatProofGenTimes, floatTxnSubmissions, floatTxnVerifications []float64

	challenges := apiClient.GetAllChallengesForAllocation(t, allocationID, client.HttpOkStatus)

	for i := 0; i < len(challenges); i++ {
		challenge := challenges[i]
		blobberUrl := blobberUrls[challenge.BlobberID]

		url := blobberUrl + "/challenge-timings-by-challengeId?challenge_id=" + challenge.ChallengeID

		resp, err := http.Get(url) //nolint:gosec
		if err != nil {
			t.Log("Error while getting challenge timings:", err)
			continue // Skip this iteration and move to the next blobber
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Log("Error while reading challenge timings response:", err)
			continue // Skip this iteration and move to the next blobber
		}

		var challengeTiming model.ChallengeTiming
		err = json.Unmarshal(body, &challengeTiming)
		if err != nil {
			t.Log("Error while unmarshalling challenge timings:", err)
			continue // Skip this iteration and move to the next blobber
		}

		if challengeTiming.TxnSubmission == 0 || challengeTiming.TxnVerification == 0 {
			continue
		}

		proofGenTimes = append(proofGenTimes, challengeTiming.ProofGenTime) // proof gen time in milliseconds

		// Calculate the time difference in milliseconds
		txnSubmission := challengeTiming.TxnSubmission.ToTime().Sub(challengeTiming.CreatedAtBlobber.ToTime())
		txnSubmissions = append(txnSubmissions, int64(txnSubmission.Seconds()))

		txnVerification := challengeTiming.TxnVerification.ToTime().Sub(challengeTiming.CreatedAtBlobber.ToTime())
		txnVerifications = append(txnVerifications, int64(txnVerification.Seconds()))
	}

	sort.Slice(proofGenTimes, func(i, j int) bool {
		return proofGenTimes[i] < proofGenTimes[j]
	})

	sort.Slice(txnSubmissions, func(i, j int) bool {
		return txnSubmissions[i] < txnSubmissions[j]
	})

	sort.Slice(txnVerifications, func(i, j int) bool {
		return txnVerifications[i] < txnVerifications[j]
	})

	t.Log("Proof Gen Times:", proofGenTimes)
	t.Log("Txn Submissions:", txnSubmissions)
	t.Log("Txn Verifications:", txnVerifications)

	// Max timings
	maxProofGenTime := proofGenTimes[len(proofGenTimes)-1]
	maxTxnSubmission := txnSubmissions[len(txnSubmissions)-1]
	maxTxnVerification := txnVerifications[len(txnVerifications)-1]

	// Log max timings
	t.Log("Max Proof Gen Time:", maxProofGenTime)
	t.Log("Max Txn Submission:", maxTxnSubmission)
	t.Log("Max Txn Verification:", maxTxnVerification)

	// Median timings
	medianProofGenTime := proofGenTimes[len(proofGenTimes)/2]
	medianTxnSubmission := txnSubmissions[len(txnSubmissions)/2]
	medianTxnVerification := txnVerifications[len(txnVerifications)/2]

	// Log mean timings
	t.Log("Median Proof Gen Time:", medianProofGenTime)
	t.Log("Median Txn Submission:", medianTxnSubmission)
	t.Log("Median Txn Verification:", medianTxnVerification)

	// Min Timings
	minProofGenTime := proofGenTimes[0]
	minTxnSubmission := txnSubmissions[0]
	minTxnVerification := txnVerifications[0]

	t.Log("Min Proof Gen Time:", minProofGenTime)
	t.Log("Min Txn Submission:", minTxnSubmission)
	t.Log("Min Txn Verification:", minTxnVerification)

	// Mean Timings
	for i := range proofGenTimes {
		floatProofGenTimes = append(floatProofGenTimes, float64(proofGenTimes[i]))
		floatTxnSubmissions = append(floatTxnSubmissions, float64(txnSubmissions[i]))
		floatTxnVerifications = append(floatTxnVerifications, float64(txnVerifications[i]))
	}

	t.Log("Mean Proof Gen Time:", int64(calculateMean(floatProofGenTimes)))
	t.Log("Mean Txn Submission:", int64(calculateMean(floatTxnSubmissions)))
	t.Log("Mean Txn Verification:", int64(calculateMean(floatTxnVerifications)))

	// Standard Deviation
	stdDevProofGenTime := calculateStandardDeviation(floatProofGenTimes)
	stdDevTxnSubmission := calculateStandardDeviation(floatTxnSubmissions)
	stdDevTxnVerification := calculateStandardDeviation(floatTxnVerifications)

	t.Log("Standard Deviation Proof Gen Time:", int64(stdDevProofGenTime))
	t.Log("Standard Deviation Txn Submission:", int64(stdDevTxnSubmission))
	t.Log("Standard Deviation Txn Verification:", int64(stdDevTxnVerification))

	// Variance
	varianceProofGenTime := calculateVariance(floatProofGenTimes)
	varianceTxnSubmission := calculateVariance(floatTxnSubmissions)
	varianceTxnVerification := calculateVariance(floatTxnVerifications)

	t.Log("Variance Proof Gen Time:", int64(varianceProofGenTime))
	t.Log("Variance Txn Submission:", int64(varianceTxnSubmission))
	t.Log("Variance Txn Verification:", int64(varianceTxnVerification))

	return []int64{maxProofGenTime, maxTxnSubmission, maxTxnVerification}
}

func calculateStandardDeviation(data []float64) float64 {
	// Step 1: Calculate the mean
	mean := calculateMean(data)

	// Step 2: Calculate the sum of squared differences from the mean
	var sumSquaredDiff float64
	for _, value := range data {
		diff := value - mean
		sumSquaredDiff += diff * diff
	}

	// Step 3: Calculate the variance
	variance := sumSquaredDiff / float64(len(data))

	// Step 4: Take the square root to get the standard deviation
	stdDev := math.Sqrt(variance)

	return stdDev
}

func calculateMean(data []float64) float64 {
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func calculateVariance(data []float64) float64 {
	mean := calculateMean(data)

	var sumSquaredDiff float64
	for _, value := range data {
		diff := value - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(data))

	return variance
}
