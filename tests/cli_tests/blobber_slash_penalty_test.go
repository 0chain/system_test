package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlobberSlashPenalty(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	prevBlock := getLatestFinalizedBlock(t)

	fmt.Println("prevBlock", prevBlock)

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

	t.RunSequentiallyWithTimeout("Upload 10% of allocation and Kill blobber in the middle, One blobber should get approx double rewards than other", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// check allocation remaining time
		allocation := getAllocation(t, allocationId)
		remainingTime := allocation.ExpirationDate - time.Now().Unix()

		// sleep for half of the remaining time
		time.Sleep(time.Duration(remainingTime/2) * time.Second)

		// 2. Kill a blobber
		_, err = killBlobber(t, configPath, createParams(map[string]interface{}{
			"id": blobberList[1].Id,
		}), true)
		require.Nil(t, err, "error killing blobber", strings.Join(output, "\n"))

		// 3. Sleep for the remaining time
		time.Sleep(time.Duration(remainingTime/2) * time.Second)

		allocation = getAllocation(t, allocationId)

		fmt.Println(allocation.MovedToChallenge)

		blobberRewards := getAllocationChallengeRewards(t, allocationId)

		fmt.Println(blobberRewards)

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

	t.RunSequentiallyWithTimeout("Upload 50% of allocation and Kill blobber in the middle, One blobber should get approx double rewards than other", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "20m",
		})

		remotepath := "/dir/"
		filesize := 0.5 * GB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// check allocation remaining time
		allocation := getAllocation(t, allocationId)
		remainingTime := allocation.ExpirationDate - time.Now().Unix()

		// sleep for half of the remaining time
		time.Sleep(time.Duration(remainingTime/2) * time.Second)

		// 2. Kill a blobber
		_, err = killBlobber(t, configPath, createParams(map[string]interface{}{
			"id": blobberList[1].Id,
		}), true)
		require.Nil(t, err, "error killing blobber", strings.Join(output, "\n"))

		// 3. Sleep for the remaining time
		time.Sleep(time.Duration(remainingTime/2) * time.Second)

		allocation = getAllocation(t, allocationId)

		fmt.Println(allocation.MovedToChallenge)

		blobberRewards := getAllocationChallengeRewards(t, allocationId)

		fmt.Println(blobberRewards)

		unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)
	})

}

func killBlobber(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("kill blobber...")
	cmd := fmt.Sprintf("./zbox kill-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, scOwnerWallet, cliConfigFilename)

	fmt.Println(cmd)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
