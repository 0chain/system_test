package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	var blobberList []climodel.BlobberInfo
	output, err := listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	t.RunSequentiallyWithTimeout("Test Cancel Allocation After Expiry Rewards", (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   1000 * MB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
			"expire": "5m",
		})

		time.Sleep(6 * time.Minute)

		//allocationID := "90a8400e8b544b9afe1a41d61480dd3c1e554477f2b0ade4712d7ecc4d054008"

		fmt.Println("Allocation ID : ", allocationID)

		cancelAllocation(t, configPath, allocationID, true)

		fmt.Println("Blobbers List : ", blobberList)

		var blobberWallet string
		count := 0

		if count > 0 {
			blobberWallet = blobber2Wallet
		} else {
			blobberWallet = blobber1Wallet
		}

		for _, blobber := range blobberList {
			collectRewardsForWallet(t, configPath, createParams(map[string]interface{}{
				"provider_type": "blobber",
				"provider_id":   blobber.Id,
			}), blobberWallet, true)

			blobberWalletBalance, _ := getBalanceForWallet(t, configPath, blobberWallet)

			fmt.Println(blobber.Id, " > ", blobberWalletBalance)
		}
	})
}
