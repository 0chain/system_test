package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func NewTestNFTCollection() map[string]string {
	return map[string]string{
		"allocation_id": "165f0f8e557c430929784035df7eeacf7a3ff795f10d76c8707409bba31cb617",
		"auth_ticket": "eyJjbGllbnRfaWQiOiIiLCJvd25lcl9pZCI6IjMxZjc0MGZiMTJjZjcyNDY0NDE5YTdlODYwNTkxMDU4YTI0OGIwMWUzNGIxM2NiZjcxZDVhMTA3YjdiZGMxZTkiLCJhbGxvY2F0aW9uX2lkIjoiZTBjMmNkMmQ1ZmFhYWQxM2ZjNTM3MzNkZDc1OTc0OWYyYjJmMDFhZjQ2Mz" +
			"MyMDA5YzY3ODIyMWEyYzQ4ODE1MyIsImZpbGVfcGF0aF9oYXNoIjoiZTcyNGEyMjAxZTIyNjUzZDMyMTY3ZmNhMWJmMTJiMmU0NGJhYzYzMzdkM2ViZGI3NDI3ZmJhNGVlY2FhNGM5ZCIsImFjdHVhbF9maWxlX2hhc2giOiIxZjExMjA4M2YyNDA1YzM5NWRlNTFiN2YxM2Y5Zjc5NWFhMTQxYzQwZjFkNDdkNzhjODNhNDk5MzBmMmI5YTM0IiwiZmlsZV9uYW1lIjoiSU1HXzQ4NzQuUE5HIiwicmVmZXJlbmNlX3R5cGUiOiJmIiwiZXhwaXJhdGlvbiI6MCwidGltZXN0YW1wIjoxNjY3MjE4MjcwLCJlbmNyeXB0ZWQiOmZhbHNlLCJzaWduYXR1cmUiOiIzMzllNTUyOTliNDhlMjI5ZGRlOTAyZjhjOTY1ZDE1YTk0MGIyNzc3YzVkOTMyN2E0Yzc5MTMxYjhhNzcxZTA3In0=",
		"collection_id":   "165f0f8e557c430929784035df7eeacf7a3ff795f10d76c8707409bba31cb617",
		"created_by":      client.X_APP_CLIENT_ID,
		"collection_name": "test_nft_collection",
	}
}

func NewTestNFT() map[string]string {
	return map[string]string{
		"allocation_id": "165f0f8e557c430929784035df7eeacf7a3ff795f10d76c8707409bba31cb617",
		"auth_ticket": "eyJjbGllbnRfaWQiOiIiLCJvd25lcl9pZCI6IjMxZjc0MGZiMTJjZjcyNDY0NDE5YTdlODYwNTkxMDU4YTI0OGIwMWUzNGIxM2NiZjcxZDVhMTA3YjdiZGMxZTkiLCJhbGxvY2F0aW9uX2lkIjoiZTBjMmNkMmQ1ZmFhYWQxM2ZjNTM3MzNkZDc1OTc0OWYyYjJmMDFhZjQ2Mz" +
			"MyMDA5YzY3ODIyMWEyYzQ4ODE1MyIsImZpbGVfcGF0aF9oYXNoIjoiZTcyNGEyMjAxZTIyNjUzZDMyMTY3ZmNhMWJmMTJiMmU0NGJhYzYzMzdkM2ViZGI3NDI3ZmJhNGVlY2FhNGM5ZCIsImFjdHVhbF9maWxlX2hhc2giOiIxZjExMjA4M2YyNDA1YzM5NWRlNTFiN2YxM2Y5Zjc5NWFhMTQxYzQwZjFkNDdkNzhjODNhNDk5MzBmMmI5YTM0IiwiZmlsZV9uYW1lIjoiSU1HXzQ4NzQuUE5HIiwicmVmZXJlbmNlX3R5cGUiOiJmIiwiZXhwaXJhdGlvbiI6MCwidGltZXN0YW1wIjoxNjY3MjE4MjcwLCJlbmNyeXB0ZWQiOmZhbHNlLCJzaWduYXR1cmUiOiIzMzllNTUyOTliNDhlMjI5ZGRlOTAyZjhjOTY1ZDE1YTk0MGIyNzc3YzVkOTMyN2E0Yzc5MTMxYjhhNzcxZTA3In0=",
		"collection_id":    "165f0f8e557c430929784035df7eeacf7a3ff795f10d76c8707409bba31cb617",
		"owned_by":         client.X_APP_CLIENT_ID,
		"stage":            "deploy_contract",
		"reference":        "test_reference",
		"nft_activity":     "test_nft_activity",
		"meta_data":        "test_nft_metadata",
		"created_by":       client.X_APP_CLIENT_ID,
		"collection_name":  "test_nft_collection",
		"contract_address": "165f0f8e557c430929784035df7eeacf7a3ff795f10d76c8707409bba31cb617",
		"token_id":         "test_token_id",
		"token_standard":   "test_token_standard",
		"tx_hash":          "165f0f8e557c430929784035df7eeacf7a3ff795f10d76c8707409bba31cb617",
	}
}

func Test0BoxNFTCollection(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("List nft collections with zero nft collections should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		nftCollectionList, response, err := zboxClient.GetNftCollections(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, int64(0), nftCollectionList.NftCollectionCount)
	})

	t.RunSequentially("List nft collections with nft collections should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestAllocation(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		nftCollectionData := NewTestNFTCollection()
		_, response, err := zboxClient.CreateNftCollection(t, headers, nftCollectionData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		nftCollectionList, response, err := zboxClient.GetNftCollections(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, int64(1), nftCollectionList.NftCollectionCount)
	})

	t.RunSequentially("update nft collection with collection present should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestAllocation(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		nftCollectionData := NewTestNFTCollection()
		_, response, err := zboxClient.CreateNftCollection(t, headers, nftCollectionData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		nftCollectionData["collection_name"] = "new_collection_name"
		_, response, err = zboxClient.UpdateNftCollection(t, headers, nftCollectionData)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		nftCollectionList, response, err := zboxClient.GetNftCollections(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "new_collection_name", nftCollectionList.ZboxNftCollection[0].CollectionName)
	})

	t.RunSequentially("update nft collection with no collection present should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestAllocation(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		nftCollectionData := NewTestNFTCollection()
		nftCollectionData["collection_name"] = "new_collection_name"
		updateResponse, response, err := zboxClient.UpdateNftCollection(t, headers, nftCollectionData)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "no nft collection was updated for these details", updateResponse.Message)
	})
}

func Test0BoxNFT(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("List nfts with zero nfts should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		nftList, response, err := zboxClient.GetAllNfts(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, int64(0), nftList.NftCount)
	})

	t.RunSequentially("List nfts with nfts should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestAllocation(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		nftCollectionData := NewTestNFTCollection()
		_, response, err := zboxClient.CreateNftCollection(t, headers, nftCollectionData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		nftData := NewTestNFT()
		_, response, err = zboxClient.CreateNft(t, headers, nftData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		nftList, response, err := zboxClient.GetAllNfts(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, int64(1), nftList.NftCount)
	})

	t.RunSequentially("update nft with nft present should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestAllocation(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		nftCollectionData := NewTestNFTCollection()
		_, response, err := zboxClient.CreateNftCollection(t, headers, nftCollectionData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		nftData := NewTestNFT()
		nft, response, err := zboxClient.CreateNft(t, headers, nftData)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		nftData["stage"] = "mint_nft"
		_, response, err = zboxClient.UpdateNft(t, headers, nftData, nft.Id)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		nftList, response, err := zboxClient.GetAllNfts(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, int64(1), nftList.NftCount)
		require.Equal(t, "mint_nft", nftList.NftList[0].Stage)
	})

	t.RunSequentially("update nft with no nft present should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		// Refresh CSRF token after teardown to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		err := Create0boxTestAllocation(t, headers)
		require.NoError(t, err)

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		nftData := NewTestNFT()
		nftData["stage"] = "mint_nft"
		_, response, err := zboxClient.UpdateNft(t, headers, nftData, 1)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}
