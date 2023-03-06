package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBlobberReadReward(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	t.RunSequentiallyWithTimeout("Case 1 : 1 delegate each, equal stake", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// 1. Create an allocation with 1 data shard and 1 parity shard.
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

		file, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}), true)
		if err != nil {
			fmt.Println("Error downloading file", err)
			return
		}

		fmt.Println(file)

		time.Sleep(2 * time.Minute)

		//
		//
		//
		//
		//// 2. Wait for 6 minutes.
		//time.Sleep(6 * time.Minute)
		//
		//// 3. Cancel the allocation.
		//cancelAllocation(t, configPath, allocationID, true)
		//
		//// 4. Collect rewards for the delegate wallets.
		//collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
		//	"provider_type": "delegate",
		//	"provider_id":   delegate1ID,
		//}), delegate1Wallet, true)
		//
		//collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
		//	"provider_type": "delegate",
		//	"provider_id":   delegate2ID,
		//}), delegate2Wallet, true)
		//
		//// 5. Get the balances of the delegate wallets.
		//delegate1WalletBalance, _ := getBalanceForWallet(t, configPath, delegate1Wallet)
		//delegate2WalletBalance, _ := getBalanceForWallet(t, configPath, delegate2Wallet)
		//
		//// 6. Check if the balances are equal.
		//require.Equal(t, delegate1WalletBalance, delegate2WalletBalance)
	})
}

func getAllRewardsByRewardType(rewardType int) int {
	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward_type=" + strconv.Itoa(rewardType)

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println(err)
		return 0
	}

	var response map[string]interface{}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	err = json.Unmarshal(body, &response)

	sum := response["sum"].(float64)

	return int(sum)

}
