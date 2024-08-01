package tokenomics_tests

import (
	"crypto/rand"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/wait"
	"github.com/stretchr/testify/require"
)

func TestReplaceEnterpriseBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// Change time unit to 10 minutes

	// Try replacing blobber with 2x price, 0.5x price and same price. Check cost in all scenarios.

	t.RunSequentially("Replace blobber in allocation, should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxSuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.True(t, isBlobberExist(newBlobberID, allocation.Blobbers))
	})

	t.RunSequentially("Replace blobber with the same one in allocation, shouldn't work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbers(t, wallet, oldBlobberID, oldBlobberID, allocationID, client.TxUnsuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.RunSequentially("Replace blobber with incorrect blobber ID of an old blobber, shouldn't work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		sdkClient.SetWallet(t, wallet)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "Old blobber ID contains zero value")

		result, err := rand.Int(rand.Reader, big.NewInt(10))
		require.Nil(t, err)

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, result.String(), allocationID, client.TxUnsuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.RunSequentially("Check token accounting of a blobber replacing in allocation, should work", func(t *test.SystemTest) {
		wallet := createWallet(t)

		blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBeforeAllocationUpdate := walletBalance.Balance

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxSuccessfulStatus)

		var numberOfBlobbersAfter int

		wait.PoolImmediately(t, time.Second*30, func() bool {
			allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			numberOfBlobbersAfter = len(allocation.Blobbers)

			return numberOfBlobbersAfter == numberOfBlobbersBefore
		})

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfterAllocationUpdate := walletBalance.Balance

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.Greater(t, balanceBeforeAllocationUpdate, balanceAfterAllocationUpdate)
	})

	//t.RunSequentiallyWithTimeout("Replace blobber in allocation with repair should work", 90*time.Second, func(t *test.SystemTest) {
	//	wallet := createWallet(t)
	//
	//	sdkClient.SetWallet(t, wallet)
	//
	//	blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
	//	allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
	//	allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
	//
	//	uploadOp := sdkClient.AddUploadOperation(t, "", "")
	//	sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{uploadOp})
	//
	//	allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	//	apiClient.CreateReadPool(t, wallet, 1.0, client.TxSuccessfulStatus)
	//
	//	oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
	//	require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")
	//	newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
	//	require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
	//	apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxSuccessfulStatus)
	//
	//	time.Sleep(10 * time.Second)
	//
	//	alloc, err := sdk.GetAllocation(allocationID)
	//	require.Nil(t, err)
	//	// Check for blobber replacement
	//	notFound := true
	//	require.True(t, len(alloc.Blobbers) > 0)
	//	// Check if new blobber is in the same position as old blobber
	//	require.True(t, alloc.Blobbers[0].ID == newBlobberID)
	//	for _, blobber := range alloc.Blobbers {
	//		if blobber.ID == oldBlobberID {
	//			notFound = false
	//			break
	//		}
	//	}
	//	require.True(t, notFound, "old blobber should not be in the list")
	//	// check for repair
	//	_, _, req, _, err := alloc.RepairRequired("/")
	//	require.Nil(t, err)
	//	require.True(t, req)
	//
	//	// do repair
	//	sdkClient.RepairAllocation(t, allocationID)
	//
	//	_, err = sdk.GetFileRefFromBlobber(allocationID, newBlobberID, uploadOp.RemotePath)
	//	require.Nil(t, err)
	//})
}

func createWallet(t *test.SystemTest) *model.Wallet {
	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error creating wallet", strings.Join(output, "\n"))

	wallet, err := utils.GetFullWalletForName(t, configPath, utils.EscapedTestName(t))
	require.Nil(t, err, "Error getting created wallet")
	require.NotNil(t, wallet, "Error empty wallet")

	//Execute faucet
	output, err = utils.ExecuteFaucetWithTokens(t, configPath, 1000)
	require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

	//TODO: fix this to properly format the wallet.
	return &model.Wallet{}
}

func getNotUsedStorageNodeID(availableStorageNodeIDs *[]string, usedStorageNodes []*model.StorageNode) string {
	for _, availableStorageNodeID := range *availableStorageNodeIDs {
		var found bool
		for _, usedStorageNode := range usedStorageNodes {
			if usedStorageNode.ID == availableStorageNodeID {
				found = true
			}
		}
		if !found {
			return availableStorageNodeID
		}
	}
	return ""
}

func getFirstUsedStorageNodeID(availableStorageNodeIDs *[]string, usedStorageNodes []*model.StorageNode) string {
	for _, availableStorageNodeID := range *availableStorageNodeIDs {
		for _, usedStorageNode := range usedStorageNodes {
			if usedStorageNode.ID == availableStorageNodeID {
				return availableStorageNodeID
			}
		}
	}
	return ""
}

func getBlobberURL(blobberID string, blobbers []*model.StorageNode) string {
	for _, blobber := range blobbers {
		if blobber.ID == blobberID {
			return blobber.BaseURL
		}
	}
	return ""
}

func isBlobberExist(blobberID string, blobbers []*model.StorageNode) bool {
	return getBlobberURL(blobberID, blobbers) != ""
}
