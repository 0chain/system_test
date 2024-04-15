package api_tests

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0boxNftCollection(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("List NFT collection id with zero nft collection id  should work")

	var firebaseToken *model.FirebaseToken
	t.TestSetup("Autenticate with firebase", func() {
		firebaseToken = authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)
	})

	defaultBatchSize := 1
	defaultMaxMint := 1
	defaultCurrMint := 1
	defaultPricePerPack := 1
	defaultTotalNFT := "1"
	collection_id := "collectionId1"

	t.RunSequentially("List NFT collection id with zero nft collection id  should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		// require.Equal(t, description, zboxWallet.Description, "Description does not match expected") // FIXME: Description is not persisted see: https://github.com/0chain/0box/issues/377

		zboxNftCollectionIdList, response, err := zboxClient.GetAllNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftCollectionIdList)
		require.Equal(t, zboxNftCollectionIdList.NftCollectionCount, 0)
	})

	t.RunSequentially("Get NFT by collection with invalid collection id should give empty array", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		_, response, err = zboxClient.GetNftCollectionById(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"invalid collection_id",
		)

		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, `{"error":"400: error getting nft collection"}`, response.String())
	})

	t.RunSequentially("Get NFT collection id with one nft collection id present should should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		// require.Equal(t, description, zboxWallet.Description, "Description does not match expected") // FIXME: Description is not persisted see: https://github.com/0chain/0box/issues/377

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationId := "allocationid1"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, "", allocationObjCreatedResponse.ID)

		collection_name := "collection"
		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
			"auth_ticket1",
			defaultTotalNFT,
			"collection_type",
			allocationId,
			"base_url",
			"symbol",
			defaultPricePerPack,
			defaultMaxMint,
			defaultCurrMint,
			defaultBatchSize,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftCollectionId)

		zboxNftCollectionIdList, response, err := zboxClient.GetAllNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.Equal(t, zboxNftCollectionIdList.NftCollectionCount, 1)
	})

	t.RunSequentially("Get NFT collection by collection id with one nft present should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		zboxNFTList, response, err := zboxClient.GetNftCollectionById(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			collection_id,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, zboxNFTList)
		require.Equal(t, zboxNFTList.NftCollection.CollectionId, collection_id)
	})
}

func Test0boxNft(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get NFT collection with zero nft collection should work")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	defaultBatchSize := 1
	defaultMaxMint := 1
	defaultCurrMint := 1
	defaultPricePerPack := 1
	defaultTotalNFT := "1"
	defaultCollectionId := "default collection id for testing"

	t.RunSequentially("Get NFT collection with zero nft collection should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		zboxNftCollectionId, response, err := zboxClient.GetAllNft(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftCollectionId)
	})

	t.RunSequentially("Get NFT collection by collection id with zero nft collection should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationId := "allocationid2"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, "", allocationObjCreatedResponse.ID)

		collection_name := "collection"
		collection_id := "collectionId2"

		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
			"auth_ticket2",
			defaultTotalNFT,
			"collection_type",
			allocationId,
			"base_url",
			"symbol",
			defaultPricePerPack,
			defaultMaxMint,
			defaultCurrMint,
			defaultBatchSize,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftCollectionId)

		zboxNftByCollectionId, response, err := zboxClient.GetAllNftByCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			collection_id,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftByCollectionId)
	})

	t.RunSequentially("Get NFT collection by wallet id with zero nft collection should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		zboxNftByCollectionId, response, err := zboxClient.GetAllNftByWalletId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			strconv.Itoa(zboxWallet.WalletId),
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftByCollectionId)
	})

	t.RunSequentially("Post NFT collection with valid arguments should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationId := "allocationid3"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, "", allocationObjCreatedResponse.ID)

		collection_name := "collection"

		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			defaultCollectionId,
			"auth_ticket3",
			defaultTotalNFT,
			"collection_type",
			allocationId,
			"base_url",
			"symbol",
			defaultPricePerPack,
			defaultMaxMint,
			defaultCurrMint,
			defaultBatchSize,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftCollectionId)

		zboxNft, response, err := zboxClient.PostNftCollection(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"stage_nft_upload",
			"nft_reference",
			zboxNftCollectionId.CollectionId,
			"auth_ticket4",
			"owned_by",
			"nft_activity",
			"meta_data",
			allocationId,
			"created_by",
			"contract_Address",
			"token_id",
			"token_standard",
			"tx_hash",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNft)
	})

	t.RunSequentially("Get NFT by collection with invalid collection id should give empty array", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		zboxNftCollectionId, response, err := zboxClient.GetAllNftByCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"invalid collection_id",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, zboxNftCollectionId.NftList, 0)
		require.Equal(t, 0, zboxNftCollectionId.NftCount)
	})

	t.RunSequentially("Get NFT collection by wallet id should give empty array for invalid wallet id", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		zboxNftByWalletId, response, err := zboxClient.GetAllNftByWalletId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"invalid wallet id",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, zboxNftByWalletId.NftList, 0)
		require.Equal(t, 0, zboxNftByWalletId.NftCount)
	})

	t.RunSequentially("Get NFT collection with one nft present should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		zboxNFTList, response, err := zboxClient.GetAllNft(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, zboxNFTList)
	})

	t.RunSequentially("Get NFT collection by wallet id with one nft present should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		wallet, _, _ := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp") // This endpoint used instead of list wallet as list wallet doesn't return the required data
		zboxNFTList, response, err := zboxClient.GetAllNftByWalletId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			strconv.Itoa(wallet.WalletId),
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, zboxNFTList)
	})

	t.RunSequentially("Get NFT collection by collection id with one nft present should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		zboxNFTList, response, err := zboxClient.GetAllNftByCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			defaultCollectionId,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, zboxNFTList)
	})

	t.RunSequentially("Post NFT collection with Invalid collectionId should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationId := "allocationid4"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, "", allocationObjCreatedResponse.ID)

		_, response, err = zboxClient.PostNftCollection(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"stage_nft_upload",
			"nft_reference",
			"invalid_collection_id",
			"auth_ticket5",
			"owned_by",
			"nft_activity",
			"meta_data",
			allocationId,
			"created_by",
			"contract_Address",
			"token_id",
			"token_standard",
			"tx_hash",
		)
		errMssg := `{"error":"400: collectionID not valid"}`
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, errMssg, response.String())
	})

	t.RunSequentially("Post NFT collection with Invalid allocation Id should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationId := "allocationid5"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, "", allocationObjCreatedResponse.ID)

		collection_name := "collection"
		collection_id := "collectionId3"
		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
			"auth_ticket6",
			defaultTotalNFT,
			"collection_type",
			allocationId,
			"base_url",
			"symbol",
			defaultPricePerPack,
			defaultMaxMint,
			defaultCurrMint,
			defaultBatchSize,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftCollectionId)

		allocationId = "allocationId is being changed here"
		_, response, err = zboxClient.PostNftCollection(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"stage_nft_upload",
			"nft_reference",
			zboxNftCollectionId.CollectionId,
			"auth_ticket7",
			"owned_by",
			"nft_activity",
			"meta_data",
			allocationId,
			"created_by",
			"contract_Address",
			"token_id",
			"token_standard",
			"tx_hash",
		)

		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, `{"error":"400: allocationID not valid"}`, response.String())
	})

	t.RunSequentially("Update NFT collection with valid argument should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "direct_storage"
		allocationId := "allocationid6"
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			allocationId,
			allocationName,
			allocationDescription,
			allocationType,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, "", allocationObjCreatedResponse.ID)

		collection_name := "collection"
		collection_id := "collectionId4"

		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
			"auth_ticket8",
			defaultTotalNFT,
			"collection_type",
			allocationId,
			"base_url",
			"symbol",
			defaultPricePerPack,
			defaultMaxMint,
			defaultCurrMint,
			defaultBatchSize,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNftCollectionId)

		zboxNft, response, err := zboxClient.PostNftCollection(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"stage_nft_upload",
			"nft_reference",
			zboxNftCollectionId.CollectionId,
			"auth_ticket9",
			"owned_by",
			"nft_activity",
			"meta_data",
			allocationId,
			"created_by",
			"contract_Address",
			"token_id",
			"token_standard",
			"tx_hash",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxNft)

		zboxNftUpdated, response, err := zboxClient.UpdateNftCollection(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
			"auth_ticket10",
			defaultTotalNFT,
			"collection_type",
			allocationId,
			"base_url",
			"symbol",
			zboxNft.Id,
			defaultPricePerPack,
			defaultMaxMint,
			defaultCurrMint,
			defaultBatchSize,
		)
		require.NoError(t, err)
		require.NotNil(t, zboxNftUpdated)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Update NFT collection with missing params should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected") // FIXME: Description is not persisted see: https://github.com/0chain/0box/issues/377

		_, response, err = zboxClient.UpdateNftCollection(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			"invalid_name",
			"invalid_collection_id",
			"auth_ticket11",
			defaultTotalNFT,
			"collection_type",
			"invalid_alloc",
			"base_url",
			"symbol",
			390,
			defaultPricePerPack,
			defaultMaxMint,
			defaultCurrMint,
			defaultBatchSize,
		)
		errMssg := `{"error":"400: allocationID not valid"}`
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, errMssg, response.String())
	})
}

func Test0Box_share_info(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Post ShareInfo with correct AuthTicket should work properly")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Post ShareInfo with correct AuthTicket should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAuthTicket,
		)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Nil(t, err)
		require.NotNil(t, shareInfoDeletionMssg)

		shareMessage := "Massege created as a part of " + t.Name()
		fromInfo := "FromInfo created as a part of " + t.Name()
		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
			zboxClient.DefaultAuthTicket,
			shareMessage,
			fromInfo,
			zboxClient.DefaultRecieverId,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, "shareinfo added successfully", shareInfoSuccessMssg.Message)

		shareInfoData, response, err := zboxClient.GetShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoData)
		require.Equal(t, shareInfoData.Message, "Data is present for the given details")
		require.Len(t, shareInfoData.Data, 1)
		require.Equal(t, shareInfoData.Data[0].Message, shareMessage)
		require.Equal(t, shareInfoData.Data[0].Receiver, zboxClient.DefaultRecieverId)
	})

	t.RunSequentially("Post ShareInfo with Incorrect AuthTicket should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAuthTicket,
		)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Nil(t, err)
		require.NotNil(t, shareInfoDeletionMssg)

		shareMessage := "Massege created as a part of " + t.Name()
		fromInfo := "FromInfo created as a part of " + t.Name()
		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
			"abc",
			shareMessage,
			fromInfo,
			zboxClient.DefaultRecieverId,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		errorString := `{"error":"illegal base64 data at input byte 0"}`
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, shareInfoSuccessMssg.Message, "")
		require.Equal(t, errorString, response.String())
	})

	t.RunSequentially("Delete ShareInfo without AUthTicket should not work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAuthTicket,
		)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Nil(t, err)
		require.NotNil(t, shareInfoDeletionMssg)

		shareMessage := "Massege created as a part of " + t.Name()
		fromInfo := "FromInfo created as a part of " + t.Name()
		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
			zboxClient.DefaultAuthTicket,
			shareMessage,
			fromInfo,
			zboxClient.DefaultRecieverId,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, "shareinfo added successfully", shareInfoSuccessMssg.Message, "Error adding ShareInfo")

		shareInfoDeletionMssg, response, err = zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"",
		)
		errorString := `{"error":"invalid params: pass atleast one of lookuphash or authticket"}`
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, shareInfoDeletionMssg.Message, "")
		require.Equal(t, errorString, response.String())
	})

	t.RunSequentially("Delete ShareInfo with correct parameter should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		shareInfoDeletionMssg, response, err := zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAuthTicket,
		)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Nil(t, err)
		require.NotNil(t, shareInfoDeletionMssg)

		shareMessage := "Massege created as a part of " + t.Name()
		fromInfo := "FromInfo created as a part of " + t.Name()
		shareInfoSuccessMssg, response, err := zboxClient.PostShareInfo(t,
			zboxClient.DefaultAuthTicket,
			shareMessage,
			fromInfo,
			zboxClient.DefaultRecieverId,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, "shareinfo added successfully", shareInfoSuccessMssg.Message, "Error adding shareInfo")

		shareInfoDeletionMssg, response, err = zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAuthTicket,
		)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Nil(t, err)
		require.NotNil(t, shareInfoDeletionMssg)
		require.Equal(t, shareInfoDeletionMssg.Message, "deleting shareinfo successful", "Error deleting ShareInfo")
	})
}

func Test0Box(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create a wallet with valid phone number should work")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Create a wallet with valid phone number should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")
	})

	t.RunSequentially("List wallet should work with zero initialisedWallets", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		wallets, _, err := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 0, len(wallets.Data), "wallet list should be empty")
	})

	t.RunSequentially("List wallet should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		_, response, err = zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		wallets, response, err := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, wallets)
		require.Equal(t, 1, len(wallets.Data), "Expected 1 wallet only to be present")
	})

	t.RunSequentially("Phone exists should work with existing phone number", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", "userName")
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, "userName", zboxOwner.UserName, "owner name does not match expected")

		data, response, err := zboxClient.CheckPhoneExists(t, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, true, data.Exists, "Expected phone number to exist")
	})

	t.RunSequentially("Phone exists check should return error with non-existing phone number", func(t *test.SystemTest) {
		phoneNumber := fmt.Sprintf("%s%d", zboxClient.DefaultPhoneNumber, 0)
		teardown(t, firebaseToken.IdToken, phoneNumber)
		csrfToken := createCsrfToken(t, phoneNumber)

		data, response, err := zboxClient.CheckPhoneExists(t, csrfToken, phoneNumber)
		require.NoError(t, err)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, false, data.Exists, "Expected phone number to not exist")
	})

	t.RunSequentially("Wallet exists should work with zero wallet", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		walletName := "wallet_name"

		data, response, err := zboxClient.CheckWalletExists(t, walletName, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, false, data.Exists, "Expected wallet to not exist")
	})

	t.RunSequentially("Wallet exists should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		walletName := "wallet_name"
		description := "wallet created as part of " + t.Name()
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		_, response, err = zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		t.Logf("Should return true when wallet exists")
		data, response, err := zboxClient.CheckWalletExists(t, walletName, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, true, data.Exists, "Expected wallet to exist")
	})
}

func Test0BoxFCM(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Creating FCM Token with valid credentials should work")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	teardownFCM(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Creating FCM Token with valid credentials should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", "userName")
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, "userName", zboxOwner.UserName, "owner name does not match expected")

		response, err = zboxClient.CreateFCMToken(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Creating FCM Token with existing credentials should fail", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", "userName")
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, "userName", zboxOwner.UserName, "owner name does not match expected")

		response, err = zboxClient.CreateFCMToken(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Updating FCM Token should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		data, response, err := zboxClient.UpdateFCMToken(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, data, "response object should not be nil")
	})

	t.RunSequentially("Deleting FCM Token should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		data, response, err := zboxClient.DeleteFCMToken(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, data, "response object should not be nil")
		require.Equal(t, "Firebase token register deleted successfully", data.Message, "response object should match input")
	})
}

func Test0BoxWallet(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Get wallet keys should work with wallet present")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Get wallet keys should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		_, response, err = zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		zboxWalletKeys, response, err := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWalletKeys)
		require.NotEqual(t, 0, len(response.String()), "Response body is empty")
	})

	t.RunSequentially("Get wallet keys should not work with wrong phone number ", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		_, response, err := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Delete Wallet should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// Create Wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		_, response, err = zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Get Wallet
		wallet, _, _ := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")

		// Delete Wallet
		_, response, err = zboxClient.DeleteWallet(t, wallet.WalletId, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		var responseJson map[string]interface{}
		err = json.Unmarshal([]byte(response.String()), &responseJson)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "Wallet info deleted successfully", responseJson["message"], "Response message does not match expected. Output: [%v]", response.String())

		// Get Wallet
		_, response, err = zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Update Wallet with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// Create Wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		wallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Update Wallet
		_, response, err = zboxClient.UpdateWallet(t, wallet.Mnemonic, "new_wallet_name", "new_wallet_description", firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Get Wallet
		_, resp, _ := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		var walletList model.ZboxWalletList

		// store data to responseJson and read and println it
		_ = json.Unmarshal([]byte(resp.String()), &walletList)

		wallets := walletList.Data
		require.Equal(t, 1, len(wallets), "Wallet not updated")
		newWallet := wallets[0]
		require.Equal(t, "new_wallet_name", newWallet.Name, "Wallet name not updated")
		require.Equal(t, "new_wallet_description", newWallet.Description, "Wallet description not updated")
	})
}

func TestDexState(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create a DEX state with valid phone number should work")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	postData := map[string]string{
		"stage":     "burn",
		"reference": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
	}

	updateData := map[string]string{
		"stage":     "burn",
		"reference": "{\"test_2\":\"test1\", \"test4\":\"test3\"}",
	}

	// POST DEX STATE
	t.RunSequentially("Create a DEX state with valid phone number should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		dexState, response, err := zboxClient.PostDexState(t,
			postData,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, dexState)
	})

	t.RunSequentially("Create a DEX state with invalid csrf token should fail", func(t *test.SystemTest) {
		dexState, response, err := zboxClient.PostDexState(t,
			postData,
			firebaseToken.IdToken,
			"abcd",
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Empty(t, dexState)
	})

	t.RunSequentially("Create a DEX state with invalid field should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		wrongData := map[string]string{
			"stage":        "burn",
			"refe3r72t981": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
		}

		dexState, response, err := zboxClient.PostDexState(t,
			wrongData,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Empty(t, dexState)
	})

	t.RunSequentially("Create a DEX state 2 times with same phone number should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PostDexState(t,
			postData,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Empty(t, dexState)
	})

	// GET DEX STATE
	t.RunSequentially("Get DEX state with valid phone number should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		data := map[string]string{
			"stage":     "burn",
			"reference": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
		}

		_, response, err = zboxClient.PostDexState(t,
			data,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		dexState, response, err := zboxClient.GetDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, dexState)
		require.Equal(t, postData["stage"], dexState.Stage)
		require.Equal(t, postData["reference"], dexState.Reference)
	})

	// UPDATE DEX STATE
	t.RunSequentially("Update DEX state with valid phone number should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		userName := "user_name"

		zboxOwner, response, err := zboxClient.PostOwner(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp", userName)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxOwner)
		require.Equal(t, userName, zboxOwner.UserName, "owner name does not match expected")

		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
			userName,
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		data := map[string]string{
			"stage":     "burn",
			"reference": "{\"test_1\":\"test2\", \"test3\":\"tes4\"}",
		}

		_, response, err = zboxClient.PostDexState(t,
			data,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		dexState, response, err := zboxClient.GetDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, postData["reference"], dexState.Reference)

		dexState, response, err = zboxClient.PutDexState(t,
			updateData,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, dexState)

		// get dex state
		dexState, response, err = zboxClient.GetDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, updateData["reference"], dexState.Reference)
	})

	t.RunSequentially("Update DEX state with invalid data should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PutDexState(t,
			map[string]string{
				"stage": "burn",
			},
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Empty(t, dexState)
	})
}

func teardown(t *test.SystemTest, idToken, phoneNumber string) {
	t.Logf("Tearing down existing test data for [%v]", phoneNumber)
	csrfToken := createCsrfToken(t, phoneNumber)

	var clientId string
	var clientKey string
	var clientSignature string
	if phoneNumber == zboxClient.DefaultPhoneNumber {
		clientId = X_APP_CLIENT_ID
		clientKey = X_APP_CLIENT_KEY
		clientSignature = X_APP_CLIENT_SIGNATURE
	}
	if phoneNumber == "+919876543210" {
		clientId = X_APP_CLIENT_ID_R
		clientKey = X_APP_CLIENT_KEY_R
		clientSignature = X_APP_CLIENT_SIGNATURE_R
	}

	message, response, err := zboxClient.DeleteOwner(t, clientId, clientKey, clientSignature, idToken, csrfToken, phoneNumber)
	println(message, response, err)
}

func teardownFCM(t *test.SystemTest, idToken, phoneNumber string) {
	t.Logf("Tearing down existing fcm test data for [%v]", phoneNumber)
	csrfToken := createCsrfToken(t, phoneNumber)

	_, _, err := zboxClient.DeleteFCMToken(t, idToken, csrfToken, phoneNumber)
	require.NoError(t, err, "Cleanup failed for 0box FCM Token")
}

func createCsrfToken(t *test.SystemTest, phoneNumber string) string {
	csrfToken, response, err := zboxClient.CreateCSRFToken(t, phoneNumber)
	require.NoError(t, err, "CSRF token creation failed with output: %v and error %v ", response, err)

	require.NotNil(t, csrfToken, "CSRF token container was nil!", response)
	require.NotNil(t, csrfToken.CSRFToken, "CSRF token was nil!", response)

	return csrfToken.CSRFToken
}

func authenticateWithFirebase(t *test.SystemTest, phoneNumber string) *model.FirebaseToken {
	session, response, err := zboxClient.FirebaseSendSms(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", phoneNumber)
	require.NoError(t, err, "Firebase send SMS failed: ", response.RawResponse)
	token, response, err := zboxClient.FirebaseCreateToken(t, "AIzaSyAhySl9LVEFtCgnzbxtmB_T3hiLdECmAGY", session.SessionInfo)
	require.NoError(t, err, "Firebase create token failed: ", response.RawResponse)

	return token
}
