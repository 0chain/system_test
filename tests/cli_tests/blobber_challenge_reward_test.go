package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strconv"
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

	t.RunSequentiallyWithTimeout("Test Blobber Challenge Rewards", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// wallet balance before
		blobberOwnerWalletBalances, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)
		fmt.Println("blobberOwnerWalletBalance : ", blobberOwnerWalletBalances)
		blobberOwnerWalletBalance := blobberOwnerWalletBalances[0][9:14]

		// start block
		startBlock := getLatestFinalizedBlock(t)

		// convert blobberOwnerWalletBalance to float
		blobberOwnerWalletBalanceFloat, _ := strconv.ParseFloat(blobberOwnerWalletBalance, 64)
		fmt.Println("blobberOwnerWalletBalanceFloat : ", blobberOwnerWalletBalanceFloat)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   100 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

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

		//sleep for 30 seconds to allow the challenges to be created
		time.Sleep(30 * time.Second)

		// end block
		endBlock := getLatestFinalizedBlock(t)

		fmt.Println(startBlock, endBlock)

		blobberWallet, _ := getWalletForName(t, configPath, blobberOwnerWallet)

		fmt.Println(blobberWallet)

		//getBlockContainingTransactions(t, startBlock, endBlock, blobberWallet, "")

		fmt.Println("Client Wallet Balance : ")
		fmt.Println(getBalance(t, configPath))

		// list validators before
		output, err = listValidators(t, configPath, "--json")
		require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))

		//collect reward for each blobber
		for _, blobber := range blobberList {
			output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
				"provider_type": "blobber",
				"provider_id":   blobber.Id,
			}), blobberOwnerWallet, true)

			blobberWalletBalance, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)

			fmt.Println(blobber.Id, " > ", blobberWalletBalance)
		}

		fmt.Println("--------------------------------------------------")

		output, err = listValidators(t, configPath, "--json")
		require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))

		var validatorList []climodel.Validator
		json.Unmarshal([]byte(output[0]), &validatorList)

		fmt.Println(getBalanceForWallet(t, configPath, validatorOwnerWallet))

		// collect reward for each validator
		for _, validator := range validatorList {
			output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
				"provider_type": "validator",
				"provider_id":   validator.ID,
			}), validatorOwnerWallet, true)

			validatorWalletBalance, _ := getBalanceForWallet(t, configPath, validatorOwnerWallet)

			fmt.Println(validator.ID, " > ", validatorWalletBalance)
		}

		fmt.Println("--------------------------------------------------")

		blobberOwnerWalletBalancesAfter, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)
		fmt.Println("blobberOwnerWalletBalanceAfter : ", blobberOwnerWalletBalancesAfter)
		blobberOwnerWalletBalanceAfter := blobberOwnerWalletBalancesAfter[0][9:14]

		// convert blobberOwnerWalletBalanceAfter to float
		blobberOwnerWalletBalanceAfterFloat, _ := strconv.ParseFloat(blobberOwnerWalletBalanceAfter, 64)

		require.Greater(t, blobberOwnerWalletBalanceAfterFloat, blobberOwnerWalletBalanceFloat, "blobber wallet balance should be greater than before")
	})
}
