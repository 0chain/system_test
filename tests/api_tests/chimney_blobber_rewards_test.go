package api_tests

import (
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestChimneyBlobberRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Replace blobber in allocation, should work")

	chimneyClient.ExecuteFaucetWithTokens(t, sdkWallet, 9000, client.TxSuccessfulStatus)

	allBlobbers, resp, err := chimneyClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	for _, blobber := range allBlobbers {
		// stake tokens to this blobber
		chimneyClient.CreateStakePool(t, sdkWallet, 3, blobber.ID, client.TxSuccessfulStatus)
	}

	lenBlobbers := int64(len(allBlobbers))

	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	blobberRequirements.DataShards = (lenBlobbers + 1) / 2
	blobberRequirements.ParityShards = lenBlobbers / 2
	blobberRequirements.Size = 107374182400

	allocationBlobbers := chimneyClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
	allocationID := chimneyClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

	uploadOp := sdkClient.AddUploadOperation(t, allocationID, 1024)
	sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})

	time.Sleep(20 * time.Minute)

	chimneyClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

	time.Sleep(10 * time.Second)

	alloc := chimneyClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	require.Equal(testSetup, true, alloc.Cancelled)
	require.Equal(testSetup, true, alloc.Finalized)

	t.RunWithTimeout("Chimney blobber rewards", 1*time.Hour, func(t *test.SystemTest) {
		allocCreatedAt := alloc.StartTime
		allocExpiredAt := alloc.Expiration

		allocDuration := allocExpiredAt - allocCreatedAt
		durationInTimeUnits := allocDuration / int64(alloc.TimeUnit)
		t.Logf("Alloc duration: %v", durationInTimeUnits)

	})
}

func durationInTimeUnits(dur, timeUnit int64) (float64, error) {
	return float64(dur) / float64(timeUnit), nil
}
