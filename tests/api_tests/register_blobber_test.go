package api_tests

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/google/uuid"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestRegisterBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	// write a test case to register a blobber with storage version
	t.Run("Register blobber with storage version", func(t *test.SystemTest) {
		wallet := createWallet(t)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}

		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()

		sn.Capacity = 10240 * GB
		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		//todo: make check to this
		sn.StorageVersion = 2
		sn.ManagingWallet = wallet.Id

		apiClient.RegisterBlobberWithIdVerification(t, wallet, sn, 1, wallet.Id)

		var killBlobberReq = &model.KillBlobberRequest{
			ProviderID: wallet.Id,
		}

		scWallet := initialiseSCWallet()

		// get wallet balance
		walletBalance = apiClient.GetWalletBalance(t, scWallet, client.HttpOkStatus)
		scWallet.Nonce = int(walletBalance.Nonce)

		// todo: check logic
		apiClient.KillBlobber(t, scWallet, killBlobberReq, 1)
	})

	t.Run("Write price lower than min_write_price should not allow register", func(t *test.SystemTest) {
		wallet := createWallet(t)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10 * GB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 1

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: write_price is less than min_write_price allowed")
	})

	t.Run("Write price higher than max_write_price should not allow register", func(t *test.SystemTest) {
		wallet := createWallet(t)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 100000000000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: write_price is greater than max_write_price allowed")
	})

	t.Run("Read price higher than max_read_price should not allow register", func(t *test.SystemTest) {
		wallet := createWallet(t)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		sn.Terms.ReadPrice = 100000000000000000
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: read_price is greater than max_read_price allowed")
	})

	t.Run("Service charge higher than max_service_charge should not allow register", func(t *test.SystemTest) {
		wallet := createWallet(t)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.6

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: creating stake pool: invalid stake_pool settings: service_charge (0.600000) is greater than max allowed by SC (0.500000)")
	})

	t.Run("Capacity lower than min_blobber_capacity should not allow register", func(t *test.SystemTest) {
		wallet := createWallet(t)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 1 * MB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: insufficient blobber capacity")
	})
}

func generateRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	result := make([]byte, length)
	for i := range result {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		result[i] = charset[randomIndex.Int64()]
	}
	return string(result)
}

func generateRandomURL() string {
	return fmt.Sprintf("http://%s.com/%s", generateRandomString(10), generateRandomString(8))
}
