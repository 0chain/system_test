package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

// 1687440537
func TestChallengeTimings(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
		"time_unit": "5m",
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))

	output, err = utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	numData := 2
	numParity := 10

	t.RunWithTimeout("Case 1: 1 10mb allocation, 1mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 99,
			"data":   numData,
			"parity": numParity,
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 3 * MB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, fmt.Sprintf("error uploading file %s", allocationId), strings.Join(output, "\n"))

		time.Sleep(20 * time.Minute)

		// cancel allocation
		_, err = utils.CancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, fmt.Sprintf("error cancelling allocation %s", allocationId), strings.Join(output, "\n"))

		result := getChallengeTimings(t, blobberList, []string{allocationId})

		proofGenTime := result[0]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 50, "It is taking more than 50000 milliseconds to generate proof")

		require.True(t, txnVerificationTime < 7000, "It is taking more than 7000 milliseconds to verify txn")

		require.True(t, false)
	})

	t.RunWithTimeout("Case 2: 1 100mb allocation, 10mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 99,
			"data":   numData,
			"parity": numParity,
		})

		// Uploading 10% of allocation

		remotepath := "/dir/"
		filesize := 30 * MB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, fmt.Sprintf("error uploading file %s", allocationId), strings.Join(output, "\n"))

		time.Sleep(20 * time.Minute)
		_, err = utils.CancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, fmt.Sprintf("error cancelling allocation %s", allocationId), strings.Join(output, "\n"))

		result := getChallengeTimings(t, blobberList, []string{allocationId})

		proofGenTime := result[0]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 90, "It is taking more than 90000 milliseconds to generate proof")

		require.True(t, txnVerificationTime < 10000, "It is taking more than 10000 milliseconds to verify txn")

		require.True(t, false)
	})

	t.RunWithTimeout("Case 3: 10 100mb allocation, 10mb file each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		var allocationIDs []string

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// range of 10 allocations
		for i := 0; i < 10; i++ {
			// 1. Create an allocation with 1 data shard and 1 parity shard.
			allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   100 * MB,
				"tokens": 99,
				"data":   numData,
				"parity": numParity,
			})

			allocationIDs = append(allocationIDs, allocationId)

			// Uploading 10% of allocation
			remotepath := "/dir/"
			filesize := 30 * MB
			filename := utils.GenerateRandomTestFileName(t)

			err = utils.CreateFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = utils.UploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, fmt.Sprintf("error uploading file %s", allocationId), strings.Join(output, "\n"))
		}

		time.Sleep(20 * time.Minute)
		for _, allocationId := range allocationIDs {
			_, err = utils.CancelAllocation(t, configPath, allocationId, true)
			require.Nil(t, err, fmt.Sprintf("error cancelling allocation %s", allocationId), strings.Join(output, "\n"))
		}

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTime := result[0]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 110, "It is taking more than 110000 milliseconds to generate proof")
		require.True(t, txnVerificationTime < 10000, "It is taking more than 10000 milliseconds to verify txn")

		require.True(t, false)
	})

	t.RunWithTimeout("Case 4: 10 1gb allocation, 100mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "20m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		var allocationIDs []string

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// range of 10 allocations
		for i := 0; i < 10; i++ {
			// 1. Create an allocation with 1 data shard and 1 parity shard.
			allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   1 * GB,
				"tokens": 99,
				"data":   numData,
				"parity": numParity,
			})

			allocationIDs = append(allocationIDs, allocationId)

			// Uploading 10% of allocation

			remotepath := "/dir/"
			filesize := 300 * MB
			filename := utils.GenerateRandomTestFileName(t)

			err = utils.CreateFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = utils.UploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, fmt.Sprintf("error uploading file %s", allocationId), strings.Join(output, "\n"))
		}

		time.Sleep(20 * time.Minute)
		for _, allocationId := range allocationIDs {
			_, err = utils.CancelAllocation(t, configPath, allocationId, true)
			require.Nil(t, err, fmt.Sprintf("error cancelling allocation %s", allocationId), strings.Join(output, "\n"))
		}

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTime := result[0]
		txnVerificationTime := result[2]

		require.True(t, proofGenTime < 350, "It is taking more than 320000 milliseconds to generate proof")
		require.True(t, txnVerificationTime < 15000, "It is taking more than 10000 milliseconds to verify txn")

		require.True(t, false)
	})

	t.RunWithTimeout("Case 5: 1 10gb allocation, 1gb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "20m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		var allocationIDs []string

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 99)
		require.Nil(t, err, "Error executing faucet with tokens", strings.Join(output, "\n"))

		// range of 10 allocations
		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * GB,
			"tokens": 99,
			"data":   numData,
			"parity": numParity,
		})
		allocationIDs = append(allocationIDs, allocationId)

		// Uploading 10% of allocation
		remotepath := "/dir/"
		filesize := 3 * GB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, fmt.Sprintf("error uploading file %s", allocationId), strings.Join(output, "\n"))

		time.Sleep(20 * time.Minute)
		_, err = utils.CancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, fmt.Sprintf("error cancelling allocation %s", allocationId), strings.Join(output, "\n"))

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTime := result[0]
		txnVerificationTime := result[2]
		require.True(t, proofGenTime < 4200, "It is taking more than 4000000 milliseconds to generate proof")
		require.True(t, txnVerificationTime < 30000, "It is taking more than 30000 milliseconds to verify txn")

		require.True(t, false)
	})

	t.RunWithTimeout("Case 6: 1 100gb allocation, 10gb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "20m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		var allocationIDs []string

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 99)
		require.Nil(t, err, "Error executing faucet with tokens", strings.Join(output, "\n"))

		// range of 10 allocations
		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   100 * GB,
			"tokens": 99,
			"data":   numData,
			"parity": numParity,
		})
		allocationIDs = append(allocationIDs, allocationId)

		// Uploading 10% of allocation
		remotepath := "/dir/"
		filesize := 30 * GB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, fmt.Sprintf("error uploading file %s", allocationId), strings.Join(output, "\n"))

		time.Sleep(20 * time.Minute)
		_, err = utils.CancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, fmt.Sprintf("error cancelling allocation %s", allocationId), strings.Join(output, "\n"))

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTime := result[0]
		txnVerificationTime := result[2]
		require.True(t, proofGenTime < 4200, "It is taking more than 4000000 milliseconds to generate proof")
		require.True(t, txnVerificationTime < 30000, "It is taking more than 30000 milliseconds to verify txn")

		require.True(t, false)
	})

	t.RunWithTimeout("Case 7: 1 1000gb allocation, 100gb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err = utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "20m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		var allocationIDs []string

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 99)
		require.Nil(t, err, "Error executing faucet with tokens", strings.Join(output, "\n"))

		// range of 10 allocations
		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * GB,
			"tokens": 99,
			"data":   numData,
			"parity": numParity,
		})
		allocationIDs = append(allocationIDs, allocationId)

		// Uploading 10% of allocation
		remotepath := "/dir/"
		filesize := 300 * GB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, fmt.Sprintf("error uploading file %s", allocationId), strings.Join(output, "\n"))

		time.Sleep(20 * time.Minute)
		_, err = utils.CancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, fmt.Sprintf("error cancelling allocation %s", allocationId), strings.Join(output, "\n"))

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTime := result[0]
		txnVerificationTime := result[2]
		require.True(t, proofGenTime < 4200, "It is taking more than 4000000 milliseconds to generate proof")
		require.True(t, txnVerificationTime < 30000, "It is taking more than 30000 milliseconds to verify txn")

		require.True(t, false)
	})
}

func getChallengeTimings(t *test.SystemTest, blobbers []climodel.BlobberInfo, allocationIDs []string) []int64 {
	blobberUrls := make(map[string]string)

	for i := 0; i < len(blobbers); i++ {
		blobber := blobbers[i]
		blobberUrls[blobber.Id] = blobber.Url
	}

	var proofGenTimes, txnSubmissions, txnVerifications []int64
	var floatProofGenTimes, floatTxnSubmissions, floatTxnVerifications []float64

	for i := 0; i < len(allocationIDs); i++ {
		allocationID := allocationIDs[i]
		challenges, err := getAllChallenges(t, allocationID)
		require.Nil(t, err, "Error getting all challenges")

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

			var challengeTiming ChallengeTiming
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
			txnSubmissions = append(txnSubmissions, txnSubmission.Milliseconds())

			txnVerification := challengeTiming.TxnVerification.ToTime().Sub(challengeTiming.CreatedAtBlobber.ToTime())
			txnVerifications = append(txnVerifications, txnVerification.Milliseconds())
		}
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

type ChallengeTiming struct {
	// ChallengeID is the challenge ID generated on blockchain.
	ChallengeID string `gorm:"column:challenge_id;size:64;primaryKey" json:"id"`

	// CreatedAtChain is when generated on blockchain.
	CreatedAtChain common.Timestamp `gorm:"created_at_chain" json:"created_at_chain"`
	// CreatedAtBlobber is when synchronized and created at blobber.
	CreatedAtBlobber common.Timestamp `gorm:"created_at_blobber" json:"created_at_blobber"`
	// FileSize is size of file that was randomly selected for challenge
	FileSize int64 `gorm:"file_size" json:"file_size"`
	// ProofGenTime is the time taken in millisecond to generate challenge proof for the file
	ProofGenTime int64 `gorm:"proof_gen_time" json:"proof_gen_time"`
	// CompleteValidation is when all validation tickets are all received.
	CompleteValidation common.Timestamp `gorm:"complete_validation" json:"complete_validation"`
	// TxnSubmission is when challenge response is first sent to blockchain.
	TxnSubmission common.Timestamp `gorm:"txn_submission" json:"txn_submission"`
	// TxnVerification is when challenge response is verified on blockchain.
	TxnVerification common.Timestamp `gorm:"txn_verification" json:"txn_verification"`
	// Canceled is when challenge is Canceled by blobber due to expiration or bad challenge data (eg. invalid ref or not a file) which is impossible to validate.
	Canceled common.Timestamp `gorm:"canceled" json:"canceled"`
	// Expiration is when challenge is marked as expired by blobber.
	Expiration common.Timestamp `gorm:"expiration" json:"expiration"`

	// ClosedAt is when challenge is closed (eg. expired, canceled, or completed/verified).
	ClosedAt common.Timestamp `gorm:"column:closed_at;index:idx_closed_at,sort:desc;" json:"closed"`

	// UpdatedAt is when row is last updated.
	UpdatedAt common.Timestamp `gorm:"column:updated_at;index:idx_updated_at,sort:desc;" json:"updated"`
}

func getAllChallenges(t *test.SystemTest, allocationID string) ([]Challenge, error) {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/all-challenges?allocation_id=" + allocationID)
	t.Log("Allocation challenge list url: ", url)
	var result []Challenge
	res, _ := http.Get(url) //nolint:gosec
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		require.Nil(t, err, "Error closing response body")
	}(res.Body)
	body, _ := io.ReadAll(res.Body)
	err := json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
