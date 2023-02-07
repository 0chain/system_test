package api_tests

import (
	"fmt"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestNftInfoState(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	t.RunSequentially("Posting Allocation with valid form-data should work", func(t *test.SystemTest) {
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
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, zboxWallet)
		require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

		CreateAllocation, response, err := zboxClient.CreateAllocation(t,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode())
		require.NotNil(t, CreateAllocation)
		fmt.Println(string(response.Body()))
	})

	// t.RunSequentially("Posting NFT Info with valid form-data should work", func(t *test.SystemTest) {
	// 	teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
	// 	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	// 	PostNftInfo, response, err := zboxClient.PostNftInfo(t,
	// 		firebaseToken.IdToken,
	// 		csrfToken,
	// 		zboxClient.DefaultPhoneNumber,
	// 	)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 201, response.StatusCode())
	// 	require.NotNil(t, PostNftInfo)
	// })

	// t.RunSequentially("Putting NFT Info with valid form-data should work", func(t *test.SystemTest) {
	// 	teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
	// 	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	// 	PutNftInfo, response, err := zboxClient.PutNftInfo(t,
	// 		firebaseToken.IdToken,
	// 		csrfToken,
	// 		zboxClient.DefaultPhoneNumber,
	// 	)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 201, response.StatusCode())
	// 	require.NotNil(t, PutNftInfo)
	// })

	// t.RunSequentially("Putting NFT State with valid form-data should work", func(t *test.SystemTest) {
	// 	teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
	// 	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	// 	PutNftState, response, err := zboxClient.PutNftState(t,
	// 		firebaseToken.IdToken,
	// 		csrfToken,
	// 		zboxClient.DefaultPhoneNumber,
	// 	)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 201, response.StatusCode())
	// 	require.NotNil(t, PutNftState)
	// })

	// t.RunSequentially("Getting NFT State with valid form-data should work", func(t *test.SystemTest) {
	// 	teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
	// 	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	// 	response, err := zboxClient.GetNftState(t,
	// 		firebaseToken.IdToken,
	// 		csrfToken,
	// 		zboxClient.DefaultPhoneNumber,
	// 	)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode())
	// })
}
