package cli_tests

import (
	"strconv"
	"strings"
	"testing"
	"time"

	cliutil "github.com/0chain/system_test/internal/cli/util"

	"github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestBlobberAvailability(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("blobber is available switch controls blobber use for allocations")

	t.RunSequentially("blobber is available switch controls blobber use for allocations", func(t *test.SystemTest) {
		// create and faucet pour tokens
		output, err := createWallet(t, configPath)
		require.NoError(t, err, output)

		// update blobber info use owner wallet, so need to faucet pour tokens
		output, err = createWalletForName(t, configPath, blobberOwnerWallet)
		require.NoError(t, err, output)

		startBlobbers := getBlobbers(t)
		var blobberToDeactivate *model.BlobberDetails
		var activeBlobbers int
		for i := range startBlobbers {
			if !startBlobbers[i].NotAvailable && !startBlobbers[i].IsKilled && !startBlobbers[i].IsShutdown {
				activeBlobbers++
				if blobberToDeactivate == nil {
					blobberToDeactivate = &startBlobbers[i]
				}
			}
		}
		require.NotEqual(t, blobberToDeactivate, "", "no active blobbers")
		require.True(t, activeBlobbers > 2, "need at least three active blobbers")
		dataShards := 1
		parityShards := activeBlobbers - dataShards

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   "3.0",
			"size":   "10000",
		}))
		require.NoError(t, err, strings.Join(output, "\n"))
		beforeAllocationId, err := getAllocationID(output[0])
		require.NoError(t, err, "error getting allocation id")
		beforeAllocation := getAllocation(t, beforeAllocationId)

		setNotAvailability(t, blobberToDeactivate.ID, true)
		t.Cleanup(func() { setNotAvailability(t, blobberToDeactivate.ID, false) })
		cliutil.Wait(t, 1*time.Second)
		betweenBlobbers := getBlobbers(t)
		for i := range betweenBlobbers {
			if betweenBlobbers[i].ID == blobberToDeactivate.ID {
				require.Falsef(t, !betweenBlobbers[i].NotAvailable, "blobber %s should be deactivated", blobberToDeactivate.ID)
			}
		}

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   "3.0",
			"size":   "10000",
		}))
		require.Error(t, err, "create allocation should fail")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "not enough blobbers to honor the allocation"))

		output, err = updateAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": beforeAllocationId,
			"extend":     true,
		}), true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))

		afterAlloc := getAllocation(t, beforeAllocationId)
		require.Greater(t, afterAlloc.ExpirationDate, beforeAllocation.ExpirationDate)
		createAllocationTestTeardown(t, beforeAllocationId)

		setNotAvailability(t, blobberToDeactivate.ID, false)
		cliutil.Wait(t, 1*time.Second)
		afterBlobbers := getBlobbers(t)
		for i := range betweenBlobbers {
			if afterBlobbers[i].ID == blobberToDeactivate.ID {
				require.Truef(t, !afterBlobbers[i].NotAvailable, "blobber %s should be activated", blobberToDeactivate.ID)
			}
		}

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":   strconv.Itoa(dataShards),
			"parity": strconv.Itoa(parityShards),
			"lock":   "3.0",
			"size":   "10000",
		}))
		require.NoError(t, err, strings.Join(output, "\n"))
		afterAllocationId, err := getAllocationID(output[0])
		require.NoError(t, err, "error getting allocation id")
		createAllocationTestTeardown(t, afterAllocationId)
	})
}

func setNotAvailability(t *test.SystemTest, blobberId string, availability bool) {
	output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
		"blobber_id":    blobberId,
		"not_available": availability,
	}))
	require.NoError(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
}
