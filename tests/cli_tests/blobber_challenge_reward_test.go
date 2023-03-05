package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestBlobberChallengeRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	//output, err := registerWallet(t, configPath)
	//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))
	//
	var blobberList []climodel.BlobberInfo
	output, err := listBlobbers(t, configPath, "--json")
	//require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	//require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[3]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	t.RunSequentiallyWithTimeout("Test Blobber Challenge Rewards", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		count := 0

		//collect reward for each blobber
		for _, blobber := range blobberList {
			var blobberWallet string

			if count > 0 {
				blobberWallet = blobber2Wallet
			} else {
				blobberWallet = blobber1Wallet
			}
			count++

			output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
				"provider_type": "blobber",
				"provider_id":   blobber.Id,
			}), blobberWallet, true)

			blobberWalletBalance, _ := getBalanceForWallet(t, configPath, blobberWallet)

			fmt.Println(blobber.Id, " > ", blobberWalletBalance)
		}
		//
		//output, err := listValidators(t, configPath, "--json")
		//require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
		//
		//return
		//
		//var validatorList []climodel.Validator
		//json.Unmarshal([]byte(output[3]), &validatorList)
		//
		//fmt.Println("Validator List: ", output[3])
		//
		//fmt.Println(getBalanceForWallet(t, configPath, validatorOwnerWallet))
		//
		//// collect reward for each validator
		//for _, validator := range validatorList {
		//	output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
		//		"provider_type": "validator",
		//		"provider_id":   validator.ID,
		//	}), validatorOwnerWallet, true)
		//
		//	validatorWalletBalance, _ := getBalanceForWallet(t, configPath, validatorOwnerWallet)
		//
		//	fmt.Println(validator.ID, " > ", validatorWalletBalance)
		//}
		//
		//return
		//
		//output, err := registerWallet(t, configPath)
		//require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		//allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
		//	"size":   1000 * MB,
		//	"tokens": 1,
		//	"data":   1,
		//	"parity": 1,
		//	"expire": "5m",
		//})
		//
		//allocationID := "90a8400e8b544b9afe1a41d61480dd3c1e554477f2b0ade4712d7ecc4d054008"
		//
		//fmt.Println("Allocation ID : ", allocationID)
		//
		//cancelAllocation(t, configPath, allocationID, true)

		return
		//
		//// wallet balance before
		//blobberOwnerWalletBalances, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)
		//fmt.Println("blobberOwnerWalletBalance : ", blobberOwnerWalletBalances)
		//blobberOwnerWalletBalance := blobberOwnerWalletBalances[0][9:14]
		//
		//// start block
		//startBlock := getLatestFinalizedBlock(t)
		//
		//// convert blobberOwnerWalletBalance to float
		//blobberOwnerWalletBalanceFloat, _ := strconv.ParseFloat(blobberOwnerWalletBalance, 64)
		//fmt.Println("blobberOwnerWalletBalanceFloat : ", blobberOwnerWalletBalanceFloat)
		//
		//allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
		//	"size":   100 * MB,
		//	"tokens": 1,
		//	"data":   1,
		//	"parity": 1,
		//	"expire": "5m",
		//})
		//
		//remotepath := "/dir/"
		//filesize := 30 * MB
		//filename := generateRandomTestFileName(t)
		//
		//err = createFileWithSize(filename, int64(filesize))
		//require.Nil(t, err)
		//
		//output, err = uploadFile(t, configPath, map[string]interface{}{
		//	// fetch the latest block in the chain
		//	"allocation": allocationId,
		//	"remotepath": remotepath + filepath.Base(filename),
		//	"localpath":  filename,
		//}, true)
		//require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))
		//
		////sleep for 60 seconds to allow the challenges to be created
		//time.Sleep(30 * time.Second)
		//
		//// end block
		//endBlock := getLatestFinalizedBlock(t)
		//
		//fmt.Println(startBlock, endBlock)
		//
		//blobberWallet, _ := getWalletForName(t, configPath, blobberOwnerWallet)
		//
		//fmt.Println(blobberWallet)
		//
		////getBlockContainingTransactions(t, startBlock, endBlock, blobberWallet, "")
		//
		//fmt.Println("Client Wallet Balance : ")
		//fmt.Println(getBalance(t, configPath))
		//
		//// list validators before
		//output, err = listValidators(t, configPath, "--json")
		//require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
		//
		////collect reward for each blobber
		//for _, blobber := range blobberList {
		//	//output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
		//	//	"provider_type": "blobber",
		//	//	"provider_id":   blobber.Id,
		//	//}), blobberOwnerWallet, true)
		//
		//	blobberWalletBalance, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)
		//
		//	fmt.Println(blobber.Id, " > ", blobberWalletBalance)
		//}
		//
		//fmt.Println("--------------------------------------------------")
		//
		//output, err = listValidators(t, configPath, "--json")
		//require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
		//
		//var validatorList []climodel.Validator
		//json.Unmarshal([]byte(output[0]), &validatorList)
		//
		//fmt.Println(getBalanceForWallet(t, configPath, validatorOwnerWallet))
		//
		//// collect reward for each validator
		//for _, validator := range validatorList {
		//	//output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
		//	//	"provider_type": "validator",
		//	//	"provider_id":   validator.ID,
		//	//}), validatorOwnerWallet, true)
		//
		//	validatorWalletBalance, _ := getBalanceForWallet(t, configPath, validatorOwnerWallet)
		//
		//	fmt.Println(validator.ID, " > ", validatorWalletBalance)
		//}
		//
		//fmt.Println("--------------------------------------------------")
		//
		//blobberOwnerWalletBalancesAfter, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)
		//fmt.Println("blobberOwnerWalletBalanceAfter : ", blobberOwnerWalletBalancesAfter)
		//blobberOwnerWalletBalanceAfter := blobberOwnerWalletBalancesAfter[0][9:14]
		//
		//// convert blobberOwnerWalletBalanceAfter to float
		//blobberOwnerWalletBalanceAfterFloat, _ := strconv.ParseFloat(blobberOwnerWalletBalanceAfter, 64)
		//
		//require.Greater(t, blobberOwnerWalletBalanceAfterFloat, blobberOwnerWalletBalanceFloat, "blobber wallet balance should be greater than before")
	})
}
