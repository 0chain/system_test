package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"path/filepath"
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

	t.RunSequentiallyWithTimeout("Case 1: 1 10mb allocation, 1mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})
		fmt.Println("Allocation ID : ", allocationId)

		// Uploading 10% of allocation

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

		time.Sleep(1 * time.Minute)

		result := getChallengeTimings(t, blobberList, []string{allocationId})

		fmt.Println("ProofGenTimes : ", result[0])
		fmt.Println("TxnSubmissions : ", result[1])
		fmt.Println("TxnVerifications : ", result[2])

		//proofGenTimes := result[0]
		//txnSubmissions := result[1]
		//txnVerifications := result[2]

		//for _, proofGenTime := range proofGenTimes {
		//	require.True(t, proofGenTime < 1681327150, "Proof generation time is more than 2 seconds")
		//}
		//
		//for _, txnSubmission := range txnSubmissions {
		//	require.True(t, txnSubmission == 0, "Transaction submission time is more than 2 seconds")
		//}
		//
		//for _, txnVerification := range txnVerifications {
		//	require.True(t, txnVerification < 18446744072028224700, "Transaction verification time is more than 2 seconds")
		//}
	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 2: 1 100mb allocation, 10mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   100 * MB,
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

		time.Sleep(1 * time.Minute)

		result := getChallengeTimings(t, blobberList, []string{allocationId})

		proofGenTimes := result[0]
		txnSubmissions := result[1]
		txnVerifications := result[2]

		for _, proofGenTime := range proofGenTimes {
			require.True(t, proofGenTime < 10, "Proof generation time is more than 2 seconds")
		}

		for _, txnSubmission := range txnSubmissions {
			require.True(t, txnSubmission < 1680789860, "Transaction submission time is more than 2 seconds")
		}

		for _, txnVerification := range txnVerifications {
			require.True(t, txnVerification < 1680789865, "Transaction verification time is more than 2 seconds")
		}

	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 3: 10 100mb allocation, 10mb file each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		var allocationIDs []string

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		// range of 10 allocations
		for i := 0; i < 10; i++ {

			// 1. Create an allocation with 1 data shard and 1 parity shard.
			allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   100 * MB,
				"tokens": 1,
				"data":   1,
				"parity": 1,
				"expire": "5m",
			})
			fmt.Println("Allocation ID : ", allocationId)

			allocationIDs = append(allocationIDs, allocationId)

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

		time.Sleep(1 * time.Minute)

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTimes := result[0]
		txnSubmissions := result[1]
		txnVerifications := result[2]

		for _, proofGenTime := range proofGenTimes {
			require.True(t, proofGenTime < 10, "Proof generation time is more than 2 seconds")
		}

		for _, txnSubmission := range txnSubmissions {
			require.True(t, txnSubmission < 1680789860, "Transaction submission time is more than 2 seconds")
		}

		for _, txnVerification := range txnVerifications {
			require.True(t, txnVerification < 1680789865, "Transaction verification time is more than 2 seconds")
		}

	})

	t.RunSequentiallyWithTimeout("Case 4: 10 1gb allocation, 100mb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		var allocationIDs []string

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
				"expire": "1h",
			})
			fmt.Println("Allocation ID : ", allocationId)

			allocationIDs = append(allocationIDs, allocationId)

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

		time.Sleep(1 * time.Minute)

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTimes := result[0]
		txnSubmissions := result[1]
		txnVerifications := result[2]

		for _, proofGenTime := range proofGenTimes {
			require.True(t, proofGenTime < 10, "Proof generation time is more than 2 seconds")
		}

		for _, txnSubmission := range txnSubmissions {
			require.True(t, txnSubmission < 1680789860, "Transaction submission time is more than 2 seconds")
		}

		for _, txnVerification := range txnVerifications {
			require.True(t, txnVerification < 1680789865, "Transaction verification time is more than 2 seconds")
		}

	})

	t.Skip()

	t.RunSequentiallyWithTimeout("Case 5: 10 10gb allocation, 1gb each", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {

		var allocationIDs []string

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		executeFaucetWithTokens(t, configPath, 100)

		// range of 10 allocations
		for i := 0; i < 10; i++ {

			// 1. Create an allocation with 1 data shard and 1 parity shard.
			allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
				"size":   10 * GB,
				"tokens": 1,
				"data":   1,
				"parity": 1,
				"expire": "1h",
			})
			fmt.Println("Allocation ID : ", allocationId)

			allocationIDs = append(allocationIDs, allocationId)

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

		time.Sleep(1 * time.Minute)

		result := getChallengeTimings(t, blobberList, allocationIDs)

		proofGenTimes := result[0]
		txnSubmissions := result[1]
		txnVerifications := result[2]

		for _, proofGenTime := range proofGenTimes {
			require.True(t, proofGenTime < 10, "Proof generation time is more than 2 seconds")
		}

		for _, txnSubmission := range txnSubmissions {
			require.True(t, txnSubmission < 1680789860, "Transaction submission time is more than 2 seconds")
		}

		for _, txnVerification := range txnVerifications {
			require.True(t, txnVerification < 1680789865, "Transaction verification time is more than 2 seconds")
		}

	})

}

func getChallengeTimings(t *test.SystemTest, blobbers []climodel.BlobberInfo, allocationIDs []string) [][]common.Timestamp {

	blobberUrls := []string{}
	for _, blobber := range blobbers {
		blobberUrls = append(blobberUrls, blobber.Url)
	}

	var proofGenTimes []common.Timestamp
	var txnSubmissions []common.Timestamp
	var txnVerifications []common.Timestamp

	for _, allocationID := range allocationIDs {
		challenges, _ := getAllChallenges(t, allocationID)

		for _, challenge := range challenges {

			for _, blobberUrl := range blobberUrls {
				url := blobberUrl + "/challenge-timings-by-challengeId?challenge_id=" + challenge.ChallengeID

				resp, err := http.Get(url)
				if err != nil {
					fmt.Println("Error while getting challenge timings : ", err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)

				var challengeTiming ChallengeTiming
				err = json.Unmarshal(body, &challengeTiming)
				if err != nil {
					fmt.Println("Error while unmarshalling challenge timings : ", err)
				}

				fmt.Println(challengeTiming.ProofGenTime)

				challengeTiming.ProofGenTime = challengeTiming.ProofGenTime / 1000
				proofGenTime := common.Timestamp(challengeTiming.ProofGenTime)

				fmt.Println("Proof Gen Time : ", proofGenTime)

				proofGenTimes = append(proofGenTimes, challengeTiming.CompleteValidation-proofGenTime-challengeTiming.CreatedAtBlobber)
				txnSubmissions = append(txnSubmissions, challengeTiming.TxnSubmission-challengeTiming.CompleteValidation)
				txnVerifications = append(txnVerifications, challengeTiming.TxnVerification-challengeTiming.TxnSubmission)
			}
		}
	}

	fmt.Println("Proof Gen Times : ", proofGenTimes)
	fmt.Println("Txn Submissions : ", txnSubmissions)
	fmt.Println("Txn Verifications : ", txnVerifications)

	// max values from all the lists
	var maxProofGenTime, maxTxnSubmission, maxTxnVerification common.Timestamp

	for _, proofGenTime := range proofGenTimes {
		if proofGenTime > maxProofGenTime {
			maxProofGenTime = proofGenTime
		}
	}

	for _, txnSubmission := range txnSubmissions {
		if txnSubmission > maxTxnSubmission {
			maxTxnSubmission = txnSubmission
		}
	}

	for _, txnVerification := range txnVerifications {
		if txnVerification > maxTxnVerification {
			maxTxnVerification = txnVerification
		}
	}

	fmt.Println("Max Proof Gen Time : ", maxProofGenTime)
	fmt.Println("Max Txn Submission : ", maxTxnSubmission)
	fmt.Println("Max Txn Verification : ", maxTxnVerification)

	var result [][]common.Timestamp
	result = append(result, proofGenTimes)
	result = append(result, txnSubmissions)
	result = append(result, txnVerifications)

	return result
}

type ChallengeTiming struct {
	// ChallengeID is the challenge ID generated on blockchain.
	ChallengeID string `json:"id"`

	// CreatedAtChain is when generated on blockchain.
	CreatedAtChain common.Timestamp `json:"created_at_chain"`
	// CreatedAtBlobber is when synchronized and created at blobber.
	CreatedAtBlobber common.Timestamp `json:"created_at_blobber"`
	// FileSize is size of file that was randomly selected for challenge
	FileSize int64 `json:"file_size"`
	// ProofGenTime is the time taken in millisecond to generate challenge proof for the file
	ProofGenTime int64 `json:"proof_gen_time"`
	// CompleteValidation is when all validation tickets are all received.
	CompleteValidation common.Timestamp `json:"complete_validation"`
	// TxnSubmission is when challenge response is first sent to blockchain.
	TxnSubmission common.Timestamp `json:"txn_submission"`
	// TxnVerification is when challenge response is verified on blockchain.
	TxnVerification common.Timestamp `json:"txn_verification"`
	// Cancelled is when challenge is cancelled by blobber due to expiration or bad challenge data (eg. invalid ref or not a file) which is impossible to validate.
	Cancelled common.Timestamp `json:"cancelled"`
	// Expiration is when challenge is marked as expired by blobber.
	Expiration common.Timestamp `json:"expiration"`
}
