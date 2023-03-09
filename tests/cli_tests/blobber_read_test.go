package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBlobberReadReward(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	setupWalletWithCustomTokens(t, configPath, 9)

	t.RunSequentiallyWithTimeout("Case 1 : 1 delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		setupWalletWithCustomTokens(t, configPath, 9)

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

		err = os.Remove(filename)
		require.Nil(t, err)

		remoteFilepath := remotepath + filepath.Base(filename)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remoteFilepath,
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))

		fmt.Println("Sleeping for 30 seconds")
		time.Sleep(30 * time.Second)

		downloadCost, _ := getDownloadCost(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
		}), true)

		fmt.Println(downloadCost)

		totalDownloadRewards := getTotalRewardsByRewardType(7)

		fmt.Println(totalDownloadRewards)

		require.Equal(t, downloadCost, totalDownloadRewards, "Download cost and total download rewards are not equal")
	})
}

func getTotalRewardsByRewardType(rewardType int) int {
	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward_type=" + strconv.Itoa(rewardType)

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println(err)
		return 0
	}

	fmt.Println(resp.Body)

	var response map[string]interface{}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	err = json.Unmarshal(body, &response)

	fmt.Println(response)

	sum := response["sum"].(float64)

	return int(sum)

}

//
//func sizeInGB(size int64) float64 {
//	return float64(size) / GB
//}
//
//func calculateDownloadCost(alloc *sdk.Allocation, fileSize int64, blocksPerMarker int) common.Balance {
//	chunkSize := fileref.CHUNK_SIZE
//	dataShards := alloc.DataShards
//
//	// singleShardSize collection of data-shards
//	singleShardSize := fileref.CHUNK_SIZE * dataShards
//	totalShards := int(math.Ceil(float64(fileSize) / float64(singleShardSize)))
//
//	// Currently if for example, blocksPerMarker is 10, and there are say 11 shards. First 10 shards will be covered
//	// by first readmarker. Second readmarker is signed with blocksPerMarker number of blocks i.e. 10
//	// so user ends up paying for 20 shards
//	totRMsForEachBlobber := int(math.Ceil(float64(totalShards) / float64(blocksPerMarker)))
//
//	var cost float64
//
//	for _, _ = range alloc.BlobberDetails {
//		readPrice := 0.1
//		for i := 0; i < totRMsForEachBlobber; i++ {
//			cost += sizeInGB(int64(chunkSize)*int64(blocksPerMarker)) * float64(readPrice)
//		}
//	}
//
//	balance := common.ToBalance(cost)
//	return balance
//}
