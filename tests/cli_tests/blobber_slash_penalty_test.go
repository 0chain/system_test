package cli_tests

import (
	"encoding/json"
	"fmt"
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

func TestBlobberSlashPenalty(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	t.RunSequentiallyWithTimeout("Test Cancel Allocation After Expiry Rewards when client uploads 10% of allocation", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "50m",
		})

		remotepath := "/dir/"
		filesize := 50 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// 2. Wait for 2 minutes 30 seconds
		time.Sleep(2*time.Minute + 30*time.Second)

		// 3. Kill the blobber
		killProvider(t, blobberList[0].Id)

		// 4. Wait for 2 minutes 30 seconds
		time.Sleep(2*time.Minute + 30*time.Second)

		// 5. Check the total challenge reward of the blobber
		for _, blobber := range blobberList {
			totalChallengeReward := getTotalChallengeRewardByProviderID(blobber.Id)
			fmt.Println(totalChallengeReward)
		}
	})

	t.RunSequentiallyWithTimeout("Test Cancel Allocation After Expiry Rewards when client uploads 50% of allocation", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   500 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "50m",
		})

		remotepath := "/dir/"
		filesize := 250 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// 2. Wait for 2 minutes 30 seconds
		time.Sleep(2*time.Minute + 30*time.Second)

		// 3. Kill the blobber
		killProvider(t, blobberList[0].Id)

		// 4. Wait for 2 minutes 30 seconds
		time.Sleep(2*time.Minute + 30*time.Second)

		// 5. Check the total challenge reward of the blobber
		for _, blobber := range blobberList {
			totalChallengeReward := getTotalChallengeRewardByProviderID(blobber.Id)
			fmt.Println(totalChallengeReward)
		}
	})
}

func killProvider(t *test.SystemTest, providerID string) {
	res, err := http.Get("https://test2.zus.network/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/kill_blobber?provider_id=" + providerID)

	fmt.Println(res, err)
}

func getTotalChallengeRewardByProviderID(providerID string) int {
	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/all-challenges?provider_id=" + providerID
	res, _ := http.Get(url)
	body, _ := io.ReadAll(res.Body)

	var response map[string]interface{}

	if err := json.Unmarshal(body, &response); err != nil {
		panic(err)
	}

	return response["sum"].(int)
}
