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

func TestBlobberChallengeRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

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

	t.RunWithTimeout("Case 1 : Client Uploads 10% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[0].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[1].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[0].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[1].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		remotepath := "/dir/"
		filesize := 50 * MB
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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		challenges, _ := getAllChallenges(t, configPath)

		totalPossibleReward := 5000000000 / (365 * 25 * 12 * 10)
		totalReward := 0

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			totalReward += challengeReward

			if challengeReward != 0 {
				fmt.Println("Challenge ID : ", challenge.ChallengeID)
				fmt.Println("Challenge Reward : ", challengeReward)
			}
		}

		fmt.Println("Total possible reward : ", totalPossibleReward)
		fmt.Println("Total reward : ", totalReward)
	})

	t.RunWithTimeout("Case 2 : Client Uploads 30% of Allocation and 1 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[0].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[1].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[0].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[1].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		remotepath := "/dir/"
		filesize := 300 * MB
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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		challenges, _ := getAllChallenges(t, configPath)

		totalPossibleReward := 1000000000 / (365 * 25 * 12 * 10)
		totalReward := 0

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			totalReward += challengeReward

			if challengeReward != 0 {
				fmt.Println("Challenge ID : ", challenge.ChallengeID)
				fmt.Println("Challenge Reward : ", challengeReward)
			}
		}

		fmt.Println("Total possible reward : ", totalPossibleReward)
		fmt.Println("Total reward : ", totalReward)
	})

	t.RunWithTimeout("Case 3 : Client Uploads 10% of Allocation and 1 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[0].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[1].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[0].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[1].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		challenges, _ := getAllChallenges(t, configPath)

		totalPossibleReward := 1000000000 / (365 * 25 * 12 * 10)
		totalReward := 0

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			totalReward += challengeReward

			if challengeReward != 0 {
				fmt.Println("Challenge ID : ", challenge.ChallengeID)
				fmt.Println("Challenge Reward : ", challengeReward)
			}
		}

		fmt.Println("Total possible reward : ", totalPossibleReward)
		fmt.Println("Total reward : ", totalReward)
	})

	t.RunWithTimeout("Case 4 : Client Uploads 10% of Allocation and 2 delegate each (equal stake)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[0].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[1].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[0].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[1].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		challenges, _ := getAllChallenges(t, configPath)

		totalPossibleReward := 1000000000 / (365 * 25 * 12 * 10)
		totalReward := 0

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			totalReward += challengeReward

			if challengeReward != 0 {
				fmt.Println("Challenge ID : ", challenge.ChallengeID)
				fmt.Println("Challenge Reward : ", challengeReward)
			}
		}

		fmt.Println("Total possible reward : ", totalPossibleReward)
		fmt.Println("Total reward : ", totalReward)
	})

	t.RunWithTimeout("Case 5 : Client Uploads 10% of Allocation and 2 delegate each (unequal stake 2:1)", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[0].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[1].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[0].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[1].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		challenges, _ := getAllChallenges(t, configPath)

		totalPossibleReward := 1000000000 / (365 * 25 * 12 * 10)
		totalReward := 0

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			totalReward += challengeReward

			if challengeReward != 0 {
				fmt.Println("Challenge ID : ", challenge.ChallengeID)
				fmt.Println("Challenge Reward : ", challengeReward)
			}
		}

		fmt.Println("Total possible reward : ", totalPossibleReward)
		fmt.Println("Total reward : ", totalReward)
	})

	t.RunWithTimeout("Case 6 : Client Uploads 10% of Allocation and 1 delegate each (equal stake) with Uploading in starting and in the middle", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[0].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"blobber_id": blobberList[1].Id,
		//	"tokens":     1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[0].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
		//
		//stakeTokens(t, configPath, createParams(map[string]interface{}{
		//	"validator_id": validatorList[1].ID,
		//	"tokens":       1,
		//}), true)
		//
		//output, err = registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

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

		// sleep for 2 minutes
		time.Sleep(2 * time.Minute)

		remotepath = "/dir/"
		filesize = 100 * MB
		filename = generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		challenges, _ := getAllChallenges(t, configPath)

		totalPossibleReward := 2 * (1000000000 / (365 * 25 * 12 * 10))
		totalReward := 0

		challengeUrl := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?challenge_id="
		for _, challenge := range challenges {

			res, _ := http.Get(challengeUrl + challenge.ChallengeID)

			var response map[string]interface{}
			body, _ := io.ReadAll(res.Body)
			err := json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println(err)
			}

			challengeReward := int(response["sum"].(float64))

			totalReward += challengeReward

			if challengeReward != 0 {
				fmt.Println("Challenge ID : ", challenge.ChallengeID)
				fmt.Println("Challenge Reward : ", challengeReward)
			}
		}

		fmt.Println("Total possible reward : ", totalPossibleReward)
		fmt.Println("Total reward : ", totalReward)
	})

}

func getAllChallenges(t *test.SystemTest, configPath string) ([]Challenge, error) {

	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/all-challenges"

	var result []Challenge

	res, _ := http.Get(url)

	fmt.Println(res.Body)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(res.Body)

	body, _ := io.ReadAll(res.Body)

	err := json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type Challenge struct {
	ChallengeID    string           `json:"challenge_id"`
	CreatedAt      common.Timestamp `json:"created_at"`
	AllocationID   string           `json:"allocation_id"`
	BlobberID      string           `json:"blobber_id"`
	ValidatorsID   string           `json:"validators_id"`
	Seed           int64            `json:"seed"`
	AllocationRoot string           `json:"allocation_root"`
	Responded      bool             `json:"responded"`
	Passed         bool             `json:"passed"`
	RoundResponded int64            `json:"round_responded"`
	ExpiredN       int              `json:"expired_n"`
}
