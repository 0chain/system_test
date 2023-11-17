package api_tests

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/google/uuid"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestRegisterBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Write price lower than min_write_price should not allow register", func(t *test.SystemTest) {
		sdkWalletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		t.Logf("sdkWallet balance: %v", sdkWallet)
		sdkWallet.Nonce = int(sdkWalletBalance.Nonce)

		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 9.0, client.TxSuccessfulStatus)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10 * GB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 1

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, sdkWallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: write_price is less than min_write_price allowed")
	})

	t.RunSequentially("Write price higher than max_write_price should not allow register", func(t *test.SystemTest) {
		sdkWalletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		t.Logf("sdkWallet balance: %v", sdkWallet)
		sdkWallet.Nonce = int(sdkWalletBalance.Nonce)

		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 9.0, client.TxSuccessfulStatus)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 100000000000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, sdkWallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: write_price is greater than max_write_price allowed")
	})

	t.RunSequentially("Read price higher than max_read_price should not allow register", func(t *test.SystemTest) {
		sdkWalletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		t.Logf("sdkWallet balance: %v", sdkWallet)
		sdkWallet.Nonce = int(sdkWalletBalance.Nonce)

		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 9.0, client.TxSuccessfulStatus)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		sn.Terms.ReadPrice = 100000000000000000
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, sdkWallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: read_price is greater than max_read_price allowed")
	})

	t.RunSequentially("Service charge higher than max_service_charge should not allow register", func(t *test.SystemTest) {
		sdkWalletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		t.Logf("sdkWallet balance: %v", sdkWallet)
		sdkWallet.Nonce = int(sdkWalletBalance.Nonce)

		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 9.0, client.TxSuccessfulStatus)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.6

		apiClient.RegisterBlobber(t, sdkWallet, sn, 2, "add_or_update_blobber_failed: creating stake pool: invalid stake_pool settings: service_charge (0.600000) is greater than max allowed by SC (0.500000)")
	})

	t.RunSequentially("Capacity lower than min_blobber_capacity should not allow register", func(t *test.SystemTest) {
		sdkWalletBalance := apiClient.GetWalletBalance(t, sdkWallet, client.HttpOkStatus)
		t.Logf("sdkWallet balance: %v", sdkWallet)
		sdkWallet.Nonce = int(sdkWalletBalance.Nonce)

		apiClient.ExecuteFaucetWithTokens(t, sdkWallet, 9.0, client.TxSuccessfulStatus)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 1 * MB

		sn.Terms.ReadPrice = 1000000000
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2

		apiClient.RegisterBlobber(t, sdkWallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: insufficient blobber capacity")
	})
}

func generateRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[random.Intn(len(charset))]
	}
	return string(result)
}

func generateRandomURL() string {
	return fmt.Sprintf("http://%s.com/%s", generateRandomString(10), generateRandomString(8))
}
