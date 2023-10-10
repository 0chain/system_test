package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		// require.Equal(t, description, zboxWallet.Description, "Description does not match expected") // FIXME: Description is not persisted see: https://github.com/0chain/0box/issues/377

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
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
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
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
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
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
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

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
		wallets, _, _ := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp") // This endpoint used instead of list wallet as list wallet doesn't return the required data
		zboxNFTList, response, err := zboxClient.GetAllNftByWalletId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			strconv.Itoa(wallets[0].WalletId),
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
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
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

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
		errMssg := `{\"error\":\"400: collectionID not valid\"}`
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, errMssg, response.String())
	})

	t.RunSequentially("Post NFT collection with Invalid allocation Id should not work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
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
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

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
		require.Equal(t, `{\"error\":\"400: allocationID not valid\"}`, response.String())
	})

	t.RunSequentially("Update NFT collection with valid argument should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")

		allocationName := "allocation_name"
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
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
		require.Equal(t, "creating allocation successful", allocationObjCreatedResponse.Message)

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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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
		errMssg := `{\"error\":\"400: allocationID not valid\"}`
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		require.Equal(t, shareInfoDeletionMssg.Message, "Share info deleted successfully", "ShareInfo not deleted properly")

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
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, "Share info added successfully", shareInfoSuccessMssg.Message)

		shareInfoData, response, err := zboxClient.GetShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			shareMessage,
			fromInfo,
			zboxClient.DefaultAuthTicket,
			zboxClient.DefaultRecieverId,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoData)
		require.Equal(t, shareInfoData.Message, "Data is present for the given details")
		require.Len(t, shareInfoData.Data, 1)
		require.Equal(t, shareInfoData.Data[0].Message, shareMessage)
		require.Equal(t, shareInfoData.Data[0].FromInfo, fromInfo)
		require.Equal(t, shareInfoData.Data[0].Receiver, zboxClient.DefaultRecieverId)
	})

	t.RunSequentially("Post ShareInfo with Incorrect AuthTicket should work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		require.Equal(t, shareInfoDeletionMssg.Message, "Share info deleted successfully", "ShareInfo not deleted properly")

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

		errorString := `{"error":"share_info_insert_fail: error getting lookupHash from auth_ticket"}`
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, "Share info added successfully", shareInfoSuccessMssg.Message, "Error adding ShareInfo")

		shareInfoData, response, err := zboxClient.GetShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			shareMessage,
			fromInfo,
			zboxClient.DefaultAuthTicket,
			zboxClient.DefaultRecieverId,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoData)
		require.Equal(t, shareInfoData.Message, "Data is present for the given details")
		require.Len(t, shareInfoData.Data, 1)
		require.Equal(t, shareInfoData.Data[0].Message, shareMessage)
		require.Equal(t, shareInfoData.Data[0].FromInfo, fromInfo)
		require.Equal(t, shareInfoData.Data[0].Receiver, zboxClient.DefaultRecieverId)

		shareInfoDeletionMssg, response, err = zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"",
		)
		errorString := `{"error":"invalid_body: Invalid body parameter. [{AuthTicket  required }]"}`
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, "Share info added successfully", shareInfoSuccessMssg.Message, "Error adding shareInfo")

		shareInfoData, response, err := zboxClient.GetShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			shareMessage,
			fromInfo,
			zboxClient.DefaultAuthTicket,
			zboxClient.DefaultRecieverId,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoData)
		require.Equal(t, shareInfoData.Message, "Data is present for the given details")
		require.Len(t, shareInfoData.Data, 1)
		require.Equal(t, shareInfoData.Data[0].Message, shareMessage)
		require.Equal(t, shareInfoData.Data[0].FromInfo, fromInfo)
		require.Equal(t, shareInfoData.Data[0].Receiver, zboxClient.DefaultRecieverId)

		shareInfoDeletionMssg, response, err = zboxClient.DeleteShareInfo(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			zboxClient.DefaultAuthTicket,
		)

		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Nil(t, err)
		require.NotNil(t, shareInfoDeletionMssg)
		require.Equal(t, shareInfoDeletionMssg.Message, "Share info deleted successfully", "Error deleting ShareInfo")
	})

	t.RunSequentially("Get ShareInfo with Incorrect clientRecieverId should not work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		require.Equal(t, "Share info deleted successfully", shareInfoDeletionMssg.Message, "Error deleting ShareInfo")

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
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		shareInfoData, response, err := zboxClient.GetShareInfo(t,
			zboxClient.DefaultAuthTicket,
			shareMessage,
			fromInfo,
			"xyz",
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		errorString := `{"error":{"code":"invalid_header","msg":"Invalid signature."}}`
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoData)
		require.Equal(t, errorString, response.String())
	})

	t.RunSequentially("Get ShareInfo with Incorrect AuthTicket should not work properly", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		require.Equal(t, "Share info deleted successfully", shareInfoDeletionMssg.Message, "Error deleting shareInfo")

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
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoSuccessMssg)
		require.Equal(t, "Share info added successfully", shareInfoSuccessMssg.Message, "Error adding shareInfo")

		shareInfoData, response, err := zboxClient.GetShareInfo(t,
			"abc",
			shareMessage,
			fromInfo,
			zboxClient.DefaultRecieverId,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, shareInfoData)
		require.Equal(t, `{"error":{"code":"invalid_header","msg":"Invalid signature."}}`, response.String())
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")
		require.Equal(t, description, zboxWallet.Description, "Description does not match expected")
	})

	t.RunSequentially("List wallet should work with zero wallets", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		wallets, response, err := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, wallets)
		require.Equal(t, 0, len(wallets.Data), "More wallets present than expected")
	})

	t.RunSequentially("List wallet should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		wallets, response, err := zboxClient.ListWallets(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, wallets)
		require.Equal(t, 1, len(wallets.Data), "Expected 1 wallet only to be present")
	})

	t.RunSequentially("Get empty user info should not work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		userInfo, response, err := zboxClient.GetUserInfo(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, userInfo)
		require.Equal(t, "", userInfo.Username, "output not as expected", response.String())
		require.Equal(t, "", userInfo.Biography, "output not as expected", response.String()) //FIXME: should be null
		require.NotNil(t, userInfo.Avatar, "output not as expected", response.String())       //FIXME: should be null
		require.Equal(t, "", userInfo.Avatar.LargeLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.Avatar.MedLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.Avatar.SmallLoc, "output not as expected", response.String())
		require.NotNil(t, userInfo.BackgroundImage, "output not as expected", response.String()) //FIXME: should be null
		require.Equal(t, "", userInfo.BackgroundImage.LargeLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.BackgroundImage.MedLoc, "output not as expected", response.String())
		require.Equal(t, "", userInfo.BackgroundImage.SmallLoc, "output not as expected", response.String())
		require.NotNil(t, userInfo.CreatedAt, "output not as expected", response.String()) // FIXME: weird that this is present on a blank object
	})

	t.RunSequentially("Create User Info Biography should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		success, response, err := zboxClient.PostUserInfoBiography(t, "bio from "+escapedTestName(t), firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, success)
		require.Equal(t, "updating bio successful", success.Message, "output not as expected", response.String())
	})

	t.RunSequentially("Create User Info Avatar should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		avatarImagePath := escapedTestName(t) + "avatar.png"
		generateImage(t, avatarImagePath)
		success, response, err := zboxClient.PostUserInfoAvatar(t, avatarImagePath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, success)
		require.Equal(t, "updating avatar successful", success.Message, "output not as expected", response.String())
	})

	t.RunSequentially("Create User Info background image should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		backgroundImagePath := escapedTestName(t) + "background.png"
		generateImage(t, backgroundImagePath)

		success, response, err := zboxClient.PostUserInfoBackgroundImage(t, backgroundImagePath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, success)
		require.Equal(t, "updating bgimage successful", success.Message, "output not as expected", response.String())
	})

	t.RunSequentially("Create User Info username should work", func(t *test.SystemTest) {
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		username := cliutils.RandomAlphaNumericString(10)

		usernameResponse, response, err := zboxClient.PutUsername(t, username, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, username)
		require.Equal(t, username, usernameResponse.Username, "output not as expected", response.String())
	})

	t.RunSequentially("Get fully populated user info from username should work", func(t *test.SystemTest) {
		t.Skip("skip till fixed")
		// FIXME: there are no delete endpoints so we can't teardown
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		username := cliutils.RandomAlphaNumericString(10)
		_, _, err := zboxClient.PutUsername(t, username, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		bio := "bio from " + escapedTestName(t)
		_, _, err = zboxClient.PostUserInfoBiography(t, bio, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		avatarImagePath := escapedTestName(t) + "avatar.png"
		generateImage(t, avatarImagePath)
		_, _, err = zboxClient.PostUserInfoAvatar(t, avatarImagePath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		thumbnailPath := escapedTestName(t) + "background.png"
		generateImage(t, thumbnailPath)
		_, _, err = zboxClient.PostUserInfoBackgroundImage(t, thumbnailPath, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)

		userInfo, response, err := zboxClient.GetUserInfo(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, userInfo)
		require.Equal(t, username, userInfo.Username, "output not as expected", response.String())
		require.Equal(t, bio, userInfo.Biography, "output not as expected", response.String())
		require.NotNil(t, userInfo.Avatar, "output not as expected", response.String())
		require.NotNil(t, userInfo.CreatedAt, "output not as expected", response.String())
		require.NotNil(t, userInfo.BackgroundImage, "output not as expected", response.String())
	})
	// FIXME: Missing field description does not match field name (Pascal case instead of snake case)
	// [{ClientID  required } {PublicKey  required } {Timestamp  required } {TokenInput  required } {AppType  required } {PhoneNumber  required }]

	t.RunSequentially("Phone exists should work with existing phone number", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		data, response, err := zboxClient.CheckPhoneExists(t, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, data.Exist)
		require.Equal(t, true, *data.Exist, "Expected phone number to exist")
	})

	t.RunSequentially("Phone exists check should return error with non-existing phone number", func(t *test.SystemTest) {
		phoneNumber := fmt.Sprintf("%s%d", zboxClient.DefaultPhoneNumber, 0)
		teardown(t, firebaseToken.IdToken, phoneNumber)
		csrfToken := createCsrfToken(t, phoneNumber)

		data, response, err := zboxClient.CheckPhoneExists(t, csrfToken, phoneNumber)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, data.Error)
		require.Equal(t, "404: User not found", *data.Error, "Expected error message to match")
	})

	t.RunSequentially("Wallet exists should work with zero wallet", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		walletName := "wallet_name"

		data, response, err := zboxClient.CheckWalletExists(t, walletName, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, data.Exist)
		require.Equal(t, false, *data.Exist, "Expected wallet to not exist")
	})

	t.RunSequentially("Wallet exists should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		walletName := "wallet_name"

		description := "wallet created as part of " + t.Name()
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		t.Logf("Should return true when wallet exists")
		data, response, err := zboxClient.CheckWalletExists(t, walletName, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, data.Exist)
		require.Equal(t, true, *data.Exist, "Expected wallet to exist")
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
		response, err := zboxClient.CreateFCMToken(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Creating FCM Token with existing credentials should fail", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		response, err := zboxClient.CreateFCMToken(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Updating FCM Token should work", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		data, response, err := zboxClient.UpdateFCMToken(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, data, "response object should not be nil")
		require.Equal(t, "zorro", data.DeviceType, "response object should match input")
	})

	t.RunSequentially("Updating Someone else's FCM Token should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		_, response, err := zboxClient.UpdateFCMToken(t, firebaseToken.IdToken, csrfToken, "+917696229926")
		require.NoError(t, err)
		require.Equal(t, 401, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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
		_, _, err := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, "+910123456789", "blimp")

		require.Error(t, err)
	})

	t.RunSequentially("Get wallet keys should return empty with wallet not present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		_, response, _ := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")

		// convert response to json
		var responseJson []string
		err := json.Unmarshal([]byte(response.String()), &responseJson)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 0, len(responseJson), "Response body is empty")
	})

	t.RunSequentially("Delete Wallet should work with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// Create Wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		// Get Wallet
		wallets, _, _ := zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")
		require.Equal(t, 1, len(wallets), "Wallet not created")
		wallet := wallets[0]

		// Delete Wallet
		_, response, _ = zboxClient.DeleteWallet(t, wallet.WalletId, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		var responseJson map[string]interface{}
		err = json.Unmarshal([]byte(response.String()), &responseJson)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "Wallet info deleted successfully", responseJson["message"], "Response message does not match expected. Output: [%v]", response.String())

		// Get Wallet
		wallets, _, _ = zboxClient.GetWalletKeys(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber, "blimp")
		require.Equal(t, 0, len(wallets), "Wallet not deleted")
	})

	t.RunSequentially("Update Wallet with wallet present", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// Create Wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		wallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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

	t.RunSequentially("Contact Wallet should work with for single user", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// create wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		type contactResponse struct {
			Message string              `json:"message"`
			Data    []map[string]string `json:"data"`
		}

		var cr contactResponse

		reqBody := "[{\"user_name\":\"artem\",\"phone_number\":\"+917696229925\"}]"

		response, err = zboxClient.ContactWallet(t, reqBody, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		_ = json.Unmarshal([]byte(response.String()), &cr)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 1, len(cr.Data), "Response data does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Contact Wallet should work with for multiple users", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// create wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		reqBody := "[{\"user_name\":\"artem\",\"phone_number\":\"+917696229925\"},{\"user_name\":\"artem2\",\"phone_number\":\"+917696229925\"}]"

		response, err = zboxClient.ContactWallet(t, reqBody, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		type contactResponse struct {
			Message string              `json:"message"`
			Data    []map[string]string `json:"data"`
		}

		var cr contactResponse
		_ = json.Unmarshal([]byte(response.String()), &cr)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 2, len(cr.Data), "Response data does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Contact Wallet should not work without phone", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// create wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		reqBody := "[{\"user_name\":\"artem\"}]"

		response, err = zboxClient.ContactWallet(t, reqBody, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		type contactResponse struct {
			Message string              `json:"message"`
			Data    []map[string]string `json:"data"`
		}

		var cr contactResponse
		_ = json.Unmarshal([]byte(response.String()), &cr)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 0, len(cr.Data), "Response data does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Contact Wallet should work without user_name", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// create wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		type contactResponse struct {
			Message string              `json:"message"`
			Data    []map[string]string `json:"data"`
		}

		var cr contactResponse

		reqBody := "[{\"phone_number\":\"+917696229925\"}]"

		response, err = zboxClient.ContactWallet(t, reqBody, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		_ = json.Unmarshal([]byte(response.String()), &cr)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, 1, len(cr.Data), "Response data does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Contact Wallet should not work with wrong phone number", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// create wallet
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		_, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		type contactResponse struct {
			Message string              `json:"message"`
			Data    []map[string]string `json:"data"`
		}

		var cr contactResponse

		reqBody := "[{\"user_name\":\"artem\",\"phone_number\":\"+917696232325\"}]"

		response, err = zboxClient.ContactWallet(t, reqBody, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)

		_ = json.Unmarshal([]byte(response.String()), &cr)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, cr.Message, "No data present for the given details", "Response data does not match expected. Output: [%v]", response.String())
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
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

	t.RunSequentially("Create a DEX state with invalid phone number should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PostDexState(t,
			postData,
			firebaseToken.IdToken,
			csrfToken,
			"123456789",
		)
		require.NoError(t, err)
		require.Equal(t, 401, response.StatusCode())
		require.Empty(t, dexState)
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

	t.RunSequentially("Create a DEX state with invalid firebase token should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PostDexState(t,
			postData,
			"abed",
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 401, response.StatusCode())
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
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

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

	t.RunSequentially("Get DEX state with invalid phone number should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.GetDexState(t, firebaseToken.IdToken, csrfToken, "123456789")
		require.NoError(t, err)
		require.Equal(t, 401, response.StatusCode())
		require.Empty(t, dexState)
	})

	t.RunSequentially("Get a DEX state with invalid csrf token should fail", func(t *test.SystemTest) {
		csrfToken := "rg483biecoq23dce2bou" //nolint:gosec

		dexState, response, err := zboxClient.GetDexState(t, firebaseToken.IdToken, csrfToken, "123456789")
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Empty(t, dexState)
	})

	// UPDATE DEX STATE
	t.RunSequentially("Update DEX state with valid phone number should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
		description := "wallet created as part of " + t.Name()
		walletName := "wallet_name"
		zboxWallet, response, err := zboxClient.PostWallet(t,
			zboxClient.DefaultMnemonic,
			walletName,
			description,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"blimp",
		)

		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		// get dex state
		dexState, response, err := zboxClient.GetDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, postData["reference"], dexState.Reference)

		// update dex state
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

	t.RunSequentially("Update DEX state with invalid phone number should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PutDexState(t,
			map[string]string{
				"stage":     "burn",
				"reference": "{\"test_2\":\"test1\", \"test4\":\"test3\"}",
			},
			firebaseToken.IdToken,
			csrfToken,
			"123456789",
		)
		require.NoError(t, err)
		require.Equal(t, 401, response.StatusCode())
		require.Empty(t, dexState)
	})

	t.RunSequentially("Update DEX state with invalid csrf token should fail", func(t *test.SystemTest) {
		csrfToken := "fhkjfhno2" //nolint:gosec

		dexState, response, err := zboxClient.PutDexState(t,
			updateData,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
		require.Empty(t, dexState)
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

	appType := [5]string{"blimp", "vult", "chimney", "bolt", "chalk"}
	for _, app := range appType {
		wallets, _, _ := zboxClient.GetWalletKeysForNumber(t, clientId, clientKey, clientSignature, idToken, csrfToken, phoneNumber, app) // This endpoint used instead of list wallet as list wallet doesn't return the required data

		if len(wallets) != 0 {
			t.Logf("Found [%v] existing wallets for [%v] for the app type [%v]", len(wallets), phoneNumber, app)
			for _, wallet := range wallets {
				message, response, err := zboxClient.DeleteWalletForNumber(t, wallet.WalletId, clientId, clientKey, clientSignature, idToken, csrfToken, phoneNumber)
				println(message, response, err)
			}
		} else {
			t.Logf("No wallets found for [%v] teardown", phoneNumber)
		}
	}
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

func escapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func generateImage(t *test.SystemTest, localpath string) {
	//nolint
	thumbnailBytes, _ := base64.StdEncoding.DecodeString(`iVBORw0KGgoAAAANSUhEUgAAANgAAADpCAMAAABx2AnXAAAAwFBMVEX///8REiQAAADa2ttlZWWlpaU5OTnIyMiIiIhzc3ODg4OVlZXExMT6+vr39/fOzs7v7+9dXV0rKyvf399GRkbn5+dBQUEREREAABp5eXmxsbFsbGxaWlqfn59gYGC4uLgAABWrq6sAAByXl5dOTk4LCwscHBwvLy88PDwkJCR5eYGUlJpBQUxtbnYAAA8ZGyojJTNiY2sAAB82N0OFhYxSU10uLjxKSlQeHy1+f4ebnaRNUFmLjZNdXWWqq7JoaXKY6lzbAAAMKUlEQVR4nO2dC1u6PhvHETARORlhchA8ZYVa+tM0+2u9/3f17N5AUdG0ELBnn666pgzal+3e4d4GDEOhUCgUCoVCoVAoFAqFQqFQKBQKhUKhUCiUP4pqPrNst2NknY6E0Rw2oJh1Us7FsIotST508IFdY6aarN+i1oJUa3FHlWc2QiftxP0CYZNsNeZwBQ48Whwn4ijXY2eVaIbo+8fh6y4uphIEhbTT91NULOjRde5xoPYU4AQVRSmSTXAPnrNL6nncQcItFNBsdps7BY63IMOCuBx8rcRdRZMqQkM9VP1kgQ5pbZFwd0eZCF8WUcANIhvwbUwNIxPzY5+tlFJ9AthugnBrR9gzZI6FAjeRyA/719A37YGTm0wDMU4QBg01iWCFmYNzqYGPy7VIsdygRW+Gs3c4I0DAUxCOljplXeqwEQqo+ijh5s4L4nZrIaSd4wUcMTedEzViNm5oV0yQDdo6xpoaOeyw2zhQatUeCt3HVi7pI4N9kGbKimRIRBjOyJCesfcV8EhMC9eaUvoiYsH9jhtP54R1fQFEhBHFmKegQYutPxmSkblpwXvRFIYZtiWM0UQcqbauzcGcKkE140bEdFC4nGbij6Hfb3Rt7vaWMGJoN5tzQFgpCAuRHBMj4ewx1gUrUqPtCJP2hYW2BPYW9rPgpNbFE3w6Eo+qkOdKtE9xujB9k9VlCMb0o7Nkt8dwujCmClHdkuHhhoy/dEp/yRnC9K0KMnawmiPOEMZ4EV1xQ9VccY4wphR6D2pcikn8GWcJY5SW+/xwY+el03GM84QhZDk3I5ajnC3sWqDCro2/LUxhDE5VOc7ATri/IQxcAw/8DWmeHm6628K6eW+KFZQh8UjsEfBA56brOLxdNkVBqHQaiGKxZVmeJ0kllcvWP2DtDoQT5C670YtROymF988P30eK4yaj6Qv9+6SxrkcSp/8sbzPpOMq3+H8/3+xzR7Ko24iOQLjAsy9gq4RKpeJZrWKjUxEE0TTLts3zrus4Trd7V7shneJeFpaGJ4+eVEXeI3BK7bku9Cf8Pa4Moz6PfWRZUe9ir5ECOE9ij2DnYOzMpYmPQOk8oR3D4+r0+8XRWa8dcBltxB6qhLfjBGG4hU+/EYe5iLvYIzjxh5ye2FvT+q4oEpwD+X5ZDno2tcNlFIBao2cJ4D8VveO1XtTfmB6VQ8KEw2UU2J6hYMUj2vIlTOl9k5zd+VznoLR8CcNdxGMeNG6vGT5kj/kSBjX6cZcnilErFy3BdMIuWS3+RuRL2CNLlhAcQV/7sI0i6b7cxirLlTAZ0nmG811uYGWPcX2nXAmDnvHzWU5q4/ZQ+5AbYZxXEXl2Pct8Kgo2NVsUi+r2HcmHMKXyGNZyh1vneLT16riHatRdkAthnUj1Hd/TOkJ0ZBdx3udAmHYTbZfOn+DaWj+3dglkL0wPptd75UrF7jk/mOCqOGJFDAfZYYOdubBgZaz4+ylWj+R8hXzKXBhOzU0yM8ekUJJRWNbCcL2R2KI1PLlJfB0ZC8Pjr6fkhvDWujBmLAwXniQ9gHyYZdkKk8HCEl1Mj9c3wsqlbIXpSWcYGYrCpbMV1jq/c/gdUH/0mKyFCUmXxKAQMFkLMzcNalJoMMmkZS0MHIXxztEfo/WI2WYrTGQTXxIaLs7P3sYSXhLK5cLGcBWW7NQBuEFgwXu2wnC5SXaa/C4o3Rl3qWAUda4z4ChqeKsyFuaFPaCk6IVNftbDFuw+S262uLy+UVkLw976+6SU4UlP4g7KWhhD9n4lstdGJ74B4jXJXBiZLWYfG/qvJvllQwqmmIJKNnthcri16DZmbcTJrB2ucTsoshG2tWH4tzwa0YtmLYzhqsnI6kU61LkQhqQJt7+WxVtRK82JMARX+hW7nsn8CEsYKixR/qywFPYcZiMMtuldeC829EMS9hOdAO76XnSdpAzOqiTHQ6eBN6Zf9DkxuDeTwS45PG6Kf5ZMEih4zOB+HzFxgicfdPmL0CWzpJms4z66YyAZ0rewdJRlpAuVRvOSsuxMH4ckWcUjwJKbu9b+9y3w2d0fO9M6+PSuPIDng2LXYa99h9eGoSMM6Do8xt95WBjm4Fh6nrNmh1LEUg44r6xIlPw8DeIbtlb9Huh1ydGHgOTmySTfIJ6SG1vrwtJM3S+AhRoP98BD97ABOSQK3vuX9+cmBICwhqwAx6LhCIpxf13CTnZ4a1RY9lBhwLUJE3Ruza4j1OAilK5M2Bbb+yB2tyNdj7D9qZfoXu393UhX00Brexu6oyNGY19Xnp6wdRSDv91iu1/V2j54W8tsoPwDSL8jYLdbtXXweO+EQqFQKBQKhUKhUCgUCoVCoVAoFMoB5PC5xmtXu3zhR8KmNGdWqlYdoLt+rpvUvdCyO3LHODedyaVSVTUw66kTqXohYVIXMkvn03l5XKm6O5N8OWHVNGdut4RpXtGTS0SY2ipKgd2prVZkCaIsFS0ujG7pJKDAmYxabAU3hUNn4zLgkQiWjH5dFT54GnxGcYsqs32ZiwlTed60+YZrwCLyatl0bTimmK5pukJYVA2IVIVtbpK7Cdl22RUrbpl3seZO1TZ5OFvh8YY41eGYMm/zVY7RwJol1+TLtotXx5HLJP46uRIvIkz8VklXNOBtSDz62+HR7TRMHskRTQNMPrAMuQwfJVthdBdemWRVPTingnIClBhl2IvQciU4G0VSbJxiFSlSUI4Z8N5eD/6rAOe6KKhX8WWcpOd10b/odDoVWAfr8TjzIMc0HlddHEqgQR6y2go2T0ASGfzCpAZPHjJlgvWsM6fBo4M4GxkDaY4IC2yMCCMZa4roBFsjl0l4QWqkKHZI2lXHYDiiRrZbqHyaZYRtE4OzqmF0kUyteyhhuL6R+WIgTHeI9ZQbO8KMjTA9vCkmWa3puQnPWUeENcoy+cYIkwbJUnkLv/4tsHSrGt5ZgQizQmFKRBjZGIzOPphja2GiEFz3csJK5OmOUCg0Gz9SuoTSqmyXfq4art5u8bgGhOK0K8zFm6hUR2JkExcDzz2YY+Fl+KSFuZIerrk27ZJiNHDKi25RU6Qy3O9W1VMYbv2kZoGXFM1CajTe5BSjAndjVxjPdzSlxIPZeG4DXcjmObA5gdOIMGkjTOPL6DJCOXFhkS6VVkHh4P1MDd5xylwZ0mqhYFUIG1e54joO7j0YphNEx70wGVfZxSpUdJ6AThHxKQ0U3W44uAXjnQaq7iHHSLdNgK2FHFymmLiNyeFqNXxdY/OWDhSUNR4XQ41To50RQw0ftqoH0UkvUMcmpIOwEjqkb6KjHGfIhVB0eHBB0NHWDHI2unzDTmeZvoAr7MZPHoJJhJ2Mire6GG5KL3yVqqblidWftZphrXgSillteEXXTGuFElcp28IPN6kYzjknKpZom60UV1794nVo56byinbBUCgUCoVCoVAoFAqFQqFQKBQK5fJwfxQmZuf/n4Ap/FGosGvjqLB6e+tT8HsdBMIm6Hf0ugljmqu35mz96XVeL4xWk8KVQIS1v8b15rLZbBbqTXb5Wm826yjQ+vz8HH6wLyxbqLPsTGXZyXSQcXpPJsix92XzfeH3p+yi7y/6s37fn3/8x/3HskNtteTU2YDj5tKAmw1SzbF6XMnfMY92uw3fwd961FQCYc1l4Ws4bA6HY5ad/lsW2KH/9jJQ9cWwP1LZ8ac0YUcGF/uPLsdsuJq811/fB81RuzBY/jeoj+qF1ylK/gz9FF7fm+PV9G25mE9Xk+V4OZuu2M+2v6hHhdVRlFV//OUP6s3pv4+X5td03n5h29yiM/fYiVd6eRkZ6qh9JBnJ0576w8/hdP658v3PwXLyOfS/lnNvyPqr4XDR7y/GPuu/fS5Zf7zq+NNFcfhWZP2vdlRYof3pvy/rs1G/8L4aD1eF/uqt/TFcllDx44aS3/f8QWnOvaQqrL5AyubLwYc/XnZmX8uP6XjxMfmcjpbzxbj/tZx8vPn+YPkxHE6m1r/+23LpS7NVv7ktbPjeni39+mjpv4zZr+n7bFZ/qyzqzdX8X3/18jLsz4bsMOWqAxW2QWE2eS0MUNEbtGdtVCgno9mkOa8P6u+jwmA0exvMXtGfl9Fo0pyNXkbtMInrdgwyEGyoWQeLxKrbzTr+rgmGiSrMPLZi9fWfHf4/ex7XDBV2bfwPF18HmekEj6sAAAAASUVORK5CYII=`)
	err := os.WriteFile(localpath, thumbnailBytes, os.ModePerm)
	require.Nil(t, err, "failed to generate thumbnail", err)
}
