package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestUpdateBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Update blobber in allocation without correct delegated client, shouldn't work")

	t.Parallel()

	t.RunWithTimeout("update blobber version should work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		require.NotEqual(t, wallet.Id, blobber.StakePoolSettings.DelegateWallet)

		blobber.StorageVersion = 1

		apiClient.FundWallet(t, wallet, 1.0, client.TxSuccessfulStatus)
		apiClient.UpdateBlobber(t, wallet, blobber, client.TxUnsuccessfulStatus)

		blobber = apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		require.NotEqual(t, wallet.Id, blobber.StakePoolSettings.DelegateWallet)

		require.Equal(t, int64(1), blobber.StorageVersion)
	})

	t.RunWithTimeout("update blobber: degrade version should not work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)

		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		require.NotEqual(t, wallet.Id, blobber.StakePoolSettings.DelegateWallet)

		blobber.StorageVersion = 0

		apiClient.FundWallet(t, wallet, 1.0, client.TxSuccessfulStatus)
		apiClient.UpdateBlobber(t, wallet, blobber, client.TxUnsuccessfulStatus)

		blobber = apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		require.NotEqual(t, wallet.Id, blobber.StakePoolSettings.DelegateWallet)

		require.Equal(t, int64(1), blobber.StorageVersion)
	})

	t.RunWithTimeout("Update blobber in allocation without correct delegated client, shouldn't work", 10*time.Minute, func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		require.NotEqual(t, wallet.Id, blobber.StakePoolSettings.DelegateWallet)

		apiClient.FundWallet(t, wallet, 1.0, client.TxSuccessfulStatus)
		apiClient.UpdateBlobber(t, wallet, blobber, client.TxUnsuccessfulStatus)
	})
}
