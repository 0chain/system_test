package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestChallengeTimings(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var validatorList []climodel.Validator
	output, err = listValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	t.RunSequentiallyWithTimeout("Case 1: 1 allocation, 1mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		for i := 0; i < 5; i++ {
			remotepath := "/dir/"
			filesize := 1 * MB
			filename := generateRandomTestFileName(t)

			err = createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = uploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
		}

		getChallengeTimings()
	})

	t.RunSequentiallyWithTimeout("Case 2: 1 allocation, 10mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

		for i := 0; i < 5; i++ {
			remotepath := "/dir/"
			filesize := 10 * MB
			filename := generateRandomTestFileName(t)

			err = createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = uploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
		}

	})

	t.RunSequentiallyWithTimeout("Case 3: 10 allocation, 10mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// range of 10 allocations
		for i := 0; i < 10; i++ {

			// 1. Create an allocation with 1 data shard and 1 parity shard.
			allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   1 * GB,
				"tokens": 1,
				"data":   1,
				"parity": 1,
				"expire": "5m",
			})
			fmt.Println("Allocation ID : ", allocationId)

			// Uploading 10% of allocation

			remotepath := "/dir/"
			filesize := 10 * MB
			filename := generateRandomTestFileName(t)

			err = createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = uploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		}

	})

	t.RunSequentiallyWithTimeout("Case 4: 10 allocation, 100mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// range of 10 allocations
		for i := 0; i < 10; i++ {

			// 1. Create an allocation with 1 data shard and 1 parity shard.
			allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   1 * GB,
				"tokens": 1,
				"data":   1,
				"parity": 1,
				"expire": "5m",
			})
			fmt.Println("Allocation ID : ", allocationId)

			// Uploading 10% of allocation

			remotepath := "/dir/"
			filesize := 100 * MB
			filename := generateRandomTestFileName(t)

			err = createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = uploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		}

	})

	t.RunSequentiallyWithTimeout("Case 4: 10 allocation, 1gb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, true, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// range of 10 allocations
		for i := 0; i < 10; i++ {

			// 1. Create an allocation with 1 data shard and 1 parity shard.
			allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * GB,
				"tokens": 1,
				"data":   1,
				"parity": 1,
				"expire": "20m",
			})
			fmt.Println("Allocation ID : ", allocationId)

			// Uploading 10% of allocation

			remotepath := "/dir/"
			filesize := 1 * GB
			filename := generateRandomTestFileName(t)

			err = createFileWithSize(filename, int64(filesize))
			require.Nil(t, err)

			output, err = uploadFile(t, configPath, map[string]interface{}{
				// fetch the latest block in the chain
				"allocation": allocationId,
				"remotepath": remotepath + filepath.Base(filename),
				"localpath":  filename,
			}, true)
			require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		}

	})

}

func getChallengeTimings() [][]int64 {
	blobber1URL := "https://test2.zus.network/blobber01/challengetimings"
	blobber2URL := "https://test2.zus.network/blobber02/challengetimings"

	var blobber1Response, blobber2Response []map[string]string

	// make get request to blobber1 and store response to blobber1Response

	res, _ := http.Get(blobber1URL)
	json.NewDecoder(res.Body).Decode(&blobber1Response)

	// make get request to blobber2 and store response to blobber2Response

	res, _ = http.Get(blobber2URL)
	json.NewDecoder(res.Body).Decode(&blobber2Response)

	var proofGenTimes []int64
	var txnSubmissions []int64
	var txnVerifications []int64

	for _, blobber := range blobber1Response {
		proofGenTime, _ := strconv.ParseInt(blobber["proof_gen_time"], 10, 64)
		proofGenTimes = append(proofGenTimes, proofGenTime)

		txnSubmission, _ := strconv.ParseInt(blobber["txn_submission"], 10, 64)
		txnSubmissions = append(txnSubmissions, txnSubmission)

		txnVerification, _ := strconv.ParseInt(blobber["txn_verification"], 10, 64)
		txnVerifications = append(txnVerifications, txnVerification)
	}

	for _, blobber := range blobber2Response {
		proofGenTime, _ := strconv.ParseInt(blobber["proof_gen_time"], 10, 64)
		proofGenTimes = append(proofGenTimes, proofGenTime)

		txnSubmission, _ := strconv.ParseInt(blobber["txn_submission"], 10, 64)
		txnSubmissions = append(txnSubmissions, txnSubmission)

		txnVerification, _ := strconv.ParseInt(blobber["txn_verification"], 10, 64)
		txnVerifications = append(txnVerifications, txnVerification)
	}

	fmt.Println("Proof Gen Times : ", proofGenTimes)
	fmt.Println("Txn Submissions : ", txnSubmissions)
	fmt.Println("Txn Verifications : ", txnVerifications)

	var result [][]int64
	result = append(result, proofGenTimes)
	result = append(result, txnSubmissions)
	result = append(result, txnVerifications)

	return result
}
