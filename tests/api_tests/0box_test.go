package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/api/util/tokenomics"
	"github.com/0chain/system_test/internal/api/util/wait"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0boxNftCollection(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("List NFT collection id with zero nft collection id  should work")

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	defaultBatchSize := 1
	defaultMaxMint := 1
	defaultCurrMint := 1
	defaultPricePerPack := 1
	defaultTotalNFT := "1"
	collection_id := "collection id created as a part of testing nft collection"

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

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationId := "allocation id created as a part of" + t.Name()
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

		collection_name := "collection as a part of" + t.Name()
		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
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

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationId := "allocation id created as a part of" + t.Name()
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

		collection_name := "collection as a part of " + t.Name()
		collection_id := "collectionId as a part of " + t.Name()

		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
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

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationId := "allocation id created as a part of" + t.Name()
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

		collection_name := "collection as a part of" + t.Name()

		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			defaultCollectionId,
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

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationId := "allocation id created as a part of" + t.Name()
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

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationId := "allocation id created as a part of" + t.Name()
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

		collection_name := "collection as a part of" + t.Name()
		collection_id := "collection id as a part of" + t.Name()
		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
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

		allocationName := "allocation created as part of " + t.Name()
		allocationDescription := "allocation description created as part of " + t.Name()
		allocationType := "allocation type created as part of " + t.Name()
		allocationId := "allocation id created as a part of" + t.Name()
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

		collection_name := "collection as a part of " + t.Name()
		collection_id := "collectionId as a part of " + t.Name()

		zboxNftCollectionId, response, err := zboxClient.CreateNftCollectionId(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
			"created_by",
			collection_name,
			collection_id,
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
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
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
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.PostDexState(t,
			postData,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode())
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
		require.Equal(t, 400, response.StatusCode())
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
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.GetDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, dexState)
		require.Equal(t, postData["stage"], dexState.Stage)
		require.Equal(t, postData["reference"], dexState.Reference)
	})

	t.RunSequentially("Get DEX state with invalid phone number should fail", func(t *test.SystemTest) {
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		dexState, response, err := zboxClient.GetDexState(t, firebaseToken.IdToken, csrfToken, "123456789")
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode())
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
		csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)

		// get dex state
		dexState, response, err := zboxClient.GetDexState(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)

		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
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
		require.Equal(t, 200, response.StatusCode())
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
		require.Equal(t, 400, response.StatusCode())
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

func Test0boxGraphAndTotalEndpoints(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	ownerBalance := apiClient.GetWalletBalance(t, ownerWallet, client.HttpOkStatus)
	t.Logf("Owner balance: %v", ownerBalance)
	ownerWallet.Nonce = int(ownerBalance.Nonce)
	for i := 0; i < 10; i++ {
		apiClient.ExecuteFaucet(t, blobberOwnerWallet, client.TxSuccessfulStatus)
	}
	blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
	t.Logf("Blobber owner balance: %v", blobberOwnerBalance)
	blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)

	// Faucet the used wallets
	for i := 0; i < 10; i++ {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)
	}

	// Stake 6 blobbers, each with 1 token
	targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 6, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
	require.Len(t, targetBlobbers, 6)
	for _, blobber := range targetBlobbers {
		confHash := apiClient.CreateStakePool(t, sdkWallet, 3, blobber.ID, float64(1.0), client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)
	}

	// Create the free allocation marker (ownerWallet -> sdkWallet)
	apiClient.ExecuteFaucet(t, ownerWallet, client.TxSuccessfulStatus)
	apiClient.AddFreeStorageAssigner(t, ownerWallet, client.TxSuccessfulStatus)
	marker := config.CreateFreeStorageMarker(t, sdkWallet.ToSdkWallet(sdkWalletMnemonics), ownerWallet.ToSdkWallet(ownerWalletMnemonics))
	t.Logf("Free allocation marker: %v", marker)

	t.Run("test /v2/graph-write-price", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphWritePrice))

		t.Run("test graph data", func(t *test.SystemTest) {
			data, resp, err := zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			priceBeforeStaking := (*data)[0]

			targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 2, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, targetBlobbers, 2)

			targetBlobbers[0].Capacity += 10 * 1024 * 1024 * 1024
			targetBlobbers[1].Capacity -= 10 * 1024 * 1024 * 1024

			targetBlobbers[0].Terms.WritePrice += *tokenomics.IntToZCN(0.1)
			targetBlobbers[1].Terms.WritePrice += *tokenomics.IntToZCN(0.1)

			apiClient.ExecuteFaucet(t, blobberOwnerWallet, client.TxSuccessfulStatus)
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)

			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				// get all blobbers
				allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				printBlobbers(t, "After Update", allBlobbers)

				expectedAWP := calculateExpectedAvgWritePrice(allBlobbers)
				roundingError := int64(1000)

				data, resp, err := zboxClient.GetGraphWritePrice(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				priceAfterStaking := (*data)[0]

				latest, resp, err := zboxClient.GetAverageWritePrice(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())

				diff := priceAfterStaking - expectedAWP
				t.Logf("priceBeforeStaking: %d, priceAfterStaking: %d, expectedAWP: %d, diff: %d", priceBeforeStaking, priceAfterStaking, expectedAWP, diff)
				return priceAfterStaking != priceBeforeStaking && diff >= -roundingError && diff <= roundingError && priceAfterStaking == int64(*latest)
			})

			// Cleanup: Revert write price to 0.1
			targetBlobbers[0].Terms.WritePrice = *tokenomics.IntToZCN(0.1)
			targetBlobbers[1].Terms.WritePrice = *tokenomics.IntToZCN(0.1)
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)
		})
	})

	t.Run("test /v2/graph-total-challenge-pools", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphTotalChallengePools))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get initial total challenge pools
			data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			totalChallengePools := (*data)[0]

			// Create a new allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			// Upload a file
			sdkClient.UploadFile(t, allocationID)

			var totalChallengePoolsAfterAllocation int64
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				// Get total challenge pools
				data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalChallengePoolsAfterAllocation = (*data)[0]
				return totalChallengePoolsAfterAllocation > totalChallengePools
			})

			// Cancel the second allocation
			apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				// Get total challenge pools
				data, resp, err := zboxClient.GetGraphTotalChallengePools(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalChallengePoolsAfterCancel := (*data)[0]
				return totalChallengePoolsAfterCancel < totalChallengePoolsAfterAllocation
			})
		})
	})

	t.Run("test /v2/graph-allocated-storage", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphAllocatedStorage))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get initial total challenge pools
			data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			allocatedStorage := (*data)[0]

			// Create a new allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				allocatedStorageAfterAllocation := (*data)[0]
				latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := allocatedStorageAfterAllocation > allocatedStorage && allocatedStorageAfterAllocation == int64(*latest)
				allocatedStorage = allocatedStorageAfterAllocation
				return cond
			})

			// Reduce allocation size
			apiClient.UpdateAllocation(t, sdkWallet, allocationID, &model.UpdateAllocationRequest{
				Size: -1024,
			}, client.TxSuccessfulStatus)

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				allocatedStorageAfter := (*data)[0]
				cond := allocatedStorageAfter < allocatedStorage
				allocatedStorage = allocatedStorageAfter
				return cond
			})

			// Add blobber to the allocation
			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
			require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
			apiClient.UpdateAllocationBlobbers(t, sdkWallet, newBlobberID, "", allocationID, client.TxSuccessfulStatus)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				allocatedStorageAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := allocatedStorageAfter > allocatedStorage && allocatedStorageAfter == int64(*latest)
				allocatedStorage = allocatedStorageAfter
				return cond
			})

			// Cancel allocation
			apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

			// Check decreased + consistency
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphAllocatedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				allocatedStorageAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalAllocatedStorage(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				// FIXME: allocated and saved_data of the blobbers table doesn't decrease when the allocation is canceled. Check https://github.com/0chain/0chain/issues/2211
				cond := (allocatedStorageAfter == allocatedStorage) && (allocatedStorageAfter == int64(*latest))
				allocatedStorage = allocatedStorageAfter

				// get all blobbers
				allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				expectedAllocatedStorage := calculateExpectedAllocated(allBlobbers)
				cond = cond && (allocatedStorageAfter == expectedAllocatedStorage)

				return cond
			})
		})
	})

	t.Run("test /v2/graph-used-storage", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphUsedStorage))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get initial used storage
			data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			usedStorage := (*data)[0]

			// Create a new allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			// Upload a file
			fpath, fsize := sdkClient.UploadFile(t, allocationID)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorageAfter - usedStorage) == fsize
				usedStorage = usedStorageAfter
				return cond
			})

			// Update with a bigger file
			fpath, newFsize := sdkClient.UpdateFileBigger(t, allocationID, fpath, fsize)
			t.Logf("Filename after update bigger : %v", fpath)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorageAfter - usedStorage) == (newFsize - fsize)
				usedStorage = usedStorageAfter
				fsize = newFsize
				return cond
			})

			// Update with a smaller file
			fpath, newFsize = sdkClient.UpdateFileSmaller(t, allocationID, fpath, newFsize)
			t.Logf("Filename after update smaller : %v", fpath)

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorage - usedStorageAfter) == (fsize - newFsize)
				usedStorage = usedStorageAfter
				fsize = newFsize
				return cond
			})

			// Remove a file
			sdkClient.DeleteFile(t, allocationID, fpath)

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorage - usedStorageAfter) == fsize
				if cond {
					usedStorage = usedStorageAfter
				}
				return cond
			})

			// Upload another file
			_, fsize = sdkClient.UploadFile(t, allocationID)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				cond := (usedStorageAfter - usedStorage) == fsize
				usedStorage = usedStorageAfter
				return cond
			})

			// Cancel the allocation
			apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

			// Check decreased + consistency
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				// Get total challenge pools
				data, resp, err := zboxClient.GetGraphUsedStorage(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				usedStorageAfter := (*data)[0]
				// FIXME: allocated and saved_data of the blobbers table doesn't decrease when the allocation is canceled. Check https://github.com/0chain/0chain/issues/2211
				cond := usedStorage == usedStorageAfter
				usedStorage = usedStorageAfter

				// get all blobbers
				allBlobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())

				expectedSavedData := calculateExpectedSavedData(allBlobbers)
				cond = cond && usedStorageAfter == expectedSavedData

				return cond
			})
		})
	})

	t.Run("test /v2/graph-total-staked", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphTotalStaked))

		t.Run("test graph data", func(t *test.SystemTest) {
			data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			totalStaked := (*data)[0]

			// Stake a blobbers
			targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, targetBlobbers, 1)
			confHash := apiClient.CreateStakePool(t, sdkWallet, 3, targetBlobbers[0].ID, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := (totalStakedAfter-totalStaked) == *(tokenomics.IntToZCN(1)) && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})

			// Get blobber balance before unlocking
			blobberBalanceBefore := getClientStakeForSSCProvider(t, sdkWallet, targetBlobbers[0].ID)

			// Unlock a stake pool => should decrease
			restake := unstakeBlobber(t, sdkWallet, targetBlobbers[0].ID)
			defer restake()

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := (totalStaked-totalStakedAfter) == blobberBalanceBefore && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})

			// Stake a validator
			vs, resp, err := apiClient.V1SCRestGetAllValidators(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, vs)
			validatorId := vs[0].ValidatorID
			confHash = apiClient.CreateStakePool(t, sdkWallet, 4, validatorId, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStakedAfter-totalStaked == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})

			// Unstake the validator
			confHash = apiClient.UnlockStakePool(t, sdkWallet, 4, validatorId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStaked-totalStakedAfter == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})

			// Stake a miner
			miners, resp, err := apiClient.V1SCRestGetAllMiners(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, miners)
			minerId := miners[0].SimpleNodeResponse.ID
			t.Logf("Staking miner %s", minerId)
			confHash = apiClient.CreateMinerStakePool(t, sdkWallet, 1, minerId, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStakedAfter-totalStaked == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})

			// Unstake the miner
			confHash = apiClient.UnlockMinerStakePool(t, sdkWallet, 1, minerId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStaked-totalStakedAfter == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})

			// Stake a sharder
			sharders, resp, err := apiClient.V1SCRestGetAllSharders(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, sharders)
			sharderId := sharders[0].SimpleNodeResponse.ID
			confHash = apiClient.CreateMinerStakePool(t, sdkWallet, 2, sharderId, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStakedAfter-totalStaked == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})

			// Unstake the sharder
			confHash = apiClient.UnlockMinerStakePool(t, sdkWallet, 2, sharderId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalStaked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalStakedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalStaked(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalStaked-totalStakedAfter == *tokenomics.IntToZCN(1.0) && totalStakedAfter == int64(*latest)
				totalStaked = totalStakedAfter
				return cond
			})
		})
	})

	t.Run("test /v2/graph-total-minted", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphTotalMinted))

		t.Run("test graph data", func(t *test.SystemTest) {
			data, resp, err := zboxClient.GetGraphTotalMinted(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			totalMinted := (*data)[0]

			// Create a new allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			// Upload a file
			sdkClient.UploadFile(t, allocationID)

			// Add/Remove blobber to the allocation
			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
			require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
			apiClient.UpdateAllocationBlobbers(t, sdkWallet, newBlobberID,
				allocation.Blobbers[0].ID, allocationID, client.TxSuccessfulStatus)

			// Unlock the stake pool of the removed blobber
			restake1 := unstakeBlobber(t, sdkWallet, allocation.Blobbers[0].ID)
			defer restake1()

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalMinted(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalMintedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalMinted(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalMintedAfter == int64(*latest) && totalMintedAfter > totalMinted
				totalMinted = totalMintedAfter
				return cond
			})

			// Cancel the allocation
			apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

			// Unlock the stake pool of the other blobbers
			restake2 := unstakeBlobber(t, sdkWallet, allocation.Blobbers[1].ID)
			restake3 := unstakeBlobber(t, sdkWallet, newBlobberID)
			defer restake2()
			defer restake3()

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalMinted(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalMintedAfter := (*data)[0]
				cond := totalMintedAfter > totalMinted
				totalMinted = totalMintedAfter
				return cond
			})

			// Create free allocation marker
			apiClient.AddFreeStorageAssigner(t, ownerWallet, client.TxSuccessfulStatus)
			marker := config.CreateFreeStorageMarker(t, sdkWallet.ToSdkWallet(sdkWalletMnemonics), ownerWallet.ToSdkWallet(ownerWalletMnemonics))

			// Create a new allocation
			freeAllocData := &model.FreeAllocationData{
				RecipientPublicKey: sdkWallet.PublicKey,
				Marker:             marker,
			}
			freeAllocationBlobbers := apiClient.GetFreeAllocationBlobbers(t, sdkWallet, freeAllocData, client.HttpOkStatus)
			freeAllocationBlobbers.FreeAllocationData = *freeAllocData
			apiClient.CreateFreeAllocation(t, sdkWallet, freeAllocationBlobbers, client.TxSuccessfulStatus)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalMinted(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalMintedAfter := (*data)[0]
				latest, resp, err := zboxClient.GetTotalMinted(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalMintedAfter == int64(*latest) && totalMintedAfter > totalMinted
				totalMinted = totalMintedAfter
				return cond
			})
		})
	})

	t.Run("test /v2/graph-total-locked", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphTotalLocked))

		t.Run("test graph data", func(t *test.SystemTest) {
			data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			graphTotalLocked := (*data)[0]

			// Some more stake. It's gonna be tough
			// for i := 0; i < 10; i++ {
			// 	apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)
			// }

			// Stake blobber
			blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len(blobbers))
			blobberId := blobbers[0].ID
			t.Logf("Staking blobber %s", blobberId)
			confHash := apiClient.CreateStakePool(t, sdkWallet, 3, blobberId, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter-graphTotalLocked == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Get blobber balance before unlocking
			blobberBalanceBefore := getClientStakeForSSCProvider(t, sdkWallet, blobberId)

			// Unstake the blobber
			restake := unstakeBlobber(t, sdkWallet, blobberId)
			defer restake()

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := graphTotalLocked-totalLockedAfter == blobberBalanceBefore
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Stake a validator
			vs, resp, err := apiClient.V1SCRestGetAllValidators(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, vs)
			validatorId := vs[0].ValidatorID
			confHash = apiClient.CreateStakePool(t, sdkWallet, 4, validatorId, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter-graphTotalLocked == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Unstake the validator
			confHash = apiClient.UnlockStakePool(t, sdkWallet, 4, validatorId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := graphTotalLocked-totalLockedAfter == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Stake a miner
			miners, resp, err := apiClient.V1SCRestGetAllMiners(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, miners)
			minerId := miners[0].SimpleNodeResponse.ID
			t.Logf("Staking miner %s", minerId)
			confHash = apiClient.CreateMinerStakePool(t, sdkWallet, 1, minerId, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter-graphTotalLocked == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Unstake the miner
			confHash = apiClient.UnlockMinerStakePool(t, sdkWallet, 1, minerId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := graphTotalLocked-totalLockedAfter == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Stake a sharder
			sharders, resp, err := apiClient.V1SCRestGetAllSharders(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, sharders)
			sharderId := sharders[0].SimpleNodeResponse.ID
			confHash = apiClient.CreateMinerStakePool(t, sdkWallet, 2, sharderId, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter-graphTotalLocked == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Unstake the sharder
			confHash = apiClient.UnlockMinerStakePool(t, sdkWallet, 2, sharderId, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := graphTotalLocked-totalLockedAfter == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Create allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)

			// Check increase by locked value
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter-graphTotalLocked == *tokenomics.IntToZCN(0.2)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Create write pool for the allocation
			confHash = apiClient.CreateWritePool(t, sdkWallet, allocationID, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter-graphTotalLocked == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Cancel the allocation
			confHash = apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease and calculate cancellation charge
			var cancellationCharge int64
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter < graphTotalLocked
				cancellationCharge = graphTotalLocked - totalLockedAfter
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Unlock the write pool
			confHash = apiClient.UnlockWritePool(t, sdkWallet, allocationID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease by (initial locked value + write pool value - cancellation charge)
			t.Logf("Cancellation charge: %d", cancellationCharge)
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := graphTotalLocked-totalLockedAfter == (*tokenomics.IntToZCN(1.0) + *tokenomics.IntToZCN(0.2) - cancellationCharge)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Create read pool
			confHash = apiClient.CreateReadPool(t, sdkWallet, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := totalLockedAfter-graphTotalLocked == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})

			// Unlock the read pool
			confHash = apiClient.UnlockReadPool(t, sdkWallet, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTotalLocked(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalLockedAfter := (*data)[0]
				cond := graphTotalLocked-totalLockedAfter == *tokenomics.IntToZCN(1.0)
				graphTotalLocked = totalLockedAfter
				return cond
			})
		})
	})

	t.Run("test /v2/graph-challenges", func(t *test.SystemTest) {
		t.Run("endpoint parameters", func(t *test.SystemTest) {
			// should fail for invalid parameters
			_, resp, _ := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "invalid from param")

			_, resp, _ = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "10", To: "AX", DataPoints: "5"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "invalid to param")

			_, resp, _ = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "10", To: "20", DataPoints: "AX"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "invalid data-points query param")

			// should fail for invalid parameters (end - start < points + 1)
			_, resp, _ = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "10000", To: "10010", DataPoints: "10"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "there must be at least one interval")

			// should fail for invalid parameters (end < start)
			_, resp, _ = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: "10000", To: "1000", DataPoints: "10"})
			require.Equal(t, 400, resp.StatusCode())
			require.Contains(t, resp.String(), "to 1000 less than from 10000")

			// should succeed in case of 1 point
			data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(data.TotalChallenges)))
			require.Equal(t, 1, len([]int64(data.SuccessfulChallenges)))

			// should succeed in case of multiple points
			minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
			require.NoError(t, err)
			latestRound := minerStats.LastFinalizedRound
			data, resp, err = zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{From: strconv.FormatInt(latestRound-int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 10, len([]int64(data.TotalChallenges)))
			require.Equal(t, 10, len([]int64(data.SuccessfulChallenges)))
		})

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get initial graph data
			data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(data.TotalChallenges)))
			require.Equal(t, 1, len([]int64(data.SuccessfulChallenges)))
			totalChallenges, successfulChallenges := data.TotalChallenges[0], data.SuccessfulChallenges[0]

			// Create an allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			// Upload a file
			sdkClient.UploadFile(t, allocationID)

			// Check total challenges increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(data.TotalChallenges)))
				totalChallengesAfter := data.TotalChallenges[0]
				successfulChallengesAfter := data.SuccessfulChallenges[0]
				latestTotal, resp, err := zboxClient.GetTotalChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				latestSuccessful, resp, err := zboxClient.GetSuccessfulChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond := totalChallengesAfter > totalChallenges && int64(*latestTotal) == totalChallengesAfter && int64(*latestSuccessful) == successfulChallengesAfter
				totalChallenges = totalChallengesAfter
				successfulChallenges = data.SuccessfulChallenges[0]
				return cond
			})

			// Add blobber to the allocation
			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
			require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
			apiClient.UpdateAllocationBlobbers(t, sdkWallet, newBlobberID,
				allocation.Blobbers[0].ID, allocationID, client.TxSuccessfulStatus)

			// Check total challenges increase + successful challenges increase because time has passed since the upload
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphChallenges(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(data.TotalChallenges)))
				totalChallengesAfter := data.TotalChallenges[0]
				cond := totalChallengesAfter > totalChallenges
				totalChallenges = totalChallengesAfter
				successfulChallengesAfter := data.SuccessfulChallenges[0]
				cond = cond && successfulChallengesAfter > successfulChallenges
				successfulChallenges = successfulChallengesAfter
				latestTotal, resp, err := zboxClient.GetTotalChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				latestSuccessful, resp, err := zboxClient.GetSuccessfulChallenges(t)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				cond = cond && int64(*latestTotal) == totalChallengesAfter && int64(*latestSuccessful) == successfulChallengesAfter
				return cond
			})
		})
	})

	t.Run("test /v2/graph-token-supply", func(t *test.SystemTest) {
		t.Run("endpoint parameters", graphEndpointTestCases(zboxClient.GetGraphTotalLocked))

		t.Run("test graph data", func(t *test.SystemTest) {
			data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Equal(t, 1, len([]int64(*data)))
			totalSupply := (*data)[0]

			// Create a new allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.2, client.TxSuccessfulStatus)

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter < totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Create a write pool for the allocation
			confHash := apiClient.CreateWritePool(t, sdkWallet, allocationID, 1.0, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decreased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter < totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Upload a file
			sdkClient.UploadFile(t, allocationID)

			// Add/Remove blobber to the allocation
			allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
			newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
			require.NotZero(t, newBlobberID, "New blobber ID contains zero value")
			apiClient.UpdateAllocationBlobbers(t, sdkWallet, newBlobberID,
				allocation.Blobbers[0].ID, allocationID, client.TxSuccessfulStatus)

			// Unlock the stake pool of the removed blobber
			restake1 := unstakeBlobber(t, sdkWallet, allocation.Blobbers[0].ID)
			defer restake1()

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter > totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Cancel the allocation
			apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)

			// Unlock the write pool
			confHash = apiClient.UnlockWritePool(t, sdkWallet, allocationID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increased
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter > totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Unlock the stake pool of the other blobbers
			restake2 := unstakeBlobber(t, sdkWallet, allocation.Blobbers[1].ID)
			restake3 := unstakeBlobber(t, sdkWallet, newBlobberID)
			defer restake2()
			defer restake3()

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter > totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Create free allocation marker
			apiClient.AddFreeStorageAssigner(t, ownerWallet, client.TxSuccessfulStatus)
			marker := config.CreateFreeStorageMarker(t, sdkWallet.ToSdkWallet(sdkWalletMnemonics), ownerWallet.ToSdkWallet(ownerWalletMnemonics))

			// Create a new allocation
			freeAllocData := &model.FreeAllocationData{
				RecipientPublicKey: sdkWallet.PublicKey,
				Marker:             marker,
			}
			freeAllocationBlobbers := apiClient.GetFreeAllocationBlobbers(t, sdkWallet, freeAllocData, client.HttpOkStatus)
			freeAllocationBlobbers.FreeAllocationData = *freeAllocData
			apiClient.CreateFreeAllocation(t, sdkWallet, freeAllocationBlobbers, client.TxSuccessfulStatus)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter > totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Stake a Miner
			miners, resp, err := apiClient.V1SCRestGetAllMiners(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, miners)
			minerID := miners[0].ID
			confHash = apiClient.CreateMinerStakePool(t, sdkWallet, 1, minerID, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decreased (staked tokens are burnt)
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter < totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Unsake the Miner
			confHash = apiClient.UnlockMinerStakePool(t, sdkWallet, 1, minerID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increased (unstaked tokens are minted)
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter > totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Stake a Sharder
			sharders, resp, err := apiClient.V1SCRestGetAllSharders(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.NotEmpty(t, sharders)
			sharderID := sharders[0].ID
			confHash = apiClient.CreateMinerStakePool(t, sdkWallet, 2, sharderID, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decreased (staked tokens are burnt)
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter < totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Unsake the Sharder
			confHash = apiClient.UnlockMinerStakePool(t, sdkWallet, 2, sharderID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increased (unstaked tokens are minted)
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter > totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Create read pool
			confHash = apiClient.CreateReadPool(t, sdkWallet, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decrease
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter < totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// Unlock the read pool
			confHash = apiClient.UnlockReadPool(t, sdkWallet, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check increase
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Equal(t, 1, len([]int64(*data)))
				totalSupplyAfter := (*data)[0]
				cond := totalSupplyAfter > totalSupply
				totalSupply = totalSupplyAfter
				return cond
			})

			// TODO: Burn is not working, investigate why
			// // Burn ZCN
			// confHash = apiClient.BurnZcn(t, sdkWallet, parsedConfig.EthereumAddress, float64(1.0), client.TxSuccessfulStatus)
			// require.NotEmpty(t, confHash)

			// // Check decrease
			// wait.PoolImmediately(t, 2 * time.Minute, func() bool {
			// 	data, resp, err := zboxClient.GetGraphTokenSupply(t, &model.ZboxGraphRequest{ DataPoints: "1" })
			// 	require.NoError(t, err)
			// 	require.Equal(t, 200, resp.StatusCode())
			// 	require.Equal(t, 1, len([]int64(*data)))
			// 	totalSupplyAfter := (*data)[0]
			// 	cond := totalSupplyAfter < totalSupply
			// 	totalSupply = totalSupplyAfter
			// 	return cond
			// })
		})
	})

	t.Run("test /v2/total-blobber-capacity", func(t *test.SystemTest) {
		// Get initial
		data, resp, err := zboxClient.GetTotalBlobberCapacity(t)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		totalBlobberCapacity := int64(*data)

		// Faucet the blobber owner wallet
		apiClient.ExecuteFaucet(t, blobberOwnerWallet, client.TxSuccessfulStatus)

		// Increase capacity of 2 blobber
		targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 2, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, targetBlobbers, 2)

		targetBlobbers[0].Capacity += 10 * 1024 * 1024 * 1024
		targetBlobbers[1].Capacity += 5 * 1024 * 1024 * 1024
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)

		// Check increase
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetTotalBlobberCapacity(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			totalBlobberCapacityAfter := int64(*data)
			cond := totalBlobberCapacityAfter > totalBlobberCapacity
			totalBlobberCapacity = totalBlobberCapacityAfter
			return cond
		})

		// Decrease them back
		targetBlobbers[0].Capacity -= 10 * 1024 * 1024 * 1024
		targetBlobbers[1].Capacity -= 5 * 1024 * 1024 * 1024
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[0], client.TxSuccessfulStatus)
		apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobbers[1], client.TxSuccessfulStatus)

		// Check decrease
		wait.PoolImmediately(t, 2*time.Minute, func() bool {
			data, resp, err := zboxClient.GetTotalBlobberCapacity(t)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			totalBlobberCapacityAfter := int64(*data)
			cond := totalBlobberCapacityAfter < totalBlobberCapacity
			totalBlobberCapacity = totalBlobberCapacityAfter

			blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			expectedCapacity := calculateCapacity(blobbers)
			require.Equal(t, expectedCapacity, totalBlobberCapacityAfter)
			return cond
		})
	})
}

func Test0boxGraphBlobberEndpoints(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	// Faucet the used wallets
	for i := 0; i < 100; i++ {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)
	}
	for i := 0; i < 100; i++ {
		apiClient.ExecuteFaucet(t, blobberOwnerWallet, client.TxSuccessfulStatus)
	}
	blobberOwnerBalance := apiClient.GetWalletBalance(t, blobberOwnerWallet, client.HttpOkStatus)
	t.Logf("Blobber owner balance: %v", blobberOwnerBalance)
	blobberOwnerWallet.Nonce = int(blobberOwnerBalance.Nonce)

	// Stake 6 blobbers, each with 1 token
	targetBlobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 6, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
	require.Len(t, targetBlobbers, 6)
	for _, blobber := range targetBlobbers {
		confHash := apiClient.CreateStakePool(t, sdkWallet, 3, blobber.ID, float64(1.0), client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)
	}

	t.Run("test /v2/graph-blobber-challenges-passed and /v2/graph-blobber-challenges-completed", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberChallengesPassed, blobbers[0].ID))

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberChallengesCompleted, blobbers[0].ID))

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberChallengesOpen, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Create allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			targetBlobber := (*allocationBlobbers.Blobbers)[0]
			allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

			// Get initial value of one of the blobbers
			data, resp, err := zboxClient.GetGraphBlobberChallengesPassed(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			challnegesPassed := (*data)[0]

			data, resp, err = zboxClient.GetGraphBlobberChallengesCompleted(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			challnegesCompleted := (*data)[0]

			data, resp, err = zboxClient.GetGraphBlobberChallengesOpen(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			challnegesOpen := (*data)[0]

			// Upload file
			fpath, fsize := sdkClient.UploadFile(t, allocationID)
			require.NotEmpty(t, fpath)
			require.NotZero(t, fsize)

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberChallengesPassed(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				challnegesPassedAfter := (*data)[0]
				cond := challnegesPassedAfter > challnegesPassed

				data, resp, err = zboxClient.GetGraphBlobberChallengesCompleted(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				challnegesCompletedAfter := (*data)[0]
				cond = cond && challnegesCompletedAfter > challnegesCompleted

				data, resp, err = zboxClient.GetGraphBlobberChallengesOpen(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				challnegesOpenAfter := (*data)[0]
				cond = cond && challnegesOpenAfter < challnegesOpen

				if cond {
					challnegesPassed = challnegesPassedAfter
					challnegesCompleted = challnegesCompletedAfter
					challnegesOpen = challnegesOpenAfter
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-inactive-rounds", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberInactiveRounds, blobbers[0].ID))

		// TODO: Complete if needed
		// t.Run("test graph data", func(t *test.SystemTest) {})
	})

	t.Run("test /v2/graph-blobber-write-price", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberWritePrice, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get a single blobber to use in graph parameters test
			targetBlobber := blobbers[0]

			// Get initial value of one of the blobbers
			data, resp, err := zboxClient.GetGraphBlobberWritePrice(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			writePrice := (*data)[0]

			// Faucet blobberOwner wallet
			apiClient.ExecuteFaucet(t, blobberOwnerWallet, client.TxSuccessfulStatus)

			// Increase write price
			targetBlobber.Terms.WritePrice += 1000000000
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberWritePrice(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue > writePrice
				if cond {
					writePrice = afterValue
				}
				return cond
			})

			// Decrease write price
			targetBlobber.Terms.WritePrice -= 1000000000
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

			// Check decreased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberWritePrice(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue < writePrice
				if cond {
					writePrice = afterValue
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-capacity", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberCapacity, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get a single blobber to use in graph parameters test
			targetBlobber := blobbers[0]

			// Get initial value of one of the blobbers
			data, resp, err := zboxClient.GetGraphBlobberCapacity(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			require.Len(t, *data, 1)
			capacity := (*data)[0]

			// Faucet blobberOwner wallet
			apiClient.ExecuteFaucet(t, blobberOwnerWallet, client.TxSuccessfulStatus)

			// Increase capacity
			targetBlobber.Capacity += 1000000000
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberCapacity(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue > capacity
				if cond {
					capacity = afterValue
				}
				return cond
			})

			// Decrease capacity
			targetBlobber.Capacity -= 1000000000
			apiClient.UpdateBlobber(t, blobberOwnerWallet, targetBlobber, client.TxSuccessfulStatus)

			// Check decreased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberCapacity(t, targetBlobber.ID, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue < capacity
				if cond {
					capacity = afterValue
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-allocated", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberAllocated, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get allocated of all blobbers
			blobberAllocated := make(map[string]int64)

			blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			for _, blobber := range blobbers {
				blobberAllocated[blobber.ID] = blobber.Allocated
			}

			// Create allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

			// Value before allocation
			targetBlobber := (*allocationBlobbers.Blobbers)[0]
			allocated := blobberAllocated[targetBlobber]

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberAllocated(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue > allocated
				if cond {
					allocated = afterValue
				}
				return cond
			})

			// Cancel the allocation
			confHash := apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decreased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberAllocated(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				// FIXME: allocated and saved_data of the blobbers table doesn't decrease when the allocation is canceled. Check https://github.com/0chain/0chain/issues/2211
				cond := afterValue == allocated
				if cond {
					allocated = afterValue
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-saved-data", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberSavedData, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get saved data of all blobbers
			blobberSavedData := make(map[string]int64)

			blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			for _, blobber := range blobbers {
				blobberSavedData[blobber.ID] = blobber.SavedData
			}

			// Create allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

			// Value before allocation
			targetBlobber := (*allocationBlobbers.Blobbers)[0]
			savedData := blobberSavedData[targetBlobber]

			// Upload a file
			fpath, fsize := sdkClient.UploadFile(t, allocationID)

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue-savedData == fsize
				if cond {
					savedData = afterValue
				}
				return cond
			})

			// Delete the file
			sdkClient.DeleteFile(t, allocationID, fpath)

			// Check decreased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := savedData-afterValue == fsize
				if cond {
					savedData = afterValue
				}
				return cond
			})

			// Upload another file
			_, fsize = sdkClient.UploadFile(t, allocationID)

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue-savedData == fsize
				if cond {
					savedData = afterValue
				}
				return cond
			})

			// Cancel the allocation
			confHash := apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decreased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberSavedData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]

				// FIXME: allocated and saved_data of the blobbers table doesn't decrease when the allocation is canceled. Check
				cond := savedData == afterValue
				if cond {
					savedData = afterValue
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-read-data", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberReadData, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get read data of all blobbers
			blobberReadData := make(map[string]int64)

			blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			for _, blobber := range blobbers {
				blobberReadData[blobber.ID] = blobber.ReadData
			}

			// Create allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

			// Value before allocation
			targetBlobber := (*allocationBlobbers.Blobbers)[0]
			readData := blobberReadData[targetBlobber]

			// Upload a file
			fpath, fsize := sdkClient.UploadFile(t, allocationID)

			// Download the file
			sdkClient.DownloadFile(t, allocationID, fpath, ".")
			defer os.Remove(path.Join(".", fpath))

			// // Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberReadData(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue-readData == fsize
				if cond {
					readData = afterValue
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-offers-total", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberOffersTotal, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get offers of all blobbers
			blobberOffersTotal := make(map[string]int64)

			blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			for _, blobber := range blobbers {
				data, resp, err := apiClient.V1SCRestGetStakePoolStat(t, model.SCRestGetStakePoolStatRequest{
					ProviderType: "3",
					ProviderID:   blobber.ID,
				}, client.HttpOkStatus)
				t.Logf("SP for blobber %v: %+v", blobber.ID, data)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				blobberOffersTotal[blobber.ID] = data.OffersTotal
			}

			// Create allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

			// Value before allocation
			targetBlobber := (*allocationBlobbers.Blobbers)[0]
			offersTotal := blobberOffersTotal[targetBlobber]

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberOffersTotal(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue > offersTotal
				if cond {
					offersTotal = afterValue
				}
				return cond
			})

			// Cancel the allocation
			confHash := apiClient.CancelAllocation(t, sdkWallet, allocationID, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check decreased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberOffersTotal(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue < offersTotal
				if cond {
					offersTotal = afterValue
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-unstake-total and /v2/graph-blobber-stake-total", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberTotalStake, blobbers[0].ID))

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberUnstakeTotal, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			targetBlobber := blobbers[0].ID
			data, resp, err := apiClient.V1SCRestGetStakePoolStat(t, model.SCRestGetStakePoolStatRequest{
				ProviderType: "3",
				ProviderID:   targetBlobber,
			}, client.HttpOkStatus)
			t.Logf("SP for blobber %v: %+v", targetBlobber, data)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			unstakeTotal := data.UnstakeTotal
			stakeTotal := data.Balance

			// Stake the blobber
			confHash := apiClient.CreateStakePool(t, sdkWallet, 3, targetBlobber, float64(1.0), client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check stake increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberTotalStake(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue > stakeTotal
				if cond {
					unstakeTotal = afterValue
				}
				return cond
			})

			// Unstake the blobber
			confHash = apiClient.UnlockStakePool(t, sdkWallet, 3, targetBlobber, client.TxSuccessfulStatus)
			require.NotEmpty(t, confHash)

			// Check unstake increased and stake decrease for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberTotalStake(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue < stakeTotal

				data, resp, err = zboxClient.GetGraphBlobberUnstakeTotal(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue = (*data)[0]
				cond = cond && afterValue > unstakeTotal
				if cond {
					stakeTotal = afterValue
					unstakeTotal = afterValue
				}
				return cond
			})
		})
	})

	t.Run("test /v2/graph-blobber-total-rewards", func(t *test.SystemTest) {
		// Get a single blobber to use in graph parameters test
		blobbers, resp, err := apiClient.V1SCRestGetFirstBlobbers(t, 1, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, blobbers, 1)

		t.Run("endpoint parameters", graphBlobberEndpointTestCases(zboxClient.GetGraphBlobberTotalRewards, blobbers[0].ID))

		t.Run("test graph data", func(t *test.SystemTest) {
			// Get read data of all blobbers
			blobberRewards := make(map[string]int64)

			blobbers, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			for _, blobber := range blobbers {
				sp, resp, err := apiClient.V1SCRestGetStakePoolStat(t, model.SCRestGetStakePoolStatRequest{
					ProviderType: "3",
					ProviderID:   blobber.ID,
				}, client.HttpOkStatus)
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				blobberRewards[blobber.ID] = sp.Rewards
			}

			// Create allocation
			blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
			blobberRequirements.DataShards = 1
			blobberRequirements.ParityShards = 1
			allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
			allocationID := apiClient.CreateAllocationWithLockValue(t, sdkWallet, allocationBlobbers, 0.1, client.TxSuccessfulStatus)

			// Value before allocation
			targetBlobber := (*allocationBlobbers.Blobbers)[0]
			rewards := blobberRewards[targetBlobber]

			// Upload a file
			sdkClient.UploadFile(t, allocationID)

			// Check increased for the same blobber
			wait.PoolImmediately(t, 2*time.Minute, func() bool {
				data, resp, err := zboxClient.GetGraphBlobberTotalRewards(t, targetBlobber, &model.ZboxGraphRequest{DataPoints: "1"})
				require.NoError(t, err)
				require.Equal(t, 200, resp.StatusCode())
				require.Len(t, *data, 1)
				afterValue := (*data)[0]
				cond := afterValue > rewards
				if cond {
					rewards = afterValue
				}
				return cond
			})
		})
	})

}

func teardown(t *test.SystemTest, idToken, phoneNumber string) {
	t.Logf("Tearing down existing test data for [%v]", phoneNumber)
	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	appType := [5]string{"blimp", "vult", "chimney", "bolt", "chalk"}
	for _, app := range appType {
		wallets, _, _ := zboxClient.GetWalletKeys(t, idToken, csrfToken, phoneNumber, app) // This endpoint used instead of list wallet as list wallet doesn't return the required data

		if len(wallets) != 0 {
			t.Logf("Found [%v] existing wallets for [%v] for the app type [%v]", len(wallets), phoneNumber, app)
			for _, wallet := range wallets {
				message, response, err := zboxClient.DeleteWallet(t, wallet.WalletId, idToken, csrfToken, phoneNumber)
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

func graphEndpointTestCases(endpoint model.ZboxGraphEndpoint) func(*test.SystemTest) {
	return func(t *test.SystemTest) {
		// should fail for invalid parameters
		_, resp, err := endpoint(t, &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid from param")

		_, resp, err = endpoint(t, &model.ZboxGraphRequest{From: "10", To: "AX", DataPoints: "5"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid to param")

		_, resp, err = endpoint(t, &model.ZboxGraphRequest{From: "10", To: "20", DataPoints: "AX"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid data-points query param")

		// should fail for invalid parameters (end - start < points + 1)
		_, resp, err = endpoint(t, &model.ZboxGraphRequest{From: "10000", To: "10010", DataPoints: "10"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "there must be at least one interval")

		// should fail for invalid parameters (end < start)
		_, resp, err = endpoint(t, &model.ZboxGraphRequest{From: "10000", To: "1000", DataPoints: "10"})
		require.Error(t, err)
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "to 1000 less than from 10000")

		// should succeed in case of 1 point
		data, resp, err := endpoint(t, &model.ZboxGraphRequest{DataPoints: "1"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))

		// should succeed in case of multiple points
		minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
		require.NoError(t, err)
		latestRound := minerStats.LastFinalizedRound
		data, resp, err = endpoint(t, &model.ZboxGraphRequest{From: strconv.FormatInt(latestRound-int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 10, len([]int64(*data)))
	}
}

func graphBlobberEndpointTestCases(endpoint model.ZboxGraphBlobberEndpoint, blobberId string) func(*test.SystemTest) {
	return func(t *test.SystemTest) {
		// should fail for invalid parameters
		_, resp, _ := endpoint(t, "", &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "provider id not provided")

		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "AX", To: "20", DataPoints: "5"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid from param")

		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "10", To: "AX", DataPoints: "5"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid to param")

		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "10", To: "20", DataPoints: "AX"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "invalid data-points query param")

		// should fail for invalid parameters (end < start)
		_, resp, _ = endpoint(t, blobberId, &model.ZboxGraphRequest{From: "10000", To: "1000", DataPoints: "10"})
		require.Equal(t, 400, resp.StatusCode())
		require.Contains(t, resp.String(), "to 1000 less than from 10000")

		// should succeed in case of 1 point
		data, resp, _ := endpoint(t, blobberId, &model.ZboxGraphRequest{DataPoints: "1"})
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 1, len([]int64(*data)))

		// should succeed in case of multiple points
		minerStats, _, err := apiClient.V1MinerGetStats(t, 200)
		require.NoError(t, err)
		latestRound := minerStats.LastFinalizedRound
		time.Sleep(5 * time.Second)
		data, resp, err = endpoint(t, blobberId, &model.ZboxGraphRequest{From: strconv.FormatInt(latestRound-int64(20), 10), To: strconv.FormatInt(latestRound, 10), DataPoints: "10"})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, 10, len([]int64(*data)))
	}
}

func printBlobbers(t *test.SystemTest, tag string, blobbers []*model.SCRestGetBlobberResponse) {
	t.Logf("%v: \n", tag)
	for _, blobber := range blobbers {
		t.Logf("ID: %s", blobber.ID)
		t.Logf("URL: %s", blobber.BaseURL)
		t.Logf("ReadPrice: %+v", blobber.Terms.ReadPrice)
		t.Logf("WritePrice: %+v", blobber.Terms.WritePrice)
		t.Logf("MinLockDemand: %+v", blobber.Terms.MinLockDemand)
		//t.Logf("MaxOfferDuration: %+v", blobber.Terms.MaxOfferDuration)
		t.Logf("Capacity: %+v", blobber.Capacity)
		t.Logf("Allocated: %+v", blobber.Allocated)
		t.Logf("LastHealthCheck: %+v", blobber.LastHealthCheck)

		t.Logf("TotalStake: %+v", blobber.TotalStake)
		t.Logf("DelegateWallet: %+v", blobber.StakePoolSettings.DelegateWallet)
		t.Logf("MinStake: %+v", blobber.StakePoolSettings.MinStake)
		t.Logf("MaxStake: %+v", blobber.StakePoolSettings.MaxStake)
		t.Logf("NumDelegates: %+v", blobber.StakePoolSettings.NumDelegates)
		t.Logf("ServiceCharge: %+v", blobber.StakePoolSettings.ServiceCharge)
		t.Logf("----------------------------------")
	}
}

func calculateExpectedAvgWritePrice(blobbers []*model.SCRestGetBlobberResponse) (expectedAvgWritePrice int64) {
	var totalWritePrice int64

	totalStakedStorage := int64(0)
	stakedStorage := make([]int64, 0, len(blobbers))
	for _, blobber := range blobbers {
		ss := (float64(blobber.TotalStake) / float64(blobber.Terms.WritePrice)) * model.GB
		stakedStorage = append(stakedStorage, int64(ss))
		totalStakedStorage += int64(ss)
	}

	for i, blobber := range blobbers {
		totalWritePrice += int64((float64(stakedStorage[i]) / float64(totalStakedStorage)) * float64(blobber.Terms.WritePrice))
	}
	return totalWritePrice
}

func calculateExpectedAllocated(blobbers []*model.SCRestGetBlobberResponse) int64 {
	var totalAllocatedData int64

	for _, blobber := range blobbers {
		totalAllocatedData += blobber.Allocated
	}
	return totalAllocatedData
}

func calculateExpectedSavedData(blobbers []*model.SCRestGetBlobberResponse) int64 {
	var totalSavedData int64

	for _, blobber := range blobbers {
		totalSavedData += blobber.SavedData
	}
	return totalSavedData
}

func calculateCapacity(blobbers []*model.SCRestGetBlobberResponse) int64 {
	var totalCapacity int64

	for _, blobber := range blobbers {
		totalCapacity += blobber.Capacity
	}
	return totalCapacity
}

func unstakeBlobber(t *test.SystemTest, wallet *model.Wallet, blobberId string) func() {
	confHash := apiClient.UnlockStakePool(t, wallet, 3, blobberId, client.TxSuccessfulStatus)
	require.NotEmpty(t, confHash)
	return func() {
		// Re-stake the blobber
		confHash := apiClient.CreateStakePool(t, wallet, 3, blobberId, float64(1.0), client.TxSuccessfulStatus)
		require.NotEmpty(t, confHash)
	}
}

// getClientStakeForSSCProvider returns the stake of the client for the given Storage Smart Contract provider (Blobber/Validator)
func getClientStakeForSSCProvider(t *test.SystemTest, wallet *model.Wallet, providerId string) int64 {
	stake, resp, err := apiClient.V1SCRestGetUserStakePoolStat(t, model.SCRestGetUserStakePoolStatRequest{
		ClientId: wallet.Id,
	}, client.HttpOkStatus)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
	require.NotEmpty(t, stake)

	providerStake := (*stake.Pools[providerId])[0].Balance
	return providerStake
}
