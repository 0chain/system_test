package cli_tests

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestBlobberAvailability(testSetup *testing.T) {
	//t := test.NewSystemTest(testSetup)
	//t.SetSmokeTests("blobber is available switch controls blobber use for allocations")
	//
	//t.RunSequentially("blobber is available switch controls blobber use for allocations", func(t *test.SystemTest) {
	//	// create and faucet pour tokens
	//	output, err := createWallet(t, configPath)
	//	require.NoError(t, err, output)
	//
	//	// update blobber info use owner wallet, so need to faucet pour tokens
	//	output, err = createWalletForName(t, configPath, blobberOwnerWallet)
	//	require.NoError(t, err, output)
	//
	//	startBlobbers := getBlobbers(t)
	//	var blobberToDeactivate *model.BlobberDetails
	//	var activeBlobbers int
	//	for i := range startBlobbers {
	//		if startBlobbers[i].IsAvailable && !startBlobbers[i].IsKilled && !startBlobbers[i].IsShutdown {
	//			activeBlobbers++
	//			if blobberToDeactivate == nil {
	//				blobberToDeactivate = &startBlobbers[i]
	//			}
	//		}
	//	}
	//	require.NotEqual(t, blobberToDeactivate, "", "no active blobbers")
	//	require.True(t, activeBlobbers > 2, "need at least three active blobbers")
	//	dataShards := 1
	//	parityShards := activeBlobbers - dataShards
	//
	//	output, err = executeFaucetWithTokens(t, configPath, 9.0)
	//	require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))
	//
	//	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
	//		"data":   strconv.Itoa(dataShards),
	//		"parity": strconv.Itoa(parityShards),
	//		"lock":   "3.0",
	//		"expire": "1h",
	//		"size":   "10000",
	//	}))
	//	require.NoError(t, err, strings.Join(output, "\n"))
	//	beforeAllocationId, err := getAllocationID(output[0])
	//	require.NoError(t, err, "error getting allocation id")
	//	defer createAllocationTestTeardown(t, beforeAllocationId)
	//
	//	setAvailability(t, blobberToDeactivate.ID, false)
	//	t.Cleanup(func() { setAvailability(t, blobberToDeactivate.ID, true) })
	//	cliutil.Wait(t, 1*time.Second)
	//	betweenBlobbers := getBlobbers(t)
	//	for i := range betweenBlobbers {
	//		if betweenBlobbers[i].ID == blobberToDeactivate.ID {
	//			require.Falsef(t, betweenBlobbers[i].IsAvailable, "blobber %s should be deactivated", blobberToDeactivate.ID)
	//		}
	//	}
	//
	//	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
	//		"data":   strconv.Itoa(dataShards),
	//		"parity": strconv.Itoa(parityShards),
	//		"lock":   "3.0",
	//		"expire": "1h",
	//		"size":   "10000",
	//	}))
	//	require.Error(t, err, "create allocation should fail")
	//	require.Len(t, output, 1)
	//	require.True(t, strings.Contains(output[0], "not enough blobbers to honor the allocation"))
	//
	//	setAvailability(t, blobberToDeactivate.ID, true)
	//	cliutil.Wait(t, 1*time.Second)
	//	afterBlobbers := getBlobbers(t)
	//	for i := range betweenBlobbers {
	//		if afterBlobbers[i].ID == blobberToDeactivate.ID {
	//			require.Truef(t, afterBlobbers[i].IsAvailable, "blobber %s should be activated", blobberToDeactivate.ID)
	//		}
	//	}
	//
	//	output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
	//		"data":   strconv.Itoa(dataShards),
	//		"parity": strconv.Itoa(parityShards),
	//		"lock":   "3.0",
	//		"expire": "1h",
	//		"size":   "10000",
	//	}))
	//	require.NoError(t, err, strings.Join(output, "\n"))
	//	afterAllocationId, err := getAllocationID(output[0])
	//	require.NoError(t, err, "error getting allocation id")
	//	createAllocationTestTeardown(t, afterAllocationId)
	//})
}

func setAvailability(t *test.SystemTest, blobberId string, availability bool) {
	output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
		"blobber_id":   blobberId,
		"is_available": availability,
	}))
	require.NoError(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
}
