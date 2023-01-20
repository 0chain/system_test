package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxAllocation(testSetup *testing.T) {
	// todo: These tests are sequential and start with teardown as they all share a common phone number
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	firebaseToken := authenticateWithFirebase(t, zboxClient.DefaultPhoneNumber)

	// t.RunSequentially("Create an allocation with zero allocation should work", func(t *test.SystemTest) {
	// 	teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
	// 	time.Sleep(1 * time.Second)
	// 	csrfToken := createCsrfToken(t, zboxClient.DefaultPhoneNumber)
	// 	description := "wallet created as part of " + t.Name()
	// 	walletName := "wallet_name"
	// 	zboxWallet, response, err := zboxClient.PostWallet(t,
	// 		zboxClient.DefaultMnemonic,
	// 		walletName,
	// 		description,
	// 		firebaseToken.IdToken,
	// 		csrfToken,
	// 		zboxClient.DefaultPhoneNumber,
	// 	)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	// 	require.NotNil(t, zboxWallet)
	// 	require.Equal(t, walletName, zboxWallet.Name, "Wallet name does not match expected")

	// 	allocationList, response, err := zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
	// 	require.NoError(t, err)
	// 	require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	// 	require.Nil(t, allocationList)
	// })

	t.RunSequentially("Create an allocation with valid phone number should work", func(t *test.SystemTest) {
		teardown(t, firebaseToken.IdToken, zboxClient.DefaultPhoneNumber)
		time.Sleep(1 * time.Second)
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

		allocationName := "allocation created as part of " + t.Name()
		allocationObjCreatedResponse, response, err := zboxClient.PostAllocation(t,
			zboxClient.DefaultAllocationId,
			allocationName,
			firebaseToken.IdToken,
			csrfToken,
			zboxClient.DefaultPhoneNumber,
		)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, allocationObjCreatedResponse)

		allocationList, response, err := zboxClient.ListAllocation(t, firebaseToken.IdToken, csrfToken, zboxClient.DefaultPhoneNumber)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotNil(t, allocationList)
		require.Equal(t, 1, len(allocationList), "Response status code does not match expected. Output: [%v]", response.String())
	})

}
