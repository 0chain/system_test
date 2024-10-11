package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestUpdateBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Update blobber in allocation without correct delegated client, shouldn't work")

	t.Parallel()

	t.Run("update blobber version should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		require.NotEqual(t, wallet.Id, blobber.StakePoolSettings.DelegateWallet)

		blobber.StorageVersion = 2

		apiClient.UpdateBlobber(t, wallet, blobber, client.TxUnsuccessfulStatus)
	})

	t.Run("update blobber: degrade version should not work", func(t *test.SystemTest) {
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

		apiClient.UpdateBlobber(t, wallet, blobber, client.TxUnsuccessfulStatus)
	})

	t.Run("Update blobber in allocation without correct delegated client, shouldn't work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)

		blobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, blobberID)

		blobber := apiClient.GetBlobber(t, blobberID, client.HttpOkStatus)
		require.NotEqual(t, wallet.Id, blobber.StakePoolSettings.DelegateWallet)

		apiClient.UpdateBlobber(t, wallet, blobber, client.TxUnsuccessfulStatus)
	})
}
