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

	t.RunSequentiallyWithTimeout("Test Blobber Challenge Rewards", (50*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		// wallet balance before
		blobberOwnerWalletBalances, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)
		blobberOwnerWalletBalance := blobberOwnerWalletBalances[0][9:14]

		// convert blobberOwnerWalletBalance to float
		blobberOwnerWalletBalanceFloat, _ := strconv.ParseFloat(blobberOwnerWalletBalance, 64)

		fmt.Println("blobberOwnerWalletBalance : ", blobberOwnerWalletBalanceFloat)

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 0.5 * MB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		//sleep for 1 minute to allow the challenges to be created
		time.Sleep(1 * time.Minute)

		// collect reward for each blobber
		for _, blobber := range blobberList {
			output, err = collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
				"provider_type": "blobber",
				"provider_id":   blobber.Id,
			}), blobberOwnerWallet, true)
		}

		// wallet balance after
		blobberOwnerWalletBalancesAfter, _ := getBalanceForWallet(t, configPath, blobberOwnerWallet)
		blobberOwnerWalletBalanceAfter := blobberOwnerWalletBalancesAfter[0][9:14]

		// convert blobberOwnerWalletBalanceAfter to float
		blobberOwnerWalletBalanceAfterFloat, _ := strconv.ParseFloat(blobberOwnerWalletBalanceAfter, 64)

		fmt.Println("blobberOwnerWalletBalanceAfter : ", blobberOwnerWalletBalanceAfterFloat)

		require.Greater(t, blobberOwnerWalletBalanceAfterFloat, blobberOwnerWalletBalanceFloat, "blobber wallet balance should be greater than before")
	})
}
