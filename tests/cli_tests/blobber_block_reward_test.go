package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlobberBlockRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Test Blobber Block Rewards", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		t.Skip("Skipping this test for now")

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		var blobberList []climodel.BlobberInfo
		output, err = listBlobbers(t, configPath, "--json")

		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		remotepath := "/dir/"
		filesize := 30 * MB
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

		getAllBlockRewards(t, configPath)

	})

	t.RunSequentiallyWithTimeout("Get All Block Rewards", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		getAllBlockRewards(t, configPath)
	})
}

func getAllBlockRewards(t *test.SystemTest, configPath string) {

	url := "https://test2.zus.network/sharder01/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/reward-providers?reward_type=3"
	var response map[string]interface{}

	res, _ := http.Get(url)

	// decode and save the res body to response
	json.NewDecoder(res.Body).Decode(&response)

	fmt.Println(response)

	fmt.Println("Total Block Rewards : ", int64(response["sum"].(float64)))

	var blockData map[float64][]map[string]interface{}

	// initialize blockData
	blockData = make(map[float64][]map[string]interface{})

	// run loop to get all the amount in rps in the response
	for _, rps := range response["rps"].([]interface{}) {
		amount := int64(rps.(map[string]interface{})["amount"].(float64))
		blockNumber := rps.(map[string]interface{})["block_number"].(float64)
		providerId := rps.(map[string]interface{})["provider_id"].(string)

		// check if the block number is present in the map an if not then create a new entry
		if _, ok := blockData[blockNumber]; !ok {
			blockData[blockNumber] = make([]map[string]interface{}, 0)
		}

		// append the provider id and amount to the block number
		blockData[blockNumber] = append(blockData[blockNumber], map[string]interface{}{
			"provider_id": providerId,
			"amount":      amount,
		})

		//blockData[blockNumber] = append(blockData[blockNumber], map[string]interface{}{
		//	"provider_id": providerId,
		//	"amount":      amount,
		//})

		//fmt.Println("Block Number : ", blockNumber)
		//fmt.Println("Provider ID : ", providerId)
		//fmt.Println("Amount : ", amount)
		//
		//fmt.Println("\n\n----------------------------------------------------")
	}

	for blockNumber, data := range blockData {
		fmt.Println("Block Number : ", blockNumber)

		for _, d := range data {
			fmt.Println("Provider ID : ", d["provider_id"])
			fmt.Println("Amount : ", d["amount"])
		}
		fmt.Println("\n\n----------------------------------------------------")
	}

	fmt.Println("Block Data : ", blockData)
}
