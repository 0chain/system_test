package api_tests

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/google/uuid"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestRegisterBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// write a test case to register a blobber with storage version
	t.RunSequentiallyWithTimeout("Register blobber with storage version", 20*time.Minute, func(t *test.SystemTest) {
		// Use a fresh wallet (not from the pre-initialized pool) because the SC uses the
		// wallet's client ID as the blobber ID. Re-using pool wallets across runs causes
		// "blobber already exists" since the previous run's blobber persists on-chain.
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.CreateWalletForMnemonic(t, mnemonic)
		// 5 ZCN is sufficient for the registration tx fee; avoid 15 faucet calls which
		// adds 15+ minutes on chains with high DKG restart counts.
		apiClient.FundWallet(t, wallet, 5.0, client.TxSuccessfulStatus)

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

		apiClient.RegisterBlobber(t, wallet, sn, 1, "", false)

		// Immediately kill the fake blobber so it doesn't get selected for allocations
		// in concurrently-running tests (e.g. TestObjectTree). Without this cleanup the
		// fake blobber stays in the active pool for ~1 hour (health_check_period) and
		// can be assigned to real allocations whose subsequent blobber API calls fail.
		killBlobber(t, wallet.Id)
	})

	t.RunSequentiallyWithTimeout("Write price lower than min_write_price should not allow register", 10*time.Minute, func(t *test.SystemTest) {
		// min_write_price is 0.001 ZCN on the test chain. A write price below that
		// should be rejected by the SC with "write_price is less than min_write_price".
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.CreateWalletForMnemonic(t, mnemonic)
		apiClient.FundWallet(t, wallet, 5.0, client.TxSuccessfulStatus)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		// 0.0001 ZCN — below min_write_price (0.001 ZCN)
		sn.Terms.ReadPrice = 0
		sn.Terms.WritePrice = 1000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.2
		sn.ManagingWallet = wallet.Id

		apiClient.RegisterBlobber(t, wallet, sn, 2, "write_price is less than min_write_price", false)
	})

	t.RunSequentiallyWithTimeout("Write price higher than max_write_price should not allow register", 10*time.Minute, func(t *test.SystemTest) {
		// Use fresh random wallets (not pool wallets) to avoid "blobber already exists" on
		// re-runs: the SC uses the wallet's client_id as the blobber ID, so reusing a pool
		// wallet that previously registered (even in a timed-out tx) causes cross-run failures.
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.CreateWalletForMnemonic(t, mnemonic)
		apiClient.FundWallet(t, wallet, 5.0, client.TxSuccessfulStatus)

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

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: write_price is greater than max_write_price allowed", false)
	})

	t.RunSequentiallyWithTimeout("Read price higher than max_read_price should not allow register", 10*time.Minute, func(t *test.SystemTest) {
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.CreateWalletForMnemonic(t, mnemonic)
		apiClient.FundWallet(t, wallet, 5.0, client.TxSuccessfulStatus)

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

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: read_price is greater than max_read_price allowed", false)
	})

	t.RunSequentiallyWithTimeout("Service charge higher than max_service_charge should not allow register", 10*time.Minute, func(t *test.SystemTest) {
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.CreateWalletForMnemonic(t, mnemonic)
		apiClient.FundWallet(t, wallet, 5.0, client.TxSuccessfulStatus)

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		t.Logf("wallet balance: %v", wallet)
		wallet.Nonce = int(walletBalance.Nonce)

		sn := &model.StorageNode{}
		sn.ID = uuid.New().String()
		sn.BaseURL = generateRandomURL()
		sn.Capacity = 10240 * GB

		// Use read_price = 0 to ensure it's within valid range (max_read_price is 0.0 in config)
		// This allows the service_charge validation to be tested without triggering read_price validation first
		sn.Terms.ReadPrice = 0
		sn.Terms.WritePrice = 1000000000

		sn.StakePoolSettings.DelegateWallet = "config.Configuration.DelegateWallet"
		sn.StakePoolSettings.NumDelegates = 2
		sn.StakePoolSettings.ServiceCharge = 0.65

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: creating stake pool: invalid stake_pool settings: service_charge (0.650000) is greater than max allowed by SC (0.500000)", false)
	})

	t.RunSequentiallyWithTimeout("Capacity lower than min_blobber_capacity should not allow register", 10*time.Minute, func(t *test.SystemTest) {
		mnemonic := crypto.GenerateMnemonics(t)
		wallet := apiClient.CreateWalletForMnemonic(t, mnemonic)
		apiClient.FundWallet(t, wallet, 5.0, client.TxSuccessfulStatus)

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

		apiClient.RegisterBlobber(t, wallet, sn, 2, "add_or_update_blobber_failed: invalid blobber params: insufficient blobber capacity", false)
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

func killBlobber(t *test.SystemTest, providerId string) {
	var killBlobberReq = &model.KillBlobberRequest{
		ProviderID: providerId,
	}

	scWallet := initialiseSCWallet()

	// get wallet balance
	walletBalance := apiClient.GetWalletBalance(t, scWallet, client.HttpOkStatus)
	scWallet.Nonce = int(walletBalance.Nonce)

	// Best-effort cleanup: use non-fatal variant so kill-blobber timeout doesn't fail the test.
	// "execution consensus is not reached" is a transient chain error; the registration already succeeded.
	apiClient.KillBlobberNonFatal(t, scWallet, killBlobberReq)
}
